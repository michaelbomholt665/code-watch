// # internal/output/tsv.go
package output

import (
	"circular/internal/graph"
	"fmt"
	"strings"
)

type TSVGenerator struct {
	graph *graph.Graph
}

func NewTSVGenerator(g *graph.Graph) *TSVGenerator {
	return &TSVGenerator{graph: g}
}

func (t *TSVGenerator) Generate() (string, error) {
	var buf strings.Builder

	buf.WriteString("From\tTo\tFile\tLine\tColumn\n")

	imports := t.graph.GetImports()
	for from, targets := range imports {
		for to, edge := range targets {
			buf.WriteString(fmt.Sprintf("%s\t%s\t%s\t%d\t%d\n",
				from, to, edge.ImportedBy, edge.Location.Line, edge.Location.Column))
		}
	}

	return buf.String(), nil
}
