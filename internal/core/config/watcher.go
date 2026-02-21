package config

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a configuration file for changes.
type Watcher struct {
	path     string
	callback func(*Config)
	stop     chan struct{}
	wg       sync.WaitGroup
}

// NewWatcher creates a new configuration watcher.
func NewWatcher(path string, callback func(*Config)) *Watcher {
	return &Watcher{
		path:     path,
		callback: callback,
		stop:     make(chan struct{}),
	}
}

// Start begins watching the configuration file.
func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// We watch the directory to handle atomic saves (where the file is replaced)
	dir := filepath.Dir(w.path)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return err
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer watcher.Close()

		log.Printf("Starting config watcher on %s", w.path)

		var timer *time.Timer
		const debounce = 100 * time.Millisecond

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Only care about our specific file
				if filepath.Clean(event.Name) != filepath.Clean(w.path) {
					continue
				}

				// Trigger on write or rename/create (common in atomic saves)
				if event.Op&fsnotify.Write == fsnotify.Write || 
				   event.Op&fsnotify.Create == fsnotify.Create {
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(debounce, func() {
						w.reload()
					})
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Config watcher error: %v", err)

			case <-w.stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	close(w.stop)
	w.wg.Wait()
}

func (w *Watcher) reload() {
	log.Printf("Config file change detected, reloading %s", w.path)
	cfg, err := Load(w.path)
	if err != nil {
		log.Printf("Failed to reload configuration: %v", err)
		return
	}

	if w.callback != nil {
		w.callback(cfg)
	}
}
