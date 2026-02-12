// # internal/watcher/watcher_test.go
package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
