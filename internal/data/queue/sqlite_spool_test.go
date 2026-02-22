package queue

import (
	"circular/internal/core/ports"
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteSpool_PersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spool.db")

	spool, err := OpenSQLiteSpool(path, "project-a")
	if err != nil {
		t.Fatalf("open spool: %v", err)
	}
	if err := spool.Enqueue(ports.WriteRequest{Operation: ports.WriteOperationDeleteFile, FilePath: "one.go"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := spool.Close(); err != nil {
		t.Fatalf("close spool: %v", err)
	}

	spool, err = OpenSQLiteSpool(path, "project-a")
	if err != nil {
		t.Fatalf("reopen spool: %v", err)
	}
	defer spool.Close()

	rows, err := spool.DequeueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Request.FilePath != "one.go" {
		t.Fatalf("expected file path one.go, got %q", rows[0].Request.FilePath)
	}
}

func TestSQLiteSpool_AckDeletesRows(t *testing.T) {
	spool := newTestSpool(t)
	defer spool.Close()
	if err := spool.Enqueue(ports.WriteRequest{Operation: ports.WriteOperationDeleteFile, FilePath: "one.go"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	rows, err := spool.DequeueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if err := spool.Ack([]int64{rows[0].ID}); err != nil {
		t.Fatalf("ack: %v", err)
	}
	count, err := spool.PendingCount(context.Background())
	if err != nil {
		t.Fatalf("pending count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected pending count 0, got %d", count)
	}
}

func TestSQLiteSpool_NackSchedulesRetry(t *testing.T) {
	spool := newTestSpool(t)
	defer spool.Close()

	if err := spool.Enqueue(ports.WriteRequest{Operation: ports.WriteOperationDeleteFile, FilePath: "one.go"}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	rows, err := spool.DequeueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("dequeue: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	next := time.Now().Add(150 * time.Millisecond)
	if err := spool.Nack(rows, next, "busy"); err != nil {
		t.Fatalf("nack: %v", err)
	}

	rows, err = spool.DequeueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("dequeue after nack: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no immediately retryable rows, got %d", len(rows))
	}

	time.Sleep(180 * time.Millisecond)
	rows, err = spool.DequeueBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("dequeue after retry window: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row after retry window, got %d", len(rows))
	}
	if rows[0].Attempts != 1 {
		t.Fatalf("expected attempts=1, got %d", rows[0].Attempts)
	}
}

func newTestSpool(t *testing.T) *SQLiteSpool {
	t.Helper()
	dir := t.TempDir()
	spool, err := OpenSQLiteSpool(filepath.Join(dir, "spool.db"), "project-a")
	if err != nil {
		t.Fatalf("open spool: %v", err)
	}
	return spool
}
