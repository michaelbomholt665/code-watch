package app

import (
	"circular/internal/engine/resolver"
	"fmt"
	"path/filepath"
)

type goModuleCacheEntry struct {
	Found      bool
	ModuleRoot string
	ModulePath string
}

func (a *App) resolveGoModule(path string) (string, bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	dir := filepath.Dir(absPath)
	visited := []string{}
	for {
		if cached, ok := a.goModCache[dir]; ok {
			if !cached.Found {
				return "", false, nil
			}
			moduleName, err := moduleNameFromCache(cached, absPath)
			if err != nil {
				return "", false, err
			}
			return moduleName, true, nil
		}
		visited = append(visited, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	r := resolver.NewGoResolver()
	if err := r.FindGoMod(absPath); err != nil {
		for _, d := range visited {
			a.goModCache[d] = goModuleCacheEntry{Found: false}
		}
		return "", false, nil
	}

	cached := goModuleCacheEntry{
		Found:      true,
		ModuleRoot: r.GetModuleRoot(),
		ModulePath: r.ModulePath(),
	}
	for _, d := range visited {
		a.goModCache[d] = cached
	}

	moduleName, err := moduleNameFromCache(cached, absPath)
	if err != nil {
		return "", false, err
	}
	return moduleName, true, nil
}

func moduleNameFromCache(cached goModuleCacheEntry, filePath string) (string, error) {
	rel, err := filepath.Rel(cached.ModuleRoot, filePath)
	if err != nil {
		return "", fmt.Errorf("resolve module name from cache entry %+v for %q: %w", cached, filePath, err)
	}
	dir := filepath.Dir(rel)
	if dir == "." {
		return cached.ModulePath, nil
	}
	return cached.ModulePath + "/" + dir, nil
}
