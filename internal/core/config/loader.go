package config

import (
	"circular/internal/shared/version"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

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
	normalizeProjects(&cfg)
	normalizeMCP(&cfg)

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
	if err := validateDynamicGrammars(&cfg); err != nil {
		return nil, err
	}
	if err := validateSecrets(&cfg); err != nil {
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
	if strings.TrimSpace(cfg.MCP.ConfigPath) == "" && strings.TrimSpace(cfg.ConfigFiles.ActiveFile) != "" {
		cfg.MCP.ConfigPath = cfg.ConfigFiles.ActiveFile
	}
	if strings.TrimSpace(cfg.MCP.ServerName) == "" {
		cfg.MCP.ServerName = "circular"
	}
	if strings.TrimSpace(cfg.MCP.ServerVersion) == "" {
		cfg.MCP.ServerVersion = version.Version
	}
	if cfg.MCP.MaxResponseItems == 0 {
		cfg.MCP.MaxResponseItems = 500
	}
	if cfg.MCP.RequestTimeout <= 0 {
		cfg.MCP.RequestTimeout = 30 * time.Second
	}
	if cfg.MCP.AutoManageOutputs == nil {
		enabled := true
		cfg.MCP.AutoManageOutputs = &enabled
	}
	if cfg.MCP.AutoSyncConfig == nil {
		enabled := true
		cfg.MCP.AutoSyncConfig = &enabled
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
	if strings.TrimSpace(cfg.Output.Mermaid) == "" {
		cfg.Output.Mermaid = "graph.mmd"
	}
	if cfg.Output.Diagrams.FlowConfig.MaxDepth == 0 {
		cfg.Output.Diagrams.FlowConfig.MaxDepth = 8
	}
	if strings.TrimSpace(cfg.Output.Report.Verbosity) == "" {
		cfg.Output.Report.Verbosity = "standard"
	}
	if cfg.Output.Report.TableOfContents == nil {
		enabled := true
		cfg.Output.Report.TableOfContents = &enabled
	}
	if cfg.Output.Report.CollapsibleSections == nil {
		enabled := true
		cfg.Output.Report.CollapsibleSections = &enabled
	}
	if cfg.Output.Report.IncludeMermaid == nil {
		enabled := false
		cfg.Output.Report.IncludeMermaid = &enabled
	}
	if cfg.Secrets.EntropyThreshold <= 0 {
		cfg.Secrets.EntropyThreshold = 4.0
	}
	if cfg.Secrets.MinTokenLength <= 0 {
		cfg.Secrets.MinTokenLength = 20
	}
}

func normalizeProjects(cfg *Config) {
	cfg.Projects.Active = strings.TrimSpace(cfg.Projects.Active)
	cfg.Projects.RegistryFile = strings.TrimSpace(cfg.Projects.RegistryFile)
	for i := range cfg.Projects.Entries {
		entry := &cfg.Projects.Entries[i]
		entry.Name = strings.TrimSpace(entry.Name)
		entry.Root = strings.TrimSpace(entry.Root)
		entry.DBNamespace = normalizeProjectNamespace(entry.DBNamespace, entry.Name)
		entry.ConfigFile = strings.TrimSpace(entry.ConfigFile)
	}
}

func normalizeProjectNamespace(raw, fallback string) string {
	namespace := strings.TrimSpace(raw)
	if namespace == "" {
		namespace = strings.TrimSpace(fallback)
	}
	return namespace
}

func normalizeMCP(cfg *Config) {
	cfg.MCP.Mode = strings.TrimSpace(cfg.MCP.Mode)
	cfg.MCP.Transport = strings.TrimSpace(cfg.MCP.Transport)
	cfg.MCP.Address = strings.TrimSpace(cfg.MCP.Address)
	cfg.MCP.ConfigPath = strings.TrimSpace(cfg.MCP.ConfigPath)
	cfg.MCP.OpenAPISpecPath = strings.TrimSpace(cfg.MCP.OpenAPISpecPath)
	cfg.MCP.OpenAPISpecURL = strings.TrimSpace(cfg.MCP.OpenAPISpecURL)
	cfg.MCP.ServerName = strings.TrimSpace(cfg.MCP.ServerName)
	cfg.MCP.ServerVersion = strings.TrimSpace(cfg.MCP.ServerVersion)
	cfg.MCP.ExposedToolName = strings.TrimSpace(cfg.MCP.ExposedToolName)
	if len(cfg.MCP.OperationAllowlist) == 0 {
		return
	}
	normalized := make([]string, 0, len(cfg.MCP.OperationAllowlist))
	for _, op := range cfg.MCP.OperationAllowlist {
		op = strings.ToLower(strings.TrimSpace(op))
		if op == "" {
			continue
		}
		normalized = append(normalized, op)
	}
	cfg.MCP.OperationAllowlist = normalized
}
