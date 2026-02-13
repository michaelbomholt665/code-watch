// # internal/output/output_test.go
package output

import (
	"circular/internal/graph"
	"circular/internal/parser"
	"circular/internal/resolver"
	"strings"
	"testing"
)

func TestDOTGenerator(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:    "a.go",
		Module:  "modA",
		Imports: []parser.Import{{Module: "modB"}},
	})
	g.AddFile(&parser.File{
		Path:    "b.go",
		Module:  "modB",
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

func TestDOTGenerator_WithModuleMetrics(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Imports: []parser.Import{
			{Module: "modB"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "b.go",
		Module: "modB",
	})

	gen := NewDOTGenerator(g)
	gen.SetModuleMetrics(map[string]graph.ModuleMetrics{
		"modA": {Depth: 1, FanIn: 0, FanOut: 1},
		"modB": {Depth: 0, FanIn: 1, FanOut: 0},
	})

	dot, err := gen.Generate(nil)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(dot, "(d=1 in=0 out=1)") {
		t.Fatalf("Expected metrics annotation for modA, got: %s", dot)
	}
	if !strings.Contains(dot, "fillcolor=\"lemonchiffon\"") {
		t.Fatalf("Expected depth color for depth=1, got: %s", dot)
	}
}

func TestTSVGenerator(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Imports: []parser.Import{
			{
				Module:   "modB",
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

func TestTSVGenerator_GenerateUnusedImports(t *testing.T) {
	g := graph.NewGraph()
	gen := NewTSVGenerator(g)

	tsv, err := gen.GenerateUnusedImports([]resolver.UnusedImport{
		{
			File:       "a.py",
			Language:   "python",
			Module:     "math",
			Alias:      "",
			Item:       "pow",
			Location:   parser.Location{Line: 3, Column: 1},
			Confidence: "high",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(tsv), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines in unused-import TSV, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Type\tFile\tLanguage\tModule\tAlias\tItem\tLine\tColumn\tConfidence") {
		t.Fatalf("Unexpected unused-import TSV header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "unused_import\ta.py\tpython\tmath\t\tpow\t3\t1\thigh") {
		t.Fatalf("Unexpected unused-import TSV row: %s", lines[1])
	}
}
