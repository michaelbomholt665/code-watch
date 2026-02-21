// # internal/resolver/resolver.go
package resolver

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/shared/observability"
	"context"
	"io"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type BridgeResolutionConfig struct {
	ConfirmedThreshold int
	ProbableThreshold  int
	Weights            BridgeScoreWeights
}

type BridgeScoreWeights struct {
	ExplicitRuleMatch       int
	BridgeContext           int
	BridgeImportEvidence    int
	UniqueCrossLangMatch    int
	AmbiguousCrossLangMatch int
	LocalOrModuleConflict   int
	StdlibConflict          int
}

type ProbableBridgeReference struct {
	Reference  parser.Reference
	File       string
	Score      int
	Confidence string
	Reasons    []string
}

type referenceResolution uint8

const (
	referenceUnresolved referenceResolution = iota
	referenceResolved
	referenceProbableBridge
)

type resolutionResult struct {
	status referenceResolution
	bridge bridgeAssessment
}

type bridgeAssessment struct {
	score      int
	confidence string
	reasons    []string
}

type UnresolvedReference struct {
	Reference parser.Reference
	File      string
}

type UnusedImport struct {
	File       string
	Language   string
	Module     string
	Alias      string
	Item       string
	Location   parser.Location
	Confidence string
}

type Resolver struct {
	graph            *graph.Graph
	symbolTable      graph.SymbolLookupTable
	stdlibByLanguage map[string]map[string]bool
	excludedSymbols  []string
	excludedImports  []string
	explicitBridges  []ExplicitBridge
	bridgeConfig     BridgeResolutionConfig
	closer           io.Closer
}

func NewResolver(g *graph.Graph, excludedSymbols, excludedImports []string) *Resolver {
	var symbolTable graph.SymbolLookupTable
	if g != nil {
		symbolTable = g.BuildUniversalSymbolTable()
	}
	return newResolver(g, excludedSymbols, excludedImports, symbolTable, nil)
}

func NewResolverWithSQLite(g *graph.Graph, excludedSymbols, excludedImports []string, dbPath, projectKey string) *Resolver {
	store, err := graph.OpenSQLiteSymbolStore(dbPath, projectKey)
	if err != nil {
		return NewResolver(g, excludedSymbols, excludedImports)
	}
	if err := store.SyncFromGraph(g); err != nil {
		_ = store.Close()
		return NewResolver(g, excludedSymbols, excludedImports)
	}
	return newResolver(g, excludedSymbols, excludedImports, store, store)
}

func NewResolverWithSymbolLookup(g *graph.Graph, excludedSymbols, excludedImports []string, symbolTable graph.SymbolLookupTable) *Resolver {
	return newResolver(g, excludedSymbols, excludedImports, symbolTable, nil)
}

func newResolver(g *graph.Graph, excludedSymbols, excludedImports []string, symbolTable graph.SymbolLookupTable, closer io.Closer) *Resolver {
	if symbolTable == nil && g != nil {
		symbolTable = g.BuildUniversalSymbolTable()
	}
	return &Resolver{
		graph:            g,
		symbolTable:      symbolTable,
		stdlibByLanguage: getStdlibByLanguage(),
		excludedSymbols:  excludedSymbols,
		excludedImports:  excludedImports,
		bridgeConfig:     defaultBridgeResolutionConfig(),
		closer:           closer,
	}
}

func (r *Resolver) Close() error {
	if r == nil || r.closer == nil {
		return nil
	}
	return r.closer.Close()
}

func (r *Resolver) resolveReference(ctx context.Context, file *parser.File, ref parser.Reference) bool {
	result := r.resolveReferenceResult(ctx, file, ref)
	return result.status == referenceResolved
}

