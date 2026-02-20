package graph

// internal/engine/graph/importance_test.go

import (
	"circular/internal/engine/parser"
	"testing"
)

func TestCalculateImportanceScore_Formula(t *testing.T) {
	tests := []struct {
		name       string
		fanIn      int
		fanOut     int
		complexity int
		module     string
		wantMin    float64 // minimum expected score
	}{
		{
			name:       "zero everything gives zero",
			fanIn:      0,
			fanOut:     0,
			complexity: 0,
			module:     "leaf",
			wantMin:    0,
		},
		{
			name:       "fan-in weighted double fan-out",
			fanIn:      4,
			fanOut:     2,
			complexity: 0,
			module:     "core",
			// score = 4*2 + 2*1 = 10
			wantMin: 10,
		},
		{
			name:       "complexity contributes 0.5 factor",
			fanIn:      0,
			fanOut:     0,
			complexity: 20,
			module:     "internal/algo",
			// score = 20*0.5 = 10
			wantMin: 10,
		},
		{
			name:       "API module gets bonus 10",
			fanIn:      2,
			fanOut:     1,
			complexity: 0,
			module:     "internal/api/v1",
			// score = 2*2 + 1*1 + 10 = 15
			wantMin: 15,
		},
		{
			name:       "gateway module gets bonus 10",
			fanIn:      0,
			fanOut:     0,
			complexity: 0,
			module:     "cmd/gateway",
			wantMin:    10,
		},
		{
			name:       "handler module gets bonus 10",
			fanIn:      0,
			fanOut:     0,
			complexity: 0,
			module:     "http_handler",
			wantMin:    10,
		},
		{
			name:       "server module gets bonus 10",
			fanIn:      0,
			fanOut:     0,
			complexity: 0,
			module:     "grpc/server",
			wantMin:    10,
		},
		{
			name:       "combined all factors high-traffic API",
			fanIn:      10,
			fanOut:     5,
			complexity: 40,
			module:     "gateway",
			// score = 10*2 + 5*1 + 40*0.5 + 10 = 20+5+20+10 = 55
			wantMin: 55,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateImportanceScore(tc.fanIn, tc.fanOut, tc.complexity, tc.module)
			if got < tc.wantMin {
				t.Errorf("CalculateImportanceScore(%d,%d,%d,%q) = %.2f, want >= %.2f",
					tc.fanIn, tc.fanOut, tc.complexity, tc.module, got, tc.wantMin)
			}
		})
	}
}

func TestCalculateImportanceScore_NonAPINoBonus(t *testing.T) {
	score := CalculateImportanceScore(0, 0, 0, "internal/core/domain")
	if score != 0 {
		t.Errorf("expected 0 for plain non-API module with zero metrics, got %.2f", score)
	}
}

func TestIsAPIModule(t *testing.T) {
	apiModules := []string{
		"api", "my/api", "internal/api/v1",
		"gateway", "cmd/gateway",
		"handler", "http_handler", "handlers",
		"server", "grpc/server",
		"SERVICE", "myService",
	}
	for _, m := range apiModules {
		if !isAPIModule(m) {
			t.Errorf("expected %q to be detected as API module", m)
		}
	}

	nonAPI := []string{"internal/core", "data/query", "engine/parser", "util"}
	for _, m := range nonAPI {
		if isAPIModule(m) {
			t.Errorf("expected %q NOT to be detected as API module", m)
		}
	}
}

func TestGraph_ComputeModuleMetrics_ImportanceScore(t *testing.T) {
	g := NewGraph()

	// api module: high fan-in should get high importance.
	// A -> api (fan-in 1)
	// B -> api (fan-in 2)
	// api -> C (fan-out 1)
	addSimpleFile(g, "a.go", "A", "api")
	addSimpleFile(g, "b.go", "B", "api")
	addSimpleFile(g, "api.go", "api", "C")
	addSimpleFile(g, "c.go", "C")

	metrics := g.ComputeModuleMetrics()

	apiMetrics, ok := metrics["api"]
	if !ok {
		t.Fatal("expected metrics for 'api' module")
	}
	if apiMetrics.ImportanceScore <= 0 {
		t.Errorf("expected positive ImportanceScore for API module, got %.2f", apiMetrics.ImportanceScore)
	}
	// api gets: (2*2) + (1*1) + 10(API bonus) = 15; no complexity -> >=15
	if apiMetrics.ImportanceScore < 15 {
		t.Errorf("expected ImportanceScore >= 15 for api module with fan-in=2, fan-out=1, got %.2f", apiMetrics.ImportanceScore)
	}

	// C has fan-in=1 (api imports it), fan-out=0, no complexity, non-API: score = 1*2 = 2.0
	cMetrics := metrics["C"]
	if cMetrics.ImportanceScore != 2.0 {
		t.Errorf("expected ImportanceScore=2.0 for module C (fan-in=1), got %.2f", cMetrics.ImportanceScore)
	}
}

// addSimpleFile is a test helper that adds a file with optional imports.
func addSimpleFile(g *Graph, path, module string, importModules ...string) {
	imps := make([]parser.Import, 0, len(importModules))
	for _, im := range importModules {
		imps = append(imps, parser.Import{Module: im})
	}
	g.AddFile(&parser.File{Path: path, Module: module, Imports: imps})
}
