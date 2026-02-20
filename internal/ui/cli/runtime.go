package cli

import (
	coreapp "circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"circular/internal/data/history"
	"circular/internal/engine/parser"
	mcpruntime "circular/internal/mcp/runtime"
	"circular/internal/shared/util"
	"circular/internal/ui/report"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func Run(args []string) int {
	opts, err := parseOptions(args)
	if err != nil {
		return 2
	}

	if opts.version {
		fmt.Printf("circular v%s\n", versionString)
		return 0
	}

	cleanupLogs := configureLogging(opts.ui, opts.verbose)
	defer cleanupLogs()

	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("failed to detect working directory", "error", err)
		return 1
	}

	cfg, cfgPath, err := loadConfig(opts.configPath, cwd)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return 1
	}

	paths, err := config.ResolvePaths(cfg, cwd)
	if err != nil {
		slog.Error("failed to resolve runtime paths", "error", err)
		return 1
	}

	if err := applyModeOptions(&opts, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	if opts.reportMarkdown && strings.TrimSpace(cfg.Output.Markdown) == "" {
		cfg.Output.Markdown = "analysis-report.md"
	}

	if err := validateModeCompatibility(opts, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	if err := normalizeGrammarsPath(cfg, paths.ProjectRoot); err != nil {
		slog.Error("failed to normalize grammars path", "error", err, "grammarsPath", cfg.GrammarsPath)
		return 1
	}

	if err := runMCPModeIfEnabled(opts, cfg, cfgPath); err != nil {
		slog.Error("failed to start MCP mode", "error", err)
		return 1
	}
	if cfg.MCP.Enabled {
		return 0
	}

	if opts.verifyGrammars {
		registry, err := buildGrammarRegistry(cfg)
		if err != nil {
			slog.Error("invalid language registry", "error", err)
			return 1
		}
		if !cfg.GrammarVerification.IsEnabled() {
			fmt.Println("Grammar verification is disabled in config (grammar_verification.enabled=false); no checks were run.")
			return 0
		}
		issues, err := parser.VerifyLanguageRegistryArtifacts(cfg.GrammarsPath, registry)
		if err != nil {
			slog.Error("grammar verification failed", "error", err)
			return 1
		}
		if len(issues) == 0 {
			fmt.Println("Grammar verification passed: all enabled language artifacts match manifest checksums and allowed AIB versions.")
			return 0
		}
		for _, issue := range issues {
			fmt.Printf("%s: %s (%s)\n", issue.Language, issue.Reason, issue.ArtifactPath)
		}
		fmt.Printf("Grammar verification failed: %d issues detected.\n", len(issues))
		return 1
	}

	activeProject, err := resolveRuntimeProject(cfg, paths, cwd)
	if err != nil {
		slog.Error("failed to resolve active project", "error", err)
		return 1
	}

	analysis, err := initializeAnalysis(cfg, opts.includeTests, coreAnalysisFactory{})
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		return 1
	}

	if _, err := analysis.RunScan(context.Background(), ports.ScanRequest{}); err != nil {
		slog.Error("initial scan failed", "error", err)
		return 1
	}

	if stop, code := runSingleCommand(analysis, opts); stop {
		return code
	}

	queryHistoryStore, err := openHistoryStoreIfEnabled(opts.history, cfg, paths)
	if err != nil {
		slog.Error("history setup failed", "error", err)
		return 1
	}
	if queryHistoryStore != nil {
		defer queryHistoryStore.Close()
	}

	summary, err := analysis.SummarySnapshot(context.Background())
	if err != nil {
		slog.Error("failed to collect summary snapshot", "error", err)
		return 1
	}

	if _, err := analysis.SyncOutputs(context.Background(), ports.SyncOutputsRequest{}); err != nil {
		slog.Error("failed to generate outputs", "error", err)
	}

	report, err := runHistoryMode(
		opts,
		analysis,
		activeProject,
		queryHistoryStore,
	)
	if err != nil {
		slog.Error("history mode failed", "error", err)
		return 1
	}

	if stop, code := runQueryCommand(analysis, opts, queryHistoryStore, activeProject.Key); stop {
		return code
	}

	if !opts.ui {
		if err := analysis.PrintSummary(context.Background(), ports.SummaryPrintRequest{
			Duration: 0,
			Snapshot: summary,
		}); err != nil {
			slog.Error("failed to print summary", "error", err)
			return 1
		}
	}

	if opts.once {
		return 0
	}

	watch := analysis.WatchService()
	if watch == nil {
		slog.Error("watch service unavailable")
		return 1
	}
	if err := watch.Start(context.Background()); err != nil {
		slog.Error("failed to start watcher", "error", err)
		return 1
	}

	if opts.ui {
		if err := runUI(analysis, queryHistoryStore, activeProject.Key, report); err != nil {
			slog.Error("failed to run UI", "error", err)
			return 1
		}
		return 0
	}

	select {}
}

