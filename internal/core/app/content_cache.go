package app

func (a *App) contentForPath(path string) []byte {
	content, ok := a.fileContents.Get(path)
	if !ok {
		return nil
	}
	out := make([]byte, len(content))
	copy(out, content)
	return out
}

func (a *App) cacheContent(path string, content []byte) {
	next := make([]byte, len(content))
	copy(next, content)
	a.fileContents.Put(path, next)
}

func (a *App) dropContent(path string) {
	a.fileContents.Evict(path)
}
