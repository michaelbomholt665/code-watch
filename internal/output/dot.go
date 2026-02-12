// # internal/output/dot.go
package output

import (
	"circular/internal/graph"
	"fmt"
	"strings"
)

type DOTGenerator struct {
	graph *graph.Graph
}

func NewDOTGenerator(g *graph.Graph) *DOTGenerator {
	return &DOTGenerator{graph: g}
}

func (d *DOTGenerator) Generate(cycles [][]string) (string, error) {
	var buf strings.Builder

	buf.WriteString("digraph dependencies {\n")
	buf.WriteString("  rankdir=LR;\n")
	buf.WriteString("  node [shape=box, style=rounded, fontname=\"Helvetica\", fontsize=10];\n")
	buf.WriteString("  edge [fontname=\"Helvetica\", fontsize=8, penwidth=1.2];\n")
	buf.WriteString("  ranksep=1.5;\n")
	buf.WriteString("  nodesep=0.6;\n")
	buf.WriteString("  splines=polyline;\n")
	buf.WriteString("  overlap=false;\n\n")

	// Build cycle edge set for highlighting
	cycleEdges := make(map[string]map[string]bool)
	for _, cycle := range cycles {
		for i := 0; i < len(cycle); i++ {
			from := cycle[i]
			to := cycle[(i+1)%len(cycle)]
			if cycleEdges[from] == nil {
				cycleEdges[from] = make(map[string]bool)
			}
			cycleEdges[from][to] = true
		}
	}

	modules := d.graph.Modules()
	allImports := d.graph.GetImports()

	// Categorize modules
	internalModules := make(map[string]*graph.Module)
	externalModules := make(map[string]bool)

	for modName, mod := range modules {
		internalModules[modName] = mod
	}

	for _, targets := range allImports {
		for to := range targets {
			if _, ok := internalModules[to]; !ok {
				externalModules[to] = true
			}
		}
	}

	// Internal modules cluster
	buf.WriteString("  subgraph cluster_internal {\n")
	buf.WriteString("    label=\"Internal Modules\";\n")
	buf.WriteString("    style=filled;\n")
	buf.WriteString("    color=\"whitesmoke\";\n")
	buf.WriteString("    node [fillcolor=\"white\", style=\"rounded,filled\"];\n")

	for modName, mod := range internalModules {
		funcCount := len(mod.Exports)
		fileCount := len(mod.Files)
		label := fmt.Sprintf("%s\\n(%d funcs, %d files)", modName, funcCount, fileCount)

		inCycle := false
		for _, cycle := range cycles {
			for _, m := range cycle {
				if m == modName {
					inCycle = true
					break
				}
			}
		}

		if inCycle {
			buf.WriteString(fmt.Sprintf("    \"%s\" [label=\"%s\", fillcolor=\"mistyrose\", color=\"red\", penwidth=2.0];\n", modName, label))
		} else {
			buf.WriteString(fmt.Sprintf("    \"%s\" [label=\"%s\", color=\"darkslategrey\"];\n", modName, label))
		}
	}
	buf.WriteString("  }\n\n")

	// External modules
	buf.WriteString("  // External and Standard Library\n")
	buf.WriteString("  node [fillcolor=\"gainsboro\", style=\"rounded,filled\", color=\"grey\"];\n")
	for modName := range externalModules {
		buf.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\"];\n", modName, modName))
	}
	buf.WriteString("\n")

	// Edges
	for from, targets := range allImports {
		for to := range targets {
			isCycle := cycleEdges[from] != nil && cycleEdges[from][to]
			isInternalFrom := internalModules[from] != nil
			isInternalTo := internalModules[to] != nil

			if isCycle {
				buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [color=\"red\", penwidth=3.0, label=\"CYCLE\"];\n", from, to))
			} else if isInternalFrom && isInternalTo {
				buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [color=\"forestgreen\", penwidth=1.8];\n", from, to))
			} else {
				buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [color=\"grey\", style=dashed];\n", from, to))
			}
		}
	}

	// Legend
	buf.WriteString("\n  subgraph cluster_legend {\n")
	buf.WriteString("    label=\"Legend\";\n")
	buf.WriteString("    style=dashed;\n")
	buf.WriteString("    legend_internal [label=\"Internal Module\", fillcolor=\"white\", style=\"rounded,filled\"];\n")
	buf.WriteString("    legend_external [label=\"External/Stdlib\", fillcolor=\"gainsboro\", style=\"rounded,filled\"];\n")
	buf.WriteString("    legend_cycle [label=\"Circular Import\", fillcolor=\"mistyrose\", color=\"red\", style=\"rounded,filled\"];\n")
	buf.WriteString("    legend_edge_internal [label=\"Internal Edge\", shape=plaintext, fontcolor=\"forestgreen\"];\n")
	buf.WriteString("    legend_edge_external [label=\"External Edge\", shape=plaintext, fontcolor=\"grey\"];\n")
	buf.WriteString("  }\n")

	buf.WriteString("}\n")

	return buf.String(), nil
}
