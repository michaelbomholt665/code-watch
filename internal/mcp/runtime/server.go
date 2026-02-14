package runtime

import (
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"circular/internal/mcp/adapters"
	"circular/internal/mcp/contracts"
	"circular/internal/mcp/registry"
	"circular/internal/mcp/tools/graph"
	"circular/internal/mcp/tools/query"
	"circular/internal/mcp/tools/report"
	"circular/internal/mcp/tools/scan"
	"circular/internal/mcp/tools/secrets"
	"circular/internal/mcp/tools/system"
	"circular/internal/mcp/transport"
	"circular/internal/mcp/validate"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

type Dependencies struct {
	Analysis     ports.AnalysisService
	WatchService ports.WatchService
	Logger       *slog.Logger
	ConfigPath   string
}

type AppDeps = Dependencies

type Server struct {
	cfg       *config.Config
	deps      Dependencies
	project   ProjectContext
	registry  *registry.Registry
	transport transport.Adapter
	adapter   *adapters.Adapter
	watch     ports.WatchService
	history   historyStore
	allowlist OperationAllowlist
	toolName  string

	mu        sync.Mutex
	running   bool
	projectMu sync.RWMutex
	watchMu   sync.Mutex
	watching  bool
}

type historyStore interface {
	Close() error
}

func New(cfg *config.Config, deps Dependencies, reg *registry.Registry, adapter transport.Adapter, project ProjectContext, toolName string, allowlist OperationAllowlist, toolAdapter *adapters.Adapter, history historyStore) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if deps.Analysis == nil {
		return nil, fmt.Errorf("analysis service dependency is required")
	}
	if deps.WatchService == nil {
		watch := deps.Analysis.WatchService()
		if watch != nil {
			deps.WatchService = watch
		}
	}
	if deps.WatchService == nil {
		return nil, fmt.Errorf("watch service dependency is required")
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if reg == nil {
		reg = registry.New()
	}
	if adapter == nil {
		return nil, fmt.Errorf("transport is required")
	}
	if toolAdapter == nil {
		return nil, fmt.Errorf("tool adapter is required")
	}
	if strings.TrimSpace(toolName) == "" {
		toolName = contracts.ToolNameCircular
	}

	return &Server{
		cfg:       cfg,
		deps:      deps,
		project:   project,
		registry:  reg,
		transport: adapter,
		adapter:   toolAdapter,
		watch:     deps.WatchService,
		history:   history,
		allowlist: allowlist,
		toolName:  toolName,
	}, nil
}

func (s *Server) ProjectContext() ProjectContext {
	s.projectMu.RLock()
	defer s.projectMu.RUnlock()
	return s.project
}

func (s *Server) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		<-ctx.Done()
		return ctx.Err()
	}
	s.running = true
	s.mu.Unlock()

	s.deps.Logger.Info("mcp runtime active", "mode", s.cfg.MCP.Mode, "transport", s.cfg.MCP.Transport, "project", s.project.Name)

	if err := s.registerDefaultTool(); err != nil {
		return err
	}

	err := s.transport.Start(ctx, s.handleToolCall)

	s.mu.Lock()
	s.running = false
	s.mu.Unlock()

	return err
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		if s.history != nil {
			return s.history.Close()
		}
		return nil
	}
	stopErr := s.transport.Stop()
	if s.history != nil {
		if err := s.history.Close(); err != nil && stopErr == nil {
			stopErr = err
		}
	}
	return stopErr
}

func (s *Server) Run(ctx context.Context) error {
	return s.Start(ctx)
}

func (s *Server) SyncOutputs(ctx context.Context, formats []string) ([]string, error) {
	return s.adapter.SyncOutputs(ctx, formats)
}

func (s *Server) SyncConfig(_ context.Context) (string, error) {
	s.projectMu.RLock()
	project := s.project
	s.projectMu.RUnlock()

	if err := SyncProjectConfig(project); err != nil {
		return "", err
	}
	return project.ConfigFile, nil
}

