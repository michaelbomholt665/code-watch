package helpers

import (
	"circular/internal/core/config"
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
	roots := make([]string, 0, len(paths))
	for _, p := range paths {
		normalized := filepath.Clean(p)
		if abs, err := filepath.Abs(normalized); err == nil {
			normalized = filepath.Clean(abs)
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		roots = append(roots, normalized)
	}
	sort.Strings(roots)
	return roots
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
		model.Rules = append(model.Rules, graph.ArchitectureRule{
			Name:  rule.Name,
			From:  rule.From,
			Allow: append([]string(nil), rule.Allow...),
		})
	}
	return model
}