func runSingleCommand(analysis ports.AnalysisService, opts cliOptions) (bool, int) {
	if analysis == nil {
		fmt.Fprintln(os.Stderr, "analysis service unavailable")
		return true, 1
	}

	if opts.trace {
		out, err := analysis.TraceImportChain(context.Background(), opts.args[0], opts.args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		fmt.Println(out)
		return true, 0
	}

	if opts.impact != "" {
		report, err := analysis.AnalyzeImpact(context.Background(), opts.impact)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		fmt.Print(coreapp.FormatImpactReport(report))
		return true, 0
	}

	return false, 0
}

func runQueryCommand(analysis ports.AnalysisService, opts cliOptions, historyStore ports.HistoryStore, projectKey string) (bool, int) {
	if !opts.queryModules && opts.queryModule == "" && opts.queryTrace == "" && !opts.queryTrends {
		return false, 0
	}

	if analysis == nil {
		fmt.Fprintln(os.Stderr, "analysis service unavailable")
		return true, 1
	}
	svc := analysis.QueryService(historyStore, projectKey)
	if svc == nil {
		fmt.Fprintln(os.Stderr, "query service unavailable")
		return true, 1
	}
	ctx := context.Background()

	switch {
	case opts.queryModule != "":
		details, err := svc.ModuleDetails(ctx, strings.TrimSpace(opts.queryModule))
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		fmt.Printf("Module: %s\n", details.Name)
		fmt.Printf("Files: %d, Exports: %d, Dependencies: %d, ReverseDependencies: %d\n",
			len(details.Files), len(details.ExportedSymbols), len(details.Dependencies), len(details.ReverseDependencies))
		if len(details.Files) > 0 {
			fmt.Println("File list:")
			for _, file := range details.Files {
				fmt.Printf("  - %s\n", file)
			}
		}
		return true, 0
	case opts.queryTrace != "":
		from, to, err := parseQueryTrace(opts.queryTrace)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		trace, err := svc.DependencyTrace(ctx, from, to, opts.queryLimit)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		fmt.Printf("Trace depth=%d: %s\n", trace.Depth, strings.Join(trace.Path, " -> "))
		return true, 0
	case opts.queryTrends:
		if historyStore == nil {
			fmt.Fprintln(os.Stderr, "--query-trends requires --history")
			return true, 1
		}
		since, err := parseSince(opts.since)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		slice, err := svc.TrendSlice(ctx, since, opts.queryLimit)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		fmt.Printf("Trend slice: scans=%d since=%s until=%s\n", slice.ScanCount, slice.Since, slice.Until)
		for _, snapshot := range slice.Snapshots {
			fmt.Printf("  %s modules=%d cycles=%d unresolved=%d fan_in=%.2f fan_out=%.2f\n",
				snapshot.Timestamp.Format(time.RFC3339),
				snapshot.ModuleCount,
				snapshot.CycleCount,
				snapshot.UnresolvedCount,
				snapshot.AvgFanIn,
				snapshot.AvgFanOut,
			)
		}
		return true, 0
	default:
		filter := strings.TrimSpace(opts.queryFilter)
		modules, err := svc.ListModules(ctx, filter, opts.queryLimit)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		fmt.Printf("Modules (%d):\n", len(modules))
		for _, module := range modules {
			fmt.Printf("  %s files=%d exports=%d deps=%d imported_by=%d\n",
				module.Name,
				module.FileCount,
				module.ExportCount,
				module.DependencyCount,
				module.ReverseDependencyCount,
			)
		}
		return true, 0
	}
}

func loadConfig(path, cwd string) (*config.Config, string, error) {
	if path != defaultConfigPath {
		cfg, err := config.Load(path)
		if err != nil {
			return nil, "", err
		}
		return cfg, path, nil
	}

	candidates, err := discoverDefaultConfig(cwd)
	if err != nil {
		return nil, "", err
	}

	var lastErr error
	for _, candidate := range candidates {
		cfg, loadErr := config.Load(candidate)
		if loadErr == nil {
			if candidate == filepath.Clean(filepath.Join(cwd, "circular.toml")) {
				fmt.Fprintln(os.Stderr, "warning: using deprecated config path ./circular.toml; migrate to ./data/config/circular.toml")
			}
			return cfg, candidate, nil
		}
		if os.IsNotExist(loadErr) {
			lastErr = loadErr
			continue
		}
		return nil, "", loadErr
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("no default config file found")
}

func discoverDefaultConfig(cwd string) ([]string, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, fmt.Errorf("cwd must not be empty")
	}
	return []string{
		filepath.Clean(filepath.Join(cwd, "data/config/circular.toml")),
		filepath.Clean(filepath.Join(cwd, "circular.toml")),
		filepath.Clean(filepath.Join(cwd, "data/config/circular.example.toml")),
		filepath.Clean(filepath.Join(cwd, "circular.example.toml")),
	}, nil
}

func applyModeOptions(opts *cliOptions, cfg *config.Config) error {
	modeCount := 0
	if opts.verifyGrammars {
		modeCount++
	}
	if opts.trace {
		modeCount++
	}
	if opts.impact != "" {
		modeCount++
	}
	if opts.queryModules || opts.queryModule != "" || opts.queryTrace != "" || opts.queryTrends {
		modeCount++
	}
	if modeCount > 1 {
		return fmt.Errorf("--verify-grammars, --trace, --impact, and --query-* modes cannot be combined")
	}

	if opts.verifyGrammars {
		if len(opts.args) > 0 {
			return fmt.Errorf("--verify-grammars does not accept positional path arguments")
		}
		return nil
	}

	if opts.trace {
		if len(opts.args) != 2 {
			return fmt.Errorf("trace mode requires two module arguments: circular --trace <from> <to>")
		}
		return nil
	}

	if len(opts.args) > 0 {
		cfg.WatchPaths = []string{opts.args[0]}
	}

	if opts.queryTrace != "" {
		if _, _, err := parseQueryTrace(opts.queryTrace); err != nil {
			return err
		}
	}

	if (opts.historyTSV != "" || opts.historyJSON != "") && !opts.history {
		return fmt.Errorf("--history-tsv/--history-json require --history")
	}
	if opts.queryTrends && !opts.history {
		return fmt.Errorf("--query-trends requires --history")
	}
	if opts.history {
		if _, err := parseHistoryWindow(opts.historyWindow); err != nil {
			return err
		}
	}
	return nil
}

func normalizeGrammarsPath(cfg *config.Config, base string) error {
	if filepath.IsAbs(cfg.GrammarsPath) {
		return nil
	}
	cfg.GrammarsPath = filepath.Join(base, cfg.GrammarsPath)
	return nil
}

func parseSince(value string) (time.Time, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return time.Time{}, nil
	}

	rfc3339, err := time.Parse(time.RFC3339, raw)
	if err == nil {
		return rfc3339.UTC(), nil
	}

	dateOnly, err := time.Parse("2006-01-02", raw)
	if err == nil {
		return dateOnly.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("--since must be RFC3339 or YYYY-MM-DD, got %q", value)
}

func writeBytes(path string, data []byte) error {
	return util.WriteFileWithDirs(path, data, 0o644)
}

func runHistoryMode(
	opts cliOptions,
	analysis ports.AnalysisService,
	activeProject config.ActiveProject,
	store ports.HistoryStore,
) (*history.TrendReport, error) {
	if !opts.history {
		return nil, nil
	}
	if analysis == nil {
		return nil, fmt.Errorf("analysis service unavailable")
	}

	since, err := parseSince(opts.since)
	if err != nil {
		return nil, err
	}
	window, err := parseHistoryWindow(opts.historyWindow)
	if err != nil {
		return nil, err
	}

	if store == nil {
		return nil, fmt.Errorf("history store unavailable")
	}

	trend, err := analysis.CaptureHistoryTrend(context.Background(), store, ports.HistoryTrendRequest{
		ProjectKey:  activeProject.Key,
		ProjectRoot: activeProject.Root,
		Since:       since,
		Window:      window,
	})
	if err != nil {
		return nil, err
	}
	if trend.Report == nil {
		fmt.Println("History: no snapshots matched the requested time window.")
		return nil, nil
	}
	trendReport := trend.Report

	fmt.Printf(
		"History: %d snapshots from %s to %s\n",
		trendReport.ScanCount,
		trendReport.Since.Format("2006-01-02 15:04:05"),
		trendReport.Until.Format("2006-01-02 15:04:05"),
	)
	if len(trendReport.Points) > 0 {
		latest := trendReport.Points[len(trendReport.Points)-1]
		fmt.Printf(
			"Trend latest: modules=%d (%+d), cycles=%d (%+d), unresolved=%d (%+d)\n",
			latest.ModuleCount,
			latest.DeltaModules,
			latest.CycleCount,
			latest.DeltaCycles,
			latest.UnresolvedCount,
			latest.DeltaUnresolved,
		)
	}

	if opts.historyTSV != "" {
		tsv, err := report.RenderTrendTSV(*trendReport)
		if err != nil {
			return nil, fmt.Errorf("render trend TSV: %w", err)
		}
		if err := writeBytes(opts.historyTSV, tsv); err != nil {
			return nil, fmt.Errorf("write trend TSV %q: %w", opts.historyTSV, err)
		}
	}

	if opts.historyJSON != "" {
		raw, err := report.RenderTrendJSON(*trendReport)
		if err != nil {
			return nil, fmt.Errorf("render trend JSON: %w", err)
		}
		if err := writeBytes(opts.historyJSON, raw); err != nil {
			return nil, fmt.Errorf("write trend JSON %q: %w", opts.historyJSON, err)
		}
	}

	return trendReport, nil
}

func parseHistoryWindow(value string) (time.Duration, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("--history-window must be a Go duration (example: 24h), got %q", value)
	}
	if d <= 0 {
		return 0, fmt.Errorf("--history-window must be > 0, got %q", value)
	}
	return d, nil
}

