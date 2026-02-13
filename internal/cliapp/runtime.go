package cliapp

import (
	coreapp "circular/internal/app"
	"circular/internal/config"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

	cycles := app.Graph.DetectCycles()
	metrics := app.Graph.ComputeModuleMetrics()
	hotspots := app.Graph.TopComplexity(cfg.Architecture.TopComplexity)
	violations := app.ArchitectureViolations()
	hallucinations := app.AnalyzeHallucinations()
	unusedImports := app.AnalyzeUnusedImports()
	if err := app.GenerateOutputs(cycles, unusedImports, metrics, violations, hotspots); err != nil {
		slog.Error("failed to generate outputs", "error", err)
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
		if err := runUI(app); err != nil {
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
	if opts.trace && opts.impact != "" {
		return fmt.Errorf("--trace and --impact cannot be used together")
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
