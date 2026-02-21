package helpers

import (
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"circular/internal/engine/graph"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gobwas/glob"
)

func CompileGlobs(patterns []string, label string) ([]glob.Glob, error) {
	out := make([]glob.Glob, 0, len(patterns))
	for _, p := range patterns {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid %s pattern %q: %w", label, p, err)
		}
		out = append(out, g)
	}
	return out, nil
}

func UniqueScanRoots(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	normalized := make([]string, 0, len(paths))
	for _, p := range paths {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		cleaned := filepath.Clean(trimmed)
		if abs, err := filepath.Abs(cleaned); err == nil {
			cleaned = filepath.Clean(abs)
		}
		if seen[cleaned] {
			continue
		}
		seen[cleaned] = true
		normalized = append(normalized, cleaned)
	}
	if len(normalized) <= 1 {
		return normalized
	}
	sort.Slice(normalized, func(i, j int) bool {
		if len(normalized[i]) == len(normalized[j]) {
			return normalized[i] < normalized[j]
		}
		return len(normalized[i]) < len(normalized[j])
	})
	roots := make([]string, 0, len(normalized))
	for _, candidate := range normalized {
		isChild := false
		for _, root := range roots {
			if isSubpath(root, candidate) {
				isChild = true
				break
			}
		}
		if !isChild {
			roots = append(roots, candidate)
		}
	}
	sort.Strings(roots)
	return roots
}

func isSubpath(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." {
		return false
	}
	if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}
	return true
}

func FindContainingWatchPath(path string, watchPaths []string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve file path %q: %w", path, err)
	}

	for _, watchPath := range watchPaths {
		absWatchPath, err := filepath.Abs(watchPath)
		if err != nil {
			return "", fmt.Errorf("resolve watch path %q: %w", watchPath, err)
		}

		rel, err := filepath.Rel(absWatchPath, absPath)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
			return absWatchPath, nil
		}
	}

	return "", fmt.Errorf("python file %q is not under any configured watch path", path)
}

func ArchitectureModelFromConfig(arch config.Architecture) graph.ArchitectureModel {
	model := graph.ArchitectureModel{
		Enabled: arch.Enabled,
		Layers:  make([]graph.ArchitectureLayer, 0, len(arch.Layers)),
		Rules:   make([]graph.ArchitectureRule, 0, len(arch.Rules)),
	}
	for _, layer := range arch.Layers {
		model.Layers = append(model.Layers, graph.ArchitectureLayer{
			Name:  layer.Name,
			Paths: append([]string(nil), layer.Paths...),
		})
	}
	for _, rule := range arch.Rules {
		kind := strings.TrimSpace(strings.ToLower(rule.Kind))
		if kind == "" && (rule.From != "" || len(rule.Allow) > 0) {
			kind = "layer"
		} else if kind == "" && len(rule.Modules) > 0 {
			kind = "package"
		}
		if kind != "" && kind != "layer" {
			continue
		}
		model.Rules = append(model.Rules, graph.ArchitectureRule{
			Name:  rule.Name,
			From:  rule.From,
			Allow: append([]string(nil), rule.Allow...),
		})
	}
	return model
}

func ArchitectureRulesFromConfig(arch config.Architecture) []ports.ArchitectureRule {
	if len(arch.Rules) == 0 {
		return nil
	}
	out := make([]ports.ArchitectureRule, 0, len(arch.Rules))
	for _, rule := range arch.Rules {
		kind := strings.TrimSpace(strings.ToLower(rule.Kind))
		if kind == "" && (rule.From != "" || len(rule.Allow) > 0) {
			kind = "layer"
		} else if kind == "" && len(rule.Modules) > 0 {
			kind = "package"
		}
		if kind != "package" {
			continue
		}
		out = append(out, ports.ArchitectureRule{
			Name:     rule.Name,
			Kind:     ports.ArchitectureRuleKindPackage,
			Modules:  append([]string(nil), rule.Modules...),
			MaxFiles: rule.MaxFiles,
			Imports: ports.ArchitectureImportRule{
				Allow: append([]string(nil), rule.Imports.Allow...),
				Deny:  append([]string(nil), rule.Imports.Deny...),
			},
			Exclude: ports.ArchitectureRuleExclude{
				Tests: rule.Exclude.Tests,
				Files: append([]string(nil), rule.Exclude.Files...),
			},
		})
	}
	return out
}
