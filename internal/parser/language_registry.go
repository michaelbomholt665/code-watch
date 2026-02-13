package parser

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

type LanguageSpec struct {
	Name                string
	GrammarDir          string
	Extensions          []string
	Filenames           []string
	TestFileSuffixes    []string
	Enabled             bool
	ExtractorReady      bool
	RequireVerification bool
}

type LanguageOverride struct {
	Enabled    *bool
	Extensions []string
	Filenames  []string
}

func DefaultLanguageRegistry() map[string]LanguageSpec {
	return map[string]LanguageSpec{
		"css": {
			Name:                "css",
			GrammarDir:          "css",
			Extensions:          []string{".css"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"go": {
			Name:                "go",
			GrammarDir:          "go",
			Extensions:          []string{".go"},
			TestFileSuffixes:    []string{"_test.go"},
			Enabled:             true,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"gomod": {
			Name:                "gomod",
			GrammarDir:          "gomod",
			Filenames:           []string{"go.mod"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"gosum": {
			Name:                "gosum",
			GrammarDir:          "gosum",
			Filenames:           []string{"go.sum"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"html": {
			Name:                "html",
			GrammarDir:          "html",
			Extensions:          []string{".html", ".htm"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"java": {
			Name:                "java",
			GrammarDir:          "java",
			Extensions:          []string{".java"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"javascript": {
			Name:                "javascript",
			GrammarDir:          "javascript",
			Extensions:          []string{".js", ".cjs", ".mjs"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"python": {
			Name:                "python",
			GrammarDir:          "python",
			Extensions:          []string{".py"},
			TestFileSuffixes:    []string{"_test.py"},
			Enabled:             true,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"rust": {
			Name:                "rust",
			GrammarDir:          "rust",
			Extensions:          []string{".rs"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"tsx": {
			Name:                "tsx",
			GrammarDir:          "tsx",
			Extensions:          []string{".tsx"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
		"typescript": {
			Name:                "typescript",
			GrammarDir:          "typescript",
			Extensions:          []string{".ts"},
			Enabled:             false,
			ExtractorReady:      true,
			RequireVerification: true,
		},
	}
}

func BuildLanguageRegistry(overrides map[string]LanguageOverride) (map[string]LanguageSpec, error) {
	registry := cloneLanguageRegistry(DefaultLanguageRegistry())
	if overrides == nil {
		return registry, nil
	}

	for language, override := range overrides {
		spec, ok := registry[language]
		if !ok {
			return nil, fmt.Errorf("unknown language override %q", language)
		}
		if override.Enabled != nil {
			spec.Enabled = *override.Enabled
		}
		if len(override.Extensions) > 0 {
			spec.Extensions = normalizeExtensions(override.Extensions)
		}
		if len(override.Filenames) > 0 {
			spec.Filenames = normalizeFilenames(override.Filenames)
		}
		registry[language] = spec
	}

	if err := validateLanguageRegistry(registry); err != nil {
		return nil, err
	}
	return registry, nil
}

func cloneLanguageRegistry(in map[string]LanguageSpec) map[string]LanguageSpec {
	out := make(map[string]LanguageSpec, len(in))
	for id, spec := range in {
		copySpec := spec
		copySpec.Extensions = append([]string(nil), spec.Extensions...)
		copySpec.Filenames = append([]string(nil), spec.Filenames...)
		copySpec.TestFileSuffixes = append([]string(nil), spec.TestFileSuffixes...)
		out[id] = copySpec
	}
	return out
}

func validateLanguageRegistry(registry map[string]LanguageSpec) error {
	extOwner := make(map[string]string)
	filenameOwner := make(map[string]string)

	for _, id := range sortedRegistryIDs(registry) {
		spec := registry[id]
		if !spec.Enabled {
			continue
		}
		for _, ext := range normalizeExtensions(spec.Extensions) {
			if existing, ok := extOwner[ext]; ok && existing != id {
				return fmt.Errorf("duplicate extension %q owned by %q and %q", ext, existing, id)
			}
			extOwner[ext] = id
		}
		for _, filename := range normalizeFilenames(spec.Filenames) {
			if existing, ok := filenameOwner[filename]; ok && existing != id {
				return fmt.Errorf("duplicate filename %q owned by %q and %q", filename, existing, id)
			}
			filenameOwner[filename] = id
		}
	}
	return nil
}

func normalizeExtensions(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		raw := strings.TrimSpace(strings.ToLower(value))
		if raw == "" {
			continue
		}
		if !strings.HasPrefix(raw, ".") {
			raw = "." + raw
		}
		if seen[raw] {
			continue
		}
		seen[raw] = true
		out = append(out, raw)
	}
	sort.Strings(out)
	return out
}

func normalizeFilenames(values []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(values))
	for _, value := range values {
		raw := strings.TrimSpace(strings.ToLower(path.Base(value)))
		if raw == "" {
			continue
		}
		if seen[raw] {
			continue
		}
		seen[raw] = true
		out = append(out, raw)
	}
	sort.Strings(out)
	return out
}

func sortedRegistryIDs(registry map[string]LanguageSpec) []string {
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
