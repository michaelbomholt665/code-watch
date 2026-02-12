// # internal/graph/graph.go
package graph

import (
	"circular/internal/parser"
	"sync"
)

type Graph struct {
	mu sync.RWMutex

	// Core data
	files   map[string]*parser.File // path -> file
	modules map[string]*Module      // module name -> module info

	// Relationships
	imports    map[string]map[string]*ImportEdge // from -> to -> edge
	importedBy map[string]map[string]bool        // to -> from

	// Symbol tables (for hallucination detection)
	definitions map[string]map[string]*parser.Definition // module -> symbol -> def

	// Invalidation tracking
	dirty map[string]bool // Files needing re-analysis
}

type Module struct {
	Name     string
	Files    []string // Paths to files in this module
	Exports  map[string]*parser.Definition
	RootPath string // For Go: module root, Python: package root
}

type ImportEdge struct {
	From       string
	To         string
	ImportedBy string // File path
	Location   parser.Location
}

func NewGraph() *Graph {
	return &Graph{
		files:       make(map[string]*parser.File),
		modules:     make(map[string]*Module),
		imports:     make(map[string]map[string]*ImportEdge),
		importedBy:  make(map[string]map[string]bool),
		definitions: make(map[string]map[string]*parser.Definition),
		dirty:       make(map[string]bool),
	}
}

func (g *Graph) AddFile(file *parser.File) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// If this file already exists, remove prior contributions first.
	// This prevents stale imports/definitions after file edits.
	if _, exists := g.files[file.Path]; exists {
		g.removeFileLocked(file.Path)
	}

	g.files[file.Path] = cloneFile(file)

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
	for i := range file.Definitions {
		def := cloneDefinition(&file.Definitions[i])
		if def.Exported {
			mod.Exports[def.Name] = def
		}
		g.definitions[file.Module][def.Name] = def
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
}

func (g *Graph) RemoveFile(path string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.removeFileLocked(path)
}

func (g *Graph) removeFileLocked(path string) {
	file, ok := g.files[path]
	if !ok {
		return
	}

	if mod, ok := g.modules[file.Module]; ok {
		for i, p := range mod.Files {
			if p == path {
				mod.Files = append(mod.Files[:i], mod.Files[i+1:]...)
				break
			}
		}

		if len(mod.Files) == 0 {
			for to := range g.imports[file.Module] {
				if g.importedBy[to] != nil {
					delete(g.importedBy[to], file.Module)
				}
			}

			delete(g.modules, file.Module)
			delete(g.imports, file.Module)
			delete(g.definitions, file.Module)
		} else {
			mod.Exports = make(map[string]*parser.Definition)
			g.definitions[file.Module] = make(map[string]*parser.Definition)

			oldImports := g.imports[file.Module]
			g.imports[file.Module] = make(map[string]*ImportEdge)

			for _, filePath := range mod.Files {
				if f, ok := g.files[filePath]; ok {
					for i := range f.Definitions {
						def := cloneDefinition(&f.Definitions[i])
						if def.Exported {
							mod.Exports[def.Name] = def
						}
						g.definitions[file.Module][def.Name] = def
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
				if _, stillImported := g.imports[file.Module][to]; !stillImported {
					if g.importedBy[to] != nil {
						delete(g.importedBy[to], file.Module)
					}
				}
			}
		}
	}

	delete(g.files, path)
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

func (g *Graph) FileCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.files)
}

func (g *Graph) GetFile(path string) (*parser.File, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	f, ok := g.files[path]
	if !ok {
		return nil, false
	}
	return cloneFile(f), true
}

func (g *Graph) GetAllFiles() []*parser.File {
	g.mu.RLock()
	defer g.mu.RUnlock()
	files := make([]*parser.File, 0, len(g.files))
	for _, f := range g.files {
		files = append(files, cloneFile(f))
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
	c.References = append([]parser.Reference(nil), file.References...)
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
