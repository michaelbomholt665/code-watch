package runtime

import (
	"circular/internal/core/app"
	"circular/internal/core/config"
	"context"
	"fmt"
	"log/slog"
)

type AppDeps struct {
	App        *app.App
	Logger     *slog.Logger
	ConfigPath string
}

type Server struct {
	cfg     *config.Config
	deps    AppDeps
	project ProjectContext
}

func Build(cfg *config.Config, deps AppDeps) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if deps.App == nil {
		return nil, fmt.Errorf("app dependency is required")
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	project, err := ResolveActiveProjectContext(cfg, "")
	if err != nil {
		return nil, err
	}
	project.SourceConfigPath = deps.ConfigPath

	return &Server{cfg: cfg, deps: deps, project: project}, nil
}

func (s *Server) ProjectContext() ProjectContext {
	return s.project
}

func (s *Server) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.deps.Logger.Info("mcp runtime active", "mode", s.cfg.MCP.Mode, "transport", s.cfg.MCP.Transport, "project", s.project.Name)
	<-ctx.Done()
	return ctx.Err()
}
