// # internal/output/output_test.go
package output

import (
	"circular/internal/graph"
	"circular/internal/parser"
	"strings"
	"testing"
)

func TestDOTGenerator(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Imports: []parser.Import{{Module: "modB"}},
	})
	g.AddFile(&parser.File{
		Path:   "b.go",
		Module: "modB",
		Imports: []parser.Import{{Module: "modA"}},
	})

	cycles := [][]string{{"modA", "modB"}}
	gen := NewDOTGenerator(g)
	dot, err := gen.Generate(cycles)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(dot, "digraph dependencies") {
		t.Error("DOT output missing digraph header")
	}
	if !strings.Contains(dot, "\"modA\" -> \"modB\"") {
		t.Error("DOT output missing edge modA -> modB")
	}
	if !strings.Contains(dot, "CYCLE") {
		t.Error("DOT output missing CYCLE label")
	}
}

func TestTSVGenerator(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Imports: []parser.Import{
			{
				Module: "modB",
				Location: parser.Location{Line: 10, Column: 5},
			},
		},
	})

	gen := NewTSVGenerator(g)
	tsv, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(tsv), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines in TSV, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "modA\tmodB\ta.go\t10\t5") {
		t.Errorf("Unexpected TSV line: %s", lines[1])
	}
}
