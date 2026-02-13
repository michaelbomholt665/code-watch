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
	Version             int                 `toml:"version"`
	Paths               Paths               `toml:"paths"`
	ConfigFiles         ConfigFiles         `toml:"config"`
	DB                  Database            `toml:"db"`
	Projects            Projects            `toml:"projects"`
	MCP                 MCP                 `toml:"mcp"`
	GrammarsPath        string              `toml:"grammars_path"`
	GrammarVerification GrammarVerification `toml:"grammar_verification"`
	Languages           map[string]Language `toml:"languages"`
	WatchPaths          []string            `toml:"watch_paths"`
	Exclude             Exclude             `toml:"exclude"`
	Watch               Watch               `toml:"watch"`
	Output              Output              `toml:"output"`
	Alerts              Alerts              `toml:"alerts"`
	Architecture        Architecture        `toml:"architecture"`
}

type Paths struct {
	ProjectRoot string `toml:"project_root"`
	ConfigDir   string `toml:"config_dir"`
	StateDir    string `toml:"state_dir"`
	CacheDir    string `toml:"cache_dir"`
	DatabaseDir string `toml:"database_dir"`
}

type ConfigFiles struct {
	ActiveFile string   `toml:"active_file"`
	Includes   []string `toml:"includes"`
}

type Database struct {
	Enabled     bool          `toml:"enabled"`
	Driver      string        `toml:"driver"`
	Path        string        `toml:"path"`
	BusyTimeout time.Duration `toml:"busy_timeout"`
	ProjectMode string        `toml:"project_mode"`
}

type Projects struct {
	Active       string         `toml:"active"`
	RegistryFile string         `toml:"registry_file"`
	Entries      []ProjectEntry `toml:"entries"`
}

type ProjectEntry struct {
	Name        string `toml:"name"`
	Root        string `toml:"root"`
	DBNamespace string `toml:"db_namespace"`
}

type MCP struct {
	Enabled    bool   `toml:"enabled"`
	Mode       string `toml:"mode"`
	Transport  string `toml:"transport"`
	Address    string `toml:"address"`
	ConfigPath string `toml:"config_path"`
}

type GrammarVerification struct {
	Enabled *bool `toml:"enabled"`
}

type Language struct {
	Enabled    *bool    `toml:"enabled"`
	Extensions []string `toml:"extensions"`
	Filenames  []string `toml:"filenames"`
}

