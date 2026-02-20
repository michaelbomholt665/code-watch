package app

import (
	"circular/internal/core/app/helpers"
	"circular/internal/engine/resolver"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

func (a *App) HandleChanges(paths []string) {
	slog.Info("detected changes", "count", len(paths))
	start := time.Now()
	affectedSet := make(map[string]bool)

	for _, path := range paths {
		if filepath.Base(path) == "go.mod" {
			a.goModCache = make(map[string]goModuleCacheEntry)
		}
		if !a.codeParser.IsSupportedPath(path) && filepath.Base(path) != "go.mod" {
			continue
		}

		if !a.IncludeTests && a.codeParser.IsTestFile(path) {
			continue
		}

		for _, f := range a.Graph.InvalidateTransitive(path) {
			affectedSet[f] = true
		}
		affectedSet[path] = true

		if _, err := os.Stat(path); os.IsNotExist(err) {
			a.Graph.RemoveFile(path)
			a.dropContent(path)
			if err := a.deleteSymbolStoreFile(path); err != nil {
				slog.Warn("failed to delete persisted symbol rows", "path", path, "error", err)
			}
			a.unresolvedMu.Lock()
			delete(a.unresolvedByFile, path)
			a.unresolvedMu.Unlock()
			a.unusedMu.Lock()
			delete(a.unusedByFile, path)
			a.unusedMu.Unlock()
			continue
		}

		if err := a.ProcessFile(path); err != nil {
			slog.Warn("failed to re-process file", "path", path, "error", err)
		}
	}

	cycles := a.Graph.DetectCycles()
	metrics := a.Graph.ComputeModuleMetrics()
	hotspots := a.Graph.TopComplexity(a.Config.Architecture.TopComplexity)
	violations := a.ArchitectureViolations()
	hallucinations := a.AnalyzeHallucinationsIncremental(affectedSet)
	unusedImports := a.AnalyzeUnusedImportsIncremental(affectedSet)

	if err := a.GenerateOutputs(cycles, hallucinations, unusedImports, metrics, violations, hotspots); err != nil {
		slog.Error("failed to generate outputs", "error", err)
	}

	duration := time.Since(start)
	a.PrintSummary(len(paths), a.Graph.ModuleCount(), duration, cycles, hallucinations, unusedImports, metrics, violations, hotspots)
	a.emitUpdate(Update{
		Cycles:         cycles,
		Hallucinations: hallucinations,
		ModuleCount:    a.Graph.ModuleCount(),
		FileCount:      a.Graph.FileCount(),
		SecretCount:    a.SecretCount(),
	})

	if a.Config.Alerts.Beep && (len(cycles) > 0 || len(hallucinations) > 0 || len(unusedImports) > 0 || len(violations) > 0) {
		fmt.Print("\a")
	}
}

func (a *App) newResolver() *resolver.Resolver {
	if a == nil {
		return resolver.NewResolver(nil, nil, nil)
	}

	excludedSymbols := []string(nil)
	excludedImports := []string(nil)
	if a.Config != nil {
		excludedSymbols = a.Config.Exclude.Symbols
		excludedImports = a.Config.Exclude.Imports
	}

	if a.Config == nil || !a.Config.DB.Enabled {
		res := resolver.NewResolver(a.Graph, excludedSymbols, excludedImports)
		res.WithBridgeResolutionConfig(a.resolverBridgeConfig())
		res.WithExplicitBridges(a.loadResolverBridges())
		return res
	}

	if a.symbolStore == nil {
		res := resolver.NewResolver(a.Graph, excludedSymbols, excludedImports)
		res.WithBridgeResolutionConfig(a.resolverBridgeConfig())
		res.WithExplicitBridges(a.loadResolverBridges())
		return res
	}
	res := resolver.NewResolverWithSymbolLookup(a.Graph, excludedSymbols, excludedImports, a.symbolStore)
	res.WithBridgeResolutionConfig(a.resolverBridgeConfig())
	res.WithExplicitBridges(a.loadResolverBridges())
	return res
}

func (a *App) resolverBridgeConfig() resolver.BridgeResolutionConfig {
	if a == nil || a.Config == nil {
		return resolver.BridgeResolutionConfig{}
	}
	scoring := a.Config.Resolver.BridgeScoring
	return resolver.BridgeResolutionConfig{
		ConfirmedThreshold: scoring.ConfirmedThreshold,
		ProbableThreshold:  scoring.ProbableThreshold,
		Weights: resolver.BridgeScoreWeights{
			ExplicitRuleMatch:       scoring.WeightExplicitRuleMatch,
			BridgeContext:           scoring.WeightBridgeContext,
			BridgeImportEvidence:    scoring.WeightBridgeImportEvidence,
			UniqueCrossLangMatch:    scoring.WeightUniqueCrossLangMatch,
			AmbiguousCrossLangMatch: scoring.WeightAmbiguousCrossLangMatch,
			LocalOrModuleConflict:   scoring.WeightLocalOrModuleConflict,
			StdlibConflict:          scoring.WeightStdlibConflict,
		},
	}
}

func (a *App) loadResolverBridges() []resolver.ExplicitBridge {
	if a == nil || a.Config == nil {
		return nil
	}

	paths := resolver.DiscoverBridgeConfigPaths(helpers.UniqueScanRoots(a.Config.WatchPaths))
	bridges := make([]resolver.ExplicitBridge, 0)

	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		loaded, err := resolver.LoadBridgeConfig(path)
		if err != nil {
			slog.Warn("failed to load bridge config", "path", path, "error", err)
			continue
		}
		bridges = append(bridges, loaded...)
	}

	return bridges
}

