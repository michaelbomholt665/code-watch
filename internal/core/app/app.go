package app

import (
	"circular/internal/core/config"
	"circular/internal/core/watcher"
	"circular/internal/data/history"
	"circular/internal/data/query"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"circular/internal/engine/secrets"
	"circular/internal/shared/util"
	"circular/internal/shared/version"
	"circular/internal/ui/report"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gobwas/glob"
)

type Update struct {
	Cycles         [][]string
	Hallucinations []resolver.UnresolvedReference
	ModuleCount    int
	FileCount      int
	SecretCount    int
}

type MarkdownReportRequest struct {
	OutputPath string
	WriteFile  bool
	Verbosity  string
}

type MarkdownReportResult struct {
	Markdown string
	Path     string
	Written  bool
}

type App struct {
	Config       *config.Config
	Parser       *parser.Parser
	Graph        *graph.Graph
	secretEngine *secrets.Detector
	archEngine   *graph.LayerRuleEngine
	goModCache   map[string]goModuleCacheEntry
	IncludeTests bool

	secretExcludeDirs  []glob.Glob
	secretExcludeFiles []glob.Glob

	updateMu sync.RWMutex
	onUpdate func(Update)

	// Cached unresolved references keyed by file path for incremental updates.
	unresolvedByFile map[string][]resolver.UnresolvedReference
	unresolvedMu     sync.RWMutex

	// Cached unused imports keyed by file path for incremental updates.
	unusedByFile map[string][]resolver.UnusedImport
	unusedMu     sync.RWMutex
}

type goModuleCacheEntry struct {
	Found      bool
	ModuleRoot string
	ModulePath string
}

func New(cfg *config.Config) (*App, error) {
	registry, err := buildParserRegistry(cfg)
	if err != nil {
		return nil, err
	}
	loader, err := parser.NewGrammarLoaderWithRegistry(cfg.GrammarsPath, registry, cfg.GrammarVerification.IsEnabled())
	if err != nil {
		return nil, err
	}

	p := parser.NewParser(loader)
	if err := p.RegisterDefaultExtractors(); err != nil {
		return nil, err
	}

	var secretEngine *secrets.Detector
	secretExcludeDirs, err := compileGlobs(cfg.Secrets.Exclude.Dirs, "secret exclude dir")
	if err != nil {
		return nil, err
	}
	secretExcludeFiles, err := compileGlobs(cfg.Secrets.Exclude.Files, "secret exclude file")
	if err != nil {
		return nil, err
	}
	if cfg.Secrets.Enabled {
		customPatterns := make([]secrets.PatternConfig, 0, len(cfg.Secrets.Patterns))
		for _, pattern := range cfg.Secrets.Patterns {
			customPatterns = append(customPatterns, secrets.PatternConfig{
				Name:     pattern.Name,
				Regex:    pattern.Regex,
				Severity: pattern.Severity,
			})
		}
		secretEngine, err = secrets.NewDetector(secrets.Config{
			EntropyThreshold: cfg.Secrets.EntropyThreshold,
			MinTokenLength:   cfg.Secrets.MinTokenLength,
			Patterns:         customPatterns,
		})
		if err != nil {
			return nil, err
		}
	}

	return &App{
		Config:             cfg,
		Parser:             p,
		Graph:              graph.NewGraph(),
		secretEngine:       secretEngine,
		archEngine:         graph.NewLayerRuleEngine(architectureModelFromConfig(cfg.Architecture)),
		goModCache:         make(map[string]goModuleCacheEntry),
		unresolvedByFile:   make(map[string][]resolver.UnresolvedReference),
		unusedByFile:       make(map[string][]resolver.UnusedImport),
		secretExcludeDirs:  secretExcludeDirs,
		secretExcludeFiles: secretExcludeFiles,
	}, nil
}

func buildParserRegistry(cfg *config.Config) (map[string]parser.LanguageSpec, error) {
	overrides := make(map[string]parser.LanguageOverride, len(cfg.Languages))
	for lang, languageCfg := range cfg.Languages {
		overrides[lang] = parser.LanguageOverride{
			Enabled:    languageCfg.Enabled,
			Extensions: append([]string(nil), languageCfg.Extensions...),
			Filenames:  append([]string(nil), languageCfg.Filenames...),
		}
	}
	return parser.BuildLanguageRegistry(overrides)
}

func compileGlobs(patterns []string, label string) ([]glob.Glob, error) {
	out := make([]glob.Glob, 0, len(patterns))
	for _, p := range patterns {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid %s pattern %q: %w", label, p, err)
		}
		out = append(out, g)
	}
	return out, nil
}

func (a *App) SetUpdateHandler(handler func(Update)) {
	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	a.onUpdate = handler
}

func (a *App) CurrentUpdate() Update {
	return Update{
		Cycles:         a.Graph.DetectCycles(),
		Hallucinations: a.AnalyzeHallucinations(),
		ModuleCount:    a.Graph.ModuleCount(),
		FileCount:      a.Graph.FileCount(),
		SecretCount:    a.SecretCount(),
	}
}

