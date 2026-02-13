package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	GrammarsPath string       `toml:"grammars_path"`
	WatchPaths   []string     `toml:"watch_paths"`
	Exclude      Exclude      `toml:"exclude"`
	Watch        Watch        `toml:"watch"`
	Output       Output       `toml:"output"`
	Alerts       Alerts       `toml:"alerts"`
	Architecture Architecture `toml:"architecture"`
}

type Exclude struct {
	Dirs    []string `toml:"dirs"`
	Files   []string `toml:"files"`
	Symbols []string `toml:"symbols"` // Prefixes to ignore (e.g., self., ctx.)
}

type Watch struct {
	Debounce time.Duration `toml:"debounce"`
}

type Output struct {
	DOT            string              `toml:"dot"`
	TSV            string              `toml:"tsv"`
	Mermaid        string              `toml:"mermaid"`
	PlantUML       string              `toml:"plantuml"`
	UpdateMarkdown []MarkdownInjection `toml:"update_markdown"`
	Paths          OutputPaths         `toml:"paths"`
}

type MarkdownInjection struct {
	File   string `toml:"file"`
	Marker string `toml:"marker"`
	Format string `toml:"format"`
}

type OutputPaths struct {
	Root        string `toml:"root"`
	DiagramsDir string `toml:"diagrams_dir"`
}

type Alerts struct {
	Beep     bool `toml:"beep"`
	Terminal bool `toml:"terminal"`
}

type Architecture struct {
	Enabled       bool                `toml:"enabled"`
	TopComplexity int                 `toml:"top_complexity"`
	Layers        []ArchitectureLayer `toml:"layers"`
	Rules         []ArchitectureRule  `toml:"rules"`
}

type ArchitectureLayer struct {
	Name  string   `toml:"name"`
	Paths []string `toml:"paths"`
}

