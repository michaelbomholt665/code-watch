package app

import (
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteWorker_AppliesQueuedUpsert(t *testing.T) {
	store := newTestSymbolStore(t)
	app := &App{
		Config:      testWriteQueueConfig(8, 2, 20*time.Millisecond),
		symbolStore: store,
	}
	if err := app.initWriteQueue(); err != nil {
		t.Fatalf("initWriteQueue failed: %v", err)
	}
	defer func() {
		_ = app.stopWriteWorker(context.Background())
		_ = store.Close()
	}()

	req := ports.WriteRequest{
		Operation: ports.WriteOperationUpsertFile,
		FilePath:  "a.go",
		File: &parser.File{
			Path:     "a.go",
			Language: "go",
			Module:   "example.com/a",
		},
	}
	if err := app.enqueueSymbolWrite(req); err != nil {
		t.Fatalf("enqueueSymbolWrite failed: %v", err)
	}

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		loaded, err := store.LoadFile("a.go")
		if err == nil && loaded != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for queued upsert to reach symbol store")
}

func TestWriteWorker_StopDrainsPendingMemoryWrites(t *testing.T) {
	store := newTestSymbolStore(t)
	app := &App{
		Config:      testWriteQueueConfig(8, 8, 5*time.Second),
		symbolStore: store,
	}
	if err := app.initWriteQueue(); err != nil {
		t.Fatalf("initWriteQueue failed: %v", err)
	}
	defer store.Close()

	if err := app.enqueueSymbolWrite(ports.WriteRequest{
		Operation: ports.WriteOperationUpsertFile,
		FilePath:  "drain.go",
		File: &parser.File{
			Path:     "drain.go",
			Language: "go",
			Module:   "example.com/drain",
		},
	}); err != nil {
		t.Fatalf("enqueueSymbolWrite failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := app.stopWriteWorker(ctx); err != nil {
		t.Fatalf("stopWriteWorker failed: %v", err)
	}

	loaded, err := store.LoadFile("drain.go")
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected pending write to be drained before worker stop")
	}
}

func newTestSymbolStore(t *testing.T) *graph.SQLiteSymbolStore {
	t.Helper()
	dir := t.TempDir()
	store, err := graph.OpenSQLiteSymbolStore(filepath.Join(dir, "history.db"), "default")
	if err != nil {
		t.Fatalf("open symbol store: %v", err)
	}
	return store
}

func testWriteQueueConfig(memoryCap, batchSize int, flushInterval time.Duration) *config.Config {
	enabled := true
	disabled := false
	return &config.Config{
		Projects: config.Projects{Active: "default"},
		WriteQueue: config.WriteQueueConfig{
			Enabled:              &enabled,
			MemoryCapacity:       memoryCap,
			PersistentEnabled:    &disabled,
			BatchSize:            batchSize,
			FlushInterval:        flushInterval,
			ShutdownDrainTimeout: 2 * time.Second,
			RetryBaseDelay:       10 * time.Millisecond,
			RetryMaxDelay:        100 * time.Millisecond,
			SyncFallback:         &enabled,
		},
	}
}
