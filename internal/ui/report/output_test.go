// # internal/output/output_test.go
package report

import (
	"circular/internal/core/ports"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/engine/resolver"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestTSVGenerator_GenerateArchitectureRuleViolations(t *testing.T) {
	g := graph.NewGraph()
	gen := NewTSVGenerator(g)

	tsv, err := gen.GenerateArchitectureRuleViolations([]ports.ArchitectureRuleViolation{
		{
			RuleName: "api-size",
			RuleKind: ports.ArchitectureRuleKindPackage,
			Module:   "internal/api",
			Type:     "file_count",
			Message:  "module exceeds file-count limit",
			Limit:    10,
			Actual:   12,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(tsv), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines in architecture rule TSV, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Type\tRule\tModule\tViolation\tTarget\tDetail\tFile\tLine\tColumn\tLimit\tActual") {
		t.Fatalf("Unexpected architecture rule TSV header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "architecture_rule_violation\tapi-size\tinternal/api\tfile_count\t-") {
		t.Fatalf("Unexpected architecture rule TSV row: %s", lines[1])
	}
}

func TestTSVGenerator_GenerateProbableBridges(t *testing.T) {
	g := graph.NewGraph()
	gen := NewTSVGenerator(g)

	tsv, err := gen.GenerateProbableBridges([]resolver.ProbableBridgeReference{
		{
			File:       "client.py",
			Reference:  parser.Reference{Name: "grpc.insecure_channel", Location: parser.Location{Line: 8, Column: 3}},
			Confidence: "medium",
			Score:      6,
			Reasons:    []string{"bridge_context", "bridge_prefix_heuristic"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(tsv), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines in probable-bridge TSV, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Type\tFile\tReference\tLine\tColumn\tConfidence\tScore\tReasons") {
		t.Fatalf("Unexpected probable-bridge TSV header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "probable_bridge\tclient.py\tgrpc.insecure_channel\t8\t3\tmedium\t6\tbridge_context,bridge_prefix_heuristic") {
		t.Fatalf("Unexpected probable-bridge TSV row: %s", lines[1])
	}
}

func TestTSVGenerator_GenerateSecrets(t *testing.T) {
	g := graph.NewGraph()
	gen := NewTSVGenerator(g)

	tsv, err := gen.GenerateSecrets([]parser.Secret{
		{
			Kind:       "aws-access-key-id",
			Severity:   "high",
			Value:      "AKIA1234567890ABCDEF",
			Entropy:    3.7,
			Confidence: 0.99,
			Location:   parser.Location{File: "a.go", Line: 7, Column: 2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(tsv), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines in secrets TSV, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "Type\tKind\tSeverity\tValue\tEntropy\tConfidence\tFile\tLine\tColumn") {
		t.Fatalf("Unexpected secrets TSV header: %s", lines[0])
	}
	if !strings.Contains(lines[1], "secret\taws-access-key-id\thigh\tAKIA...CDEF\t3.7000\t0.99\ta.go\t7\t2") {
		t.Fatalf("Unexpected secrets TSV row: %s", lines[1])
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

func TestMermaidGenerator_GenerateArchitecture(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "internal/api/a.go",
		Module: "internal/api",
		Imports: []parser.Import{
			{Module: "internal/core"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "internal/core/b.go",
		Module: "internal/core",
	})

	model := graph.ArchitectureModel{
		Enabled: true,
		Layers: []graph.ArchitectureLayer{
			{Name: "api", Paths: []string{"internal/api"}},
			{Name: "core", Paths: []string{"internal/core"}},
		},
	}
	violations := []graph.ArchitectureViolation{
		{FromLayer: "api", ToLayer: "core"},
	}

	gen := NewMermaidGenerator(g)
	out, err := gen.GenerateArchitecture(model, violations)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "flowchart LR") {
		t.Fatalf("expected flowchart output, got: %s", out)
	}
	if !strings.Contains(out, "deps:1 viol:1") {
		t.Fatalf("expected aggregated layer dependency label, got: %s", out)
	}
	if !strings.Contains(out, "linkStyle 0 stroke:#a64d00") {
		t.Fatalf("expected violation link style in architecture mode, got: %s", out)
	}
}

func TestPlantUMLGenerator_GenerateArchitecture(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "internal/api/a.go",
		Module: "internal/api",
		Imports: []parser.Import{
			{Module: "internal/core"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "internal/core/b.go",
		Module: "internal/core",
	})

	model := graph.ArchitectureModel{
		Enabled: true,
		Layers: []graph.ArchitectureLayer{
			{Name: "api", Paths: []string{"internal/api"}},
			{Name: "core", Paths: []string{"internal/core"}},
		},
	}
	violations := []graph.ArchitectureViolation{
		{FromLayer: "api", ToLayer: "core"},
	}

	gen := NewPlantUMLGenerator(g)
	out, err := gen.GenerateArchitecture(model, violations)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "@startuml") {
		t.Fatalf("expected plantuml header, got: %s", out)
	}
	if !strings.Contains(out, "rectangle \"api\"") {
		t.Fatalf("expected architecture layer node, got: %s", out)
	}
	if !strings.Contains(out, "deps:1 viol:1") {
		t.Fatalf("expected aggregated layer dependency label, got: %s", out)
	}
	if !strings.Contains(out, "-[#a64d00,dashed]->") {
		t.Fatalf("expected violation edge styling in architecture mode, got: %s", out)
	}
}

func TestMermaidGenerator_GenerateComponent(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Imports: []parser.Import{
			{Module: "modB", Alias: "b"},
		},
		References: []parser.Reference{
			{Name: "b.DoThing"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "b.go",
		Module: "modB",
		Definitions: []parser.Definition{
			{Name: "DoThing", FullName: "modB.DoThing"},
		},
	})

	gen := NewMermaidGenerator(g)
	out, err := gen.GenerateComponent(graph.ArchitectureModel{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "flowchart LR") {
		t.Fatalf("expected flowchart output, got: %s", out)
	}
	if !strings.Contains(out, "refs:1") {
		t.Fatalf("expected symbol reference edge label, got: %s", out)
	}
	if !strings.Contains(out, "sym:DoThing") {
		t.Fatalf("expected symbol preview in edge label, got: %s", out)
	}
	if !strings.Contains(out, "symbolNode") {
		t.Fatalf("expected symbol node style class when show_internal=true, got: %s", out)
	}
}

func TestPlantUMLGenerator_GenerateFlow(t *testing.T) {
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
		Imports: []parser.Import{
			{Module: "modC"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "c.go",
		Module: "modC",
	})

	gen := NewPlantUMLGenerator(g)
	out, err := gen.GenerateFlow([]string{"modA"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@startuml") {
		t.Fatalf("expected plantuml header, got: %s", out)
	}
	if !strings.Contains(out, "modA\\n(step:0)") {
		t.Fatalf("expected entry node step annotation, got: %s", out)
	}
	if !strings.Contains(out, "modB\\n(step:1)") {
		t.Fatalf("expected depth annotation for downstream node, got: %s", out)
	}
	if !strings.Contains(out, "-->") {
		t.Fatalf("expected flow edges, got: %s", out)
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

func TestMarkdownGenerator(t *testing.T) {
	gen := NewMarkdownGenerator()
	out, err := gen.Generate(MarkdownReportData{
		TotalModules: 2,
		TotalFiles:   3,
		Cycles:       [][]string{{"modA", "modB", "modA"}},
		Unresolved: []resolver.UnresolvedReference{
			{
				File: "a.go",
				Reference: parser.Reference{
					Name:     "pkg.Missing",
					Location: parser.Location{Line: 9, Column: 2},
				},
			},
		},
		UnusedImports: []resolver.UnusedImport{
			{
				File:       "a.go",
				Language:   "go",
				Module:     "fmt",
				Location:   parser.Location{Line: 3, Column: 1},
				Confidence: "medium",
			},
		},
		Violations: []graph.ArchitectureViolation{
			{
				RuleName:   "api-only-core",
				FromLayer:  "api",
				ToLayer:    "infra",
				FromModule: "app/api",
				ToModule:   "app/infra",
				File:       "internal/api/a.go",
				Line:       12,
				Column:     2,
			},
		},
		Hotspots: []graph.ComplexityHotspot{
			{Module: "app/api", Definition: "DoThing", File: "internal/api/a.go", Score: 42, Branches: 4, Parameters: 3, Nesting: 3, LOC: 120},
		},
	}, MarkdownReportOptions{
		ProjectName:         "code-watch",
		ProjectRoot:         ".",
		Version:             "1.0.0",
		GeneratedAt:         time.Date(2026, 2, 14, 11, 0, 0, 0, time.UTC),
		Verbosity:           "detailed",
		TableOfContents:     true,
		CollapsibleSections: true,
		IncludeMermaid:      true,
		MermaidDiagram:      "flowchart LR\n  A --> B",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "generated_at: 2026-02-14T11:00:00Z") {
		t.Fatalf("expected frontmatter timestamp, got: %s", out)
	}
	if !strings.Contains(out, "## Executive Summary") {
		t.Fatalf("expected summary section, got: %s", out)
	}
	if !strings.Contains(out, "## Circular Imports") {
		t.Fatalf("expected cycles section, got: %s", out)
	}
	if !strings.Contains(out, "🔴 High") && !strings.Contains(out, "🟡 Medium") {
		t.Fatalf("expected severity label, got: %s", out)
	}
	if !strings.Contains(out, "## Dependency Diagram") {
		t.Fatalf("expected mermaid section, got: %s", out)
	}
	if !strings.Contains(out, "```mermaid") {
		t.Fatalf("expected mermaid fenced code block, got: %s", out)
	}
	if !strings.Contains(out, "## Probable Bridge References") {
		t.Fatalf("expected probable bridge section, got: %s", out)
	}
	if !strings.Contains(out, "No probable bridge references detected.") {
		t.Fatalf("expected probable bridge empty-state text, got: %s", out)
	}
}
