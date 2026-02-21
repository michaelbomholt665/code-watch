package runtime

import (
	"circular/internal/core/config"
	"circular/internal/data/history"
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/openapi"
	"circular/internal/mcp/registry"
	"circular/internal/mcp/transport"
	"fmt"
	"strings"
)

func Build(cfg *config.Config, deps AppDeps) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	project, err := ResolveActiveProjectContext(cfg, "")
	if err != nil {
		return nil, err
	}
	project.SourceConfigPath = deps.ConfigPath

	paths, err := config.ResolvePaths(cfg, project.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve MCP paths: %w", err)
	}
	project.TemplatePath = config.ResolveRelative(paths.ConfigDir, "circular.example.toml")

	var historyStore *history.Store
	if cfg.DB.Enabled {
		store, err := history.Open(paths.DBPath)
		if err != nil {
			return nil, fmt.Errorf("open history store: %w", err)
		}
		historyStore = store
	}

	adapter, err := buildTransport(cfg)
	if err != nil {
		if historyStore != nil {
			_ = historyStore.Close()
		}
		return nil, err
	}

	reg := registry.New()
	toolName := strings.TrimSpace(cfg.MCP.ExposedToolName)
	allowlist := BuildOperationAllowlist(cfg)
	if err := loadOpenAPIOperations(cfg); err != nil {
		if historyStore != nil {
			_ = historyStore.Close()
		}
		return nil, err
	}
	toolAdapter := adapters.NewAdapter(deps.Analysis, historyStore, project.Key)
	server, err := New(cfg, deps, reg, adapter, project, toolName, allowlist, toolAdapter, historyStore)
	if err != nil && historyStore != nil {
		_ = historyStore.Close()
	}
	return server, err
}

func buildTransport(cfg *config.Config) (transport.Adapter, error) {
	transportName := strings.ToLower(strings.TrimSpace(cfg.MCP.Transport))
	switch transportName {
	case "", "stdio":
		return transport.NewStdio(cfg.MCP.RateLimit)
	case "sse", "http":
		addr := cfg.MCP.Address
		if addr == "" {
			addr = "127.0.0.1:8765"
		}
		return transport.NewSSE(addr, cfg.MCP.RateLimit)
	default:
		return nil, fmt.Errorf("unsupported MCP transport: %s", transportName)
	}
}

func loadOpenAPIOperations(cfg *config.Config) error {
	specSource := strings.TrimSpace(cfg.MCP.OpenAPISpecPath)
	if specSource == "" {
		specSource = strings.TrimSpace(cfg.MCP.OpenAPISpecURL)
	}
	if specSource == "" {
		return nil
	}

	spec, err := openapi.LoadSpec(specSource)
	if err != nil {
		return fmt.Errorf("load MCP OpenAPI spec: %w", err)
	}

	ops, err := openapi.Convert(spec)
	if err != nil {
		return fmt.Errorf("convert MCP OpenAPI operations: %w", err)
	}

	filtered := openapi.ApplyAllowlist(ops, cfg.MCP.OperationAllowlist)
	if len(filtered) == 0 {
		return fmt.Errorf("MCP OpenAPI conversion produced zero allowlisted operations")
	}
	return nil
}
