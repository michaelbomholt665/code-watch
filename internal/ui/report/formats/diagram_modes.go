package formats

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/shared/util"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const maxDefinitionNodesPerModule = 12

type componentEdge struct {
	From       string
	To         string
	Imports    int
	SymbolRefs int
	Symbols    []string
}

type componentDiagramData struct {
	Modules     map[string]*graph.Module
	ModuleNames []string
	Definitions map[string][]string
	Edges       []componentEdge
}

type flowNode struct {
	Name  string
	Depth int
	Entry bool
}

type flowEdge struct {
	From string
	To   string
}

type flowDiagramData struct {
	Nodes []flowNode
	Edges []flowEdge
}

func buildComponentDiagramData(g *graph.Graph, showInternal bool) componentDiagramData {
	modules := g.Modules()
	moduleNames := util.SortedStringKeys(modules)
	moduleSet := make(map[string]bool, len(moduleNames))
	for _, name := range moduleNames {
		moduleSet[name] = true
	}

	defLookup := make(map[string]map[string]bool, len(moduleNames))
	definitionNames := make(map[string][]string, len(moduleNames))
	for _, moduleName := range moduleNames {
		defs, _ := g.GetDefinitions(moduleName)
		defLookup[moduleName] = make(map[string]bool, len(defs))
		names := make([]string, 0, len(defs))
		for name := range defs {
			defLookup[moduleName][name] = true
			if showInternal {
				names = append(names, name)
			}
		}
		if showInternal {
			sort.Strings(names)
			if len(names) > maxDefinitionNodesPerModule {
				names = names[:maxDefinitionNodesPerModule]
			}
			definitionNames[moduleName] = names
		}
	}

	importCounts := make(map[string]int)
	imports := g.GetImports()
	for _, from := range util.SortedStringKeys(imports) {
		if !moduleSet[from] {
			continue
		}
		targets := util.SortedStringKeys(imports[from])
		for _, to := range targets {
			if !moduleSet[to] || from == to {
				continue
			}
			importCounts[from+"->"+to]++
		}
	}

	symbolRefCounts := make(map[string]int)
	symbolsByEdge := make(map[string]map[string]bool)
	files := g.GetAllFiles()
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	for _, file := range files {
		if file == nil || !moduleSet[file.Module] {
			continue
		}

		aliasToModule := make(map[string]string)
		directItemToModules := make(map[string][]string)
		for _, imp := range file.Imports {
			if imp.Module == "" || !moduleSet[imp.Module] {
				continue
			}
			alias := strings.TrimSpace(imp.Alias)
			if alias == "" {
				alias = parser.ModuleReferenceBase(file.Language, imp.Module)
			}
			if alias != "" && alias != "_" && alias != "." {
				if _, exists := aliasToModule[alias]; !exists {
					aliasToModule[alias] = imp.Module
				}
			}
			for _, item := range imp.Items {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				directItemToModules[item] = appendUniqueString(directItemToModules[item], imp.Module)
			}
		}
		for item := range directItemToModules {
			sort.Strings(directItemToModules[item])
		}

		for _, ref := range file.References {
			targetModule, symbol := resolveReferenceTarget(ref.Name, aliasToModule, directItemToModules)
			if targetModule == "" || symbol == "" || targetModule == file.Module {
				continue
			}
			if !defLookup[targetModule][symbol] {
				continue
			}
			key := file.Module + "->" + targetModule
			symbolRefCounts[key]++
			if symbolsByEdge[key] == nil {
				symbolsByEdge[key] = make(map[string]bool)
			}
			symbolsByEdge[key][symbol] = true
		}
	}

	keysSet := make(map[string]bool, len(importCounts)+len(symbolRefCounts))
	for key := range importCounts {
		keysSet[key] = true
	}
	for key := range symbolRefCounts {
		keysSet[key] = true
	}
	keys := util.SortedStringKeys(keysSet)
	edges := make([]componentEdge, 0, len(keys))
	for _, key := range keys {
		parts := strings.SplitN(key, "->", 2)
		if len(parts) != 2 {
			continue
		}
		symbolNames := make([]string, 0, len(symbolsByEdge[key]))
		for symbol := range symbolsByEdge[key] {
			symbolNames = append(symbolNames, symbol)
		}
		sort.Strings(symbolNames)
		edges = append(edges, componentEdge{
			From:       parts[0],
			To:         parts[1],
			Imports:    importCounts[key],
			SymbolRefs: symbolRefCounts[key],
			Symbols:    symbolNames,
		})
	}

	return componentDiagramData{
		Modules:     modules,
		ModuleNames: moduleNames,
		Definitions: definitionNames,
		Edges:       edges,
	}
}

