package grammar

import (
	"circular/internal/engine/parser/registry"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type VerificationIssue struct {
	Language     string
	ArtifactKind string
	ArtifactPath string
	ExpectedHash string
	ActualHash   string
	Reason       string
}

func VerifyGrammarArtifacts(baseDir string, manifest GrammarManifest) ([]VerificationIssue, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("baseDir must not be empty")
	}

	info, err := os.Stat(baseDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("grammar base path is not a directory: %s", baseDir)
	}

	allowed := make(map[int]bool, len(manifest.AllowedAIBVersions))
	for _, version := range manifest.AllowedAIBVersions {
		allowed[version] = true
	}

	issues := make([]VerificationIssue, 0)
	for _, artifact := range manifest.Artifacts {
		if !allowed[artifact.AIBVersion] {
			issues = append(issues, VerificationIssue{
				Language: artifact.Language,
				Reason:   fmt.Sprintf("unsupported AIB version %d", artifact.AIBVersion),
			})
		}
		issues = append(issues, verifyArtifactHash(baseDir, artifact.Language, "shared-object", artifact.SharedObjectPath, artifact.SharedObjectHash)...)
		issues = append(issues, verifyArtifactHash(baseDir, artifact.Language, "node-types", artifact.NodeTypesPath, artifact.NodeTypesHash)...)
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Language != issues[j].Language {
			return issues[i].Language < issues[j].Language
		}
		if issues[i].ArtifactKind != issues[j].ArtifactKind {
			return issues[i].ArtifactKind < issues[j].ArtifactKind
		}
		if issues[i].ArtifactPath != issues[j].ArtifactPath {
			return issues[i].ArtifactPath < issues[j].ArtifactPath
		}
		return issues[i].Reason < issues[j].Reason
	})
	return issues, nil
}

func VerifyLanguageRegistryArtifacts(baseDir string, registry map[string]registry.LanguageSpec) ([]VerificationIssue, error) {
	manifestPath := filepath.Join(baseDir, "manifest.toml")
	manifest, err := LoadGrammarManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	enabled := make(map[string]bool)
	requiredVerification := make(map[string]bool)
	for language, spec := range registry {
		if !spec.Enabled {
			continue
		}
		enabled[language] = true
		if spec.RequireVerification {
			requiredVerification[language] = true
		}
	}

	manifestLanguages := make(map[string]bool, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		manifestLanguages[artifact.Language] = true
	}

	issues, err := VerifyGrammarArtifacts(baseDir, manifest)
	if err != nil {
		return nil, err
	}

	filtered := make([]VerificationIssue, 0)
	for _, issue := range issues {
		if enabled[issue.Language] {
			filtered = append(filtered, issue)
		}
	}

	for language := range requiredVerification {
		if !manifestLanguages[language] {
			filtered = append(filtered, VerificationIssue{
				Language: language,
				Reason:   "language missing from manifest",
			})
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Language != filtered[j].Language {
			return filtered[i].Language < filtered[j].Language
		}
		if filtered[i].ArtifactKind != filtered[j].ArtifactKind {
			return filtered[i].ArtifactKind < filtered[j].ArtifactKind
		}
		if filtered[i].ArtifactPath != filtered[j].ArtifactPath {
			return filtered[i].ArtifactPath < filtered[j].ArtifactPath
		}
		return filtered[i].Reason < filtered[j].Reason
	})
	return filtered, nil
}

func verifyArtifactHash(baseDir, language, kind, relPath, expectedHash string) []VerificationIssue {
	fullPath := filepath.Join(baseDir, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return []VerificationIssue{{
			Language:     language,
			ArtifactKind: kind,
			ArtifactPath: relPath,
			ExpectedHash: expectedHash,
			ActualHash:   "<missing>",
			Reason:       "artifact missing or unreadable",
		}}
	}

	actual := fmt.Sprintf("%x", sha256.Sum256(data))
	if actual == expectedHash {
		return nil
	}
	return []VerificationIssue{{
		Language:     language,
		ArtifactKind: kind,
		ArtifactPath: relPath,
		ExpectedHash: expectedHash,
		ActualHash:   actual,
		Reason:       "checksum mismatch",
	}}
}
