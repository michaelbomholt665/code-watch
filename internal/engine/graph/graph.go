// # internal/graph/graph.go
package graph

import (
	"circular/internal/engine/parser"
	"circular/internal/shared/observability"
	"sort"
	"sync"
)

type FileLoader interface {
	LoadFile(path string) (*parser.File, error)
}

type Graph struct {
	mu sync.RWMutex

	// Core data
	fileCache      *LRUCache[string, *parser.File] // path -> file
	fileToModule   map[string]string               // path -> module name
	fileToLanguage map[string]string               // path -> language
	modules        map[string]*Module              // module name -> module info

	loader FileLoader

	// Relationships
	imports    map[string]map[string]*ImportEdge // from -> to -> edge
	importedBy map[string]map[string]bool        // to -> from

	// Symbol tables (for hallucination detection)
	definitions map[string]map[string]*parser.Definition // module -> symbol -> def

	// Invalidation tracking
	dirty map[string]bool // Files needing re-analysis
}

type Module struct {
	Name          string
	Files         []string // Paths to files in this module
	Exports       map[string]*parser.Definition
	MaxComplexity int    // Max complexity score across all files in this module
	RootPath      string // For Go: module root, Python: package root
}

type ImportEdge struct {
	From       string
	To         string
	ImportedBy string // File path
	Location   parser.Location
}

type ModuleMetrics struct {
	Depth           int
	FanIn           int
	FanOut          int
	ImportanceScore float64 // (FanIn*2) + (FanOut*1) + (Complexity*0.5) + (IsAPI?10:0)
}

func NewGraph() *Graph {
	return NewGraphWithCapacity(1000)
}

func NewGraphWithCapacity(capacity int) *Graph {
	return &Graph{
		fileCache:      NewLRUCache[string, *parser.File](capacity),
		fileToModule:   make(map[string]string),
		fileToLanguage: make(map[string]string),
		modules:        make(map[string]*Module),
		imports:        make(map[string]map[string]*ImportEdge),
		importedBy:     make(map[string]map[string]bool),
		definitions:    make(map[string]map[string]*parser.Definition),
		dirty:          make(map[string]bool),
	}
}

func (g *Graph) SetLoader(loader FileLoader) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.loader = loader
}

func (g *Graph) UpdateCapacity(capacity int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.fileCache.SetCapacity(capacity)
}

func (g *Graph) PruneCache(percentage int) int {
	return g.fileCache.Prune(percentage)
}

func (g *Graph) AddFile(file *parser.File) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// If this file already exists, remove prior contributions first.
	// This prevents stale imports/definitions after file edits.
	if _, exists := g.fileCache.Get(file.Path); exists {
		g.removeFileLocked(file.Path)
	}

	g.fileCache.Put(file.Path, cloneFile(file))
	g.fileToModule[file.Path] = file.Module
	g.fileToLanguage[file.Path] = file.Language

	mod, ok := g.modules[file.Module]
	if !ok {
		mod = &Module{
			Name:    file.Module,
			Exports: make(map[string]*parser.Definition),
		}
		g.modules[file.Module] = mod
	}

	found := false
	for _, p := range mod.Files {
		if p == file.Path {
			found = true
			break
		}
	}
	if !found {
		mod.Files = append(mod.Files, file.Path)
	}

	if g.definitions[file.Module] == nil {
		g.definitions[file.Module] = make(map[string]*parser.Definition)
	}
	fileMaxComplexity := 0
	for i := range file.Definitions {
		def := cloneDefinition(&file.Definitions[i])
		existingDef, hasExistingDef := g.definitions[file.Module][def.Name]
		if !hasExistingDef || preferDefinition(def, existingDef) {
			g.definitions[file.Module][def.Name] = def
		}
		if def.Exported {
			existingExport, hasExport := mod.Exports[def.Name]
			if !hasExport || preferDefinition(def, existingExport) {
				mod.Exports[def.Name] = def
			}
		}

		sc := def.ComplexityScore
		if sc == 0 {
			sc = (def.BranchCount * 2) + (def.NestingDepth * 2) + def.ParameterCount + (def.LOC / 10)
		}
		if sc > fileMaxComplexity {
			fileMaxComplexity = sc
		}
	}

	if fileMaxComplexity > mod.MaxComplexity {
		mod.MaxComplexity = fileMaxComplexity
	}

	if g.imports[file.Module] == nil {
		g.imports[file.Module] = make(map[string]*ImportEdge)
	}

	for _, imp := range file.Imports {
		edge := &ImportEdge{
			From:       file.Module,
			To:         imp.Module,
			ImportedBy: file.Path,
			Location:   imp.Location,
		}
		g.imports[file.Module][imp.Module] = edge

		if g.importedBy[imp.Module] == nil {
			g.importedBy[imp.Module] = make(map[string]bool)
		}
		g.importedBy[imp.Module][file.Module] = true
	}

	observability.GraphNodes.Set(float64(len(g.modules)))
	edgeCount := 0
	for _, targets := range g.imports {
		edgeCount += len(targets)
	}
	observability.GraphEdges.Set(float64(edgeCount))
}