func (s *Server) GenerateConfig(_ context.Context) (contracts.SystemGenerateConfigOutput, error) {
	s.projectMu.RLock()
	project := s.project
	s.projectMu.RUnlock()

	generated, err := GenerateProjectConfig(project)
	if err != nil {
		return contracts.SystemGenerateConfigOutput{}, err
	}
	return contracts.SystemGenerateConfigOutput{
		Generated: generated,
		Target:    project.ConfigFile,
	}, nil
}

func (s *Server) GenerateScript(_ context.Context) (contracts.SystemGenerateScriptOutput, error) {
	s.projectMu.RLock()
	project := s.project
	s.projectMu.RUnlock()

	generated, err := GenerateProjectScript(project)
	if err != nil {
		return contracts.SystemGenerateScriptOutput{}, err
	}
	return contracts.SystemGenerateScriptOutput{
		Generated: generated,
		Target:    project.ScriptFile,
	}, nil
}

func (s *Server) SelectProject(_ context.Context, name string) (contracts.ProjectSummary, error) {
	if strings.TrimSpace(name) == "" {
		return contracts.ProjectSummary{}, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "name is required"}
	}

	project, err := ResolveActiveProjectContext(s.cfg, name)
	if err != nil {
		return contracts.ProjectSummary{}, err
	}
	project.SourceConfigPath = s.deps.ConfigPath
	paths, err := config.ResolvePaths(s.cfg, project.Root)
	if err == nil {
		project.TemplatePath = config.ResolveRelative(paths.ConfigDir, "circular.example.toml")
	}

	s.projectMu.Lock()
	s.project = project
	s.projectMu.Unlock()

	if strings.TrimSpace(project.Key) != "" {
		s.adapter.SetProjectKey(project.Key)
	}

	return contracts.ProjectSummary{
		Name:        project.Name,
		Root:        project.Root,
		DBNamespace: project.DBNamespace,
		Key:         project.Key,
	}, nil
}

func (s *Server) StartWatch(_ context.Context) (contracts.SystemWatchOutput, error) {
	s.watchMu.Lock()
	if s.watching {
		s.watchMu.Unlock()
		return contracts.SystemWatchOutput{
			Status:          "watching",
			AlreadyWatching: true,
		}, nil
	}
	s.watching = true
	s.watchMu.Unlock()

	go func() {
		if err := s.watch.Start(context.Background()); err != nil {
			s.deps.Logger.Error("mcp background watcher failed", "error", err)
			s.watchMu.Lock()
			s.watching = false
			s.watchMu.Unlock()
			return
		}
	}()

	return contracts.SystemWatchOutput{Status: "watching"}, nil
}

func (s *Server) registerDefaultTool() error {
	if _, ok := s.registry.HandlerFor(s.toolName); ok {
		return nil
	}
	return s.registry.Register(s.toolName, func(ctx context.Context, input any) (any, error) {
		raw, ok := input.(map[string]any)
		if !ok {
			return nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "tool args must be an object"}
		}
		return s.dispatchOperation(ctx, raw)
	})
}

func (s *Server) handleToolCall(ctx context.Context, tool string, raw map[string]any) (any, error) {
	if strings.TrimSpace(tool) == "" {
		return nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: "tool is required"}
	}
	if !strings.EqualFold(tool, s.toolName) {
		return nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: fmt.Sprintf("unsupported tool: %s", tool)}
	}

	handler, ok := s.registry.HandlerFor(s.toolName)
	if !ok {
		return nil, contracts.ToolError{Code: contracts.ErrorUnavailable, Message: "tool handler not registered"}
	}

	timeout := s.cfg.MCP.RequestTimeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	out, err := handler(ctx, raw)
	if err != nil {
		return nil, toToolError(err)
	}
	return out, nil
}

