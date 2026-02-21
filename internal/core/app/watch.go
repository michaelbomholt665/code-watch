package app

import "circular/internal/core/watcher"

func (a *App) StartWatcher() error {
	w, err := watcher.NewWatcher(
		a.Config.Watch.Debounce,
		a.Config.Exclude.Dirs,
		a.Config.Exclude.Files,
		a.HandleChanges,
	)
	if err != nil {
		return err
	}
	w.SetLanguageFilters(
		a.codeParser.SupportedExtensions(),
		a.codeParser.SupportedFilenames(),
		a.codeParser.SupportedTestFileSuffixes(),
	)
	a.activeWatcher = w
	return w.Watch(a.Config.WatchPaths)
}