type Exclude struct {
	Dirs    []string `toml:"dirs"`
	Files   []string `toml:"files"`
	Symbols []string `toml:"symbols"` // Prefixes to ignore (e.g., self., ctx.)
	Imports []string `toml:"imports"` // Import paths to ignore for unused check
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

	applyDefaults(&cfg)

	if err := validateVersion(&cfg); err != nil {
		return nil, err
	}
	if err := validateProjects(&cfg); err != nil {
		return nil, err
	}
	if err := validateDatabase(&cfg); err != nil {
		return nil, err
	}
	if err := validateMCP(&cfg); err != nil {
		return nil, err
	}
	if err := validateArchitecture(&cfg); err != nil {
		return nil, err
	}
	if err := validateOutput(&cfg); err != nil {
		return nil, err
	}
	if err := validateLanguages(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	if strings.TrimSpace(cfg.Paths.ConfigDir) == "" {
		cfg.Paths.ConfigDir = "data/config"
	}
	if strings.TrimSpace(cfg.Paths.StateDir) == "" {
		cfg.Paths.StateDir = "data/state"
	}
	if strings.TrimSpace(cfg.Paths.CacheDir) == "" {
		cfg.Paths.CacheDir = "data/cache"
	}
	if strings.TrimSpace(cfg.Paths.DatabaseDir) == "" {
		cfg.Paths.DatabaseDir = "data/database"
	}

	if strings.TrimSpace(cfg.ConfigFiles.ActiveFile) == "" {
		cfg.ConfigFiles.ActiveFile = "circular.toml"
	}

	if strings.TrimSpace(cfg.DB.Driver) == "" {
		cfg.DB.Driver = "sqlite"
	}
	if strings.TrimSpace(cfg.DB.Path) == "" {
		cfg.DB.Path = "history.db"
	}
	if cfg.DB.BusyTimeout <= 0 {
		cfg.DB.BusyTimeout = 5 * time.Second
	}
	if strings.TrimSpace(cfg.DB.ProjectMode) == "" {
		cfg.DB.ProjectMode = "multi"
	}
	if !cfg.DB.Enabled && cfg.Version <= 1 {
		// Keep v1 compatibility where db block did not exist.
		cfg.DB.Enabled = true
	}

	if strings.TrimSpace(cfg.Projects.RegistryFile) == "" {
		cfg.Projects.RegistryFile = "projects.toml"
	}

	if strings.TrimSpace(cfg.MCP.Mode) == "" {
		cfg.MCP.Mode = "embedded"
	}
	if strings.TrimSpace(cfg.MCP.Transport) == "" {
		cfg.MCP.Transport = "stdio"
	}
	if strings.TrimSpace(cfg.MCP.Address) == "" {
		cfg.MCP.Address = "127.0.0.1:8765"
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
}

func (g GrammarVerification) IsEnabled() bool {
	if g.Enabled == nil {
		return true
	}
	return *g.Enabled
}

func validateVersion(cfg *Config) error {
	if cfg.Version < 1 {
		return fmt.Errorf("version must be >= 1, got %d", cfg.Version)
	}
	if cfg.Version > 2 {
		return fmt.Errorf("unsupported config version %d; supported versions are 1 and 2", cfg.Version)
	}
	return nil
}

func validateDatabase(cfg *Config) error {
	driver := strings.ToLower(strings.TrimSpace(cfg.DB.Driver))
	if driver != "sqlite" {
		return fmt.Errorf("db.driver must be sqlite, got %q", cfg.DB.Driver)
	}
	if strings.TrimSpace(cfg.DB.Path) == "" {
		return fmt.Errorf("db.path must not be empty")
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.DB.ProjectMode))
	if mode != "single" && mode != "multi" {
		return fmt.Errorf("db.project_mode must be one of: single, multi")
	}
	return nil
}

func validateProjects(cfg *Config) error {
	entries := cfg.Projects.Entries
	if len(entries) == 0 {
		if strings.TrimSpace(cfg.Projects.Active) != "" {
			return fmt.Errorf("projects.active is set to %q but projects.entries is empty", cfg.Projects.Active)
		}
		return nil
	}

	seenNames := make(map[string]bool, len(entries))
	for i, entry := range entries {
		ref := fmt.Sprintf("projects.entries[%d]", i)
		name := strings.TrimSpace(entry.Name)
		root := strings.TrimSpace(entry.Root)
		if name == "" {
			return fmt.Errorf("%s.name must not be empty", ref)
		}
		if root == "" {
			return fmt.Errorf("%s.root must not be empty", ref)
		}
		if seenNames[name] {
			return fmt.Errorf("duplicate project name %q", name)
		}
		seenNames[name] = true
	}

	active := strings.TrimSpace(cfg.Projects.Active)
	if active != "" && !seenNames[active] {
		return fmt.Errorf("projects.active references unknown project %q", active)
	}
	return nil
}

func validateMCP(cfg *Config) error {
	mode := strings.ToLower(strings.TrimSpace(cfg.MCP.Mode))
	switch mode {
	case "embedded", "server":
	default:
		return fmt.Errorf("mcp.mode must be one of: embedded, server")
	}

	transport := strings.ToLower(strings.TrimSpace(cfg.MCP.Transport))
	switch transport {
	case "stdio", "http":
	default:
		return fmt.Errorf("mcp.transport must be one of: stdio, http")
	}

	if transport == "http" && strings.TrimSpace(cfg.MCP.Address) == "" {
		return fmt.Errorf("mcp.address must not be empty when mcp.transport=http")
	}
	if cfg.MCP.Enabled && mode == "embedded" && transport == "http" {
		return fmt.Errorf("mcp transport http is only valid with mcp.mode=server")
	}
	return nil
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

func validateLanguages(cfg *Config) error {
	for language, settings := range cfg.Languages {
		if strings.TrimSpace(language) == "" {
			return fmt.Errorf("languages key must not be empty")
		}
		for _, ext := range settings.Extensions {
			if strings.TrimSpace(ext) == "" {
				return fmt.Errorf("languages.%s.extensions must not include empty values", language)
			}
		}
		for _, name := range settings.Filenames {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("languages.%s.filenames must not include empty values", language)
			}
		}
	}
	return nil
}
