package runtime

import (
	"circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"circular/internal/mcp/registry"
	"circular/internal/mcp/transport"
	"context"
	"log/slog"
	"reflect"
	"testing"
)

func TestServer_StartStop(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCP{
			Mode:             "embedded",
			Transport:        "stdio",
			MaxResponseItems: 10,
		},
	}

	testApp := testMCPApp(cfg)
	toolAdapter := adapters.NewAdapter(testApp, nil, "default")

	var got any
	transport := &fakeTransport{
		startFn: func(ctx context.Context, handler transport.Handler) error {
			out, err := handler(ctx, contracts.ToolNameCircular, map[string]any{
				"operation": string(contracts.OperationQueryModules),
				"params":    map[string]any{"limit": 5},
			})
			if err != nil {
				return err
			}
			got = out
			return nil
		},
	}

	server, err := New(cfg, Dependencies{
		App:    testApp,
		Logger: slog.Default(),
	}, registry.New(), transport, ProjectContext{Name: "default", Root: "."}, contracts.ToolNameCircular, OperationAllowlist{allowAll: true}, toolAdapter, nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if got == nil {
		t.Fatal("expected transport call result")
	}
	result, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected wrapped result map, got %T", got)
	}
	if result["operation"] != contracts.OperationQueryModules {
		t.Fatalf("unexpected operation result: %+v", result)
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestServer_RegisterTools(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCP{
			Mode:             "embedded",
			Transport:        "stdio",
			MaxResponseItems: 10,
		},
	}
	testApp := testMCPApp(cfg)
	toolAdapter := adapters.NewAdapter(testApp, nil, "default")
	reg := registry.New()

	server, err := New(cfg, Dependencies{
		App:    testApp,
		Logger: slog.Default(),
	}, reg, &fakeTransport{}, ProjectContext{Name: "default", Root: "."}, contracts.ToolNameCircular, OperationAllowlist{allowAll: true}, toolAdapter, nil)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	if err := server.registerDefaultTool(); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	if err := server.registerDefaultTool(); err != nil {
		t.Fatalf("second register should be idempotent: %v", err)
	}

	tools := reg.Tools()
	if !reflect.DeepEqual(tools, []string{contracts.ToolNameCircular}) {
		t.Fatalf("unexpected registered tools: %v", tools)
	}
}

type fakeTransport struct {
	startFn func(ctx context.Context, handler transport.Handler) error
	stopFn  func() error
}

func (f *fakeTransport) Start(ctx context.Context, handler transport.Handler) error {
	if f.startFn != nil {
		return f.startFn(ctx, handler)
	}
	return nil
}

func (f *fakeTransport) Stop() error {
	if f.stopFn != nil {
		return f.stopFn()
	}
	return nil
}

func testMCPApp(cfg *config.Config) *app.App {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "app/a",
	})
	return &app.App{
		Config: cfg,
		Graph:  g,
	}
}
