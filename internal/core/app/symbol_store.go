package app

import (
	"circular/internal/core/config"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"fmt"
	"os"
	"strings"
)

func (a *App) initSymbolStore() error {
	if a == nil || a.Config == nil || !a.Config.DB.Enabled {
		return nil
	}
	dbPath := strings.TrimSpace(a.Config.DB.Path)
	if dbPath == "" {
		return nil
	}
	cwd, err := os.Getwd()
	if err == nil {
		if resolved, pathErr := config.ResolvePaths(a.Config, cwd); pathErr == nil {
			dbPath = resolved.DBPath
		}
	}
	projectKey := strings.TrimSpace(a.Config.Projects.Active)
	if projectKey == "" {
		projectKey = "default"
	}
	store, err := graph.OpenSQLiteSymbolStore(dbPath, projectKey)
	if err != nil {
		return fmt.Errorf("open sqlite symbol store: %w", err)
	}
	a.symbolStore = store
	return nil
}

func (a *App) upsertSymbolStoreFile(file *parser.File) error {
	if a == nil || a.symbolStore == nil || file == nil {
		return nil
	}
	return a.symbolStore.UpsertFile(file)
}

func (a *App) deleteSymbolStoreFile(path string) error {
	if a == nil || a.symbolStore == nil {
		return nil
	}
	return a.symbolStore.DeleteFile(path)
}

func (a *App) pruneSymbolStorePaths() error {
	if a == nil || a.symbolStore == nil {
		return nil
	}
	return a.symbolStore.PruneToPaths(a.currentGraphPaths())
}