func buildFlowDiagramData(g *graph.Graph, entryPoints []string, maxDepth int) (flowDiagramData, error) {
	if maxDepth < 1 {
		return flowDiagramData{}, fmt.Errorf("flow diagram max depth must be >= 1")
	}

	modules := g.Modules()
	moduleNames := util.SortedStringKeys(modules)
	moduleSet := make(map[string]bool, len(moduleNames))
	for _, moduleName := range moduleNames {
		moduleSet[moduleName] = true
	}
	if len(moduleNames) == 0 {
		return flowDiagramData{}, fmt.Errorf("flow diagram requires at least one module")
	}

	startSet := resolveFlowEntryModules(g.GetAllFiles(), moduleSet, entryPoints)
	if len(startSet) == 0 {
		startSet = defaultFlowEntryModules(g.GetImports(), moduleSet)
	}
	if len(startSet) == 0 {
		startSet[moduleNames[0]] = true
	}

	starts := util.SortedStringKeys(startSet)
	depthByModule := make(map[string]int, len(moduleNames))
	queue := make([]string, 0, len(starts))
	for _, start := range starts {
		depthByModule[start] = 0
		queue = append(queue, start)
	}

	imports := g.GetImports()
	edgeSet := make(map[string]bool)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		currentDepth := depthByModule[current]
		if currentDepth >= maxDepth {
			continue
		}

		targets := make([]string, 0, len(imports[current]))
		for target := range imports[current] {
			if moduleSet[target] {
				targets = append(targets, target)
			}
		}
		sort.Strings(targets)

		for _, target := range targets {
			edgeSet[current+"->"+target] = true
			nextDepth := currentDepth + 1
			if previous, seen := depthByModule[target]; !seen || nextDepth < previous {
				depthByModule[target] = nextDepth
				queue = append(queue, target)
			}
		}
	}

	flowNodes := make([]flowNode, 0, len(depthByModule))
	for _, moduleName := range util.SortedStringKeys(depthByModule) {
		flowNodes = append(flowNodes, flowNode{
			Name:  moduleName,
			Depth: depthByModule[moduleName],
			Entry: startSet[moduleName],
		})
	}

	flowEdges := make([]flowEdge, 0, len(edgeSet))
	for _, key := range util.SortedStringKeys(edgeSet) {
		parts := strings.SplitN(key, "->", 2)
		if len(parts) != 2 {
			continue
		}
		flowEdges = append(flowEdges, flowEdge{From: parts[0], To: parts[1]})
	}

	return flowDiagramData{Nodes: flowNodes, Edges: flowEdges}, nil
}

func defaultFlowEntryModules(imports map[string]map[string]*graph.ImportEdge, moduleSet map[string]bool) map[string]bool {
	fanIn := make(map[string]int, len(moduleSet))
	for moduleName := range moduleSet {
		fanIn[moduleName] = 0
	}
	for from := range imports {
		if !moduleSet[from] {
			continue
		}
		for to := range imports[from] {
			if moduleSet[to] {
				fanIn[to]++
			}
		}
	}

	roots := make(map[string]bool)
	for moduleName, count := range fanIn {
		if count == 0 {
			roots[moduleName] = true
		}
	}
	return roots
}

func resolveFlowEntryModules(files []*parser.File, moduleSet map[string]bool, entryPoints []string) map[string]bool {
	entries := make([]string, 0, len(entryPoints))
	for _, entry := range entryPoints {
		trimmed := strings.TrimSpace(entry)
		if trimmed != "" {
			entries = append(entries, filepath.ToSlash(filepath.Clean(trimmed)))
		}
	}

	result := make(map[string]bool)
	for _, entry := range entries {
		if moduleSet[entry] {
			result[entry] = true
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	for _, file := range files {
		if file == nil || !moduleSet[file.Module] {
			continue
		}
		normalizedPath := filepath.ToSlash(filepath.Clean(file.Path))
		base := filepath.Base(normalizedPath)
		for _, entry := range entries {
			if normalizedPath == entry ||
				strings.HasSuffix(normalizedPath, "/"+entry) ||
				strings.HasSuffix(normalizedPath, entry) ||
				base == entry {
				result[file.Module] = true
			}
		}
	}
	return result
}

func resolveReferenceTarget(name string, aliasToModule map[string]string, directItemToModules map[string][]string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}

	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		filtered := parts[:0]
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				filtered = append(filtered, part)
			}
		}
		if len(filtered) >= 2 {
			if module := aliasToModule[filtered[0]]; module != "" {
				return module, filtered[len(filtered)-1]
			}
		}
	}

	if modules := directItemToModules[name]; len(modules) > 0 {
		return modules[0], name
	}
	return "", ""
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
