// # cmd/circular/app.go
package main

import (
	"circular/internal/config"
	"circular/internal/graph"
	"circular/internal/output"
	"circular/internal/parser"
	"circular/internal/resolver"
	"circular/internal/watcher"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gobwas/glob"
)

type App struct {
	Config     *config.Config
	Parser     *parser.Parser
	Graph      *graph.Graph
	archEngine *graph.LayerRuleEngine
	teaProgram *tea.Program
	goModCache map[string]goModuleCacheEntry

	// Cached unresolved references keyed by file path for incremental updates.
	unresolvedByFile map[string][]resolver.UnresolvedReference
	// Cached unused imports keyed by file path for incremental updates.
	unusedByFile map[string][]resolver.UnusedImport
}

type goModuleCacheEntry struct {
	Found      bool
	ModuleRoot string
	ModulePath string
}

func NewApp(cfg *config.Config) (*App, error) {
	loader, err := parser.NewGrammarLoader(cfg.GrammarsPath)
	if err != nil {
		return nil, err
	}

	p := parser.NewParser(loader)
	p.RegisterExtractor("python", &parser.PythonExtractor{})
	p.RegisterExtractor("go", &parser.GoExtractor{})

	return &App{
		Config:           cfg,
		Parser:           p,
		Graph:            graph.NewGraph(),
		archEngine:       graph.NewLayerRuleEngine(architectureModelFromConfig(cfg.Architecture)),
		goModCache:       make(map[string]goModuleCacheEntry),
		unresolvedByFile: make(map[string][]resolver.UnresolvedReference),
		unusedByFile:     make(map[string][]resolver.UnusedImport),
	}, nil
}

func (a *App) InitialScan() error {
	paths := a.Config.WatchPaths

	// If it's a Go project, we might want to expand the scan to the module root
	// to ensure all internal definitions are loaded, even if we only watch a sub-path.
	expandedPaths := make(map[string]bool)
	for _, p := range paths {
		expandedPaths[p] = true

		// Look for go.mod to find module root
		r := resolver.NewGoResolver()
		if err := r.FindGoMod(p); err == nil {
			// Get absolute path of module root
			if absRoot, err := filepath.Abs(r.GetModuleRoot()); err == nil {
				expandedPaths[absRoot] = true
			}
		}
	}

	finalPaths := make([]string, 0, len(expandedPaths))
	for p := range expandedPaths {
		finalPaths = append(finalPaths, p)
	}

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

			ext := filepath.Ext(path)
			if ext != ".py" && ext != ".go" {
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
		r := resolver.NewPythonResolver(a.Config.WatchPaths[0])
		file.Module = r.GetModuleName(path)
	} else if file.Language == "go" {
		if moduleName, ok := a.resolveGoModule(path); ok {
			file.Module = moduleName
		}
	}

	a.Graph.AddFile(file)
	return nil
}

func (a *App) HandleChanges(paths []string) {
	slog.Info("detected changes", "count", len(paths))
	start := time.Now()
	affectedSet := make(map[string]bool)

	for _, path := range paths {
		if filepath.Base(path) == "go.mod" {
			a.goModCache = make(map[string]goModuleCacheEntry)
		}

		// Determine impacted files from previous graph state before applying this update.
		for _, f := range a.Graph.InvalidateTransitive(path) {
			affectedSet[f] = true
		}
		affectedSet[path] = true

		if _, err := os.Stat(path); os.IsNotExist(err) {
			a.Graph.RemoveFile(path)
			delete(a.unresolvedByFile, path)
			continue
		}

		if err := a.ProcessFile(path); err != nil {
			slog.Warn("failed to re-process file", "path", path, "error", err)
		}
	}

	cycles := a.Graph.DetectCycles()
	metrics := a.Graph.ComputeModuleMetrics()
	hotspots := a.Graph.TopComplexity(a.Config.Architecture.TopComplexity)
	violations := a.archEngine.Validate(a.Graph)
	hallucinations := a.AnalyzeHallucinationsIncremental(affectedSet)
	unusedImports := a.AnalyzeUnusedImportsIncremental(affectedSet)

	if err := a.GenerateOutputs(cycles, unusedImports, metrics, violations, hotspots); err != nil {
		slog.Error("failed to generate outputs", "error", err)
	}

	duration := time.Since(start)
	a.PrintSummary(len(paths), a.Graph.ModuleCount(), duration, cycles, hallucinations, unusedImports, metrics, violations, hotspots)

	if a.teaProgram != nil {
		a.teaProgram.Send(updateMsg{
			cycles:         cycles,
			hallucinations: hallucinations,
			moduleCount:    a.Graph.ModuleCount(),
			fileCount:      a.Graph.FileCount(), // Need to implement this or use a count
		})
	}

	if a.Config.Alerts.Beep && (len(cycles) > 0 || len(hallucinations) > 0 || len(unusedImports) > 0 || len(violations) > 0) {
		fmt.Print("\a")
	}
}

