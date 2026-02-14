package formats

import (
	"circular/internal/engine/graph"
	"circular/internal/shared/util"
	"fmt"
	"strings"
)

type PlantUMLGenerator struct {
	graph   *graph.Graph
	metrics map[string]graph.ModuleMetrics
	hotspot map[string]int
}

func NewPlantUMLGenerator(g *graph.Graph) *PlantUMLGenerator {
	return &PlantUMLGenerator{graph: g}
}

func (p *PlantUMLGenerator) SetModuleMetrics(metrics map[string]graph.ModuleMetrics) {
	if len(metrics) == 0 {
		p.metrics = nil
		return
	}
	p.metrics = make(map[string]graph.ModuleMetrics, len(metrics))
	for mod, metric := range metrics {
		p.metrics[mod] = metric
	}
}

func (p *PlantUMLGenerator) SetComplexityHotspots(hotspots []graph.ComplexityHotspot) {
	if len(hotspots) == 0 {
		p.hotspot = nil
		return
	}
	p.hotspot = make(map[string]int, len(hotspots))
	for _, h := range hotspots {
		if current, ok := p.hotspot[h.Module]; !ok || h.Score > current {
			p.hotspot[h.Module] = h.Score
		}
	}
}

func (p *PlantUMLGenerator) Generate(cycles [][]string, violations []graph.ArchitectureViolation, model graph.ArchitectureModel) (string, error) {
	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("skinparam componentStyle rectangle\n")
	b.WriteString("skinparam packageStyle rectangle\n")
	b.WriteString("skinparam linetype ortho\n")
	b.WriteString("skinparam nodesep 80\n")
	b.WriteString("skinparam ranksep 100\n")
	b.WriteString("left to right direction\n\n")

	modules := p.graph.Modules()
	imports := p.graph.GetImports()
	moduleNames := util.SortedStringKeys(modules)
	moduleSet := make(map[string]bool, len(moduleNames))
	for _, name := range moduleNames {
		moduleSet[name] = true
	}

	externalSet := make(map[string]bool)
	for _, targets := range imports {
		for to := range targets {
			if !moduleSet[to] {
				externalSet[to] = true
			}
		}
	}
	externalNames := util.SortedStringKeys(externalSet)
	aggregateExternal := len(externalNames) > externalAggregationThreshold

	allNames := append(append([]string{}, moduleNames...), externalNames...)
	if aggregateExternal {
		allNames = append(allNames, externalAggregateNodeID)
	}
	aliases := makeIDs(allNames)
	cycleEdges := cycleEdgeSet(cycles)
	violationEdges := violationEdgeSet(violations)
	layerByModule := classifyLayers(moduleNames, modules, model)
	externalEdgeCounts := countExternalEdges(imports, moduleSet)

	if model.Enabled && len(model.Layers) > 0 {
		for _, layer := range model.Layers {
			layerModules := modulesInLayer(moduleNames, layerByModule, layer.Name)
			if len(layerModules) == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("package \"%s\" {\n", escapeLabel(layer.Name)))
			for _, modName := range layerModules {
				b.WriteString(fmt.Sprintf("  component \"%s\" as %s\n", escapeLabel(moduleLabel(modName, modules[modName], p.metrics, p.hotspot)), aliases[modName]))
			}
			b.WriteString("}\n")
		}
		unlayered := modulesInLayer(moduleNames, layerByModule, "")
		for _, modName := range unlayered {
			b.WriteString(fmt.Sprintf("component \"%s\" as %s\n", escapeLabel(moduleLabel(modName, modules[modName], p.metrics, p.hotspot)), aliases[modName]))
		}
	} else {
		for _, modName := range moduleNames {
			b.WriteString(fmt.Sprintf("component \"%s\" as %s\n", escapeLabel(moduleLabel(modName, modules[modName], p.metrics, p.hotspot)), aliases[modName]))
		}
	}

	if aggregateExternal {
		b.WriteString(fmt.Sprintf("component \"External/Stdlib\\n(%d modules)\" as %s #DDDDDD\n", len(externalNames), aliases[externalAggregateNodeID]))
	} else {
		for _, modName := range externalNames {
			b.WriteString(fmt.Sprintf("component \"%s\" as %s #DDDDDD\n", escapeLabel(modName), aliases[modName]))
		}
	}

	b.WriteString("\n")
	for _, from := range util.SortedStringKeys(imports) {
		targets := util.SortedStringKeys(imports[from])
		for _, to := range targets {
			if aggregateExternal && !moduleSet[to] {
				continue
			}
			label := ""
			arrow := "-->"
			if cycleEdges[from+"->"+to] {
				label = " : CYCLE"
				arrow = "-[#red,thickness=2]->"
			} else if violationEdges[from+"->"+to] {
				label = " : VIOLATION"
				arrow = "-[#a64d00,dashed]->"
			} else if !moduleSet[to] {
				arrow = "-[#777777,dashed]->"
			}
			b.WriteString(fmt.Sprintf("%s %s %s%s\n", aliases[from], arrow, aliases[to], label))
		}
	}
	if aggregateExternal {
		for _, from := range moduleNames {
			count := externalEdgeCounts[from]
			if count == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("%s -[#777777,dashed]-> %s : ext:%d\n", aliases[from], aliases[externalAggregateNodeID], count))
		}
	}

	b.WriteString("\nlegend right\n")
	b.WriteString("|= Item |= Meaning |\n")
	b.WriteString("|Node line 1|Module name|\n")
	b.WriteString("|Node line 2|Function/export count and file count|\n")
	b.WriteString("|d|Dependency depth|\n")
	b.WriteString("|in|Fan-in (number of internal modules importing this module)|\n")
	b.WriteString("|out|Fan-out (number of internal modules this module imports)|\n")
	b.WriteString("|cx|Top complexity hotspot score in the module|\n")
	if len(externalNames) > 0 {
		b.WriteString("|<color:#DDDDDD>Component</color>|External module|\n")
	}
	if len(cycleEdges) > 0 {
		b.WriteString("|<color:#cc0000>Red edge</color>|Cycle edge|\n")
	}
	if len(violationEdges) > 0 {
		b.WriteString("|<color:#a64d00>Brown dashed edge</color>|Architecture violation edge|\n")
	}
	b.WriteString("|ext:N|Count of external dependencies from that module (aggregated mode)|\n")
	b.WriteString("endlegend\n")

	b.WriteString("\n@enduml\n")
	return b.String(), nil
}

