package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePaths_DefaultLayout(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		WatchPaths: []string{root},
		DB: Database{
			Path: "history.db",
		},
	}
	applyDefaults(cfg)

	got, err := ResolvePaths(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ProjectRoot != filepath.Clean(root) {
		t.Fatalf("expected project root %q, got %q", root, got.ProjectRoot)
	}
	if got.DBPath != filepath.Join(root, "data/database", "history.db") {
		t.Fatalf("unexpected db path: %q", got.DBPath)
	}
}

func TestResolvePaths_MCPConfigPathRelative(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		WatchPaths: []string{root},
		Paths: Paths{
			ConfigDir: "cfg",
		},
		MCP: MCP{
			ConfigPath: "mcp.toml",
		},
	}
	applyDefaults(cfg)
	normalizeMCP(cfg)

	got, err := ResolvePaths(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.MCPConfigPath != filepath.Join(root, "cfg", "mcp.toml") {
		t.Fatalf("unexpected mcp config path: %q", got.MCPConfigPath)
	}
}

func TestResolvePaths_MCPOpenAPISpecPathRelative(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		WatchPaths: []string{root},
		Paths: Paths{
			ConfigDir: "cfg",
		},
		MCP: MCP{
			OpenAPISpecPath: "openapi.yaml",
		},
	}
	applyDefaults(cfg)
	normalizeMCP(cfg)

	got, err := ResolvePaths(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.MCPOpenAPISpecPath != filepath.Join(root, "cfg", "openapi.yaml") {
		t.Fatalf("unexpected mcp openapi spec path: %q", got.MCPOpenAPISpecPath)
	}
}

func TestResolvePaths_AbsoluteOverrides(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "custom", "history.db")
	cfg := &Config{
		Paths: Paths{
			ProjectRoot: root,
			ConfigDir:   filepath.Join(root, "cfg"),
			DatabaseDir: filepath.Join(root, "db"),
		},
		DB: Database{
			Path: dbPath,
		},
	}
	applyDefaults(cfg)

	got, err := ResolvePaths(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConfigDir != filepath.Join(root, "cfg") {
		t.Fatalf("unexpected config dir: %q", got.ConfigDir)
	}
	if got.DBPath != dbPath {
		t.Fatalf("unexpected db path: %q", got.DBPath)
	}
}

func TestDetectProjectRoot_FallbackOrder(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := DetectProjectRoot([]string{sub})
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(root) {
		t.Fatalf("expected %q, got %q", root, got)
	}
}
