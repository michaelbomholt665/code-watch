package architecture

import (
	"circular/internal/core/ports"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"testing"
)

func TestRuleEvaluator_FileCountViolation(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{Path: "internal/api/a.go", Module: "internal/api"})
	g.AddFile(&parser.File{Path: "internal/api/b.go", Module: "internal/api"})

	rules := []ports.ArchitectureRule{
		{
			Name:     "api-size",
			Kind:     ports.ArchitectureRuleKindPackage,
			Modules:  []string{"internal/api"},
			MaxFiles: 1,
			Exclude:  ports.ArchitectureRuleExclude{Tests: true},
		},
	}
	eval := NewRuleEvaluator(rules)
	result := eval.Evaluate(g)

	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	v := result.Violations[0]
	if v.Type != "file_count" {
		t.Fatalf("expected file_count violation, got %s", v.Type)
	}
	if v.Actual != 2 || v.Limit != 1 {
		t.Fatalf("unexpected file-count values: actual=%d limit=%d", v.Actual, v.Limit)
	}
}

func TestRuleEvaluator_ImportViolation(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "internal/api/a.go",
		Module: "internal/api",
		Imports: []parser.Import{
			{Module: "internal/infra", Location: parser.Location{Line: 3, Column: 1}},
		},
	})
	g.AddFile(&parser.File{Path: "internal/infra/x.go", Module: "internal/infra"})

	rules := []ports.ArchitectureRule{
		{
			Name:    "api-imports",
			Kind:    ports.ArchitectureRuleKindPackage,
			Modules: []string{"internal/api"},
			Imports: ports.ArchitectureImportRule{
				Allow: []string{"internal/core"},
			},
		},
	}
	eval := NewRuleEvaluator(rules)
	result := eval.Evaluate(g)

	if len(result.Violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(result.Violations))
	}
	v := result.Violations[0]
	if v.Type != "import" {
		t.Fatalf("expected import violation, got %s", v.Type)
	}
	if v.Target != "internal/infra" {
		t.Fatalf("expected target internal/infra, got %s", v.Target)
	}
	if v.File != "internal/api/a.go" || v.Line != 3 {
		t.Fatalf("unexpected location: %s:%d", v.File, v.Line)
	}
}
