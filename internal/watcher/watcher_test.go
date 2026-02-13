// # internal/watcher/watcher_test.go
package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher_RejectsNilCallback(t *testing.T) {
	w, err := NewWatcher(100*time.Millisecond, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil callback")
	}
	if !errors.Is(err, os.ErrInvalid) {
		t.Fatalf("expected os.ErrInvalid, got %v", err)
	}
	if w != nil {
		t.Fatal("expected nil watcher when callback is invalid")
	}
}

func TestWatcher(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "watchertest")
	defer os.RemoveAll(tmpDir)

	changedFiles := make(chan []string, 1)
	w, err := NewWatcher(100*time.Millisecond, []string{"exclude_dir"}, []string{"*.exclude"}, func(paths []string) {
		changedFiles <- paths
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	err = w.Watch([]string{tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	// Create a file
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	select {
	case paths := <-changedFiles:
		found := false
		for _, p := range paths {
			if p == testFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s in changed files %v", testFile, paths)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for file change event")
	}

	// Test exclusion
	excludeFile := filepath.Join(tmpDir, "test.exclude")
	os.WriteFile(excludeFile, []byte("exclude me"), 0644)

	select {
	case paths := <-changedFiles:
		for _, p := range paths {
			if filepath.Base(p) == "test.exclude" {
				t.Error("Excluded file triggered event")
			}
		}
	case <-time.After(500 * time.Millisecond):
		// Expected
	}

	// New directory should be recursively watched after create.
	subdir := filepath.Join(tmpDir, "newdir")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	subFile := filepath.Join(subdir, "nested.go")
	if err := os.WriteFile(subFile, []byte("package nested"), 0644); err != nil {
		t.Fatal(err)
	}

	foundNested := false
	timeout := time.After(2 * time.Second)
	for !foundNested {
		select {
		case paths := <-changedFiles:
			for _, p := range paths {
				if p == subFile {
					foundNested = true
					break
				}
			}
		case <-timeout:
			t.Fatal("timed out waiting for nested file event in newly created directory")
		}
	}
}

func TestWatcher_RenameTriggersChange(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-rename")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	changedFiles := make(chan []string, 8)
	w, err := NewWatcher(100*time.Millisecond, nil, nil, func(paths []string) {
		changedFiles <- paths
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if err := w.Watch([]string{tmpDir}); err != nil {
		t.Fatal(err)
	}

	oldPath := filepath.Join(tmpDir, "old.go")
	newPath := filepath.Join(tmpDir, "new.go")
	if err := os.WriteFile(oldPath, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	timeout := time.After(2 * time.Second)
	for {
		select {
		case paths := <-changedFiles:
			for _, p := range paths {
				if p == oldPath || p == newPath {
					return
				}
			}
		case <-timeout:
			t.Fatalf("timed out waiting for rename event, old=%s new=%s", oldPath, newPath)
		}
	}
}

func TestWatcher_LanguageFilters(t *testing.T) {
	changedFiles := make(chan []string, 1)
	w, err := NewWatcher(10*time.Millisecond, nil, nil, func(paths []string) {
		changedFiles <- paths
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	w.SetLanguageFilters([]string{".go"}, []string{"go.mod"}, []string{"_test.go"})

	if w.shouldExcludeFile("main.py") == false {
		t.Fatal("expected .py to be excluded when .go is the only enabled extension")
	}
	if w.shouldExcludeFile("go.mod") {
		t.Fatal("expected go.mod to be included via filename filter")
	}
	if w.shouldExcludeFile("main_test.go") == false {
		t.Fatal("expected _test.go files to be excluded")
	}
}
