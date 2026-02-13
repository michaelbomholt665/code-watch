package grammar

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type GrammarManifest struct {
	Version            int               `toml:"version"`
	AllowedAIBVersions []int             `toml:"allowed_aib_versions"`
	Artifacts          []GrammarArtifact `toml:"artifacts"`
}

type GrammarArtifact struct {
	Language         string `toml:"language"`
	AIBVersion       int    `toml:"aib_version"`
	SharedObjectPath string `toml:"so_path"`
	SharedObjectHash string `toml:"so_sha256"`
	NodeTypesPath    string `toml:"node_types_path"`
	NodeTypesHash    string `toml:"node_types_sha256"`
	Source           string `toml:"source"`
	ApprovedDate     string `toml:"approved_date"`
}

func LoadGrammarManifest(path string) (GrammarManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return GrammarManifest{}, err
	}

	var manifest GrammarManifest
	if _, err := toml.Decode(string(data), &manifest); err != nil {
		return GrammarManifest{}, err
	}

	if manifest.Version <= 0 {
		return GrammarManifest{}, fmt.Errorf("manifest version must be > 0")
	}
	if len(manifest.AllowedAIBVersions) == 0 {
		return GrammarManifest{}, fmt.Errorf("manifest must define allowed_aib_versions")
	}
	if len(manifest.Artifacts) == 0 {
		return GrammarManifest{}, fmt.Errorf("manifest must define at least one artifact")
	}

	seen := make(map[string]bool, len(manifest.Artifacts))
	for i, artifact := range manifest.Artifacts {
		ref := fmt.Sprintf("artifacts[%d]", i)
		artifact.Language = strings.TrimSpace(strings.ToLower(artifact.Language))
		artifact.SharedObjectPath = filepath.Clean(strings.TrimSpace(artifact.SharedObjectPath))
		artifact.NodeTypesPath = filepath.Clean(strings.TrimSpace(artifact.NodeTypesPath))
		artifact.SharedObjectHash = strings.TrimSpace(strings.ToLower(artifact.SharedObjectHash))
		artifact.NodeTypesHash = strings.TrimSpace(strings.ToLower(artifact.NodeTypesHash))
		artifact.Source = strings.TrimSpace(artifact.Source)
		artifact.ApprovedDate = strings.TrimSpace(artifact.ApprovedDate)

		if artifact.Language == "" {
			return GrammarManifest{}, fmt.Errorf("%s.language must not be empty", ref)
		}
		if seen[artifact.Language] {
			return GrammarManifest{}, fmt.Errorf("duplicate language entry %q in manifest", artifact.Language)
		}
		seen[artifact.Language] = true
		if artifact.AIBVersion <= 0 {
			return GrammarManifest{}, fmt.Errorf("%s.aib_version must be > 0", ref)
		}
		if artifact.SharedObjectPath == "" || artifact.SharedObjectHash == "" {
			return GrammarManifest{}, fmt.Errorf("%s.so_path and so_sha256 must not be empty", ref)
		}
		if artifact.NodeTypesPath == "" || artifact.NodeTypesHash == "" {
			return GrammarManifest{}, fmt.Errorf("%s.node_types_path and node_types_sha256 must not be empty", ref)
		}
		manifest.Artifacts[i] = artifact
	}

	return manifest, nil
}
