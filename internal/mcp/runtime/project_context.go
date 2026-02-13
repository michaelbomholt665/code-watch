package runtime

import (
	"circular/internal/core/config"
	"circular/internal/shared/util"
	"fmt"
	"os"
	"strings"
)

type ProjectContext struct {
	Name             string
	Root             string
	DBNamespace      string
	Key              string
	ConfigFile       string
	SourceConfigPath string
}

func ResolveActiveProjectContext(cfg *config.Config, name string) (ProjectContext, error) {
	if cfg == nil {
		return ProjectContext{}, fmt.Errorf("config is required")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ProjectContext{}, fmt.Errorf("resolve cwd: %w", err)
	}
	paths, err := config.ResolvePaths(cfg, cwd)
	if err != nil {
		return ProjectContext{}, fmt.Errorf("resolve paths: %w", err)
	}

	entries := append([]config.ProjectEntry(nil), cfg.Projects.Entries...)
	for i := range entries {
		entries[i].Root = config.ResolveRelative(paths.ProjectRoot, entries[i].Root)
		if strings.TrimSpace(entries[i].ConfigFile) != "" {
			entries[i].ConfigFile = config.ResolveRelative(paths.ConfigDir, entries[i].ConfigFile)
		}
	}

	copyCfg := *cfg
	copyCfg.Projects = cfg.Projects
	copyCfg.Projects.Entries = entries
	if strings.TrimSpace(name) != "" {
		copyCfg.Projects.Active = name
	}

	project, err := config.ResolveActiveProject(&copyCfg, cwd)
	if err != nil {
		return ProjectContext{}, err
	}

	configFile := strings.TrimSpace(project.ConfigFile)
	if configFile == "" && strings.TrimSpace(paths.MCPConfigPath) != "" {
		configFile = paths.MCPConfigPath
	}

	return ProjectContext{
		Name:        project.Name,
		Root:        project.Root,
		DBNamespace: project.DBNamespace,
		Key:         project.Key,
		ConfigFile:  configFile,
	}, nil
}

func SyncProjectConfig(ctx ProjectContext) error {
	target := strings.TrimSpace(ctx.ConfigFile)
	if target == "" {
		return nil
	}
	source := strings.TrimSpace(ctx.SourceConfigPath)
	if source == "" {
		return fmt.Errorf("source config path is required to sync %q", target)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read source config %q: %w", source, err)
	}
	if err := util.WriteFileWithDirs(target, data, 0o644); err != nil {
		return fmt.Errorf("write config sync %q: %w", target, err)
	}
	return nil
}