func openHistoryStoreIfEnabled(enabled bool, cfg *config.Config, paths config.ResolvedPaths) (*history.Store, error) {
	if !enabled {
		return nil, nil
	}
	if !cfg.DB.Enabled {
		return nil, nil
	}

	store, err := history.Open(paths.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open history store: %w", err)
	}
	return store, nil
}

func resolveRuntimeProject(cfg *config.Config, paths config.ResolvedPaths, cwd string) (config.ActiveProject, error) {
	entries := cfg.Projects.Entries
	if len(entries) == 0 && strings.TrimSpace(cfg.Projects.RegistryFile) != "" {
		registryPath := config.ResolveRelative(paths.ConfigDir, cfg.Projects.RegistryFile)
		if loaded, err := config.LoadProjectRegistry(registryPath); err == nil {
			cfg.Projects.Entries = loaded
		} else if !os.IsNotExist(err) {
			return config.ActiveProject{}, fmt.Errorf("load projects registry %q: %w", registryPath, err)
		}
	}
	for i := range cfg.Projects.Entries {
		cfg.Projects.Entries[i].Root = config.ResolveRelative(paths.ProjectRoot, cfg.Projects.Entries[i].Root)
		if strings.TrimSpace(cfg.Projects.Entries[i].ConfigFile) != "" {
			cfg.Projects.Entries[i].ConfigFile = config.ResolveRelative(paths.ConfigDir, cfg.Projects.Entries[i].ConfigFile)
		}
	}
	project, err := config.ResolveActiveProject(cfg, cwd)
	if err != nil {
		return config.ActiveProject{}, err
	}
	if strings.TrimSpace(project.Root) == "" {
		project.Root = paths.ProjectRoot
	}
	if strings.TrimSpace(project.Key) == "" {
		project.Key = "default"
	}
	return project, nil
}

