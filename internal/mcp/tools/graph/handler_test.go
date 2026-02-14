package graph

import (
	"circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"context"
	"testing"
)

func TestHandleCyclesLimits(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "app/a",
		Imports: []parser.Import{
			{Module: "app/b"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "b.go",
		Module: "app/b",
		Imports: []parser.Import{
			{Module: "app/a"},
		},
	})

	adapter := adapters.NewAdapter(&app.App{
		Config: &config.Config{},
		Graph:  g,
	}, nil, "default")

	out, err := HandleCycles(context.Background(), adapter, contracts.GraphCyclesInput{Limit: 20}, 1)
	if err != nil {
		t.Fatalf("handle cycles: %v", err)
	}
	if out.CycleCount != 1 {
		t.Fatalf("expected cycle_count=1, got %d", out.CycleCount)
	}
	if len(out.Cycles) != 1 {
		t.Fatalf("expected bounded cycles=1, got %d", len(out.Cycles))
	}
}
