package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type ActiveProject struct {
	Name        string
	Root        string
	DBNamespace string
	Key         string
	ConfigFile  string
}

func ResolveActiveProject(cfg *Config, cwd string) (ActiveProject, error) {
	entries := cfg.Projects.Entries
	if len(entries) == 0 {
		return ActiveProject{
			Name:        "default",
			Root:        filepath.Clean(cwd),
			DBNamespace: "default",
			Key:         "default",
		}, nil
	}

	active := strings.TrimSpace(cfg.Projects.Active)
	if active != "" {
		for _, entry := range entries {
			if strings.TrimSpace(entry.Name) == active {
				return materializeProject(entry, cwd), nil
			}
		}
		return ActiveProject{}, fmt.Errorf("projects.active references unknown project %q", active)
	}

	absCWD, err := filepath.Abs(cwd)
	if err == nil {
		best := ActiveProject{}
		bestLen := -1
		for _, entry := range entries {
			m := materializeProject(entry, cwd)
			rel, relErr := filepath.Rel(m.Root, absCWD)
			if relErr == nil && (rel == "." || !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
				if l := len(m.Root); l > bestLen {
					best = m
					bestLen = l
				}
			}
		}
		if bestLen >= 0 {
			return best, nil
		}
	}

	return materializeProject(entries[0], cwd), nil
}

func LoadProjectRegistry(path string) ([]ProjectEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Entries []ProjectEntry `toml:"entries"`
	}
	if _, err := toml.Decode(string(data), &payload); err != nil {
		return nil, err
	}
	return payload.Entries, nil
}

func materializeProject(entry ProjectEntry, base string) ActiveProject {
	root := ResolveRelative(base, entry.Root)
	key := normalizeProjectNamespace(entry.DBNamespace, entry.Name)
	if key == "" {
		key = strings.TrimSpace(entry.Name)
	}
	if key == "" {
		key = "default"
	}
	return ActiveProject{
		Name:        strings.TrimSpace(entry.Name),
		Root:        filepath.Clean(root),
		DBNamespace: key,
		Key:         key,
		ConfigFile:  strings.TrimSpace(entry.ConfigFile),
	}
}
