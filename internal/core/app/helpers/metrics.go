package helpers

import (
	"circular/internal/engine/graph"
	"fmt"
	"sort"
)

func MetricLeaders(
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
