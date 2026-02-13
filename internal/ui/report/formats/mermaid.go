package formats

import (
	"circular/internal/engine/graph"
	"circular/internal/shared/util"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type MermaidGenerator struct {
	graph   *graph.Graph
	metrics map[string]graph.ModuleMetrics
	hotspot map[string]int
}

const externalAggregationThreshold = 10

func NewMermaidGenerator(g *graph.Graph) *MermaidGenerator {
	return &MermaidGenerator{graph: g}
}

func (m *MermaidGenerator) SetModuleMetrics(metrics map[string]graph.ModuleMetrics) {
	if len(metrics) == 0 {
		m.metrics = nil
		return
	}
	m.metrics = make(map[string]graph.ModuleMetrics, len(metrics))
	for mod, metric := range metrics {
		m.metrics[mod] = metric
	}
}

func (m *MermaidGenerator) SetComplexityHotspots(hotspots []graph.ComplexityHotspot) {
	if len(hotspots) == 0 {
		m.hotspot = nil
		return
	}
	m.hotspot = make(map[string]int, len(hotspots))
	for _, h := range hotspots {
		if current, ok := m.hotspot[h.Module]; !ok || h.Score > current {
			m.hotspot[h.Module] = h.Score
		}
	}
}

func (m *MermaidGenerator) Generate(cycles [][]string, violations []graph.ArchitectureViolation, model graph.ArchitectureModel) (string, error) {
	var b strings.Builder
	b.WriteString("%%{init: {'theme': 'base', 'themeVariables': {'textColor': '#000000', 'primaryTextColor': '#000000', 'lineColor': '#333333'}, 'flowchart': {'nodeSpacing': 80, 'rankSpacing': 110, 'curve': 'basis'}}}%%\n")
	b.WriteString("flowchart LR\n")

	modules := m.graph.Modules()
	imports := m.graph.GetImports()
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
	ids := makeIDs(allNames)

	cycleEdges := cycleEdgeSet(cycles)
	violationEdges := violationEdgeSet(violations)
	cycleModules := cycleModuleSet(cycles)
	layerByModule := classifyLayers(moduleNames, modules, model)

	if model.Enabled && len(model.Layers) > 0 {
		for _, layer := range model.Layers {
			layerModules := modulesInLayer(moduleNames, layerByModule, layer.Name)
			if len(layerModules) == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("  subgraph layer_%s[\"%s\"]\n", sanitizeID(layer.Name), escapeLabel(layer.Name)))
			for _, modName := range layerModules {
				b.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", ids[modName], escapeLabel(moduleLabel(modName, modules[modName], m.metrics, m.hotspot))))
			}
			b.WriteString("  end\n")
		}

		unlayered := modulesInLayer(moduleNames, layerByModule, "")
		for _, modName := range unlayered {
			b.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", ids[modName], escapeLabel(moduleLabel(modName, modules[modName], m.metrics, m.hotspot))))
		}
	} else {
		for _, modName := range moduleNames {
			b.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", ids[modName], escapeLabel(moduleLabel(modName, modules[modName], m.metrics, m.hotspot))))
		}
	}

	externalEdgeCounts := countExternalEdges(imports, moduleSet)
	if aggregateExternal {
		b.WriteString(fmt.Sprintf("  %s[\"External/Stdlib\\n(%d modules)\"]\n", ids[externalAggregateNodeID], len(externalNames)))
	} else {
		for _, modName := range externalNames {
			b.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", ids[modName], escapeLabel(modName)))
		}
	}

	b.WriteString("\n")
	if len(moduleNames) > 0 {
		b.WriteString("  classDef internalNode fill:#f7fbff,stroke:#4d6480,stroke-width:1px,color:#000000;\n")
		b.WriteString("  class ")
		b.WriteString(strings.Join(toIDs(moduleNames, ids), ","))
		b.WriteString(" internalNode;\n")
	}
	if len(externalNames) > 0 {
		b.WriteString("  classDef externalNode fill:#efefef,stroke:#808080,stroke-dasharray:4 3,color:#000000;\n")
		if aggregateExternal {
			b.WriteString(fmt.Sprintf("  class %s externalNode;\n", ids[externalAggregateNodeID]))
		} else {
			b.WriteString("  class ")
			b.WriteString(strings.Join(toIDs(externalNames, ids), ","))
			b.WriteString(" externalNode;\n")
		}
	}
	if len(cycleModules) > 0 {
		cycleNames := intersectOrdered(moduleNames, cycleModules)
		if len(cycleNames) > 0 {
			b.WriteString("  classDef cycleNode fill:#ffecec,stroke:#cc0000,stroke-width:2px,color:#000000;\n")
			b.WriteString("  class ")
			b.WriteString(strings.Join(toIDs(cycleNames, ids), ","))
			b.WriteString(" cycleNode;\n")
		}
	}
	if len(m.hotspot) > 0 {
		hotspotNames := make([]string, 0, len(m.hotspot))
		for name := range m.hotspot {
			if moduleSet[name] {
				hotspotNames = append(hotspotNames, name)
			}
		}
		sort.Strings(hotspotNames)
		if len(hotspotNames) > 0 {
			b.WriteString("  classDef hotspotNode stroke:#8a4f00,stroke-width:2px,color:#000000;\n")
			b.WriteString("  class ")
			b.WriteString(strings.Join(toIDs(hotspotNames, ids), ","))
			b.WriteString(" hotspotNode;\n")
		}
	}

	b.WriteString("\n")
	linkIndex := 0
	cycleLinkIndexes := make([]int, 0)
	violationLinkIndexes := make([]int, 0)
	externalLinkIndexes := make([]int, 0)
	for _, from := range util.SortedStringKeys(imports) {
		targets := util.SortedStringKeys(imports[from])
		for _, to := range targets {
			if aggregateExternal && !moduleSet[to] {
				continue
			}
			edgeLabel := ""
			if cycleEdges[from+"->"+to] {
				edgeLabel = "|CYCLE|"
				cycleLinkIndexes = append(cycleLinkIndexes, linkIndex)
			} else if violationEdges[from+"->"+to] {
				edgeLabel = "|VIOLATION|"
				violationLinkIndexes = append(violationLinkIndexes, linkIndex)
			} else if !moduleSet[to] {
				externalLinkIndexes = append(externalLinkIndexes, linkIndex)
			}
			b.WriteString(fmt.Sprintf("  %s -->%s %s\n", ids[from], edgeLabel, ids[to]))
			linkIndex++
		}
	}
	if aggregateExternal {
		for _, from := range moduleNames {
			count := externalEdgeCounts[from]
			if count == 0 {
				continue
			}
			b.WriteString(fmt.Sprintf("  %s -->|ext:%d| %s\n", ids[from], count, ids[externalAggregateNodeID]))
			externalLinkIndexes = append(externalLinkIndexes, linkIndex)
			linkIndex++
		}
	}

	if len(cycleLinkIndexes) > 0 || len(violationLinkIndexes) > 0 || len(externalLinkIndexes) > 0 {
		b.WriteString("\n")
	}
	if len(cycleLinkIndexes) > 0 {
		b.WriteString(fmt.Sprintf("  linkStyle %s stroke:#cc0000,stroke-width:3px;\n", joinInts(cycleLinkIndexes)))
	}
	if len(violationLinkIndexes) > 0 {
		b.WriteString(fmt.Sprintf("  linkStyle %s stroke:#a64d00,stroke-width:2px,stroke-dasharray:5 3;\n", joinInts(violationLinkIndexes)))
	}
	if len(externalLinkIndexes) > 0 {
		b.WriteString(fmt.Sprintf("  linkStyle %s stroke:#777777,stroke-dasharray:4 3;\n", joinInts(externalLinkIndexes)))
	}
	b.WriteString("\n")
	b.WriteString("  subgraph legend_info[\"Legend\"]\n")
	b.WriteString("    legend_metrics[\"Node line 1: module\\nline 2: funcs/files\\n(d=depth in=fan-in out=fan-out)\\n(cx=complexity hotspot score)\"]\n")
	b.WriteString("    legend_edges[\"Edge labels: CYCLE=import cycle, VIOLATION=architecture rule violation, ext:N=external dependency count\"]\n")
	b.WriteString("  end\n")
	b.WriteString("  classDef legendNode fill:#fff8dc,stroke:#b8a24c,stroke-width:1px,color:#000000;\n")
	b.WriteString("  class legend_metrics,legend_edges legendNode;\n")

	return b.String(), nil
}