func (g *Graph) RemoveFile(path string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.removeFileLocked(path)
}

func (g *Graph) removeFileLocked(path string) {
	moduleName, ok := g.fileToModule[path]
	if !ok {
		return
	}

	if mod, ok := g.modules[moduleName]; ok {
		for i, p := range mod.Files {
			if p == path {
				mod.Files = append(mod.Files[:i], mod.Files[i+1:]...)
				break
			}
		}

		if len(mod.Files) == 0 {
			for to := range g.imports[moduleName] {
				if g.importedBy[to] != nil {
					delete(g.importedBy[to], moduleName)
				}
			}

			delete(g.modules, moduleName)
			delete(g.imports, moduleName)
			delete(g.definitions, moduleName)
		} else {
			mod.Exports = make(map[string]*parser.Definition)
			g.definitions[moduleName] = make(map[string]*parser.Definition)

			oldImports := g.imports[moduleName]
			g.imports[moduleName] = make(map[string]*ImportEdge)

			for _, filePath := range mod.Files {
				if f, ok := g.fileCache.Get(filePath); ok {
					for i := range f.Definitions {
						def := cloneDefinition(&f.Definitions[i])
						existingDef, hasExistingDef := g.definitions[moduleName][def.Name]
						if !hasExistingDef || preferDefinition(def, existingDef) {
							g.definitions[moduleName][def.Name] = def
						}
						if def.Exported {
							existingExport, hasExport := mod.Exports[def.Name]
							if !hasExport || preferDefinition(def, existingExport) {
								mod.Exports[def.Name] = def
							}
						}
					}
					for _, imp := range f.Imports {
						edge := &ImportEdge{
							From:       f.Module,
							To:         imp.Module,
							ImportedBy: f.Path,
							Location:   imp.Location,
						}
						g.imports[f.Module][imp.Module] = edge
						if g.importedBy[imp.Module] == nil {
							g.importedBy[imp.Module] = make(map[string]bool)
						}
						g.importedBy[imp.Module][f.Module] = true
					}
				}
			}

			for to := range oldImports {
				if _, stillImported := g.imports[moduleName][to]; !stillImported {
					if g.importedBy[to] != nil {
						delete(g.importedBy[to], moduleName)
					}
				}
			}
		}
	}

	g.fileCache.Evict(path)
	delete(g.fileToModule, path)
	delete(g.fileToLanguage, path)

	observability.GraphNodes.Set(float64(len(g.modules)))
	edgeCount := 0
	for _, targets := range g.imports {
		edgeCount += len(targets)
	}
	observability.GraphEdges.Set(float64(edgeCount))
}

func (g *Graph) GetModule(name string) (*Module, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	mod, ok := g.modules[name]
	if !ok {
		return nil, false
	}
	return cloneModule(mod), true
}

func (g *Graph) Modules() map[string]*Module {
	g.mu.RLock()
	defer g.mu.RUnlock()

	res := make(map[string]*Module, len(g.modules))
	for name, mod := range g.modules {
		res[name] = cloneModule(mod)
	}
	return res
}

func (g *Graph) ModuleCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.modules)
}

func (g *Graph) GetDefinitions(moduleName string) (map[string]*parser.Definition, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	defs, ok := g.definitions[moduleName]
	if !ok {
		return nil, false
	}
	res := make(map[string]*parser.Definition, len(defs))
	for k, v := range defs {
		res[k] = cloneDefinition(v)
	}
	return res, true
}

