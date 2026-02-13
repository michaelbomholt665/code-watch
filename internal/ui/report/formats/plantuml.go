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
