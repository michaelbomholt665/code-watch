// # internal/config/config_test.go
package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	content := `
grammars_path = "./grammars"
watch_paths = ["./src"]

[exclude]
dirs = [".git"]
files = ["*.log"]

[watch]
debounce = "1s"

[output]
dot = "graph.dot"
tsv = "deps.tsv"

[alerts]
beep = true
terminal = true
`
	tmpfile, err := os.CreateTemp("", "config*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.GrammarsPath != "./grammars" {
		t.Errorf("Expected GrammarsPath ./grammars, got %s", cfg.GrammarsPath)
	}
	if len(cfg.WatchPaths) != 1 || cfg.WatchPaths[0] != "./src" {
		t.Errorf("Unexpected WatchPaths: %v", cfg.WatchPaths)
	}
	if cfg.Watch.Debounce != time.Second {
		t.Errorf("Expected debounce 1s, got %v", cfg.Watch.Debounce)
	}
	if cfg.Output.DOT != "graph.dot" {
		t.Errorf("Expected DOT graph.dot, got %s", cfg.Output.DOT)
	}
}

func TestLoadDefaultDebounce(t *testing.T) {
	content := `grammars_path = "./grammars"`
	tmpfile, err := os.CreateTemp("", "config*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(content))
	tmpfile.Close()

	cfg, _ := Load(tmpfile.Name())
	if cfg.Watch.Debounce != 500*time.Millisecond {
		t.Errorf("Expected default debounce 500ms, got %v", cfg.Watch.Debounce)
	}
}

func TestLoadError(t *testing.T) {
	_, err := Load("nonexistent.toml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	tmpfile, _ := os.CreateTemp("", "badconfig*.toml")
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte("bad = toml = format"))
	tmpfile.Close()

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Error("Expected error for malformed TOML")
	}
}
