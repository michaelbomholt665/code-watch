package cliapp

import (
	"circular/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyModeOptions_RejectsTraceAndImpact(t *testing.T) {
	opts := &cliOptions{trace: true, impact: "pkg", args: []string{"a", "b"}}
	cfg := &config.Config{}

	err := applyModeOptions(opts, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "cannot be used together") {
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

func TestNormalizeGrammarsPath_MakesRelativePathAbsolute(t *testing.T) {
	cfg := &config.Config{GrammarsPath: "./grammars"}
	if err := normalizeGrammarsPath(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(cfg.GrammarsPath) {
		t.Fatalf("expected absolute path, got %q", cfg.GrammarsPath)
	}
}
