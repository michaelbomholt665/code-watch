package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewGrammarLoaderWithRegistry_VerificationFailure(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "go", "go.so"), []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "go", "node-types.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := `
version = 1
allowed_aib_versions = [15]

[[artifacts]]
language = "go"
aib_version = 15
so_path = "go/go.so"
so_sha256 = "` + strings.Repeat("0", 64) + `"
node_types_path = "go/node-types.json"
node_types_sha256 = "` + strings.Repeat("0", 64) + `"
`
	if err := os.WriteFile(filepath.Join(base, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	registry := map[string]LanguageSpec{
		"go": {
			Name:                "go",
			Enabled:             true,
			Extensions:          []string{".go"},
			RequireVerification: true,
		},
	}
	if _, err := NewGrammarLoaderWithRegistry(base, registry, true); err == nil {
		t.Fatal("expected verification failure")
	}
}

func TestNewGrammarLoaderWithRegistry_EnableExpandedLanguages(t *testing.T) {
	trueVal := true
	registry, err := BuildLanguageRegistry(map[string]LanguageOverride{
		"javascript": {Enabled: &trueVal},
		"typescript": {Enabled: &trueVal},
		"tsx":        {Enabled: &trueVal},
		"java":       {Enabled: &trueVal},
		"rust":       {Enabled: &trueVal},
		"html":       {Enabled: &trueVal},
		"css":        {Enabled: &trueVal},
		"gomod":      {Enabled: &trueVal},
		"gosum":      {Enabled: &trueVal},
	})
	if err != nil {
		t.Fatal(err)
	}

	loader, err := NewGrammarLoaderWithRegistry("./grammars", registry, false)
	if err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"go",
		"python",
		"javascript",
		"typescript",
		"tsx",
		"java",
		"rust",
		"html",
		"css",
	}
	for _, language := range cases {
		if loader.languages[language] == nil {
			t.Fatalf("expected runtime grammar for %s", language)
		}
	}

	// go.mod and go.sum use raw-text extraction and intentionally do not load runtime grammars.
	if loader.languages["gomod"] != nil {
		t.Fatal("expected gomod runtime grammar to be nil")
	}
	if loader.languages["gosum"] != nil {
		t.Fatal("expected gosum runtime grammar to be nil")
	}
}
