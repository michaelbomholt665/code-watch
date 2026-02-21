package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcher_ContentHashing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "watcher-hash-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	changedFiles := make(chan []string, 10)
	w, err := NewWatcher(50*time.Millisecond, nil, nil, func(paths []string) {
		changedFiles <- paths
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	w.Watch([]string{tmpDir})
	time.Sleep(100 * time.Millisecond)

	testFile := filepath.Join(tmpDir, "hash_target.go")
	content := []byte(`package main
func main() {}`)
	
	// Initial create
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for initial event
	select {
	case <-changedFiles:
		// OK
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for create event")
	}

	// "Touch" file (update mtime but content same)
	now := time.Now()
	if err := os.Chtimes(testFile, now, now); err != nil {
		t.Fatal(err)
	}
	
	// Trigger the watcher explicitly if Chtimes doesn't trigger it (fsnotify might ignore if only access time changed, but Chtimes changes modtime)
	// Some OS/fsnotify implementations trigger CHMOD or similar. 
	// To be sure, let's just write the SAME content again.
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case paths := <-changedFiles:
		t.Errorf("Received unexpected event for identical content: %v", paths)
	case <-time.After(200 * time.Millisecond):
		// Expected timeout - no event should fire
	}

	// Change content
	newContent := []byte(`package main
func main() { println(1) }`)
	if err := os.WriteFile(testFile, newContent, 0644); err != nil {
		t.Fatal(err)
	}

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
			t.Errorf("Expected event for %s, got %v", testFile, paths)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for content change")
	}
}
