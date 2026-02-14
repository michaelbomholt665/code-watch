package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ResolvedPaths struct {
	ProjectRoot        string
	ConfigDir          string
	StateDir           string
	CacheDir           string
	DatabaseDir        string
	DBPath             string
	MCPConfigPath      string
	MCPOpenAPISpecPath string
	OutputRoot         string
}

func ResolvePaths(cfg *Config, cwd string) (ResolvedPaths, error) {
	if strings.TrimSpace(cwd) == "" {
		return ResolvedPaths{}, fmt.Errorf("cwd must not be empty")
	}

	projectRoot := strings.TrimSpace(cfg.Paths.ProjectRoot)
	if projectRoot != "" {
		projectRoot = ResolveRelative(cwd, projectRoot)
	} else {
		root, err := DetectProjectRoot(append(append([]string(nil), cfg.WatchPaths...), cwd))
		if err != nil {
			return ResolvedPaths{}, err
		}
		projectRoot = root
	}

	configDir := ResolveRelative(projectRoot, cfg.Paths.ConfigDir)
	stateDir := ResolveRelative(projectRoot, cfg.Paths.StateDir)
	cacheDir := ResolveRelative(projectRoot, cfg.Paths.CacheDir)
	databaseDir := ResolveRelative(projectRoot, cfg.Paths.DatabaseDir)

	dbPath := strings.TrimSpace(cfg.DB.Path)
	if filepath.IsAbs(dbPath) {
		dbPath = filepath.Clean(dbPath)
	} else {
		dbPath = filepath.Join(databaseDir, dbPath)
	}

	mcpConfigPath := strings.TrimSpace(cfg.MCP.ConfigPath)
	if mcpConfigPath != "" {
		mcpConfigPath = ResolveRelative(configDir, mcpConfigPath)
	}
	openapiSpecPath := strings.TrimSpace(cfg.MCP.OpenAPISpecPath)
	if openapiSpecPath != "" {
		openapiSpecPath = ResolveRelative(configDir, openapiSpecPath)
	}

	outputRoot := strings.TrimSpace(cfg.Output.Paths.Root)
	if outputRoot == "" {
		outputRoot = projectRoot
	} else {
		outputRoot = ResolveRelative(projectRoot, outputRoot)
	}

	resolved := ResolvedPaths{
		ProjectRoot: filepath.Clean(projectRoot),
		ConfigDir:   filepath.Clean(configDir),
		StateDir:    filepath.Clean(stateDir),
		CacheDir:    filepath.Clean(cacheDir),
		DatabaseDir: filepath.Clean(databaseDir),
		DBPath:      filepath.Clean(dbPath),
		OutputRoot:  filepath.Clean(outputRoot),
	}
	if mcpConfigPath != "" {
		resolved.MCPConfigPath = filepath.Clean(mcpConfigPath)
	}
	if openapiSpecPath != "" {
		resolved.MCPOpenAPISpecPath = filepath.Clean(openapiSpecPath)
	}
	return resolved, nil
}

func ResolveRelative(base, value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return filepath.Clean(base)
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Clean(filepath.Join(base, raw))
}

func DetectProjectRoot(candidates []string) (string, error) {
	markers := []string{
		"go.mod",
		".git",
		"data/config/circular.toml",
		"circular.toml",
	}

	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}

		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		root := abs
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			root = filepath.Dir(abs)
		}

		for {
			for _, marker := range markers {
				if _, err := os.Stat(filepath.Join(root, marker)); err == nil {
					return filepath.Clean(root), nil
				}
			}
			parent := filepath.Dir(root)
			if parent == root {
				break
			}
			root = parent
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(cwd), nil
}