func classifyLayers(moduleNames []string, modules map[string]*graph.Module, model graph.ArchitectureModel) map[string]string {
	layerByModule := make(map[string]string, len(moduleNames))
	if !model.Enabled || len(model.Layers) == 0 {
		for _, name := range moduleNames {
			layerByModule[name] = ""
		}
		return layerByModule
	}

	for _, name := range moduleNames {
		mod := modules[name]
		samplePath := ""
		if mod != nil && len(mod.Files) > 0 {
			files := append([]string(nil), mod.Files...)
			sort.Strings(files)
			samplePath = util.NormalizePatternPath(files[0])
		}
		modulePath := util.NormalizePatternPath(name)
		bestLayer := ""
		bestScore := 0

		for _, layer := range model.Layers {
			for _, raw := range layer.Paths {
				pattern := util.NormalizePatternPath(raw)
				if pattern == "" {
					continue
				}
				if matchesPattern(pattern, modulePath, samplePath) {
					score := len(pattern)
					if score > bestScore || (score == bestScore && layer.Name < bestLayer) {
						bestLayer = layer.Name
						bestScore = score
					}
				}
			}
		}
		layerByModule[name] = bestLayer
	}
	return layerByModule
}

func modulesInLayer(moduleNames []string, layerByModule map[string]string, layer string) []string {
	mods := make([]string, 0)
	for _, mod := range moduleNames {
		if layerByModule[mod] == layer {
			mods = append(mods, mod)
		}
	}
	return mods
}

