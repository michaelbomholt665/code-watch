package app

func (a *App) contentForPath(path string) []byte {
	a.fileContentMu.RLock()
	defer a.fileContentMu.RUnlock()
	content, ok := a.fileContents[path]
	if !ok {
		return nil
	}
	out := make([]byte, len(content))
	copy(out, content)
	return out
}

func (a *App) cacheContent(path string, content []byte) {
	a.fileContentMu.Lock()
	defer a.fileContentMu.Unlock()
	next := make([]byte, len(content))
	copy(next, content)
	a.fileContents[path] = next
}

func (a *App) dropContent(path string) {
	a.fileContentMu.Lock()
	defer a.fileContentMu.Unlock()
	delete(a.fileContents, path)
}