func (a *App) InitialScan() error {
	finalPaths := uniqueScanRoots(a.Config.WatchPaths)
	expandedPaths := make(map[string]bool, len(finalPaths))
	for _, p := range finalPaths {
		expandedPaths[p] = true

		r := resolver.NewGoResolver()
		if err := r.FindGoMod(p); err == nil {
			if absRoot, err := filepath.Abs(r.GetModuleRoot()); err == nil {
				expandedPaths[filepath.Clean(absRoot)] = true
			}
		}
	}

	finalPaths = make([]string, 0, len(expandedPaths))
	for p := range expandedPaths {
		finalPaths = append(finalPaths, p)
	}
	sort.Strings(finalPaths)

	files, err := a.ScanDirectories(finalPaths, a.Config.Exclude.Dirs, a.Config.Exclude.Files)
	if err != nil {
		return err
	}

	for _, filePath := range files {
		if err := a.ProcessFile(filePath); err != nil {
			slog.Warn("failed to process file", "path", filePath, "error", err)
		}
	}
	return nil
}

func uniqueScanRoots(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	roots := make([]string, 0, len(paths))
	for _, p := range paths {
		normalized := filepath.Clean(p)
		if abs, err := filepath.Abs(normalized); err == nil {
			normalized = filepath.Clean(abs)
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		roots = append(roots, normalized)
	}
	sort.Strings(roots)
	return roots
}

func (a *App) ScanDirectories(paths []string, excludeDirs, excludeFiles []string) ([]string, error) {
	var files []string

	dirGlobs := make([]glob.Glob, 0, len(excludeDirs))
	for _, p := range excludeDirs {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude dir pattern %q: %w", p, err)
		}
		dirGlobs = append(dirGlobs, g)
	}

	fileGlobs := make([]glob.Glob, 0, len(excludeFiles))
	for _, p := range excludeFiles {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude file pattern %q: %w", p, err)
		}
		fileGlobs = append(fileGlobs, g)
	}

	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			base := filepath.Base(path)
			if d.IsDir() {
				for _, g := range dirGlobs {
					if g.Match(base) {
						return filepath.SkipDir
					}
				}
				return nil
			}

			if !a.Parser.IsSupportedPath(path) {
				return nil
			}

			// Exclude test files if requested
			if !a.IncludeTests && a.Parser.IsTestFile(path) {
				return nil
			}

			for _, g := range fileGlobs {
				if g.Match(base) {
					return nil
				}
			}

			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (a *App) ProcessFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	file, err := a.Parser.ParseFile(path, content)
	if err != nil {
		return err
	}

	if file.Language == "python" {
		if len(a.Config.WatchPaths) == 0 {
			return fmt.Errorf("python resolver requires at least one watch path")
		}
		matchingPath, err := findContainingWatchPath(path, a.Config.WatchPaths)
		if err != nil {
			return err
		}
		r := resolver.NewPythonResolver(matchingPath)
		file.Module = r.GetModuleName(path)
	} else if file.Language == "go" {
		moduleName, ok, err := a.resolveGoModule(path)
		if err != nil {
			return err
		}
		if ok {
			file.Module = moduleName
		}
	}
	if a.secretEngine != nil && !a.shouldSkipSecretScan(path) {
		file.Secrets = a.secretEngine.Detect(path, content)
	}

	a.Graph.AddFile(file)
	return nil
}

func (a *App) shouldSkipSecretScan(path string) bool {
	base := filepath.Base(path)
	for _, g := range a.secretExcludeFiles {
		if g.Match(base) {
			return true
		}
	}

	dir := filepath.Dir(path)
	for {
		name := filepath.Base(dir)
		for _, g := range a.secretExcludeDirs {
			if g.Match(name) {
				return true
			}
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return false
}

func findContainingWatchPath(path string, watchPaths []string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve file path %q: %w", path, err)
	}

	for _, watchPath := range watchPaths {
		absWatchPath, err := filepath.Abs(watchPath)
		if err != nil {
			return "", fmt.Errorf("resolve watch path %q: %w", watchPath, err)
		}

		rel, err := filepath.Rel(absWatchPath, absPath)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
			return absWatchPath, nil
		}
	}

	return "", fmt.Errorf("python file %q is not under any configured watch path", path)
}