func (g *Graph) HasDefinitions(moduleName string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.definitions[moduleName]
	return ok
}

func (g *Graph) FileCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.fileToModule)
}

func (g *Graph) GetFile(path string) (*parser.File, bool) {
	g.mu.RLock()
	f, ok := g.fileCache.Get(path)
	loader := g.loader
	g.mu.RUnlock()

	if ok {
		return cloneFile(f), true
	}

	if loader != nil {
		// Attempt to reload from persistent store
		reloaded, err := loader.LoadFile(path)
		if err == nil && reloaded != nil {
			g.mu.Lock()
			g.fileCache.Put(path, cloneFile(reloaded))
			g.mu.Unlock()
			return reloaded, true
		}
	}

	return nil, false
}

func (g *Graph) GetAllFiles() []*parser.File {
	g.mu.RLock()
	defer g.mu.RUnlock()
	keys := g.fileCache.Keys()
	files := make([]*parser.File, 0, len(keys))
	for _, k := range keys {
		if f, ok := g.fileCache.Peek(k); ok {
			files = append(files, cloneFile(f))
		}
	}
	return files
}

func (g *Graph) GetImports() map[string]map[string]*ImportEdge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	res := make(map[string]map[string]*ImportEdge, len(g.imports))
	for from, targets := range g.imports {
		res[from] = make(map[string]*ImportEdge, len(targets))
		for to, edge := range targets {
			res[from][to] = cloneImportEdge(edge)
		}
	}
	return res
}

func (g *Graph) ComputeModuleMetrics() map[string]ModuleMetrics {
	g.mu.RLock()
	defer g.mu.RUnlock()

	moduleNames := make([]string, 0, len(g.modules))
	for name := range g.modules {
		moduleNames = append(moduleNames, name)
	}
	sort.Strings(moduleNames)

	adjacency := make(map[string][]string, len(moduleNames))
	for _, name := range moduleNames {
		targetSet := make(map[string]bool)
		for to := range g.imports[name] {
			if _, ok := g.modules[to]; ok {
				targetSet[to] = true
			}
		}
		targets := make([]string, 0, len(targetSet))
		for to := range targetSet {
			targets = append(targets, to)
		}
		sort.Strings(targets)
		adjacency[name] = targets
	}

	fanIn := make(map[string]int, len(moduleNames))
	fanOut := make(map[string]int, len(moduleNames))
	for _, from := range moduleNames {
		fanOut[from] = len(adjacency[from])
		for _, to := range adjacency[from] {
			fanIn[to]++
		}
	}

	componentOf, components := stronglyConnectedComponents(moduleNames, adjacency)
	componentEdges := make(map[int]map[int]bool, len(components))
	for _, from := range moduleNames {
		fromComp := componentOf[from]
		for _, to := range adjacency[from] {
			toComp := componentOf[to]
			if fromComp == toComp {
				continue
			}
			if componentEdges[fromComp] == nil {
				componentEdges[fromComp] = make(map[int]bool)
			}
			componentEdges[fromComp][toComp] = true
		}
	}

	depthByComp := make(map[int]int, len(components))
	var computeDepth func(int) int
	computeDepth = func(comp int) int {
		if depth, ok := depthByComp[comp]; ok {
			return depth
		}
		maxDepth := 0
		for next := range componentEdges[comp] {
			candidate := 1 + computeDepth(next)
			if candidate > maxDepth {
				maxDepth = candidate
			}
		}
		depthByComp[comp] = maxDepth
		return maxDepth
	}

	for comp := range components {
		computeDepth(comp)
	}

	// Compute max complexity score per module for the importance formula.
	metrics := make(map[string]ModuleMetrics, len(moduleNames))
	for _, name := range moduleNames {
		fi := fanIn[name]
		fo := fanOut[name]

		maxComplexity := 0
		if defs, ok := g.definitions[name]; ok {
			for _, def := range defs {
				sc := def.ComplexityScore
				if sc == 0 {
					sc = (def.BranchCount * 2) + (def.NestingDepth * 2) + def.ParameterCount + (def.LOC / 10)
				}
				if sc > maxComplexity {
					maxComplexity = sc
				}
			}
		}

		metrics[name] = ModuleMetrics{
			Depth:           depthByComp[componentOf[name]],
			FanIn:           fi,
			FanOut:          fo,
			ImportanceScore: CalculateImportanceScore(fi, fo, maxComplexity, name),
		}
	}

	return metrics
}

