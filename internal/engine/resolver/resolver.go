// # internal/resolver/resolver.go
package resolver

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"strings"
)

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
	symbolTable      *graph.UniversalSymbolTable
	stdlibByLanguage map[string]map[string]bool
	excludedSymbols  []string
	excludedImports  []string
}

func NewResolver(g *graph.Graph, excludedSymbols, excludedImports []string) *Resolver {
	return &Resolver{
		graph:            g,
		symbolTable:      g.BuildUniversalSymbolTable(),
		stdlibByLanguage: getStdlibByLanguage(),
		excludedSymbols:  excludedSymbols,
		excludedImports:  excludedImports,
	}
}

func (r *Resolver) resolveReference(file *parser.File, ref parser.Reference) bool {
	// 0. Check local symbols (vars, params, etc)
	if r.isLocalSymbol(file, ref.Name) {
		return true
	}

	// 0.5 Cross-language bridge hints (FFI/process/service calls).
	if IsCrossLanguageBridgeReference(file.Language, ref) {
		return true
	}

	// 1. Check stdlib
	if r.isStdlibSymbol(file.Language, ref.Name) || r.isStdlibCall(file.Language, ref.Name) {
		return true
	}

	// 2. Check local module and imports
	if r.resolveQualifiedReference(file, ref) {
		return true
	}

	// 4. Check builtins
	if file.Language == "python" && pythonBuiltins[ref.Name] {
		return true
	}
	if file.Language == "go" && goBuiltins[ref.Name] {
		return true
	}

	// 5. Multi-pass cross-language probabilistic resolution.
	if r.resolveProbabilisticReference(file, ref) {
		return true
	}

	return false
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