func (a *App) HandleChanges(paths []string) {
	slog.Info("detected changes", "count", len(paths))
	start := time.Now()
	affectedSet := make(map[string]bool)

	for _, path := range paths {
		if filepath.Base(path) == "go.mod" {
			a.goModCache = make(map[string]goModuleCacheEntry)
		}
		if !a.Parser.IsSupportedPath(path) && filepath.Base(path) != "go.mod" {
			continue
		}

		if !a.IncludeTests && a.Parser.IsTestFile(path) {
			continue
		}

		for _, f := range a.Graph.InvalidateTransitive(path) {
			affectedSet[f] = true
		}
		affectedSet[path] = true

		if _, err := os.Stat(path); os.IsNotExist(err) {
			a.Graph.RemoveFile(path)
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

func (a *App) emitUpdate(update Update) {
	a.updateMu.RLock()
	handler := a.onUpdate
	a.updateMu.RUnlock()
	if handler != nil {
		handler(update)
	}
}

func (a *App) AnalyzeHallucinations() []resolver.UnresolvedReference {
	res := resolver.NewResolver(a.Graph, a.Config.Exclude.Symbols, a.Config.Exclude.Imports)
	unresolved := res.FindUnresolved()
	a.rebuildUnresolvedCache(unresolved)
	return unresolved
}

func (a *App) AnalyzeHallucinationsIncremental(affectedSet map[string]bool) []resolver.UnresolvedReference {
	if len(affectedSet) == 0 {
		return a.cachedUnresolved()
	}

	paths := make([]string, 0, len(affectedSet))
	for path := range affectedSet {
		paths = append(paths, path)
	}

	res := resolver.NewResolver(a.Graph, a.Config.Exclude.Symbols, a.Config.Exclude.Imports)
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

func (a *App) AnalyzeUnusedImports() []resolver.UnusedImport {
	paths := a.currentGraphPaths()
	res := resolver.NewResolver(a.Graph, a.Config.Exclude.Symbols, a.Config.Exclude.Imports)
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

	res := resolver.NewResolver(a.Graph, a.Config.Exclude.Symbols, a.Config.Exclude.Imports)
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

func (a *App) GenerateOutputs(
	cycles [][]string,
	unresolved []resolver.UnresolvedReference,
	unusedImports []resolver.UnusedImport,
	metrics map[string]graph.ModuleMetrics,
	violations []graph.ArchitectureViolation,
	hotspots []graph.ComplexityHotspot,
) error {
	archModel := architectureModelFromConfig(a.Config.Architecture)
	diagramModes, err := resolveDiagramModes(a.Config.Output.Diagrams)
	if err != nil {
		return err
	}
	mermaidDiagrams := make(map[diagramMode]string, len(diagramModes))
	plantUMLDiagrams := make(map[diagramMode]string, len(diagramModes))
	targets, err := a.resolveOutputTargets()
	if err != nil {
		return err
	}

	if targets.DOT != "" {
		dotGen := report.NewDOTGenerator(a.Graph)
		dotGen.SetModuleMetrics(metrics)
		dotGen.SetComplexityHotspots(hotspots)
		dot, err := dotGen.Generate(cycles)
		if err != nil {
			return fmt.Errorf("generate DOT output: %w", err)
		}
		if err := writeArtifact(targets.DOT, dot); err != nil {
			return fmt.Errorf("write DOT output %q: %w", targets.DOT, err)
		}
	}

	if targets.TSV != "" {
		tsvGen := report.NewTSVGenerator(a.Graph)
		dependenciesTSV, err := tsvGen.Generate()
		if err != nil {
			return fmt.Errorf("generate TSV output: %w", err)
		}
		tsv := dependenciesTSV

		if len(unusedImports) > 0 {
			unusedTSV, err := tsvGen.GenerateUnusedImports(unusedImports)
			if err != nil {
				return fmt.Errorf("generate unused-import TSV block: %w", err)
			}
			tsv = strings.TrimRight(dependenciesTSV, "\n") + "\n\n" + strings.TrimRight(unusedTSV, "\n") + "\n"
		}
		if len(violations) > 0 {
			violationsTSV, err := tsvGen.GenerateArchitectureViolations(violations)
			if err != nil {
				return fmt.Errorf("generate architecture-violation TSV block: %w", err)
			}
			tsv = strings.TrimRight(tsv, "\n") + "\n\n" + strings.TrimRight(violationsTSV, "\n") + "\n"
		}
		allSecrets := a.allSecrets(0)
		if len(allSecrets) > 0 {
			secretsTSV, err := tsvGen.GenerateSecrets(allSecrets)
			if err != nil {
				return fmt.Errorf("generate secrets TSV block: %w", err)
			}
			tsv = strings.TrimRight(tsv, "\n") + "\n\n" + strings.TrimRight(secretsTSV, "\n") + "\n"
		}

		if err := writeArtifact(targets.TSV, tsv); err != nil {
			return fmt.Errorf("write TSV output %q: %w", targets.TSV, err)
		}
	}

	needMermaid := targets.Mermaid != "" && a.Config.Output.MermaidEnabled()
	needPlantUML := targets.PlantUML != "" && a.Config.Output.PlantUMLEnabled()
	if targets.Markdown != "" && a.Config.Output.Report.IncludeMermaidEnabled() && a.Config.Output.MermaidEnabled() {
		needMermaid = true
	}
	for _, injection := range a.Config.Output.UpdateMarkdown {
		switch strings.ToLower(strings.TrimSpace(injection.Format)) {
		case "mermaid":
			if a.Config.Output.MermaidEnabled() {
				needMermaid = true
			}
		case "plantuml":
			if a.Config.Output.PlantUMLEnabled() {
				needPlantUML = true
			}
		}
	}

	if needMermaid {
		mermaidGen := report.NewMermaidGenerator(a.Graph)
		mermaidGen.SetModuleMetrics(metrics)
		mermaidGen.SetComplexityHotspots(hotspots)
		for _, mode := range diagramModes {
			diagram, genErr := generateMermaidByMode(mermaidGen, mode, archModel, cycles, violations, a.Config.Output.Diagrams)
			if genErr != nil {
				return fmt.Errorf("generate Mermaid output (%s): %w", mode.Suffix(), genErr)
			}
			mermaidDiagrams[mode] = diagram
			if targets.Mermaid != "" {
				outPath := diagramOutputPath(targets.Mermaid, mode, len(diagramModes) > 1)
				if err := writeArtifact(outPath, diagram); err != nil {
					return fmt.Errorf("write Mermaid output %q: %w", outPath, err)
				}
			}
		}
	}

	if needPlantUML {
		plantUMLGen := report.NewPlantUMLGenerator(a.Graph)
		plantUMLGen.SetModuleMetrics(metrics)
		plantUMLGen.SetComplexityHotspots(hotspots)
		for _, mode := range diagramModes {
			diagram, genErr := generatePlantUMLByMode(plantUMLGen, mode, archModel, cycles, violations, a.Config.Output.Diagrams)
			if genErr != nil {
				return fmt.Errorf("generate PlantUML output (%s): %w", mode.Suffix(), genErr)
			}
			plantUMLDiagrams[mode] = diagram
			if targets.PlantUML != "" {
				outPath := diagramOutputPath(targets.PlantUML, mode, len(diagramModes) > 1)
				if err := writeArtifact(outPath, diagram); err != nil {
					return fmt.Errorf("write PlantUML output %q: %w", outPath, err)
				}
			}
		}
	}

	injectionMode := preferredInjectionMode(diagramModes)
	for _, injection := range a.Config.Output.UpdateMarkdown {
		format := strings.ToLower(strings.TrimSpace(injection.Format))
		diagram := ""
		switch format {
		case "mermaid":
			diagram = markdownDiagramBlock("mermaid", mermaidDiagrams[injectionMode])
		case "plantuml":
			diagram = markdownDiagramBlock("plantuml", plantUMLDiagrams[injectionMode])
		default:
			continue
		}

		if err := report.InjectDiagram(injection.File, injection.Marker, diagram); err != nil {
			return fmt.Errorf("inject %s diagram into %q with marker %q: %w", format, injection.File, injection.Marker, err)
		}
	}

	if targets.Markdown != "" {
		if unresolved == nil {
			unresolved = a.AnalyzeHallucinations()
		}
		root, err := a.resolveOutputRoot()
		if err != nil {
			return err
		}
		reportGen := report.NewMarkdownGenerator()
		md, err := reportGen.Generate(report.MarkdownReportData{
			TotalModules:  a.Graph.ModuleCount(),
			TotalFiles:    a.Graph.FileCount(),
			Cycles:        cycles,
			Unresolved:    unresolved,
			UnusedImports: unusedImports,
			Violations:    violations,
			Hotspots:      hotspots,
		}, report.MarkdownReportOptions{
			ProjectName:         filepath.Base(root),
			ProjectRoot:         root,
			Version:             version.Version,
			GeneratedAt:         time.Now().UTC(),
			Verbosity:           a.Config.Output.Report.Verbosity,
			TableOfContents:     a.Config.Output.Report.TableOfContentsEnabled(),
			CollapsibleSections: a.Config.Output.Report.CollapsibleSectionsEnabled(),
			IncludeMermaid:      a.Config.Output.Report.IncludeMermaidEnabled(),
			MermaidDiagram:      mermaidDiagrams[preferredInjectionMode(diagramModes)],
		})
		if err != nil {
			return fmt.Errorf("generate markdown report: %w", err)
		}
		if err := writeArtifact(targets.Markdown, md); err != nil {
			return fmt.Errorf("write markdown report %q: %w", targets.Markdown, err)
		}
	}

	return nil
}

type diagramMode int

const (
	diagramModeDependency diagramMode = iota
	diagramModeArchitecture
	diagramModeComponent
	diagramModeFlow
)

func resolveDiagramMode(diagrams config.DiagramOutput) (diagramMode, error) {
	modes, err := resolveDiagramModes(diagrams)
	if err != nil {
		return diagramModeDependency, err
	}
	if len(modes) == 0 {
		return diagramModeDependency, nil
	}
	return modes[0], nil
}

func resolveDiagramModes(diagrams config.DiagramOutput) ([]diagramMode, error) {
	modes := make([]diagramMode, 0, 4)
	selected := 0
	if diagrams.Architecture {
		selected++
		modes = append(modes, diagramModeArchitecture)
	}
	if diagrams.Component {
		selected++
		modes = append(modes, diagramModeComponent)
	}
	if diagrams.Flow {
		selected++
		modes = append(modes, diagramModeFlow)
	}
	if selected == 0 {
		return []diagramMode{diagramModeDependency}, nil
	}
	if selected > 1 {
		return append([]diagramMode{diagramModeDependency}, modes...), nil
	}
	return modes, nil
}

func (m diagramMode) Suffix() string {
	switch m {
	case diagramModeArchitecture:
		return "architecture"
	case diagramModeComponent:
		return "component"
	case diagramModeFlow:
		return "flow"
	default:
		return "dependency"
	}
}

func preferredInjectionMode(modes []diagramMode) diagramMode {
	if len(modes) == 0 {
		return diagramModeDependency
	}
	for _, mode := range modes {
		if mode == diagramModeDependency {
			return mode
		}
	}
	return modes[0]
}

func diagramOutputPath(base string, mode diagramMode, suffixOutput bool) string {
	if !suffixOutput {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return stem + "-" + mode.Suffix() + ext
}

func generateMermaidByMode(
	gen *report.MermaidGenerator,
	mode diagramMode,
	archModel graph.ArchitectureModel,
	cycles [][]string,
	violations []graph.ArchitectureViolation,
	diagrams config.DiagramOutput,
) (string, error) {
	switch mode {
	case diagramModeArchitecture:
		return gen.GenerateArchitecture(archModel, violations)
	case diagramModeComponent:
		return gen.GenerateComponent(archModel, diagrams.ComponentCfg.ShowInternal)
	case diagramModeFlow:
		return gen.GenerateFlow(diagrams.FlowConfig.EntryPoints, diagrams.FlowConfig.MaxDepth)
	default:
		return gen.Generate(cycles, violations, archModel)
	}
}

func generatePlantUMLByMode(
	gen *report.PlantUMLGenerator,
	mode diagramMode,
	archModel graph.ArchitectureModel,
	cycles [][]string,
	violations []graph.ArchitectureViolation,
	diagrams config.DiagramOutput,
) (string, error) {
	switch mode {
	case diagramModeArchitecture:
		return gen.GenerateArchitecture(archModel, violations)
	case diagramModeComponent:
		return gen.GenerateComponent(archModel, diagrams.ComponentCfg.ShowInternal)
	case diagramModeFlow:
		return gen.GenerateFlow(diagrams.FlowConfig.EntryPoints, diagrams.FlowConfig.MaxDepth)
	default:
		return gen.Generate(cycles, violations, archModel)
	}
}

type outputTargets struct {
	DOT      string
	TSV      string
	Mermaid  string
	PlantUML string
	Markdown string
}

func (a *App) resolveOutputTargets() (outputTargets, error) {
	root, err := a.resolveOutputRoot()
	if err != nil {
		return outputTargets{}, err
	}
	diagramsDir := strings.TrimSpace(a.Config.Output.Paths.DiagramsDir)
	if diagramsDir == "" {
		diagramsDir = "docs/diagrams"
	}
	if !filepath.IsAbs(diagramsDir) {
		diagramsDir = filepath.Join(root, diagramsDir)
	}
	targets := outputTargets{
		DOT:      resolveOutputPath(strings.TrimSpace(a.Config.Output.DOT), root),
		TSV:      resolveOutputPath(strings.TrimSpace(a.Config.Output.TSV), root),
		Mermaid:  resolveDiagramPath(strings.TrimSpace(a.Config.Output.Mermaid), root, diagramsDir),
		PlantUML: resolveDiagramPath(strings.TrimSpace(a.Config.Output.PlantUML), root, diagramsDir),
		Markdown: resolveOutputPath(strings.TrimSpace(a.Config.Output.Markdown), root),
	}
	return targets, nil
}

func (a *App) resolveOutputRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	paths, err := config.ResolvePaths(a.Config, cwd)
	if err != nil {
		return "", fmt.Errorf("resolve output root: %w", err)
	}
	return paths.OutputRoot, nil
}

func resolveOutputPath(path, root string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(root, path)
}

func resolveDiagramPath(path, root, diagramsDir string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if strings.Contains(path, string(os.PathSeparator)) || strings.Contains(path, "/") {
		return filepath.Join(root, path)
	}
	return filepath.Join(diagramsDir, path)
}

func writeArtifact(path, content string) error {
	return util.WriteStringWithDirs(path, content, 0o644)
}

func (a *App) TraceImportChain(from, to string) (string, error) {
	if _, ok := a.Graph.GetModule(from); !ok {
		return "", fmt.Errorf("source module not found: %s", from)
	}
	if _, ok := a.Graph.GetModule(to); !ok {
		return "", fmt.Errorf("target module not found: %s", to)
	}

	chain, ok := a.Graph.FindImportChain(from, to)
	if !ok {
		return "", fmt.Errorf("no import chain found from %s to %s", from, to)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Import chain: %s -> %s\n\n", from, to))
	for i, module := range chain {
		b.WriteString(module)
		b.WriteString("\n")
		if i < len(chain)-1 {
			b.WriteString("  -> ")
		}
	}

	return strings.TrimRight(b.String(), "\n"), nil
}

func (a *App) AnalyzeImpact(path string) (graph.ImpactReport, error) {
	return a.Graph.AnalyzeImpact(path)
}

func (a *App) ArchitectureViolations() []graph.ArchitectureViolation {
	return a.archEngine.Validate(a.Graph)
}

func (a *App) BuildQueryService(historyStore interface {
	LoadSnapshots(projectKey string, since time.Time) ([]history.Snapshot, error)
}, projectKey string) *query.Service {
	return query.NewService(a.Graph, historyStore, projectKey)
}

func (a *App) SecretCount() int {
	total := 0
	for _, file := range a.Graph.GetAllFiles() {
		total += len(file.Secrets)
	}
	return total
}

func (a *App) allSecrets(limit int) []parser.Secret {
	all := make([]parser.Secret, 0)
	for _, file := range a.Graph.GetAllFiles() {
		if file == nil || len(file.Secrets) == 0 {
			continue
		}
		all = append(all, file.Secrets...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Location.File != all[j].Location.File {
			return all[i].Location.File < all[j].Location.File
		}
		if all[i].Location.Line != all[j].Location.Line {
			return all[i].Location.Line < all[j].Location.Line
		}
		if all[i].Location.Column != all[j].Location.Column {
			return all[i].Location.Column < all[j].Location.Column
		}
		return all[i].Kind < all[j].Kind
	})
	if limit > 0 && len(all) > limit {
		return append([]parser.Secret(nil), all[:limit]...)
	}
	return all
}

func (a *App) GenerateMarkdownReport(req MarkdownReportRequest) (MarkdownReportResult, error) {
	cycles := a.Graph.DetectCycles()
	metrics := a.Graph.ComputeModuleMetrics()
	hotspots := a.Graph.TopComplexity(a.Config.Architecture.TopComplexity)
	violations := a.ArchitectureViolations()
	unresolved := a.AnalyzeHallucinations()
	unused := a.AnalyzeUnusedImports()

	root, err := a.resolveOutputRoot()
	if err != nil {
		return MarkdownReportResult{}, err
	}

	var mermaidDiagram string
	if a.Config.Output.Report.IncludeMermaidEnabled() && a.Config.Output.MermaidEnabled() {
		mermaidGen := report.NewMermaidGenerator(a.Graph)
		mermaidGen.SetModuleMetrics(metrics)
		mermaidGen.SetComplexityHotspots(hotspots)
		mermaidDiagram, err = mermaidGen.Generate(cycles, violations, architectureModelFromConfig(a.Config.Architecture))
		if err != nil {
			return MarkdownReportResult{}, fmt.Errorf("generate mermaid diagram for markdown report: %w", err)
		}
	}

	verbosity := strings.TrimSpace(req.Verbosity)
	if verbosity == "" {
		verbosity = a.Config.Output.Report.Verbosity
	}
	md, err := report.NewMarkdownGenerator().Generate(report.MarkdownReportData{
		TotalModules:  a.Graph.ModuleCount(),
		TotalFiles:    a.Graph.FileCount(),
		Cycles:        cycles,
		Unresolved:    unresolved,
		UnusedImports: unused,
		Violations:    violations,
		Hotspots:      hotspots,
	}, report.MarkdownReportOptions{
		ProjectName:         filepath.Base(root),
		ProjectRoot:         root,
		Version:             version.Version,
		GeneratedAt:         time.Now().UTC(),
		Verbosity:           verbosity,
		TableOfContents:     a.Config.Output.Report.TableOfContentsEnabled(),
		CollapsibleSections: a.Config.Output.Report.CollapsibleSectionsEnabled(),
		IncludeMermaid:      a.Config.Output.Report.IncludeMermaidEnabled(),
		MermaidDiagram:      mermaidDiagram,
	})
	if err != nil {
		return MarkdownReportResult{}, fmt.Errorf("generate markdown report: %w", err)
	}

	outPath := strings.TrimSpace(req.OutputPath)
	if outPath == "" {
		outPath = strings.TrimSpace(a.Config.Output.Markdown)
	}
	if outPath == "" && req.WriteFile {
		outPath = "analysis-report.md"
	}
	if outPath != "" && !filepath.IsAbs(outPath) {
		outPath = filepath.Join(root, outPath)
	}

	result := MarkdownReportResult{Markdown: md, Path: outPath}
	if req.WriteFile || outPath != "" {
		if outPath == "" {
			return MarkdownReportResult{}, fmt.Errorf("output path is required when write_file=true")
		}
		if err := writeArtifact(outPath, md); err != nil {
			return MarkdownReportResult{}, fmt.Errorf("write markdown report %q: %w", outPath, err)
		}
		result.Written = true
	}
	return result, nil
}

func (a *App) PrintSummary(
	fileCount, moduleCount int,
	duration time.Duration,
	cycles [][]string,
	hallucinations []resolver.UnresolvedReference,
	unusedImports []resolver.UnusedImport,
	metrics map[string]graph.ModuleMetrics,
	violations []graph.ArchitectureViolation,
	hotspots []graph.ComplexityHotspot,
) {
	if !a.Config.Alerts.Terminal {
		return
	}

	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Update: %d files, %d modules in %v\n", fileCount, moduleCount, duration)

	if len(cycles) > 0 {
		fmt.Printf("⚠️  FOUND %d CIRCULAR IMPORTS:\n", len(cycles))
		for _, c := range cycles {
			fmt.Printf("   %s\n", strings.Join(c, " -> "))
		}
	} else {
		fmt.Println("✅ No circular imports found.")
	}

	if len(hallucinations) > 0 {
		fmt.Printf("❓ FOUND %d UNRESOLVED REFERENCES:\n", len(hallucinations))
		for _, h := range hallucinations {
			fmt.Printf("   %s in %s:%d\n", h.Reference.Name, h.File, h.Reference.Location.Line)
		}
	} else {
		fmt.Println("✅ No unresolved references found.")
	}

	if len(unusedImports) > 0 {
		fmt.Printf("🧹 FOUND %d UNUSED IMPORTS:\n", len(unusedImports))
		for _, u := range unusedImports {
			target := u.Module
			if u.Item != "" {
				target = target + "." + u.Item
			}
			fmt.Printf("   %s in %s:%d\n", target, u.File, u.Location.Line)
		}
	} else {
		fmt.Println("✅ No unused imports found.")
	}

	secretCount := a.SecretCount()
	if secretCount > 0 {
		fmt.Printf("🔐 FOUND %d POTENTIAL SECRETS\n", secretCount)
		for _, finding := range a.allSecrets(5) {
			fmt.Printf("   %s [%s] %s:%d (%s)\n",
				finding.Kind,
				finding.Severity,
				finding.Location.File,
				finding.Location.Line,
				secrets.MaskValue(finding.Value),
			)
		}
	} else {
		fmt.Println("✅ No hardcoded secrets found.")
	}

	if len(metrics) > 0 {
		topDepth := metricLeaders(metrics, func(m graph.ModuleMetrics) int { return m.Depth }, 3, 0)
		topFanIn := metricLeaders(metrics, func(m graph.ModuleMetrics) int { return m.FanIn }, 3, 1)
		topFanOut := metricLeaders(metrics, func(m graph.ModuleMetrics) int { return m.FanOut }, 3, 1)

		fmt.Println("📊 Dependency Metrics:")
		if len(topDepth) > 0 {
			fmt.Printf("   Deepest modules: %s\n", strings.Join(topDepth, ", "))
		}
		if len(topFanIn) > 0 {
			fmt.Printf("   Highest fan-in: %s\n", strings.Join(topFanIn, ", "))
		}
		if len(topFanOut) > 0 {
			fmt.Printf("   Highest fan-out: %s\n", strings.Join(topFanOut, ", "))
		}
	}

	if len(violations) > 0 {
		fmt.Printf("🏛️  FOUND %d ARCHITECTURE VIOLATIONS:\n", len(violations))
		for _, v := range violations {
			fmt.Printf("   %s (%s -> %s) in %s:%d\n", v.RuleName, v.FromLayer, v.ToLayer, v.File, v.Line)
		}
	} else if a.Config.Architecture.Enabled {
		fmt.Println("✅ No architecture violations found.")
	}

	if len(hotspots) > 0 {
		fmt.Println("🔥 Top complexity hotspots:")
		for _, h := range hotspots {
			fmt.Printf("   %s.%s score=%d (branches=%d params=%d depth=%d loc=%d)\n", h.Module, h.Definition, h.Score, h.Branches, h.Parameters, h.Nesting, h.LOC)
		}
	}
	fmt.Println(strings.Repeat("-", 40))
}

func (a *App) StartWatcher() error {
	w, err := watcher.NewWatcher(
		a.Config.Watch.Debounce,
		a.Config.Exclude.Dirs,
		a.Config.Exclude.Files,
		a.HandleChanges,
	)
	if err != nil {
		return err
	}
	w.SetLanguageFilters(
		a.Parser.SupportedExtensions(),
		a.Parser.SupportedFilenames(),
		a.Parser.SupportedTestFileSuffixes(),
	)
	return w.Watch(a.Config.WatchPaths)
}

func (a *App) resolveGoModule(path string) (string, bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	dir := filepath.Dir(absPath)
	visited := []string{}
	for {
		if cached, ok := a.goModCache[dir]; ok {
			if !cached.Found {
				return "", false, nil
			}
			moduleName, err := moduleNameFromCache(cached, absPath)
			if err != nil {
				return "", false, err
			}
			return moduleName, true, nil
		}
		visited = append(visited, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	r := resolver.NewGoResolver()
	if err := r.FindGoMod(absPath); err != nil {
		for _, d := range visited {
			a.goModCache[d] = goModuleCacheEntry{Found: false}
		}
		return "", false, nil
	}

	cached := goModuleCacheEntry{
		Found:      true,
		ModuleRoot: r.GetModuleRoot(),
		ModulePath: r.ModulePath(),
	}
	for _, d := range visited {
		a.goModCache[d] = cached
	}

	moduleName, err := moduleNameFromCache(cached, absPath)
	if err != nil {
		return "", false, err
	}
	return moduleName, true, nil
}

func moduleNameFromCache(cached goModuleCacheEntry, filePath string) (string, error) {
	rel, err := filepath.Rel(cached.ModuleRoot, filePath)
	if err != nil {
		return "", fmt.Errorf("resolve module name from cache entry %+v for %q: %w", cached, filePath, err)
	}
	dir := filepath.Dir(rel)
	if dir == "." {
		return cached.ModulePath, nil
	}
	return cached.ModulePath + "/" + dir, nil
}

func (a *App) rebuildUnresolvedCache(unresolved []resolver.UnresolvedReference) {
	next := make(map[string][]resolver.UnresolvedReference)
	for _, f := range a.Graph.GetAllFiles() {
		next[f.Path] = nil
	}
	for _, u := range unresolved {
		next[u.File] = append(next[u.File], u)
	}
	a.unresolvedMu.Lock()
	a.unresolvedByFile = next
	a.unresolvedMu.Unlock()
}

func (a *App) cachedUnresolved() []resolver.UnresolvedReference {
	a.unresolvedMu.RLock()
	defer a.unresolvedMu.RUnlock()

	res := make([]resolver.UnresolvedReference, 0)
	for _, refs := range a.unresolvedByFile {
		res = append(res, refs...)
	}
	return res
}

func (a *App) rebuildUnusedCache(unused []resolver.UnusedImport) {
	next := make(map[string][]resolver.UnusedImport)
	for _, f := range a.Graph.GetAllFiles() {
		next[f.Path] = nil
	}
	for _, u := range unused {
		next[u.File] = append(next[u.File], u)
	}
	a.unusedMu.Lock()
	a.unusedByFile = next
	a.unusedMu.Unlock()
}

func (a *App) cachedUnused() []resolver.UnusedImport {
	a.unusedMu.RLock()
	defer a.unusedMu.RUnlock()

	res := make([]resolver.UnusedImport, 0)
	for _, refs := range a.unusedByFile {
		res = append(res, refs...)
	}
	return res
}

func (a *App) currentGraphPaths() []string {
	files := a.Graph.GetAllFiles()
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	return paths
}

func metricLeaders(
	metrics map[string]graph.ModuleMetrics,
	scoreFn func(graph.ModuleMetrics) int,
	limit int,
	minScore int,
) []string {
	type scoredModule struct {
		module string
		score  int
	}

	scored := make([]scoredModule, 0, len(metrics))
	for module, m := range metrics {
		score := scoreFn(m)
		if score < minScore {
			continue
		}
		scored = append(scored, scoredModule{module: module, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].module < scored[j].module
		}
		return scored[i].score > scored[j].score
	})

	if len(scored) > limit {
		scored = scored[:limit]
	}

	lines := make([]string, 0, len(scored))
	for _, s := range scored {
		lines = append(lines, fmt.Sprintf("%s(%d)", s.module, s.score))
	}
	return lines
}

func architectureModelFromConfig(arch config.Architecture) graph.ArchitectureModel {
	model := graph.ArchitectureModel{
		Enabled: arch.Enabled,
		Layers:  make([]graph.ArchitectureLayer, 0, len(arch.Layers)),
		Rules:   make([]graph.ArchitectureRule, 0, len(arch.Rules)),
	}
	for _, layer := range arch.Layers {
		model.Layers = append(model.Layers, graph.ArchitectureLayer{
			Name:  layer.Name,
			Paths: append([]string(nil), layer.Paths...),
		})
	}
	for _, rule := range arch.Rules {
		model.Rules = append(model.Rules, graph.ArchitectureRule{
			Name:  rule.Name,
			From:  rule.From,
			Allow: append([]string(nil), rule.Allow...),
		})
	}
	return model
}

func markdownDiagramBlock(format, diagram string) string {
	trimmed := strings.TrimRight(diagram, "\n")
	return "```" + format + "\n" + trimmed + "\n```"
}
