package formats

import (
	"circular/internal/engine/graph"
	"circular/internal/shared/util"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

func moduleLabel(module string, mod *graph.Module, metrics map[string]graph.ModuleMetrics, hotspots map[string]int) string {
	fileCount := 0
	exports := 0
	if mod != nil {
		fileCount = len(mod.Files)
		exports = len(mod.Exports)
	}

	parts := []string{fmt.Sprintf("%s\\n(%d funcs, %d files)", module, exports, fileCount)}
	if metric, ok := metrics[module]; ok {
		parts = append(parts, fmt.Sprintf("(d=%d in=%d out=%d)", metric.Depth, metric.FanIn, metric.FanOut))
		if metric.ImportanceScore > 0 {
			parts = append(parts, fmt.Sprintf("(imp=%.1f)", metric.ImportanceScore))
		}
	}
	if score, ok := hotspots[module]; ok {
		parts = append(parts, fmt.Sprintf("(cx=%d)", score))
	}
	return strings.Join(parts, "\\n")
}

func sanitizeID(module string) string {
	if module == "" {
		return "m"
	}
	var b strings.Builder
	for _, r := range module {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	out := b.String()
	if out == "" {
		return "m"
	}
	first := rune(out[0])
	if unicode.IsDigit(first) {
		return "m_" + out
	}
	return out
}

func makeIDs(names []string) map[string]string {
	ids := make(map[string]string, len(names))
	used := make(map[string]int, len(names))
	for _, name := range names {
		base := sanitizeID(name)
		idx := used[base]
		used[base] = idx + 1
		if idx == 0 {
			ids[name] = base
			continue
		}
		ids[name] = fmt.Sprintf("%s_%d", base, idx+1)
	}
	return ids
}

func escapeLabel(s string) string {
	return strings.ReplaceAll(s, "\"", "'")
}

type layerDependency struct {
	From       string
	To         string
	Count      int
	Violations int
}

func architectureLayerDependencies(g *graph.Graph, model graph.ArchitectureModel, violations []graph.ArchitectureViolation) ([]string, []layerDependency) {
	layers := make([]string, 0, len(model.Layers))
	layerOrder := make(map[string]int, len(model.Layers))
	for i, layer := range model.Layers {
		if _, exists := layerOrder[layer.Name]; exists {
			continue
		}
		layerOrder[layer.Name] = i
		layers = append(layers, layer.Name)
	}

	modules := g.Modules()
	imports := g.GetImports()
	moduleNames := util.SortedStringKeys(modules)
	layerByModule := classifyLayers(moduleNames, modules, model)

	depMap := make(map[string]layerDependency)
	for _, from := range util.SortedStringKeys(imports) {
		fromLayer := layerByModule[from]
		if fromLayer == "" {
			continue
		}
		targets := util.SortedStringKeys(imports[from])
		for _, to := range targets {
			toLayer := layerByModule[to]
			if toLayer == "" || toLayer == fromLayer {
				continue
			}
			key := fromLayer + "->" + toLayer
			dep := depMap[key]
			dep.From = fromLayer
			dep.To = toLayer
			dep.Count++
			depMap[key] = dep
		}
	}

	for _, v := range violations {
		if v.FromLayer == "" || v.ToLayer == "" || v.FromLayer == v.ToLayer {
			continue
		}
		if _, ok := layerOrder[v.FromLayer]; !ok {
			layerOrder[v.FromLayer] = len(layers)
			layers = append(layers, v.FromLayer)
		}
		if _, ok := layerOrder[v.ToLayer]; !ok {
			layerOrder[v.ToLayer] = len(layers)
			layers = append(layers, v.ToLayer)
		}
		key := v.FromLayer + "->" + v.ToLayer
		dep := depMap[key]
		dep.From = v.FromLayer
		dep.To = v.ToLayer
		dep.Violations++
		depMap[key] = dep
	}

	deps := make([]layerDependency, 0, len(depMap))
	for _, dep := range depMap {
		deps = append(deps, dep)
	}
	sort.Slice(deps, func(i, j int) bool {
		leftFrom := layerOrder[deps[i].From]
		rightFrom := layerOrder[deps[j].From]
		if leftFrom != rightFrom {
			return leftFrom < rightFrom
		}
		leftTo := layerOrder[deps[i].To]
		rightTo := layerOrder[deps[j].To]
		if leftTo != rightTo {
			return leftTo < rightTo
		}
		if deps[i].From != deps[j].From {
			return deps[i].From < deps[j].From
		}
		return deps[i].To < deps[j].To
	})

	return layers, deps
}