func (a *App) AnalyzeHallucinations() []resolver.UnresolvedReference {
	res := resolver.NewResolver(a.Graph, a.Config.Exclude.Symbols)
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

	res := resolver.NewResolver(a.Graph, a.Config.Exclude.Symbols)
	updated := res.FindUnresolvedForPaths(paths)

	// Reset cache entries for impacted files that still exist.
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

	return a.cachedUnresolved()
}

func (a *App) AnalyzeUnusedImports() []resolver.UnusedImport {
	paths := a.currentGraphPaths()
	res := resolver.NewResolver(a.Graph, a.Config.Exclude.Symbols)
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

	res := resolver.NewResolver(a.Graph, a.Config.Exclude.Symbols)
	updated := res.FindUnusedImports(paths)

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

	return a.cachedUnused()
}

func (a *App) GenerateOutputs(
	cycles [][]string,
	unusedImports []resolver.UnusedImport,
	metrics map[string]graph.ModuleMetrics,
	violations []graph.ArchitectureViolation,
	hotspots []graph.ComplexityHotspot,
) error {
	if a.Config.Output.DOT != "" {
		dotGen := output.NewDOTGenerator(a.Graph)
		dotGen.SetModuleMetrics(metrics)
		dotGen.SetComplexityHotspots(hotspots)
		dot, err := dotGen.Generate(cycles)
		if err != nil {
			return err
		}
		if err := os.WriteFile(a.Config.Output.DOT, []byte(dot), 0644); err != nil {
			return err
		}
	}

	if a.Config.Output.TSV != "" {
		tsvGen := output.NewTSVGenerator(a.Graph)
		dependenciesTSV, err := tsvGen.Generate()
		if err != nil {
			return err
		}
		tsv := dependenciesTSV

		if len(unusedImports) > 0 {
			unusedTSV, err := tsvGen.GenerateUnusedImports(unusedImports)
			if err != nil {
				return err
			}
			tsv = strings.TrimRight(dependenciesTSV, "\n") + "\n\n" + strings.TrimRight(unusedTSV, "\n") + "\n"
		}
		if len(violations) > 0 {
			violationsTSV, err := tsvGen.GenerateArchitectureViolations(violations)
			if err != nil {
				return err
			}
			tsv = strings.TrimRight(tsv, "\n") + "\n\n" + strings.TrimRight(violationsTSV, "\n") + "\n"
		}

		if err := os.WriteFile(a.Config.Output.TSV, []byte(tsv), 0644); err != nil {
			return err
		}
	}

	return nil
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

func (a *App) RunUI() error {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	a.teaProgram = p

	// Trigger initial UI update
	go func() {
		// Wait a bit for initial scan to complete if it's still running
		// In main.go we call InitialScan before StartWatcher/RunUI
		cycles := a.Graph.DetectCycles()
		hallucinations := a.AnalyzeHallucinations()
		a.teaProgram.Send(updateMsg{
			cycles:         cycles,
			hallucinations: hallucinations,
			moduleCount:    a.Graph.ModuleCount(),
			fileCount:      a.Graph.FileCount(),
		})
	}()

	_, err := p.Run()
	return err
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
	// Note: We don't close here, it should run forever
	return w.Watch(a.Config.WatchPaths)
}

func (a *App) resolveGoModule(path string) (string, bool) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	dir := filepath.Dir(absPath)
	visited := []string{}
	for {
		if cached, ok := a.goModCache[dir]; ok {
			if !cached.Found {
				return "", false
			}
			return moduleNameFromCache(cached, absPath), true
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
		return "", false
	}

	cached := goModuleCacheEntry{
		Found:      true,
		ModuleRoot: r.GetModuleRoot(),
		ModulePath: r.ModulePath(),
	}
	for _, d := range visited {
		a.goModCache[d] = cached
	}

	return moduleNameFromCache(cached, absPath), true
}

func moduleNameFromCache(cached goModuleCacheEntry, filePath string) string {
	rel, err := filepath.Rel(cached.ModuleRoot, filePath)
	if err != nil {
		return ""
	}
	dir := filepath.Dir(rel)
	if dir == "." {
		return cached.ModulePath
	}
	return cached.ModulePath + "/" + dir
}

func (a *App) rebuildUnresolvedCache(unresolved []resolver.UnresolvedReference) {
	next := make(map[string][]resolver.UnresolvedReference)
	for _, f := range a.Graph.GetAllFiles() {
		next[f.Path] = nil
	}
	for _, u := range unresolved {
		next[u.File] = append(next[u.File], u)
	}
	a.unresolvedByFile = next
}

func (a *App) cachedUnresolved() []resolver.UnresolvedReference {
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
	a.unusedByFile = next
}

func (a *App) cachedUnused() []resolver.UnusedImport {
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
		scored = append(scored, scoredModule{
			module: module,
			score:  score,
		})
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
