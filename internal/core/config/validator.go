package config

import (
	"circular/internal/core/config/helpers"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

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
	seenNamespaces := make(map[string]bool, len(entries))
	for i, entry := range entries {
		ref := fmt.Sprintf("projects.entries[%d]", i)
		name := strings.TrimSpace(entry.Name)
		root := strings.TrimSpace(entry.Root)
		namespace := strings.TrimSpace(entry.DBNamespace)
		if name == "" {
			return fmt.Errorf("%s.name must not be empty", ref)
		}
		if root == "" {
			return fmt.Errorf("%s.root must not be empty", ref)
		}
		if namespace == "" {
			return fmt.Errorf("%s.db_namespace must not be empty", ref)
		}
		if seenNames[name] {
			return fmt.Errorf("duplicate project name %q", name)
		}
		seenNames[name] = true
		if seenNamespaces[namespace] {
			return fmt.Errorf("duplicate project db_namespace %q", namespace)
		}
		seenNamespaces[namespace] = true
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

	if cfg.MCP.MaxResponseItems < 1 || cfg.MCP.MaxResponseItems > 5000 {
		return fmt.Errorf("mcp.max_response_items must be between 1 and 5000")
	}
	if cfg.MCP.RequestTimeout < time.Second || cfg.MCP.RequestTimeout > 2*time.Minute {
		return fmt.Errorf("mcp.request_timeout must be between 1s and 2m")
	}

	exposed := strings.TrimSpace(cfg.MCP.ExposedToolName)
	if exposed != "" && strings.ContainsAny(exposed, " \t\n") {
		return fmt.Errorf("mcp.exposed_tool_name must not contain whitespace")
	}
	specPath := strings.TrimSpace(cfg.MCP.OpenAPISpecPath)
	specURL := strings.TrimSpace(cfg.MCP.OpenAPISpecURL)
	if specPath != "" && specURL != "" {
		return fmt.Errorf("mcp.openapi_spec_path cannot be set alongside mcp.openapi_spec_url")
	}
	allowlist := cfg.MCP.OperationAllowlist
	if len(allowlist) > 0 {
		seen := make(map[string]bool, len(allowlist))
		for _, op := range allowlist {
			op = strings.TrimSpace(op)
			if op == "" {
				return fmt.Errorf("mcp.operation_allowlist entries must not be empty")
			}
			key := strings.ToLower(op)
			if seen[key] {
				return fmt.Errorf("mcp.operation_allowlist contains duplicate entry %q", op)
			}
			seen[key] = true
		}
	}

	if cfg.MCP.Enabled {
		if strings.TrimSpace(cfg.MCP.ServerName) == "" {
			return fmt.Errorf("mcp.server_name must not be empty when mcp.enabled=true")
		}
		if strings.TrimSpace(cfg.MCP.ServerVersion) == "" {
			return fmt.Errorf("mcp.server_version must not be empty when mcp.enabled=true")
		}
		if exposed != "" && len(allowlist) > 0 {
			return fmt.Errorf("mcp.exposed_tool_name cannot be set alongside mcp.operation_allowlist")
		}
		if exposed == "" && len(allowlist) == 0 {
			return fmt.Errorf("mcp.operation_allowlist must not be empty when mcp.enabled=true (or set mcp.exposed_tool_name)")
		}
	}
	return nil
}

func validateOutput(cfg *Config) error {
	if strings.TrimSpace(cfg.Output.Paths.DiagramsDir) == "" {
		return fmt.Errorf("output.paths.diagrams_dir must not be empty")
	}
	verbosity := strings.ToLower(strings.TrimSpace(cfg.Output.Report.Verbosity))
	if verbosity != "summary" && verbosity != "standard" && verbosity != "detailed" {
		return fmt.Errorf("output.report.verbosity must be one of: summary, standard, detailed")
	}
	if cfg.Output.Diagrams.FlowConfig.MaxDepth < 1 {
		return fmt.Errorf("output.diagrams.flow_config.max_depth must be >= 1")
	}

	seenEntryPoints := make(map[string]bool, len(cfg.Output.Diagrams.FlowConfig.EntryPoints))
	for i, entry := range cfg.Output.Diagrams.FlowConfig.EntryPoints {
		ref := fmt.Sprintf("output.diagrams.flow_config.entry_points[%d]", i)
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return fmt.Errorf("%s must not be empty", ref)
		}
		if seenEntryPoints[entry] {
			return fmt.Errorf("duplicate flow entry point %q", entry)
		}
		seenEntryPoints[entry] = true
	}

	outputs := make(map[string]string)
	checkConflict := func(path, name string) error {
		if path == "" {
			return nil
		}
		path = filepath.Clean(path)
		if owner, exists := outputs[path]; exists {
			return fmt.Errorf("output conflict: %s and %s share the same path %q", owner, name, path)
		}
		outputs[path] = name
		return nil
	}

	if err := checkConflict(cfg.Output.DOT, "output.dot"); err != nil {
		return err
	}
	if err := checkConflict(cfg.Output.TSV, "output.tsv"); err != nil {
		return err
	}
	if err := checkConflict(cfg.Output.Mermaid, "output.mermaid"); err != nil {
		return err
	}
	if err := checkConflict(cfg.Output.PlantUML, "output.plantuml"); err != nil {
		return err
	}
	if err := checkConflict(cfg.Output.Markdown, "output.markdown"); err != nil {
		return err
	}
	if err := checkConflict(cfg.Output.SARIF, "output.sarif"); err != nil {
		return err
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

func validateSecrets(cfg *Config) error {
	if cfg.Secrets.EntropyThreshold < 1.0 || cfg.Secrets.EntropyThreshold > 8.0 {
		return fmt.Errorf("secrets.entropy_threshold must be between 1.0 and 8.0")
	}
	if cfg.Secrets.MinTokenLength < 8 || cfg.Secrets.MinTokenLength > 256 {
		return fmt.Errorf("secrets.min_token_length must be between 8 and 256")
	}

	seen := make(map[string]bool, len(cfg.Secrets.Patterns))
	for i, pattern := range cfg.Secrets.Patterns {
		ref := fmt.Sprintf("secrets.patterns[%d]", i)
		name := strings.TrimSpace(pattern.Name)
		if name == "" {
			return fmt.Errorf("%s.name must not be empty", ref)
		}
		if seen[name] {
			return fmt.Errorf("duplicate secret pattern name %q", name)
		}
		seen[name] = true

		expr := strings.TrimSpace(pattern.Regex)
		if expr == "" {
			return fmt.Errorf("%s.regex must not be empty", ref)
		}
		if _, err := regexp.Compile(expr); err != nil {
			return fmt.Errorf("%s.regex is invalid: %w", ref, err)
		}
	}

	return nil
}

func validateArchitecture(cfg *Config) error {
	arch := cfg.Architecture
	if !arch.Enabled {
		return nil
	}

	hasLayerRules := false
	for _, rule := range arch.Rules {
		kind := strings.TrimSpace(strings.ToLower(rule.Kind))
		if kind == "" && (rule.From != "" || len(rule.Allow) > 0) {
			kind = "layer"
		} else if kind == "" && len(rule.Modules) > 0 {
			kind = "package"
		}
		if kind == "" || kind == "layer" {
			hasLayerRules = true
			break
		}
	}
	if hasLayerRules && len(arch.Layers) == 0 {
		return fmt.Errorf("architecture.enabled=true requires at least one layer when layer rules are configured")
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

			if helpers.HasWildcard(path) {
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
					if helpers.WildcardPatternsOverlap(path, existing) {
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
				if helpers.IsPathOverlap(existing, path) {
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

		kind := strings.TrimSpace(strings.ToLower(rule.Kind))
		if kind == "" && (rule.From != "" || len(rule.Allow) > 0) {
			kind = "layer"
		}
		switch kind {
		case "", "layer":
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
		case "package":
			if len(rule.Modules) == 0 {
				return fmt.Errorf("architecture rule %q must define at least one module pattern", rule.Name)
			}
			if rule.MaxFiles <= 0 && len(rule.Imports.Allow) == 0 && len(rule.Imports.Deny) == 0 {
				return fmt.Errorf("architecture rule %q must set max_files or imports allow/deny rules", rule.Name)
			}
			if rule.MaxFiles < 0 {
				return fmt.Errorf("architecture rule %q max_files must be >= 0", rule.Name)
			}
			seenModule := make(map[string]bool, len(rule.Modules))
			for _, mod := range rule.Modules {
				if strings.TrimSpace(mod) == "" {
					return fmt.Errorf("architecture rule %q module pattern must not be empty", rule.Name)
				}
				if seenModule[mod] {
					return fmt.Errorf("architecture rule %q repeats module pattern %q", rule.Name, mod)
				}
				seenModule[mod] = true
			}
			seenAllow := make(map[string]bool, len(rule.Imports.Allow))
			for _, allow := range rule.Imports.Allow {
				if strings.TrimSpace(allow) == "" {
					return fmt.Errorf("architecture rule %q imports.allow must not be empty", rule.Name)
				}
				if seenAllow[allow] {
					return fmt.Errorf("architecture rule %q repeats imports.allow pattern %q", rule.Name, allow)
				}
				seenAllow[allow] = true
			}
			seenDeny := make(map[string]bool, len(rule.Imports.Deny))
			for _, deny := range rule.Imports.Deny {
				if strings.TrimSpace(deny) == "" {
					return fmt.Errorf("architecture rule %q imports.deny must not be empty", rule.Name)
				}
				if seenDeny[deny] {
					return fmt.Errorf("architecture rule %q repeats imports.deny pattern %q", rule.Name, deny)
				}
				seenDeny[deny] = true
			}
			seenExclude := make(map[string]bool, len(rule.Exclude.Files))
			for _, excl := range rule.Exclude.Files {
				if strings.TrimSpace(excl) == "" {
					return fmt.Errorf("architecture rule %q exclude.files must not contain empty values", rule.Name)
				}
				if seenExclude[excl] {
					return fmt.Errorf("architecture rule %q repeats exclude.files pattern %q", rule.Name, excl)
				}
				seenExclude[excl] = true
			}
		default:
			return fmt.Errorf("architecture rule %q has unsupported kind %q", rule.Name, rule.Kind)
		}
	}

	return nil
}

func validateResolver(cfg *Config) error {
	scoring := cfg.Resolver.BridgeScoring
	if scoring.ConfirmedThreshold < 1 {
		return fmt.Errorf("resolver.bridge_scoring.confirmed_threshold must be >= 1")
	}
	if scoring.ProbableThreshold < 1 {
		return fmt.Errorf("resolver.bridge_scoring.probable_threshold must be >= 1")
	}
	if scoring.ProbableThreshold > scoring.ConfirmedThreshold {
		return fmt.Errorf("resolver.bridge_scoring.probable_threshold must be <= resolver.bridge_scoring.confirmed_threshold")
	}
	return nil
}

func validateWriteQueue(cfg *Config) error {
	q := cfg.WriteQueue
	if q.MemoryCapacity < 1 {
		return fmt.Errorf("write_queue.memory_capacity must be >= 1")
	}
	if q.BatchSize < 1 {
		return fmt.Errorf("write_queue.batch_size must be >= 1")
	}
	if q.FlushInterval < 10*time.Millisecond {
		return fmt.Errorf("write_queue.flush_interval must be >= 10ms")
	}
	if q.ShutdownDrainTimeout < time.Second {
		return fmt.Errorf("write_queue.shutdown_drain_timeout must be >= 1s")
	}
	if q.RetryBaseDelay < 10*time.Millisecond {
		return fmt.Errorf("write_queue.retry_base_delay must be >= 10ms")
	}
	if q.RetryMaxDelay < q.RetryBaseDelay {
		return fmt.Errorf("write_queue.retry_max_delay must be >= write_queue.retry_base_delay")
	}
	if q.PersistentQueueEnabled() && strings.TrimSpace(q.SpoolPath) == "" {
		return fmt.Errorf("write_queue.spool_path must not be empty when write_queue.persistent_enabled=true")
	}
	return nil
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

func validateDynamicGrammars(cfg *Config) error {
	seen := make(map[string]bool)
	for i, dg := range cfg.DynamicGrammars {
		ref := fmt.Sprintf("dynamic_grammars[%d]", i)
		name := strings.TrimSpace(dg.Name)
		if name == "" {
			return fmt.Errorf("%s.name must not be empty", ref)
		}
		if seen[name] {
			return fmt.Errorf("duplicate dynamic grammar name %q", name)
		}
		seen[name] = true

		if strings.TrimSpace(dg.Library) == "" {
			return fmt.Errorf("%s.library must not be empty", ref)
		}
		if len(dg.Extensions) == 0 && len(dg.Filenames) == 0 {
			return fmt.Errorf("%s must define at least one extension or filename", ref)
		}
		if dg.NamespaceNode == "" {
			return fmt.Errorf("%s.namespace_node must not be empty", ref)
		}
		if dg.ImportNode == "" {
			return fmt.Errorf("%s.import_node must not be empty", ref)
		}
		if len(dg.DefinitionNodes) == 0 {
			return fmt.Errorf("%s.definition_nodes must not be empty", ref)
		}
	}
	return nil
}

func Validate(cfg *Config) []error {
	var errs []error

	if err := validateVersion(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateProjects(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateDatabase(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateMCP(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateArchitecture(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateOutput(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateLanguages(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateDynamicGrammars(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateSecrets(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateResolver(cfg); err != nil {
		errs = append(errs, err)
	}
	if err := validateWriteQueue(cfg); err != nil {
		errs = append(errs, err)
	}

	// Semantic / Cross-field validation
	errs = append(errs, validateConfigDependencies(cfg)...)

	// Path verification
	errs = append(errs, validatePaths(cfg)...)

	return errs
}

func validateConfigDependencies(cfg *Config) []error {
	var errs []error

	// MCP Dependencies
	if cfg.MCP.Enabled && cfg.MCP.Transport == "http" {
		if cfg.MCP.Address == "" {
			errs = append(errs, fmt.Errorf("mcp.address is required when mcp.transport is 'http'"))
		}
	}

	// Secrets Dependencies
	if cfg.Secrets.Enabled && len(cfg.Secrets.Patterns) == 0 && cfg.Secrets.EntropyThreshold == 0 {
		errs = append(errs, fmt.Errorf("secrets scanning enabled but no patterns or entropy threshold defined"))
	}

	return errs
}

func validatePaths(cfg *Config) []error {
	var errs []error

	// Verify crucial paths if set
	if cfg.GrammarsPath != "" {
		stat, err := os.Stat(cfg.GrammarsPath)
		if os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("grammars_path %q does not exist", cfg.GrammarsPath))
		} else if err == nil && !stat.IsDir() {
			errs = append(errs, fmt.Errorf("grammars_path %q is not a directory", cfg.GrammarsPath))
		}
	}

	for i, path := range cfg.WatchPaths {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("watch_paths[%d] %q does not exist", i, path))
		}
	}

	return errs
}
