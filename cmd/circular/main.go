// # cmd/circular/main.go
package main

import (
	"circular/internal/config"
	"circular/internal/graph"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var (
	configPath = flag.String("config", "./circular.toml", "Path to config file")
	once       = flag.Bool("once", false, "Run single scan and exit")
	ui         = flag.Bool("ui", false, "Enable terminal UI mode")
	trace      = flag.Bool("trace", false, "Trace shortest import chain between two modules")
	impact     = flag.String("impact", "", "Analyze change impact for a file path or module")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
	version    = flag.Bool("version", false, "Print version and exit")
)

const VERSION = "1.0.0"

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("circular v%s\n", VERSION)
		os.Exit(0)
	}

	// Setup logging
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	output := os.Stdout
	if *ui {
		// In UI mode, avoid stdout logs corrupting the TUI.
		logPath := resolveLogPath()
		if err := os.MkdirAll(filepath.Dir(logPath), 0700); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to create log dir for %s: %v\n", logPath, err)
		} else {
			if fi, err := os.Lstat(logPath); err == nil && (fi.Mode()&os.ModeSymlink) != 0 {
				fmt.Fprintf(os.Stderr, "warning: refusing to write logs to symlink path %s\n", logPath)
			} else {
				f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
				if err == nil {
					output = f
				} else {
					fmt.Fprintf(os.Stderr, "warning: failed to open log file %s: %v\n", logPath, err)
				}
			}
		}
	}

	logger := slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		if *configPath == "./circular.toml" {
			cfg, err = config.Load("./circular.example.toml")
		}
		if err != nil {
			slog.Error("failed to load config", "error", err)
			os.Exit(1)
		}
	}

	// Initialize app
	if *trace && *impact != "" {
		fmt.Fprintln(os.Stderr, "--trace and --impact cannot be used together")
		os.Exit(1)
	}

	if *trace {
		if flag.NArg() != 2 {
			fmt.Fprintln(os.Stderr, "trace mode requires two module arguments: circular --trace <from> <to>")
			os.Exit(1)
		}
	} else if flag.NArg() > 0 {
		cfg.WatchPaths = []string{flag.Arg(0)}
	}

	// Make grammar path absolute relative to the current working directory if it's relative
	if !filepath.IsAbs(cfg.GrammarsPath) {
		cwd, _ := os.Getwd()
		cfg.GrammarsPath = filepath.Join(cwd, cfg.GrammarsPath)
	}

	app, err := NewApp(cfg)
	if err != nil {
		slog.Error("failed to initialize app", "error", err)
		os.Exit(1)
	}

	// Initial scan
	if err := app.InitialScan(); err != nil {
		slog.Error("initial scan failed", "error", err)
		os.Exit(1)
	}

	if *trace {
		out, err := app.TraceImportChain(flag.Arg(0), flag.Arg(1))
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Println(out)
		os.Exit(0)
	}
	if *impact != "" {
		report, err := app.AnalyzeImpact(*impact)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Print(formatImpactReport(report))
		os.Exit(0)
	}

	// Analyze and Output initial state
	cycles := app.Graph.DetectCycles()
	metrics := app.Graph.ComputeModuleMetrics()
	hotspots := app.Graph.TopComplexity(cfg.Architecture.TopComplexity)
	violations := app.archEngine.Validate(app.Graph)
	hallucinations := app.AnalyzeHallucinations()
	unusedImports := app.AnalyzeUnusedImports()
	if err := app.GenerateOutputs(cycles, unusedImports, metrics, violations, hotspots); err != nil {
		slog.Error("failed to generate outputs", "error", err)
	}

	if !*ui {
		app.PrintSummary(len(app.Graph.GetAllFiles()), app.Graph.ModuleCount(), 0, cycles, hallucinations, unusedImports, metrics, violations, hotspots)
	}

	if *once {
		os.Exit(0)
	}

	// Watch mode
	if err := app.StartWatcher(); err != nil {
		slog.Error("failed to start watcher", "error", err)
		os.Exit(1)
	}

	if *ui {
		if err := app.RunUI(); err != nil {
			slog.Error("failed to run UI", "error", err)
			os.Exit(1)
		}
	} else {
		// Block forever
		select {}
	}
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

func formatImpactReport(report graph.ImpactReport) string {
	var b strings.Builder

	b.WriteString("Impact Analysis\n")
	b.WriteString("==============\n")
	b.WriteString(fmt.Sprintf("Target module: %s\n", report.TargetModule))
	if report.TargetPath != "" {
		b.WriteString(fmt.Sprintf("Target file: %s\n", report.TargetPath))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Direct importers (%d)\n", len(report.DirectImporters)))
	for _, mod := range report.DirectImporters {
		b.WriteString(fmt.Sprintf("- %s\n", mod))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Transitive impact (%d)\n", len(report.TransitiveImporters)))
	for _, mod := range report.TransitiveImporters {
		b.WriteString(fmt.Sprintf("- %s\n", mod))
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("Externally used symbols (%d)\n", len(report.ExternallyUsedSymbols)))
	for _, sym := range report.ExternallyUsedSymbols {
		b.WriteString(fmt.Sprintf("- %s\n", sym))
	}

	return b.String()
}