func (a *App) AnalyzeHallucinations() []resolver.UnresolvedReference {
	res := a.newResolver()
	defer func() { _ = res.Close() }()
	unresolved := res.FindUnresolved()
	a.rebuildUnresolvedCache(unresolved)
	return unresolved
}

func (a *App) AnalyzeProbableBridges() []resolver.ProbableBridgeReference {
	res := a.newResolver()
	defer func() { _ = res.Close() }()
	return res.FindProbableBridgeReferences()
}

func (a *App) AnalyzeHallucinationsIncremental(affectedSet map[string]bool) []resolver.UnresolvedReference {
	if len(affectedSet) == 0 {
		return a.cachedUnresolved()
	}

	paths := make([]string, 0, len(affectedSet))
	for path := range affectedSet {
		paths = append(paths, path)
	}

	res := a.newResolver()
	defer func() { _ = res.Close() }()
	updated := res.FindUnresolvedForPaths(paths)

	a.unresolvedMu.Lock()
	for _, path := range paths {
		if _, ok := a.Graph.GetFile(path); ok {
			a.unresolvedByFile[path] = nil
		} else {
			delete(a.unresolvedByFile, path)
		}
	}

	for _, u := range updated {
		a.unresolvedByFile[u.File] = append(a.unresolvedByFile[u.File], u)
	}
	a.unresolvedMu.Unlock()

	return a.cachedUnresolved()
}

func (a *App) AnalyzeProbableBridgesForPaths(paths []string) []resolver.ProbableBridgeReference {
	res := a.newResolver()
	defer func() { _ = res.Close() }()
	return res.FindProbableBridgeReferencesForPaths(paths)
}

func (a *App) AnalyzeUnusedImports() []resolver.UnusedImport {
	paths := a.currentGraphPaths()
	res := a.newResolver()
	defer func() { _ = res.Close() }()
	unused := res.FindUnusedImports(paths)
	a.rebuildUnusedCache(unused)
	return unused
}

func (a *App) AnalyzeUnusedImportsIncremental(affectedSet map[string]bool) []resolver.UnusedImport {
	if len(affectedSet) == 0 {
		return a.cachedUnused()
	}

	paths := make([]string, 0, len(affectedSet))
	for path := range affectedSet {
		paths = append(paths, path)
	}

	res := a.newResolver()
	defer func() { _ = res.Close() }()
	updated := res.FindUnusedImports(paths)

	a.unusedMu.Lock()
	for _, path := range paths {
		if _, ok := a.Graph.GetFile(path); ok {
			a.unusedByFile[path] = nil
		} else {
			delete(a.unusedByFile, path)
		}
	}

	for _, u := range updated {
		a.unusedByFile[u.File] = append(a.unusedByFile[u.File], u)
	}
	a.unusedMu.Unlock()

	return a.cachedUnused()
}
