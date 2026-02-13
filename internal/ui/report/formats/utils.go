package formats

import (
	"circular/internal/engine/graph"
	"fmt"
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
