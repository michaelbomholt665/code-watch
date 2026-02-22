package config

import (
	"circular/internal/shared/version"
	"fmt"
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

	// Apply migration from old versions
	migrate(&cfg)

	// Apply defaults first to ensure we have a base
	applyDefaults(&cfg)

	// Normalize projects and MCP
	normalizeProjects(&cfg)
	normalizeMCP(&cfg)

	// Apply environment variable overrides after defaults but before validation
	ApplyEnvOverrides(&cfg)

	if errs := Validate(&cfg); len(errs) > 0 {
		var messages []string
		for _, e := range errs {
			messages = append(messages, e.Error())
		}
		return nil, fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(messages, "\n  - "))
	}

	return &cfg, nil
}

func DefaultConfig() *Config {
	cfg := &Config{}
	applyDefaults(cfg)
	return cfg
}

func migrate(cfg *Config) {
	if cfg.Version == 0 {
		cfg.Version = 1
	}

	// In a real implementation, we could use toml.MetaData to find unknown keys
	// and suggest their replacements here.

	if cfg.Version == 1 {
		// Version 1 -> 2 migration logic
		// Example: In v1, caches might have been implicit or different.
		// If needed, move fields here.

		// For now, we just bump the version and let applyDefaults handle missing fields.
		cfg.Version = 2
	}
}

func applyDefaults(cfg *Config) {
	if cfg.Version == 0 {
		cfg.Version = 2
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

	// MCP Rate Limiting defaults
	if cfg.MCP.RateLimit.RequestsPerMinute == 0 {
		cfg.MCP.RateLimit.RequestsPerMinute = 60
	}
	if cfg.MCP.RateLimit.Burst == 0 {
		cfg.MCP.RateLimit.Burst = 10
	}
	if cfg.MCP.RateLimit.SSERequestsPerMinute == 0 {
		cfg.MCP.RateLimit.SSERequestsPerMinute = 30
	}
	if cfg.MCP.RateLimit.SSEConnectionsPerMinute == 0 {
		cfg.MCP.RateLimit.SSEConnectionsPerMinute = 5
	}
	if cfg.MCP.RateLimit.Weights == nil {
		cfg.MCP.RateLimit.Weights = map[string]int{
			"scan.run":     5,
			"secrets.scan": 3,
			"graph.cycles": 1,
		}
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

	if cfg.Resolver.BridgeScoring.ConfirmedThreshold <= 0 {
		cfg.Resolver.BridgeScoring.ConfirmedThreshold = 8
	}
	if cfg.Resolver.BridgeScoring.ProbableThreshold <= 0 {
		cfg.Resolver.BridgeScoring.ProbableThreshold = 5
	}
	if cfg.Resolver.BridgeScoring.WeightExplicitRuleMatch == 0 {
		cfg.Resolver.BridgeScoring.WeightExplicitRuleMatch = 10
	}
	if cfg.Resolver.BridgeScoring.WeightBridgeContext == 0 {
		cfg.Resolver.BridgeScoring.WeightBridgeContext = 4
	}
	if cfg.Resolver.BridgeScoring.WeightBridgeImportEvidence == 0 {
		cfg.Resolver.BridgeScoring.WeightBridgeImportEvidence = 3
	}
	if cfg.Resolver.BridgeScoring.WeightUniqueCrossLangMatch == 0 {
		cfg.Resolver.BridgeScoring.WeightUniqueCrossLangMatch = 2
	}
	if cfg.Resolver.BridgeScoring.WeightAmbiguousCrossLangMatch == 0 {
		cfg.Resolver.BridgeScoring.WeightAmbiguousCrossLangMatch = -2
	}
	if cfg.Resolver.BridgeScoring.WeightLocalOrModuleConflict == 0 {
		cfg.Resolver.BridgeScoring.WeightLocalOrModuleConflict = -4
	}
	if cfg.Resolver.BridgeScoring.WeightStdlibConflict == 0 {
		cfg.Resolver.BridgeScoring.WeightStdlibConflict = -3
	}

	if cfg.Caches.Files <= 0 {
		cfg.Caches.Files = 1000
	}
	if cfg.Caches.FileContents <= 0 {
		cfg.Caches.FileContents = 1000
	}

	if cfg.Performance.MaxHeapMB <= 0 {
		cfg.Performance.MaxHeapMB = 2048
	}

	if cfg.Observability.Port == 0 {
		cfg.Observability.Port = 9090
	}
	if cfg.Observability.ServiceName == "" {
		cfg.Observability.ServiceName = "circular"
	}

	// Write queue defaults.
	if cfg.WriteQueue.Enabled == nil {
		enabled := true
		cfg.WriteQueue.Enabled = &enabled
	}
	if cfg.WriteQueue.MemoryCapacity <= 0 {
		cfg.WriteQueue.MemoryCapacity = 2048
	}
	if cfg.WriteQueue.PersistentEnabled == nil {
		enabled := true
		cfg.WriteQueue.PersistentEnabled = &enabled
	}
	if strings.TrimSpace(cfg.WriteQueue.SpoolPath) == "" {
		cfg.WriteQueue.SpoolPath = "data/database/write_spool.db"
	}
	if cfg.WriteQueue.BatchSize <= 0 {
		cfg.WriteQueue.BatchSize = 128
	}
	if cfg.WriteQueue.FlushInterval <= 0 {
		cfg.WriteQueue.FlushInterval = 250 * time.Millisecond
	}
	if cfg.WriteQueue.ShutdownDrainTimeout <= 0 {
		cfg.WriteQueue.ShutdownDrainTimeout = 10 * time.Second
	}
	if cfg.WriteQueue.RetryBaseDelay <= 0 {
		cfg.WriteQueue.RetryBaseDelay = 500 * time.Millisecond
	}
	if cfg.WriteQueue.RetryMaxDelay <= 0 {
		cfg.WriteQueue.RetryMaxDelay = 30 * time.Second
	}
	if cfg.WriteQueue.SyncFallback == nil {
		enabled := true
		cfg.WriteQueue.SyncFallback = &enabled
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
