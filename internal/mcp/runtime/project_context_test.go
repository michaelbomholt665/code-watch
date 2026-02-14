package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateProjectConfig(t *testing.T) {
	dir := t.TempDir()
	template := filepath.Join(dir, "circular.example.toml")
	target := filepath.Join(dir, "circular.toml")

	if err := os.WriteFile(template, []byte("version = 2\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	generated, err := GenerateProjectConfig(ProjectContext{
		ConfigFile:   target,
		TemplatePath: template,
	})
	if err != nil {
		t.Fatalf("generate config: %v", err)
	}
	if !generated {
		t.Fatalf("expected generated=true")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "version = 2\n" {
		t.Fatalf("unexpected generated config: %q", string(got))
	}
}

func TestGenerateProjectConfig_Idempotent(t *testing.T) {
	dir := t.TempDir()
	template := filepath.Join(dir, "circular.example.toml")
	target := filepath.Join(dir, "circular.toml")

	if err := os.WriteFile(template, []byte("version = 2\n"), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}
	if err := os.WriteFile(target, []byte("version = 1\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	generated, err := GenerateProjectConfig(ProjectContext{
		ConfigFile:   target,
		TemplatePath: template,
	})
	if err != nil {
		t.Fatalf("generate config: %v", err)
	}
	if generated {
		t.Fatalf("expected generated=false when target already exists")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "version = 1\n" {
		t.Fatalf("expected existing config preserved, got %q", string(got))
	}
}

func TestGenerateProjectScript(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "circular-mcp")

	generated, err := GenerateProjectScript(ProjectContext{ScriptFile: target})
	if err != nil {
		t.Fatalf("generate script: %v", err)
	}
	if !generated {
		t.Fatalf("expected generated=true")
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("expected mode 0755, got %o", info.Mode().Perm())
	}
}

func TestGenerateProjectScript_Idempotent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "circular-mcp")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho existing\n"), 0o755); err != nil {
		t.Fatalf("write target: %v", err)
	}

	generated, err := GenerateProjectScript(ProjectContext{ScriptFile: target})
	if err != nil {
		t.Fatalf("generate script: %v", err)
	}
	if generated {
		t.Fatalf("expected generated=false for existing script")
	}
}