func (s *Server) dispatchOperation(ctx context.Context, raw map[string]any) (any, error) {
	operation, input, err := validate.ParseToolArgs(contracts.ToolNameCircular, raw)
	if err != nil {
		return nil, err
	}
	if !s.allowlist.Allows(operation) {
		return nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: fmt.Sprintf("operation not allowlisted: %s", operation)}
	}

	maxItems := s.cfg.MCP.MaxResponseItems
	switch operation {
	case contracts.OperationScanRun:
		out, err := scan.HandleRun(ctx, s.adapter, input.(contracts.ScanRunInput))
		return wrapToolResult(operation, out), err
	case contracts.OperationSecretsScan:
		out, err := secrets.HandleScan(ctx, s.adapter, input.(contracts.SecretsScanInput), maxItems)
		return wrapToolResult(operation, out), err
	case contracts.OperationSecretsList:
		out, err := secrets.HandleList(ctx, s.adapter, input.(contracts.SecretsListInput), maxItems)
		return wrapToolResult(operation, out), err
	case contracts.OperationGraphCycles:
		out, err := graph.HandleCycles(ctx, s.adapter, input.(contracts.GraphCyclesInput), maxItems)
		return wrapToolResult(operation, out), err
	case contracts.OperationGraphSyncDiag, contracts.OperationSystemSyncOut:
		out, err := system.HandleSyncOutputs(ctx, s, s.cfg.MCP.AllowMutations, input.(contracts.SystemSyncOutputsInput))
		return wrapToolResult(operation, out), err
	case contracts.OperationQueryModules:
		out, err := query.HandleModules(ctx, s.adapter, input.(contracts.QueryModulesInput), maxItems)
		return wrapToolResult(operation, out), err
	case contracts.OperationQueryDetails:
		out, err := query.HandleModuleDetails(ctx, s.adapter, input.(contracts.QueryModuleDetailsInput), maxItems)
		return wrapToolResult(operation, out), err
	case contracts.OperationQueryTrace:
		out, err := query.HandleTrace(ctx, s.adapter, input.(contracts.QueryTraceInput))
		return wrapToolResult(operation, out), err
	case contracts.OperationSystemSyncCfg:
		out, err := system.HandleSyncConfig(ctx, s, s.cfg.MCP.AllowMutations)
		return wrapToolResult(operation, out), err
	case contracts.OperationSystemGenCfg:
		out, err := system.HandleGenerateConfig(ctx, s, s.cfg.MCP.AllowMutations)
		return wrapToolResult(operation, out), err
	case contracts.OperationSystemGenScript:
		out, err := system.HandleGenerateScript(ctx, s, s.cfg.MCP.AllowMutations)
		return wrapToolResult(operation, out), err
	case contracts.OperationSystemSelect:
		out, err := system.HandleSelectProject(ctx, s, s.cfg.MCP.AllowMutations, input.(contracts.SystemSelectProjectInput))
		return wrapToolResult(operation, out), err
	case contracts.OperationSystemWatch:
		out, err := system.HandleWatch(ctx, s, s.cfg.MCP.AllowMutations)
		return wrapToolResult(operation, out), err
	case contracts.OperationQueryTrends:
		out, err := query.HandleTrends(ctx, s.adapter, input.(contracts.QueryTrendsInput), maxItems)
		return wrapToolResult(operation, out), err
	case contracts.OperationReportGenMD:
		out, err := report.HandleGenerateMarkdown(ctx, s.adapter, input.(contracts.ReportGenerateMarkdownInput))
		return wrapToolResult(operation, out), err
	default:
		return nil, contracts.ToolError{Code: contracts.ErrorInvalidArgument, Message: fmt.Sprintf("unsupported operation: %s", operation)}
	}
}

func wrapToolResult(operation contracts.OperationID, payload any) any {
	return map[string]any{
		"version":   contracts.ContractVersion,
		"operation": operation,
		"result":    payload,
	}
}

func toToolError(err error) error {
	var toolErr contracts.ToolError
	if errors.As(err, &toolErr) {
		return toolErr
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return contracts.ToolError{Code: contracts.ErrorUnavailable, Message: "request timed out"}
	}

	msg := err.Error()
	lower := strings.ToLower(msg)
	code := contracts.ErrorInternal
	switch {
	case strings.Contains(lower, "not found"), strings.Contains(lower, "no path"), strings.Contains(lower, "no import chain"):
		code = contracts.ErrorNotFound
	case strings.Contains(lower, "must be"), strings.Contains(lower, "invalid"), strings.Contains(lower, "required"), strings.Contains(lower, "max_depth"), strings.Contains(lower, "limit"):
		code = contracts.ErrorInvalidArgument
	}
	return contracts.ToolError{Code: code, Message: msg}
}
