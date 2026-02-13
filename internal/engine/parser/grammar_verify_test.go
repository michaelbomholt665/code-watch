package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyGrammarArtifacts_DetectsChecksumMismatch(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "javascript"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "javascript", "javascript.so"), []byte("so"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "javascript", "node-types.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := GrammarManifest{
		Version:            1,
		AllowedAIBVersions: []int{14, 15},
		Artifacts: []GrammarArtifact{
			{
				Language:         "javascript",
				AIBVersion:       14,
				SharedObjectPath: "javascript/javascript.so",
				SharedObjectHash: strings.Repeat("0", 64),
				NodeTypesPath:    "javascript/node-types.json",
				NodeTypesHash:    strings.Repeat("0", 64),
			},
		},
	}

	issues, err := VerifyGrammarArtifacts(base, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 checksum mismatch issues, got %d", len(issues))
	}
}

func TestVerifyLanguageRegistryArtifacts_MissingManifestEntry(t *testing.T) {
	base := t.TempDir()
	manifest := `
version = 1
allowed_aib_versions = [14, 15]

[[artifacts]]
language = "go"
aib_version = 15
so_path = "go/go.so"
so_sha256 = "abc"
node_types_path = "go/node-types.json"
node_types_sha256 = "def"
`
	if err := os.WriteFile(filepath.Join(base, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(base, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "go", "go.so"), []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "go", "node-types.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	enable := true
	registry := map[string]LanguageSpec{
		"go": {
			Name:                "go",
			Enabled:             true,
			RequireVerification: true,
		},
		"javascript": {
			Name:                "javascript",
			Enabled:             enable,
			RequireVerification: true,
		},
	}

	issues, err := VerifyLanguageRegistryArtifacts(base, registry)
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Fatal("expected at least one issue")
	}
	foundMissing := false
	for _, issue := range issues {
		if issue.Language == "javascript" && strings.Contains(issue.Reason, "missing") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatalf("expected missing manifest issue, got %#v", issues)
	}
}
