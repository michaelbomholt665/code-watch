package graph

import (
	"circular/internal/engine/parser"
	"sort"
)

type ComplexityHotspot struct {
	Module     string
	File       string
	Definition string
	Kind       parser.DefinitionKind
	Score      int
	Branches   int
	Parameters int
	Nesting    int
	LOC        int
}

func (g *Graph) TopComplexity(n int) []ComplexityHotspot {
	if n <= 0 {
		return nil
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	hotspots := make([]ComplexityHotspot, 0)
	for _, file := range g.files {
		for _, def := range file.Definitions {
			if def.Kind != parser.KindFunction && def.Kind != parser.KindMethod {
				continue
			}

			score := def.ComplexityScore
			if score == 0 {
				score = (def.BranchCount * 2) + (def.NestingDepth * 2) + def.ParameterCount + (def.LOC / 10)
				if score == 0 {
					score = 1
				}
			}

			hotspots = append(hotspots, ComplexityHotspot{
				Module:     file.Module,
				File:       file.Path,
				Definition: def.Name,
				Kind:       def.Kind,
				Score:      score,
				Branches:   def.BranchCount,
				Parameters: def.ParameterCount,
				Nesting:    def.NestingDepth,
				LOC:        def.LOC,
			})
		}
	}

	sort.Slice(hotspots, func(i, j int) bool {
		if hotspots[i].Score == hotspots[j].Score {
			if hotspots[i].Module == hotspots[j].Module {
				if hotspots[i].Definition == hotspots[j].Definition {
					return hotspots[i].File < hotspots[j].File
				}
				return hotspots[i].Definition < hotspots[j].Definition
			}
			return hotspots[i].Module < hotspots[j].Module
		}
		return hotspots[i].Score > hotspots[j].Score
	})

	if len(hotspots) > n {
		return hotspots[:n]
	}
	return hotspots
}