func (g *Graph) MarkDirty(paths []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, p := range paths {
		g.dirty[p] = true
	}
}

func (g *Graph) GetDirty() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	paths := make([]string, 0, len(g.dirty))
	for p := range g.dirty {
		paths = append(paths, p)
		delete(g.dirty, p)
	}
	return paths
}

func preferDefinition(candidate, existing *parser.Definition) bool {
	if candidate == nil {
		return false
	}
	if existing == nil {
		return true
	}

	candidateScore := complexityScore(candidate)
	existingScore := complexityScore(existing)
	if candidateScore != existingScore {
		return candidateScore > existingScore
	}
	if candidate.LOC != existing.LOC {
		return candidate.LOC > existing.LOC
	}
	if candidate.BranchCount != existing.BranchCount {
		return candidate.BranchCount > existing.BranchCount
	}
	if candidate.ParameterCount != existing.ParameterCount {
		return candidate.ParameterCount > existing.ParameterCount
	}
	if candidate.NestingDepth != existing.NestingDepth {
		return candidate.NestingDepth > existing.NestingDepth
	}
	return false
}

func complexityScore(def *parser.Definition) int {
	if def == nil {
		return 0
	}
	if def.ComplexityScore > 0 {
		return def.ComplexityScore
	}
	return (def.BranchCount * 2) + (def.NestingDepth * 2) + def.ParameterCount + (def.LOC / 10)
}

func cloneDefinition(def *parser.Definition) *parser.Definition {
	if def == nil {
		return nil
	}
	c := *def
	return &c
}

func cloneModule(mod *Module) *Module {
	if mod == nil {
		return nil
	}
	c := &Module{
		Name:     mod.Name,
		RootPath: mod.RootPath,
		Files:    append([]string(nil), mod.Files...),
		Exports:  make(map[string]*parser.Definition, len(mod.Exports)),
	}
	for k, v := range mod.Exports {
		c.Exports[k] = cloneDefinition(v)
	}
	return c
}

func cloneFile(file *parser.File) *parser.File {
	if file == nil {
		return nil
	}
	c := *file
	c.Imports = append([]parser.Import(nil), file.Imports...)
	c.Definitions = append([]parser.Definition(nil), file.Definitions...)
	for i := range c.Definitions {
		if len(c.Definitions[i].Decorators) == 0 {
			continue
		}
		c.Definitions[i].Decorators = append([]string(nil), c.Definitions[i].Decorators...)
	}
	c.References = append([]parser.Reference(nil), file.References...)
	c.Secrets = append([]parser.Secret(nil), file.Secrets...)
	c.LocalSymbols = append([]string(nil), file.LocalSymbols...)
	return &c
}

func cloneImportEdge(edge *ImportEdge) *ImportEdge {
	if edge == nil {
		return nil
	}
	c := *edge
	return &c
}

func stronglyConnectedComponents(nodes []string, adjacency map[string][]string) (map[string]int, [][]string) {
	index := 0
	stack := make([]string, 0, len(nodes))
	onStack := make(map[string]bool, len(nodes))
	indexByNode := make(map[string]int, len(nodes))
	lowLink := make(map[string]int, len(nodes))
	componentOf := make(map[string]int, len(nodes))
	components := make([][]string, 0)

	var strongConnect func(string)
	strongConnect = func(v string) {
		indexByNode[v] = index
		lowLink[v] = index
		index++

		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adjacency[v] {
			if _, seen := indexByNode[w]; !seen {
				strongConnect(w)
				if lowLink[w] < lowLink[v] {
					lowLink[v] = lowLink[w]
				}
			} else if onStack[w] && indexByNode[w] < lowLink[v] {
				lowLink[v] = indexByNode[w]
			}
		}

		if lowLink[v] != indexByNode[v] {
			return
		}

		component := make([]string, 0)
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == v {
				break
			}
		}
		sort.Strings(component)
		compID := len(components)
		components = append(components, component)
		for _, n := range component {
			componentOf[n] = compID
		}
	}

	for _, node := range nodes {
		if _, seen := indexByNode[node]; !seen {
			strongConnect(node)
		}
	}

	return componentOf, components
}
