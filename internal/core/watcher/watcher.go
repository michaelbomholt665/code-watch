// # internal/watcher/watcher.go
package watcher

import (
	"circular/internal/shared/observability"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gobwas/glob"
)

type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	debounce     time.Duration
	excludeDirs  []glob.Glob
	excludeFiles []glob.Glob
	extFilters   map[string]bool
	nameFilters  map[string]bool
	testSuffixes []string
	onChange     func([]string)
	callbackMu   sync.Mutex

	pending   map[string]time.Time
	pendingMu sync.Mutex
	timer     *time.Timer
}

func NewWatcher(debounce time.Duration, excludeDirs, excludeFiles []string, onChange func([]string)) (*Watcher, error) {
	if onChange == nil {
		return nil, os.ErrInvalid
	}

	compiledDirs := make([]glob.Glob, 0, len(excludeDirs))
	for _, pattern := range excludeDirs {
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, err
		}
		compiledDirs = append(compiledDirs, g)
	}

	compiledFiles := make([]glob.Glob, 0, len(excludeFiles))
	for _, pattern := range excludeFiles {
		g, err := glob.Compile(pattern)
		if err != nil {
			return nil, err
		}
		compiledFiles = append(compiledFiles, g)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher: fsw,
		debounce:  debounce,
		onChange:  onChange,
		pending:   make(map[string]time.Time),
		extFilters: map[string]bool{
			".go": true,
			".py": true,
		},
		testSuffixes: []string{"_test.go", "_test.py"},
	}

	w.excludeDirs = compiledDirs
	w.excludeFiles = compiledFiles

	return w, nil
}

func (w *Watcher) SetLanguageFilters(extensions, filenames, testSuffixes []string) {
	extFilter := make(map[string]bool, len(extensions))
	for _, ext := range extensions {
		normalized := strings.ToLower(strings.TrimSpace(ext))
		if normalized == "" {
			continue
		}
		extFilter[normalized] = true
	}

	nameFilter := make(map[string]bool, len(filenames))
	for _, name := range filenames {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized == "" {
			continue
		}
		nameFilter[normalized] = true
	}

	suffixFilter := make([]string, 0, len(testSuffixes))
	for _, suffix := range testSuffixes {
		normalized := strings.ToLower(strings.TrimSpace(suffix))
		if normalized == "" {
			continue
		}
		suffixFilter = append(suffixFilter, normalized)
	}

	w.extFilters = extFilter
	w.nameFilters = nameFilter
	if len(suffixFilter) > 0 {
		w.testSuffixes = suffixFilter
	}
}

func (w *Watcher) SetDebounce(debounce time.Duration) {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()
	w.debounce = debounce
}

func (w *Watcher) Watch(paths []string) error {
	for _, path := range paths {
		if err := w.watchRecursive(path); err != nil {
			return err
		}
	}

	go w.run()
	return nil
}

func (w *Watcher) watchRecursive(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if w.shouldExcludeDir(path) {
				return filepath.SkipDir
			}
			return w.fsWatcher.Add(path)
		}

		return nil
	})
}

func (w *Watcher) run() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			observability.WatcherEventsTotal.Inc()

			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					if !w.shouldExcludeDir(event.Name) {
						if err := w.watchRecursive(event.Name); err != nil {
							slog.Warn("failed to watch new directory", "path", event.Name, "error", err)
						} else {
							w.enqueueExistingFiles(event.Name)
						}
					}
					continue
				}
			}

			if w.shouldExcludeFile(event.Name) {
				continue
			}

			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Remove == fsnotify.Remove ||
				event.Op&fsnotify.Rename == fsnotify.Rename {
				w.scheduleChange(event.Name)
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			slog.Error("watcher error", "error", err)
		}
	}
}

func (w *Watcher) scheduleChange(path string) {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	w.pending[path] = time.Now()

	if w.timer != nil {
		w.timer.Stop()
	}

	w.timer = time.AfterFunc(w.debounce, func() {
		w.flushChanges()
	})
}

func (w *Watcher) flushChanges() {
	w.pendingMu.Lock()
	paths := make([]string, 0, len(w.pending))
	for path := range w.pending {
		paths = append(paths, path)
	}
	w.pending = make(map[string]time.Time)
	w.pendingMu.Unlock()

	if len(paths) > 0 {
		w.callbackMu.Lock()
		defer w.callbackMu.Unlock()
		w.onChange(paths)
	}
}

func (w *Watcher) shouldExcludeDir(path string) bool {
	base := filepath.Base(path)
	for _, g := range w.excludeDirs {
		if g.Match(base) {
			return true
		}
	}
	return false
}

func (w *Watcher) shouldExcludeFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))

	for _, suffix := range w.testSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}

	if len(w.extFilters) > 0 || len(w.nameFilters) > 0 {
		if w.nameFilters[base] {
			// Explicit file-name routes such as go.mod/go.sum.
		} else {
			ext := strings.ToLower(filepath.Ext(base))
			if !w.extFilters[ext] {
				return true
			}
		}
	}

	for _, g := range w.excludeFiles {
		if g.Match(base) {
			return true
		}
	}
	return false
}

func (w *Watcher) Close() error {
	if w.timer != nil {
		w.timer.Stop()
	}
	return w.fsWatcher.Close()
}

func (w *Watcher) enqueueExistingFiles(root string) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if w.shouldExcludeFile(path) {
			return nil
		}
		w.scheduleChange(path)
		return nil
	})
}