func (r *Resolver) resolveReferenceResult(ctx context.Context, file *parser.File, ref parser.Reference) resolutionResult {
	_, span := observability.Tracer.Start(ctx, "Resolver.resolveReferenceResult", trace.WithAttributes(
		attribute.String("symbol", ref.Name),
		attribute.String("file", file.Path),
	))
	defer span.End()

	// 0. Check local symbols (vars, params, etc)
	if r.isLocalSymbol(file, ref.Name) {
		return resolutionResult{status: referenceResolved}
	}

	// 0.5 Explicit bridge mappings loaded from .circular-bridge.toml.
	if r.resolveExplicitBridgeReference(file, ref) {
		return resolutionResult{
			status: referenceResolved,
			bridge: bridgeAssessment{
				score:      r.bridgeConfig.ConfirmedThreshold,
				confidence: "high",
				reasons:    []string{"explicit_bridge_rule"},
			},
		}
	}

	bridge := r.assessBridgeReference(file, ref)

	// 1. Check stdlib
	if r.isStdlibSymbol(file.Language, ref.Name) || r.isStdlibCall(file.Language, ref.Name) {
		return resolutionResult{status: referenceResolved}
	}

	// 2. Check local module and imports
	if r.resolveQualifiedReference(file, ref) {
		return resolutionResult{status: referenceResolved}
	}

	// 4. Check builtins
	if file.Language == "python" && pythonBuiltins[ref.Name] {
		return resolutionResult{status: referenceResolved}
	}
	if file.Language == "go" && goBuiltins[ref.Name] {
		return resolutionResult{status: referenceResolved}
	}

	// 5. Multi-pass cross-language probabilistic resolution.
	if r.resolveProbabilisticReference(file, ref) {
		return resolutionResult{status: referenceResolved}
	}

	if bridge.confidence == "high" {
		return resolutionResult{status: referenceResolved, bridge: bridge}
	}
	if bridge.confidence == "medium" {
		return resolutionResult{status: referenceProbableBridge, bridge: bridge}
	}

	return resolutionResult{status: referenceUnresolved, bridge: bridge}
}

func (r *Resolver) WithBridgeResolutionConfig(cfg BridgeResolutionConfig) *Resolver {
	if r == nil {
		return nil
	}
	if cfg.ConfirmedThreshold <= 0 || cfg.ProbableThreshold <= 0 || cfg.ProbableThreshold > cfg.ConfirmedThreshold {
		r.bridgeConfig = defaultBridgeResolutionConfig()
		return r
	}
	if cfg.Weights == (BridgeScoreWeights{}) {
		cfg.Weights = defaultBridgeScoreWeights()
	}
	r.bridgeConfig = cfg
	return r
}

func (r *Resolver) isLocalSymbol(file *parser.File, name string) bool {
	for _, sym := range file.LocalSymbols {
		if sym == name {
			return true
		}
	}

	// Split by dot to handle p.RegisterExtractor -> check if 'p' is local
	parts := strings.Split(name, ".")
	prefix := parts[0]
	if idx := strings.Index(prefix, "["); idx >= 0 {
		prefix = prefix[:idx]
	}
	prefix = strings.TrimLeft(prefix, "*&")

	for _, sym := range file.LocalSymbols {
		if sym == prefix {
			return true
		}
	}

	if IsKnownNonModule(name, r.excludedSymbols) {
		return true
	}

	// Also handle 'self' (Python) and 'this' (Go - though receivers are explicitly named in Go)
	if file.Language == "python" && prefix == "self" {
		return true
	}

	return false
}

func (r *Resolver) checkModule(moduleName, symbolName string, allowUnexported bool) bool {
	defs, ok := r.graph.GetDefinitions(moduleName)
	if !ok {
		return false
	}

	// Direct match
	if def, ok := defs[symbolName]; ok {
		if allowUnexported || def.Exported {
			return true
		}
	}

	// Nested: Class.method or package.Type
	for fullName, def := range defs {
		if !allowUnexported && !def.Exported {
			continue
		}
		if strings.HasPrefix(fullName, symbolName+".") ||
			strings.HasSuffix(fullName, "."+symbolName) {
			return true
		}
	}

	return false
}

func (r *Resolver) isStdlibCall(language, name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) == 0 {
		return false
	}
	return r.isStdlibSymbol(language, parts[0])
}

func (r *Resolver) isStdlibSymbol(language, name string) bool {
	langStdlib, ok := r.stdlibByLanguage[language]
	if !ok {
		return false
	}
	return langStdlib[name]
}