func matchesPattern(pattern, modulePath, samplePath string) bool {
	if strings.ContainsAny(pattern, "*?[]{}") {
		if ok, _ := filepath.Match(pattern, modulePath); ok {
			return true
		}
		if samplePath != "" {
			ok, _ := filepath.Match(pattern, samplePath)
			return ok
		}
		return false
	}
	if util.HasPathPrefix(modulePath, pattern) {
		return true
	}
	return samplePath != "" && util.HasPathPrefix(samplePath, pattern)
}

func cycleEdgeSet(cycles [][]string) map[string]bool {
	out := make(map[string]bool)
	for _, cycle := range cycles {
		if len(cycle) < 2 {
			continue
		}
		for i := 0; i < len(cycle); i++ {
			from := cycle[i]
			to := cycle[(i+1)%len(cycle)]
			out[from+"->"+to] = true
		}
	}
	return out
}

func cycleModuleSet(cycles [][]string) map[string]bool {
	out := make(map[string]bool)
	for _, cycle := range cycles {
		for _, mod := range cycle {
			out[mod] = true
		}
	}
	return out
}

func violationEdgeSet(violations []graph.ArchitectureViolation) map[string]bool {
	out := make(map[string]bool, len(violations))
	for _, v := range violations {
		out[v.FromModule+"->"+v.ToModule] = true
	}
	return out
}

const externalAggregateNodeID = "__external_aggregate__"

func toIDs(names []string, ids map[string]string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if id, ok := ids[name]; ok {
			out = append(out, id)
		}
	}
	return out
}

func intersectOrdered(ordered []string, set map[string]bool) []string {
	out := make([]string, 0)
	for _, item := range ordered {
		if set[item] {
			out = append(out, item)
		}
	}
	return out
}

func joinInts(v []int) string {
	if len(v) == 0 {
		return ""
	}
	parts := make([]string, 0, len(v))
	for _, n := range v {
		parts = append(parts, fmt.Sprintf("%d", n))
	}
	return strings.Join(parts, ",")
}

func countExternalEdges(imports map[string]map[string]*graph.ImportEdge, moduleSet map[string]bool) map[string]int {
	out := make(map[string]int)
	for from, targets := range imports {
		for to := range targets {
			if !moduleSet[to] {
				out[from]++
			}
		}
	}
	return out
}