type ArchitectureRule struct {
	Name  string   `toml:"name"`
	From  string   `toml:"from"`
	Allow []string `toml:"allow"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, err
	}

	// Default debounce if not set.
	if cfg.Watch.Debounce == 0 {
		cfg.Watch.Debounce = 500 * time.Millisecond
	}

	if len(cfg.WatchPaths) == 0 {
		cfg.WatchPaths = []string{"."}
	}

	// Keep architecture checks optional and backward compatible.
	if cfg.Architecture.TopComplexity <= 0 {
		cfg.Architecture.TopComplexity = 5
	}
	if strings.TrimSpace(cfg.Output.Paths.DiagramsDir) == "" {
		cfg.Output.Paths.DiagramsDir = "docs/diagrams"
	}

	if err := validateArchitecture(&cfg); err != nil {
		return nil, err
	}
	if err := validateOutput(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateOutput(cfg *Config) error {
	if strings.TrimSpace(cfg.Output.Paths.DiagramsDir) == "" {
		return fmt.Errorf("output.paths.diagrams_dir must not be empty")
	}

	seen := make(map[string]bool, len(cfg.Output.UpdateMarkdown))
	for i, injection := range cfg.Output.UpdateMarkdown {
		ref := fmt.Sprintf("output.update_markdown[%d]", i)
		file := strings.TrimSpace(injection.File)
		if file == "" {
			return fmt.Errorf("%s.file must not be empty", ref)
		}
		marker := strings.TrimSpace(injection.Marker)
		if marker == "" {
			return fmt.Errorf("%s.marker must not be empty", ref)
		}
		format := strings.ToLower(strings.TrimSpace(injection.Format))
		if format != "mermaid" && format != "plantuml" {
			return fmt.Errorf("%s.format must be one of: mermaid, plantuml", ref)
		}
		key := file + "|" + marker + "|" + format
		if seen[key] {
			return fmt.Errorf("duplicate markdown injection target: file=%q marker=%q format=%q", file, marker, format)
		}
		seen[key] = true
	}
	return nil
}

func validateArchitecture(cfg *Config) error {
	arch := cfg.Architecture
	if !arch.Enabled {
		return nil
	}

	if len(arch.Layers) == 0 {
		return fmt.Errorf("architecture.enabled=true requires at least one layer")
	}

	layerNames := make(map[string]bool, len(arch.Layers))
	patternOwner := make(map[string]string)
	literalPaths := make(map[string]string)
	wildcardPatterns := make(map[string]string)

	for i, layer := range arch.Layers {
		layerRef := fmt.Sprintf("architecture.layers[%d]", i)
		if strings.TrimSpace(layer.Name) == "" {
			return fmt.Errorf("%s.name must not be empty", layerRef)
		}
		if layerNames[layer.Name] {
			return fmt.Errorf("duplicate architecture layer name: %q", layer.Name)
		}
		layerNames[layer.Name] = true

		if len(layer.Paths) == 0 {
			return fmt.Errorf("%s (%s) must define at least one path pattern", layerRef, layer.Name)
		}

		for _, rawPath := range layer.Paths {
			path := strings.TrimSpace(filepath.Clean(rawPath))
			if path == "" || path == "." {
				return fmt.Errorf("layer %q has empty/invalid path pattern", layer.Name)
			}

			if owner, ok := patternOwner[path]; ok && owner != layer.Name {
				return fmt.Errorf("layer path pattern %q is declared in both %q and %q", path, owner, layer.Name)
			}
			patternOwner[path] = layer.Name

			if hasWildcard(path) {
				for existing, owner := range literalPaths {
					if owner == layer.Name {
						continue
					}
					if matched, _ := filepath.Match(path, existing); matched {
						return fmt.Errorf("layer %q path %q overlaps with layer %q path %q", layer.Name, path, owner, existing)
					}
				}

				for existing, owner := range wildcardPatterns {
					if owner == layer.Name {
						continue
					}
					if wildcardPatternsOverlap(path, existing) {
						return fmt.Errorf("layer %q path %q overlaps with layer %q path %q", layer.Name, path, owner, existing)
					}
				}

				wildcardPatterns[path] = layer.Name
				continue
			}

			for existing, owner := range literalPaths {
				if owner == layer.Name {
					continue
				}
				if isPathOverlap(existing, path) {
					return fmt.Errorf("layer %q path %q overlaps with layer %q path %q", layer.Name, path, owner, existing)
				}
			}
			for existing, owner := range wildcardPatterns {
				if owner == layer.Name {
					continue
				}
				if matched, _ := filepath.Match(existing, path); matched {
					return fmt.Errorf("layer %q path %q overlaps with layer %q path %q", layer.Name, path, owner, existing)
				}
			}
			literalPaths[path] = layer.Name
		}
	}

	ruleNames := make(map[string]bool, len(arch.Rules))
	ruleByFrom := make(map[string]string, len(arch.Rules))
	for i, rule := range arch.Rules {
		ruleRef := fmt.Sprintf("architecture.rules[%d]", i)
		if strings.TrimSpace(rule.Name) == "" {
			return fmt.Errorf("%s.name must not be empty", ruleRef)
		}
		if ruleNames[rule.Name] {
			return fmt.Errorf("duplicate architecture rule name: %q", rule.Name)
		}
		ruleNames[rule.Name] = true

		if !layerNames[rule.From] {
			return fmt.Errorf("architecture rule %q references unknown from layer %q", rule.Name, rule.From)
		}
		if previous, exists := ruleByFrom[rule.From]; exists {
			return fmt.Errorf("architecture layer %q has multiple rules (%q, %q); define exactly one", rule.From, previous, rule.Name)
		}
		ruleByFrom[rule.From] = rule.Name
		if len(rule.Allow) == 0 {
			return fmt.Errorf("architecture rule %q must include at least one allowed layer", rule.Name)
		}

		allowedSet := make(map[string]bool, len(rule.Allow))
		for _, to := range rule.Allow {
			if !layerNames[to] {
				return fmt.Errorf("architecture rule %q references unknown allowed layer %q", rule.Name, to)
			}
			if allowedSet[to] {
				return fmt.Errorf("architecture rule %q repeats allowed layer %q", rule.Name, to)
			}
			allowedSet[to] = true
		}
	}

	return nil
}

func hasWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[]{}")
}

func isPathOverlap(a, b string) bool {
	if a == b {
		return true
	}
	if strings.HasPrefix(a, b+string(os.PathSeparator)) {
		return true
	}
	if strings.HasPrefix(b, a+string(os.PathSeparator)) {
		return true
	}
	return false
}

func wildcardPatternsOverlap(a, b string) bool {
	if a == b || strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return true
	}

	aPrefix := wildcardPrefix(a)
	bPrefix := wildcardPrefix(b)
	if aPrefix != "" && bPrefix != "" && (strings.HasPrefix(aPrefix, bPrefix) || strings.HasPrefix(bPrefix, aPrefix)) {
		return true
	}

	aSample := wildcardSample(a)
	if aSample != "" {
		if matched, _ := filepath.Match(b, aSample); matched {
			return true
		}
	}

	bSample := wildcardSample(b)
	if bSample != "" {
		if matched, _ := filepath.Match(a, bSample); matched {
			return true
		}
	}

	return false
}

func wildcardPrefix(pattern string) string {
	idx := strings.IndexAny(pattern, "*?[]{}")
	if idx == -1 {
		return pattern
	}
	return pattern[:idx]
}

func wildcardSample(pattern string) string {
	var sample strings.Builder
	inSet := false
	for _, ch := range pattern {
		switch {
		case ch == '[':
			inSet = true
			sample.WriteRune('x')
		case ch == ']':
			inSet = false
		case inSet:
			continue
		case ch == '*' || ch == '?' || ch == '{' || ch == '}' || ch == ',':
			sample.WriteRune('x')
		default:
			sample.WriteRune(ch)
		}
	}
	return sample.String()
}
