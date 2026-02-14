package cli

import (
	coreapp "circular/internal/core/app"
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplyModeOptions_RejectsTraceAndImpact(t *testing.T) {
	opts := &cliOptions{trace: true, impact: "pkg", args: []string{"a", "b"}}
	cfg := &config.Config{}

	err := applyModeOptions(opts, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyModeOptions_TraceRequiresTwoArgs(t *testing.T) {
	opts := &cliOptions{trace: true, args: []string{"only-one"}}
	cfg := &config.Config{}

	err := applyModeOptions(opts, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires two module arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyModeOptions_OverridesWatchPathWithPositionalArg(t *testing.T) {
	opts := &cliOptions{args: []string{"./override"}}
	cfg := &config.Config{WatchPaths: []string{"./original"}}

	if err := applyModeOptions(opts, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.WatchPaths) != 1 || cfg.WatchPaths[0] != "./override" {
		t.Fatalf("unexpected watch paths: %v", cfg.WatchPaths)
	}
}

func TestApplyModeOptions_HistoryOutputsRequireHistoryFlag(t *testing.T) {
	opts := &cliOptions{historyTSV: "trend.tsv"}
	cfg := &config.Config{}

	err := applyModeOptions(opts, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "require --history") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyModeOptions_QueryTrendsRequiresHistory(t *testing.T) {
	opts := &cliOptions{queryTrends: true}
	cfg := &config.Config{}

	err := applyModeOptions(opts, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "requires --history") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyModeOptions_VerifyGrammarsRejectsPositionalArgs(t *testing.T) {
	opts := &cliOptions{verifyGrammars: true, args: []string{"./src"}}
	cfg := &config.Config{}

	err := applyModeOptions(opts, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not accept positional") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeGrammarsPath_MakesRelativePathAbsolute(t *testing.T) {
	cfg := &config.Config{GrammarsPath: "./grammars"}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := normalizeGrammarsPath(cfg, cwd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(cfg.GrammarsPath) {
		t.Fatalf("expected absolute path, got %q", cfg.GrammarsPath)
	}
}

func TestParseSince(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantZero  bool
		wantError bool
	}{
		{name: "empty", input: "", wantZero: true},
		{name: "date", input: "2026-02-13"},
		{name: "rfc3339", input: "2026-02-13T15:00:00Z"},
		{name: "invalid", input: "13/02/2026", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSince(tt.input)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantZero && !got.Equal(time.Time{}) {
				t.Fatalf("expected zero time, got %v", got)
			}
			if !tt.wantZero && got.IsZero() {
				t.Fatal("expected non-zero parsed time")
			}
		})
	}
}

func TestParseHistoryWindow(t *testing.T) {
	if _, err := parseHistoryWindow("24h"); err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if _, err := parseHistoryWindow("0h"); err == nil {
		t.Fatal("expected error for non-positive window")
	}
}

func TestParseQueryTrace(t *testing.T) {
	from, to, err := parseQueryTrace("a:b")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if from != "a" || to != "b" {
		t.Fatalf("unexpected parsed values: %q %q", from, to)
	}
	if _, _, err := parseQueryTrace("a-only"); err == nil {
		t.Fatal("expected query trace format error")
	}
}

func TestRunHistoryMode_SQLiteIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{DB: config.Database{Enabled: true, Path: "history.db"}}
	paths, err := config.ResolvePaths(cfg, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	store, err := openHistoryStoreIfEnabled(true, cfg, paths)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	app := &coreapp.App{Config: &config.Config{}, Graph: graph.NewGraph()}
	app.Graph.AddFile(&parser.File{Path: "a.go", Module: "app/a"})
	analysis := app.AnalysisService()

	report, err := runHistoryMode(
		cliOptions{history: true, historyWindow: "24h"},
		analysis,
		config.ActiveProject{Name: "default", Root: tmpDir, Key: "default"},
		store,
	)
	if err != nil {
		t.Fatalf("run history mode: %v", err)
	}
	if report == nil || report.ScanCount == 0 {
		t.Fatalf("expected report with snapshots, got %+v", report)
	}

	snapshots, err := store.LoadSnapshots("default", time.Time{})
	if err != nil {
		t.Fatalf("load snapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
	}
	if snapshots[0].ModuleCount != 1 || snapshots[0].FileCount != 1 {
		t.Fatalf("unexpected saved snapshot counts: %+v", snapshots[0])
	}
}

func TestLoadConfig_DefaultDiscoveryOrder(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "data", "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(tmpDir, "data", "config", "circular.toml")
	if err := os.WriteFile(cfgPath, []byte("grammars_path = \"./grammars\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := loadConfig(defaultConfigPath, tmpDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.GrammarsPath != "./grammars" {
		t.Fatalf("unexpected config payload: %+v", cfg)
	}
}

func TestLoadConfig_CustomPathNoFallback(t *testing.T) {
	tmpDir := t.TempDir()
	custom := filepath.Join(tmpDir, "custom.toml")

	_, _, err := loadConfig(custom, tmpDir)
	if err == nil {
		t.Fatal("expected missing custom config error")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}

func TestOpenHistoryStore_UsesConfiguredDBPath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Paths: config.Paths{
			ProjectRoot: tmpDir,
			DatabaseDir: filepath.Join(tmpDir, "db"),
		},
		DB: config.Database{
			Enabled: true,
			Path:    "nested/history.db",
		},
	}
	configPath, err := config.ResolvePaths(cfg, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	store, err := openHistoryStoreIfEnabled(true, cfg, configPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	if store.Path() != filepath.Join(tmpDir, "db", "nested", "history.db") {
		t.Fatalf("unexpected history path: %q", store.Path())
	}
}

func TestOpenHistoryStore_DBDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{DB: config.Database{Enabled: false}}
	paths, err := config.ResolvePaths(cfg, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	store, err := openHistoryStoreIfEnabled(true, cfg, paths)
	if err != nil {
		t.Fatal(err)
	}
	if store != nil {
		t.Fatal("expected nil store when db disabled")
	}
}

func TestValidateModeCompatibility_MCPPOC(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCP{
			Enabled:   true,
			Mode:      "embedded",
			Transport: "stdio",
		},
	}

	if err := validateModeCompatibility(cliOptions{once: true}, cfg); err == nil {
		t.Fatal("expected MCP mode/CLI conflict error")
	}
	if err := validateModeCompatibility(cliOptions{}, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMCPBootstrapDecision_Disabled(t *testing.T) {
	cfg := &config.Config{}
	if err := runMCPModeIfEnabled(cliOptions{}, cfg, "circular.toml"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunMCPModeIfEnabledWithFactory_UsesProvidedFactory(t *testing.T) {
	factory := &recordingAnalysisFactory{
		err: fmt.Errorf("factory boom"),
	}
	cfg := &config.Config{
		MCP: config.MCP{
			Enabled:   true,
			Mode:      "embedded",
			Transport: "stdio",
		},
	}

	err := runMCPModeIfEnabledWithFactory(cliOptions{includeTests: true}, cfg, "circular.toml", factory)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "init app: factory boom") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !factory.called {
		t.Fatal("expected factory to be invoked")
	}
	if !factory.includeTests {
		t.Fatal("expected include-tests option to be passed to factory")
	}
}

func TestInitializeAnalysis_RequiresFactory(t *testing.T) {
	_, err := initializeAnalysis(&config.Config{}, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "analysis factory is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateModeCompatibility_MCPEmbeddedHTTPRejected(t *testing.T) {
	cfg := &config.Config{
		MCP: config.MCP{
			Enabled:   true,
			Mode:      "embedded",
			Transport: "http",
		},
	}

	err := validateModeCompatibility(cliOptions{}, cfg)
	if err == nil {
		t.Fatal("expected compatibility error")
	}
	if !strings.Contains(err.Error(), "does not support mcp.transport=http") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type recordingAnalysisFactory struct {
	called       bool
	includeTests bool
	err          error
}

func (f *recordingAnalysisFactory) New(_ *config.Config, includeTests bool) (ports.AnalysisService, error) {
	f.called = true
	f.includeTests = includeTests
	return nil, f.err
}
