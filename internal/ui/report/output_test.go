// # internal/output/output_test.go
package report

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"os"
	"path/filepath"
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

func TestDOTGenerator_WithComplexityHotspots(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
	})

	gen := NewDOTGenerator(g)
	gen.SetComplexityHotspots([]graph.ComplexityHotspot{
		{Module: "modA", Score: 11},
	})

	dot, err := gen.Generate(nil)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(dot, "(cx=11)") {
		t.Fatalf("Expected complexity annotation in DOT output, got: %s", dot)
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

func TestTSVGenerator_GenerateArchitectureViolations(t *testing.T) {
	g := graph.NewGraph()
	gen := NewTSVGenerator(g)

	tsv, err := gen.GenerateArchitectureViolations([]graph.ArchitectureViolation{
		{
			RuleName:   "api-to-core-only",
			FromModule: "internal/core",
			FromLayer:  "core",
			ToModule:   "internal/api",
			ToLayer:    "api",
			File:       "internal/core/service.go",
			Line:       12,
			Column:     4,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(tsv), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines in architecture TSV, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Type\tRule\tFromModule\tFromLayer\tToModule\tToLayer\tFile\tLine\tColumn") {
		t.Fatalf("Unexpected architecture TSV header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "architecture_violation\tapi-to-core-only\tinternal/core\tcore\tinternal/api\tapi\tinternal/core/service.go\t12\t4") {
		t.Fatalf("Unexpected architecture TSV row: %s", lines[1])
	}
}

func TestMermaidGenerator(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Imports: []parser.Import{
			{Module: "modB"},
			{Module: "fmt"},
		},
	})
	g.AddFile(&parser.File{
		Path:    "b.go",
		Module:  "modB",
		Imports: []parser.Import{{Module: "modA"}},
	})

	gen := NewMermaidGenerator(g)
	out, err := gen.Generate(
		[][]string{{"modA", "modB"}},
		[]graph.ArchitectureViolation{{FromModule: "modA", ToModule: "modB"}},
		graph.ArchitectureModel{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "flowchart LR") {
		t.Fatalf("expected flowchart header, got: %s", out)
	}
	if !strings.Contains(out, "nodeSpacing") {
		t.Fatalf("expected mermaid spacing init block, got: %s", out)
	}
	if !strings.Contains(out, "modA -->|CYCLE| modB") {
		t.Fatalf("expected cycle edge label, got: %s", out)
	}
	if !strings.Contains(out, "stroke:#cc0000,stroke-width:3px;") {
		t.Fatalf("expected cycle link style, got: %s", out)
	}
	if !strings.Contains(out, "classDef externalNode") {
		t.Fatalf("expected external node style class, got: %s", out)
	}
	if !strings.Contains(out, "classDef cycleNode") {
		t.Fatalf("expected cycle node style class, got: %s", out)
	}
	if !strings.Contains(out, "subgraph legend_info") {
		t.Fatalf("expected mermaid legend block, got: %s", out)
	}
	if !strings.Contains(out, "fmt") {
		t.Fatalf("expected external module node, got: %s", out)
	}
}

func TestPlantUMLGenerator(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "internal/api",
		Imports: []parser.Import{
			{Module: "internal/core"},
			{Module: "fmt"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "b.go",
		Module: "internal/core",
	})

	model := graph.ArchitectureModel{
		Enabled: true,
		Layers: []graph.ArchitectureLayer{
			{Name: "api", Paths: []string{"internal/api"}},
			{Name: "core", Paths: []string{"internal/core"}},
		},
	}
	gen := NewPlantUMLGenerator(g)
	out, err := gen.Generate(nil, nil, model)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "@startuml") {
		t.Fatalf("expected @startuml header, got: %s", out)
	}
	if !strings.Contains(out, "package \"api\"") {
		t.Fatalf("expected api package cluster, got: %s", out)
	}
	if !strings.Contains(out, "-[#777777,dashed]->") {
		t.Fatalf("expected external edge style, got: %s", out)
	}
	if !strings.Contains(out, "skinparam linetype ortho") {
		t.Fatalf("expected orthogonal line routing, got: %s", out)
	}
	if !strings.Contains(out, "skinparam nodesep 80") {
		t.Fatalf("expected increased node spacing, got: %s", out)
	}
	if !strings.Contains(out, "legend right") {
		t.Fatalf("expected legend block, got: %s", out)
	}
	if !strings.Contains(out, "@enduml") {
		t.Fatalf("expected @enduml footer, got: %s", out)
	}
}

func TestMermaidGenerator_AggregatesExternalsWhenLarge(t *testing.T) {
	g := graph.NewGraph()
	imports := make([]parser.Import, 0, 12)
	for i := 0; i < 12; i++ {
		imports = append(imports, parser.Import{Module: "ext/module" + string(rune('A'+i))})
	}
	g.AddFile(&parser.File{
		Path:    "a.go",
		Module:  "modA",
		Imports: imports,
	})
	gen := NewMermaidGenerator(g)
	out, err := gen.Generate(nil, nil, graph.ArchitectureModel{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "External/Stdlib") {
		t.Fatalf("expected aggregated external node, got: %s", out)
	}
	if strings.Contains(out, "ext/moduleA") {
		t.Fatalf("expected individual external nodes to be collapsed, got: %s", out)
	}
}

func TestPlantUMLGenerator_AggregatesExternalsWhenLarge(t *testing.T) {
	g := graph.NewGraph()
	imports := make([]parser.Import, 0, 12)
	for i := 0; i < 12; i++ {
		imports = append(imports, parser.Import{Module: "ext/module" + string(rune('A'+i))})
	}
	g.AddFile(&parser.File{
		Path:    "a.go",
		Module:  "modA",
		Imports: imports,
	})
	gen := NewPlantUMLGenerator(g)
	out, err := gen.Generate(nil, nil, graph.ArchitectureModel{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "External/Stdlib") {
		t.Fatalf("expected aggregated external node, got: %s", out)
	}
	if !strings.Contains(out, "skinparam linetype ortho") {
		t.Fatalf("expected orthogonal line style, got: %s", out)
	}
}

func TestReplaceBetweenMarkers(t *testing.T) {
	content := strings.Join([]string{
		"# Docs",
		"<!-- circular:arch:start -->",
		"old",
		"<!-- circular:arch:end -->",
	}, "\n")
	got, err := ReplaceBetweenMarkers(content, "arch", "new-line")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "new-line") {
		t.Fatalf("expected replacement content, got: %s", got)
	}
	if !strings.Contains(got, "<!-- circular:arch:start -->\nnew-line\n<!-- circular:arch:end -->") {
		t.Fatalf("unexpected marker replacement result: %s", got)
	}
}

func TestReplaceBetweenMarkers_MissingMarker(t *testing.T) {
	_, err := ReplaceBetweenMarkers("no markers here", "arch", "content")
	if err == nil {
		t.Fatal("expected error for missing markers")
	}
}

func TestInjectDiagram(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "README.md")
	initial := "<!-- circular:deps:start -->\nold\n<!-- circular:deps:end -->\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	if err := InjectDiagram(path, "deps", "```mermaid\nflowchart LR\n```"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "flowchart LR") {
		t.Fatalf("expected updated markdown diagram, got: %s", string(data))
	}
}