func (p *PlantUMLGenerator) GenerateArchitecture(model graph.ArchitectureModel, violations []graph.ArchitectureViolation) (string, error) {
	if !model.Enabled || len(model.Layers) == 0 {
		return "", fmt.Errorf("architecture diagram mode requires architecture.enabled=true with at least one layer")
	}

	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("skinparam componentStyle rectangle\n")
	b.WriteString("skinparam packageStyle rectangle\n")
	b.WriteString("skinparam linetype ortho\n")
	b.WriteString("skinparam nodesep 80\n")
	b.WriteString("skinparam ranksep 100\n")
	b.WriteString("left to right direction\n\n")

	layers, deps := architectureLayerDependencies(p.graph, model, violations)
	ids := makeIDs(layers)
	for _, layer := range layers {
		b.WriteString(fmt.Sprintf("rectangle \"%s\" as %s\n", escapeLabel(layer), ids[layer]))
	}

	b.WriteString("\n")
	for _, dep := range deps {
		arrow := "-->"
		label := fmt.Sprintf(" : deps:%d", dep.Count)
		if dep.Violations > 0 {
			arrow = "-[#a64d00,dashed]->"
			label = fmt.Sprintf(" : deps:%d viol:%d", dep.Count, dep.Violations)
		}
		b.WriteString(fmt.Sprintf("%s %s %s%s\n", ids[dep.From], arrow, ids[dep.To], label))
	}

	b.WriteString("\nlegend right\n")
	b.WriteString("|= Item |= Meaning |\n")
	b.WriteString("|Rectangle|Architecture layer|\n")
	b.WriteString("|deps:N|Observed inter-layer dependency count|\n")
	b.WriteString("|viol:M|Violating dependency count for that layer pair|\n")
	b.WriteString("|<color:#a64d00>Brown dashed edge</color>|Layer pair with architecture violations|\n")
	b.WriteString("endlegend\n")
	b.WriteString("\n@enduml\n")
	return b.String(), nil
}

