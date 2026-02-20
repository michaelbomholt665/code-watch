// # internal/engine/graph/writer_test.go
package graph

import (
	"circular/internal/engine/parser"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *SQLiteSymbolStore {
	t.Helper()
	store, err := OpenSQLiteSymbolStore(filepath.Join(t.TempDir(), "symbols.db"), "test-proj")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func makeFile(path, name string) *parser.File {
	return &parser.File{
		Path:     path,
		Language: "go",
		Module:   path,
		Definitions: []parser.Definition{
			{Name: name, FullName: path + "." + name, Kind: parser.KindFunction, Exported: true},
		},
	}
}

func TestBatchWriter_FlushByCount(t *testing.T) {
	store := openTestStore(t)
	cfg := BatchWriterConfig{BatchSize: 3, FlushInterval: 10 * time.Second}
	w := NewBatchWriter(store, cfg)
	defer func() { _ = w.Close() }()

	// Submit exactly BatchSize files; the goroutine should auto-flush.
	w.Submit(makeFile("a.go", "A"))
	w.Submit(makeFile("b.go", "B"))
	w.Submit(makeFile("c.go", "C"))

	// Give the goroutine a moment to flush.
	time.Sleep(50 * time.Millisecond)

	if got := store.Lookup("A"); len(got) == 0 {
		t.Error("expected symbol A after flush-by-count, got none")
	}
	if got := store.Lookup("B"); len(got) == 0 {
		t.Error("expected symbol B after flush-by-count, got none")
	}
}

func TestBatchWriter_FlushByInterval(t *testing.T) {
	store := openTestStore(t)
	cfg := BatchWriterConfig{BatchSize: 100, FlushInterval: 50 * time.Millisecond}
	w := NewBatchWriter(store, cfg)
	defer func() { _ = w.Close() }()

	w.Submit(makeFile("x.go", "X"))

	// Wait longer than flush interval.
	time.Sleep(200 * time.Millisecond)

	if got := store.Lookup("X"); len(got) == 0 {
		t.Error("expected symbol X after flush-by-interval, got none")
	}
}

func TestBatchWriter_ExplicitFlush(t *testing.T) {
	store := openTestStore(t)
	cfg := BatchWriterConfig{BatchSize: 100, FlushInterval: 10 * time.Second}
	w := NewBatchWriter(store, cfg)
	defer func() { _ = w.Close() }()

	w.Submit(makeFile("m.go", "M"))
	if err := w.Flush(); err != nil {
		t.Fatalf("explicit flush: %v", err)
	}

	if got := store.Lookup("M"); len(got) == 0 {
		t.Error("expected symbol M after explicit flush, got none")
	}
}

func TestBatchWriter_Close_DrainsQueue(t *testing.T) {
	store := openTestStore(t)
	cfg := BatchWriterConfig{BatchSize: 100, FlushInterval: 10 * time.Second}
	w := NewBatchWriter(store, cfg)

	w.Submit(makeFile("z.go", "Z"))
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if got := store.Lookup("Z"); len(got) == 0 {
		t.Error("expected symbol Z after Close drains queue, got none")
	}
}

func TestBatchWriter_NilFile_Ignored(t *testing.T) {
	store := openTestStore(t)
	w := NewBatchWriter(store, BatchWriterConfig{})
	defer func() { _ = w.Close() }()

	// Should not panic.
	w.Submit(nil)
	if err := w.Flush(); err != nil {
		t.Fatalf("flush after nil submit: %v", err)
	}
}
