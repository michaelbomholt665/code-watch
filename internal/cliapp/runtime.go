package cliapp

import (
	coreapp "circular/internal/app"
	"circular/internal/config"
	"circular/internal/graph"
	"circular/internal/history"
	"circular/internal/output"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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

	cfg, err := loadConfig(opts.configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return 1
	}

	if err := applyModeOptions(&opts, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	if err := normalizeGrammarsPath(cfg); err != nil {
		slog.Error("failed to normalize grammars path", "error", err, "grammarsPath", cfg.GrammarsPath)
		return 1
	}

	app, err := coreapp.New(cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		return 1
	}

	if err := app.InitialScan(); err != nil {
		slog.Error("initial scan failed", "error", err)
		return 1
	}

	if stop, code := runSingleCommand(app, opts); stop {
		return code
	}

	queryHistoryStore, err := openHistoryStoreIfEnabled(opts.history)
	if err != nil {
		slog.Error("history setup failed", "error", err)
		return 1
	}
	if queryHistoryStore != nil {
		defer queryHistoryStore.Close()
	}

	cycles := app.Graph.DetectCycles()
	metrics := app.Graph.ComputeModuleMetrics()
	hotspots := app.Graph.TopComplexity(cfg.Architecture.TopComplexity)
	violations := app.ArchitectureViolations()
	hallucinations := app.AnalyzeHallucinations()
	unusedImports := app.AnalyzeUnusedImports()
	if err := app.GenerateOutputs(cycles, unusedImports, metrics, violations, hotspots); err != nil {
		slog.Error("failed to generate outputs", "error", err)
	}

	report, err := runHistoryMode(
		opts,
		app,
		metrics,
		cycles,
		len(hallucinations),
		len(unusedImports),
		len(violations),
		len(hotspots),
		queryHistoryStore,
	)
	if err != nil {
		slog.Error("history mode failed", "error", err)
		return 1
	}

	if stop, code := runQueryCommand(app, opts, queryHistoryStore); stop {
		return code
	}

	if !opts.ui {
		app.PrintSummary(len(app.Graph.GetAllFiles()), app.Graph.ModuleCount(), 0, cycles, hallucinations, unusedImports, metrics, violations, hotspots)
	}

	if opts.once {
		return 0
	}

	if err := app.StartWatcher(); err != nil {
		slog.Error("failed to start watcher", "error", err)
		return 1
	}

	if opts.ui {
		if err := runUI(app, report); err != nil {
			slog.Error("failed to run UI", "error", err)
			return 1
		}
		return 0
	}

	select {}
}

func runSingleCommand(app *coreapp.App, opts cliOptions) (bool, int) {
	if opts.trace {
		out, err := app.TraceImportChain(opts.args[0], opts.args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		fmt.Println(out)
		return true, 0
	}

	if opts.impact != "" {
		report, err := app.AnalyzeImpact(opts.impact)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return true, 1
		}
		fmt.Print(coreapp.FormatImpactReport(report))
		return true, 0
	}

	return false, 0
}

func runQueryCommand(app *coreapp.App, opts cliOptions, historyStore *history.Store) (bool, int) {
	if !opts.queryModules && opts.queryModule == "" && opts.queryTrace == "" && !opts.queryTrends {
		return false, 0
	}

	svc := app.BuildQueryService(historyStore)
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

func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err == nil {
		return cfg, nil
	}
	if path != defaultConfigPath {
		return nil, err
	}

	cfg, fallbackErr := config.Load("./circular.example.toml")
	if fallbackErr != nil {
		return nil, fallbackErr
	}
	return cfg, nil
}

func applyModeOptions(opts *cliOptions, cfg *config.Config) error {
	modeCount := 0
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
		return fmt.Errorf("--trace, --impact, and --query-* modes cannot be combined")
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

func normalizeGrammarsPath(cfg *config.Config) error {
	if filepath.IsAbs(cfg.GrammarsPath) {
		return nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg.GrammarsPath = filepath.Join(cwd, cfg.GrammarsPath)
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

func resolveHistoryPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, ".circular", "history.db"), nil
}

func writeBytes(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func runHistoryMode(
	opts cliOptions,
	app *coreapp.App,
	metrics map[string]graph.ModuleMetrics,
	cycles [][]string,
	hallucinations int,
	unusedImports int,
	violations int,
	hotspots int,
	store *history.Store,
) (*history.TrendReport, error) {
	if !opts.history {
		return nil, nil
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

	projectRoot, rootErr := os.Getwd()
	if rootErr != nil {
		projectRoot = "."
	}
	commitHash, commitTime := history.ResolveGitMetadata(projectRoot)
	avgFanIn, avgFanOut, maxFanIn, maxFanOut := summarizeFanMetrics(metrics)
	snapshot := history.Snapshot{
		Timestamp:         time.Now().UTC(),
		CommitHash:        commitHash,
		CommitTimestamp:   commitTime,
		ModuleCount:       app.Graph.ModuleCount(),
		FileCount:         app.Graph.FileCount(),
		CycleCount:        len(cycles),
		UnresolvedCount:   hallucinations,
		UnusedImportCount: unusedImports,
		ViolationCount:    violations,
		HotspotCount:      hotspots,
		AvgFanIn:          avgFanIn,
		AvgFanOut:         avgFanOut,
		MaxFanIn:          maxFanIn,
		MaxFanOut:         maxFanOut,
	}
	if err := store.SaveSnapshot(snapshot); err != nil {
		return nil, fmt.Errorf("save history snapshot: %w", err)
	}

	snapshots, err := store.LoadSnapshots(since)
	if err != nil {
		return nil, fmt.Errorf("load history snapshots: %w", err)
	}
	if len(snapshots) == 0 {
		fmt.Println("History: no snapshots matched the requested time window.")
		return nil, nil
	}

	report, err := history.BuildTrendReport(snapshots, window)
	if err != nil {
		return nil, fmt.Errorf("build trend report: %w", err)
	}

	fmt.Printf(
		"History: %d snapshots from %s to %s\n",
		report.ScanCount,
		report.Since.Format("2006-01-02 15:04:05"),
		report.Until.Format("2006-01-02 15:04:05"),
	)
	if len(report.Points) > 0 {
		latest := report.Points[len(report.Points)-1]
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
		tsv, err := output.RenderTrendTSV(report)
		if err != nil {
			return nil, fmt.Errorf("render trend TSV: %w", err)
		}
		if err := writeBytes(opts.historyTSV, tsv); err != nil {
			return nil, fmt.Errorf("write trend TSV %q: %w", opts.historyTSV, err)
		}
	}

	if opts.historyJSON != "" {
		raw, err := output.RenderTrendJSON(report)
		if err != nil {
			return nil, fmt.Errorf("render trend JSON: %w", err)
		}
		if err := writeBytes(opts.historyJSON, raw); err != nil {
			return nil, fmt.Errorf("write trend JSON %q: %w", opts.historyJSON, err)
		}
	}

	return &report, nil
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

func openHistoryStoreIfEnabled(enabled bool) (*history.Store, error) {
	if !enabled {
		return nil, nil
	}
	historyPath, err := resolveHistoryPath()
	if err != nil {
		return nil, fmt.Errorf("resolve history path: %w", err)
	}
	store, err := history.Open(historyPath)
	if err != nil {
		return nil, fmt.Errorf("open history store: %w", err)
	}
	return store, nil
}

func parseQueryTrace(raw string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("--query-trace must be formatted as <from>:<to>")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func summarizeFanMetrics(metrics map[string]graph.ModuleMetrics) (avgIn, avgOut float64, maxIn, maxOut int) {
	if len(metrics) == 0 {
		return 0, 0, 0, 0
	}
	var totalIn, totalOut int
	for _, m := range metrics {
		totalIn += m.FanIn
		totalOut += m.FanOut
		if m.FanIn > maxIn {
			maxIn = m.FanIn
		}
		if m.FanOut > maxOut {
			maxOut = m.FanOut
		}
	}
	n := float64(len(metrics))
	return float64(totalIn) / n, float64(totalOut) / n, maxIn, maxOut
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