func (p *PlantUMLGenerator) GenerateComponent(model graph.ArchitectureModel, showInternal bool) (string, error) {
	data := buildComponentDiagramData(p.graph, showInternal)
	moduleAliases := makeIDs(data.ModuleNames)

	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("skinparam componentStyle rectangle\n")
	b.WriteString("skinparam packageStyle rectangle\n")
	b.WriteString("skinparam linetype ortho\n")
	b.WriteString("skinparam nodesep 80\n")
	b.WriteString("skinparam ranksep 100\n")
	b.WriteString("left to right direction\n\n")

	layerByModule := classifyLayers(data.ModuleNames, data.Modules, model)
	if model.Enabled && len(model.Layers) > 0 {
		for _, layer := range model.Layers {
			layerModules := modulesInLayer(data.ModuleNames, layerByModule, layer.Name)
			if len(layerModules) == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("package \"%s\" {\n", escapeLabel(layer.Name)))
			for _, moduleName := range layerModules {
				b.WriteString(fmt.Sprintf("  component \"%s\" as %s\n", escapeLabel(moduleLabel(moduleName, data.Modules[moduleName], p.metrics, p.hotspot)), moduleAliases[moduleName]))
			}
			b.WriteString("}\n")
		}
		for _, moduleName := range modulesInLayer(data.ModuleNames, layerByModule, "") {
			b.WriteString(fmt.Sprintf("component \"%s\" as %s\n", escapeLabel(moduleLabel(moduleName, data.Modules[moduleName], p.metrics, p.hotspot)), moduleAliases[moduleName]))
		}
	} else {
		for _, moduleName := range data.ModuleNames {
			b.WriteString(fmt.Sprintf("component \"%s\" as %s\n", escapeLabel(moduleLabel(moduleName, data.Modules[moduleName], p.metrics, p.hotspot)), moduleAliases[moduleName]))
		}
	}

	if showInternal {
		for _, moduleName := range data.ModuleNames {
			for _, definition := range data.Definitions[moduleName] {
				defAlias := sanitizeID(moduleName + "__" + definition)
				b.WriteString(fmt.Sprintf("component \"%s\" as %s <<symbol>>\n", escapeLabel(definition), defAlias))
				b.WriteString(fmt.Sprintf("%s ..> %s\n", moduleAliases[moduleName], defAlias))
			}
		}
	}

	b.WriteString("\n")
	for _, edge := range data.Edges {
		arrow := "-->"
		labelParts := make([]string, 0, 3)
		if edge.Imports > 0 {
			labelParts = append(labelParts, fmt.Sprintf("deps:%d", edge.Imports))
		}
		if edge.SymbolRefs > 0 {
			labelParts = append(labelParts, fmt.Sprintf("refs:%d", edge.SymbolRefs))
			arrow = "-[#1f6f8b,dashed]->"
		}
		if showInternal && len(edge.Symbols) > 0 {
			preview := edge.Symbols
			if len(preview) > 3 {
				preview = preview[:3]
			}
			labelParts = append(labelParts, "sym:"+strings.Join(preview, ","))
		}
		label := ""
		if len(labelParts) > 0 {
			label = " : " + strings.Join(labelParts, " ")
		}
		b.WriteString(fmt.Sprintf("%s %s %s%s\n", moduleAliases[edge.From], arrow, moduleAliases[edge.To], label))
	}

	b.WriteString("\nlegend right\n")
	b.WriteString("|= Item |= Meaning |\n")
	b.WriteString("|Component|Module with metrics (func/file counts and optional d/in/out/cx)|\n")
	b.WriteString("|deps:N|Import edges observed between source and target modules|\n")
	b.WriteString("|refs:M|Matched symbol references from source module to target module definitions|\n")
	if showInternal {
		b.WriteString("|<<symbol>>|Definition node shown when `show_internal=true`|\n")
		b.WriteString("|sym:a,b,c|Example referenced definitions for that edge (preview)|\n")
	}
	b.WriteString("|<color:#1f6f8b>Dashed edge</color>|Edge contains matched symbol references|\n")
	b.WriteString("endlegend\n")

	b.WriteString("\n@enduml\n")
	return b.String(), nil
}

func (p *PlantUMLGenerator) GenerateFlow(entryPoints []string, maxDepth int) (string, error) {
	data, err := buildFlowDiagramData(p.graph, entryPoints, maxDepth)
	if err != nil {
		return "", err
	}

	nodeNames := make([]string, 0, len(data.Nodes))
	for _, node := range data.Nodes {
		nodeNames = append(nodeNames, node.Name)
	}
	aliases := makeIDs(nodeNames)

	var b strings.Builder
	b.WriteString("@startuml\n")
	b.WriteString("skinparam componentStyle rectangle\n")
	b.WriteString("skinparam packageStyle rectangle\n")
	b.WriteString("skinparam linetype ortho\n")
	b.WriteString("skinparam nodesep 80\n")
	b.WriteString("skinparam ranksep 100\n")
	b.WriteString("left to right direction\n\n")

	for _, node := range data.Nodes {
		color := ""
		if node.Entry {
			color = " #E9F5EC"
		}
		b.WriteString(fmt.Sprintf("component \"%s\\n(step:%d)\" as %s%s\n", escapeLabel(node.Name), node.Depth, aliases[node.Name], color))
	}

	b.WriteString("\n")
	for _, edge := range data.Edges {
		b.WriteString(fmt.Sprintf("%s --> %s\n", aliases[edge.From], aliases[edge.To]))
	}

	b.WriteString("\nlegend right\n")
	b.WriteString("|= Item |= Meaning |\n")
	b.WriteString("|step:N|Shortest hop distance from nearest entry point|\n")
	b.WriteString("|Green component|Selected flow entry module|\n")
	b.WriteString("endlegend\n")
	b.WriteString("\n@enduml\n")
	return b.String(), nil
}