func validateModeCompatibility(opts cliOptions, cfg *config.Config) error {
	if cfg.MCP.Enabled {
		if opts.ui || opts.once || opts.verifyGrammars || opts.trace || opts.impact != "" || opts.history || opts.reportMarkdown ||
			opts.queryModules || opts.queryModule != "" || opts.queryTrace != "" || opts.queryTrends || len(opts.args) > 0 {
			return fmt.Errorf("mcp.enabled=true cannot be combined with CLI modes or positional path arguments")
		}
	}
	if cfg.MCP.Enabled && strings.EqualFold(cfg.MCP.Mode, "embedded") && strings.EqualFold(cfg.MCP.Transport, "http") {
		return fmt.Errorf("mcp.mode=embedded does not support mcp.transport=http")
	}
	return nil
}

func runMCPModeIfEnabled(opts cliOptions, cfg *config.Config, configPath string) error {
	return runMCPModeIfEnabledWithFactory(opts, cfg, configPath, coreAnalysisFactory{})
}

func runMCPModeIfEnabledWithFactory(opts cliOptions, cfg *config.Config, configPath string, factory analysisFactory) error {
	if !cfg.MCP.Enabled {
		return nil
	}

	// MCP stdio requires stdout to be protocol-only JSON.
	// Route logs to stderr before any startup work can emit logs.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	analysis, err := initializeAnalysis(cfg, opts.includeTests, factory)
	if err != nil {
		return fmt.Errorf("init app: %w", err)
	}

	if _, err := analysis.RunScan(context.Background(), ports.ScanRequest{}); err != nil {
		return fmt.Errorf("initial scan: %w", err)
	}

	if cfg.MCP.AutoManageOutputsEnabled() {
		if _, err := analysis.SyncOutputs(context.Background(), ports.SyncOutputsRequest{}); err != nil {
			return fmt.Errorf("auto-manage outputs: %w", err)
		}
	}

	server, err := mcpruntime.Build(cfg, mcpruntime.AppDeps{
		Analysis:   analysis,
		Logger:     slog.Default(),
		ConfigPath: configPath,
	})
	if err != nil {
		return fmt.Errorf("build MCP runtime: %w", err)
	}

	project := server.ProjectContext()
	if cfg.MCP.AutoSyncConfigEnabled() {
		if _, err := mcpruntime.GenerateProjectConfig(project); err != nil {
			return fmt.Errorf("auto-generate project config: %w", err)
		}
		if _, err := mcpruntime.GenerateProjectScript(project); err != nil {
			return fmt.Errorf("auto-generate project script: %w", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func buildGrammarRegistry(cfg *config.Config) (map[string]parser.LanguageSpec, error) {
	overrides := make(map[string]parser.LanguageOverride, len(cfg.Languages))
	for lang, languageCfg := range cfg.Languages {
		overrides[lang] = parser.LanguageOverride{
			Enabled:    languageCfg.Enabled,
			Extensions: append([]string(nil), languageCfg.Extensions...),
			Filenames:  append([]string(nil), languageCfg.Filenames...),
		}
	}

	dynamic := make([]parser.LanguageSpec, 0, len(cfg.DynamicGrammars))
	for _, dg := range cfg.DynamicGrammars {
		dynamic = append(dynamic, parser.LanguageSpec{
			Name:       dg.Name,
			Extensions: dg.Extensions,
			Filenames:  dg.Filenames,
			IsDynamic:  true,
			LibraryPath: dg.Library,
			SymbolName:  "tree_sitter_" + dg.Name,
			DynamicConfig: &parser.DynamicExtractorConfig{
				NamespaceNode:   dg.NamespaceNode,
				ImportNode:      dg.ImportNode,
				DefinitionNodes: dg.DefinitionNodes,
			},
		})
	}

	return parser.BuildLanguageRegistry(overrides, dynamic)
}

func parseQueryTrace(raw string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("--query-trace must be formatted as <from>:<to>")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func configureLogging(uiMode, verbose bool) func() {
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	output := os.Stdout
	var closeFn func() = func() {}
	if uiMode {
		logPath := resolveLogPath()
		if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to create log dir for %s: %v\n", logPath, err)
		} else {
			if fi, err := os.Lstat(logPath); err == nil && (fi.Mode()&os.ModeSymlink) != 0 {
				fmt.Fprintf(os.Stderr, "warning: refusing to write logs to symlink path %s\n", logPath)
			} else {
				f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
				if err == nil {
					output = f
					closeFn = func() { _ = f.Close() }
				} else {
					fmt.Fprintf(os.Stderr, "warning: failed to open log file %s: %v\n", logPath, err)
				}
			}
		}
	}

	logger := slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)
	return closeFn
}

func resolveLogPath() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "circular", "circular.log")
	}

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".local", "state", "circular", "circular.log")
	}

	return "circular.log"
}
