package query

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

func TestHandleQueryModules(t *testing.T) {
	adapter := testQueryAdapter()

	out, err := HandleModules(context.Background(), adapter, contracts.QueryModulesInput{
		Filter: "app/",
		Limit:  5,
	}, 2)
	if err != nil {
		t.Fatalf("handle modules: %v", err)
	}
	if len(out.Modules) != 2 {
		t.Fatalf("expected bounded module list, got %d", len(out.Modules))
	}
}

func TestHandleQueryTrace(t *testing.T) {
	adapter := testQueryAdapter()

	out, err := HandleTrace(context.Background(), adapter, contracts.QueryTraceInput{
		From: "app/a",
		To:   "app/c",
	})
	if err != nil {
		t.Fatalf("handle trace: %v", err)
	}
	if !out.Found {
		t.Fatal("expected path found")
	}
	if out.Depth != 2 {
		t.Fatalf("expected depth=2, got %d", out.Depth)
	}
}

func testQueryAdapter() *adapters.Adapter {
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
			{Module: "app/c"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "c.go",
		Module: "app/c",
	})
	appInstance := &app.App{
		Config: &config.Config{},
		Graph:  g,
	}
	return adapters.NewAdapter(appInstance.AnalysisService(), nil, "default")
}
