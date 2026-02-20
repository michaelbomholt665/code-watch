package app

import "circular/internal/engine/resolver"

func (a *App) rebuildUnresolvedCache(unresolved []resolver.UnresolvedReference) {
	next := make(map[string][]resolver.UnresolvedReference)
	for _, f := range a.Graph.GetAllFiles() {
		next[f.Path] = nil
	}
	for _, u := range unresolved {
		next[u.File] = append(next[u.File], u)
	}
	a.unresolvedMu.Lock()
	a.unresolvedByFile = next
	a.unresolvedMu.Unlock()
}

func (a *App) cachedUnresolved() []resolver.UnresolvedReference {
	a.unresolvedMu.RLock()
	defer a.unresolvedMu.RUnlock()

	res := make([]resolver.UnresolvedReference, 0)
	for _, refs := range a.unresolvedByFile {
		res = append(res, refs...)
	}
	return res
}

func (a *App) rebuildUnusedCache(unused []resolver.UnusedImport) {
	next := make(map[string][]resolver.UnusedImport)
	for _, f := range a.Graph.GetAllFiles() {
		next[f.Path] = nil
	}
	for _, u := range unused {
		next[u.File] = append(next[u.File], u)
	}
	a.unusedMu.Lock()
	a.unusedByFile = next
	a.unusedMu.Unlock()
}

func (a *App) cachedUnused() []resolver.UnusedImport {
	a.unusedMu.RLock()
	defer a.unusedMu.RUnlock()

	res := make([]resolver.UnusedImport, 0)
	for _, refs := range a.unusedByFile {
		res = append(res, refs...)
	}
	return res
}

func (a *App) currentGraphPaths() []string {
	files := a.Graph.GetAllFiles()
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	return paths
}
