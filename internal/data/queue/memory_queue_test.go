package queue

import (
	"circular/internal/core/ports"
	"context"
	"io"
	"testing"
	"time"
)

func TestMemoryQueue_EnqueueDequeue(t *testing.T) {
	q := NewMemoryQueue(2)
	t.Cleanup(func() { _ = q.Close() })

	if got := q.Enqueue(ports.WriteRequest{Operation: ports.WriteOperationDeleteFile, FilePath: "a.go"}); got != ports.EnqueueAccepted {
		t.Fatalf("expected enqueue accepted, got %s", got)
	}
	if got := q.Enqueue(ports.WriteRequest{Operation: ports.WriteOperationDeleteFile, FilePath: "b.go"}); got != ports.EnqueueAccepted {
		t.Fatalf("expected enqueue accepted, got %s", got)
	}

	batch, err := q.DequeueBatch(context.Background(), 2, time.Millisecond)
	if err != nil {
		t.Fatalf("dequeue failed: %v", err)
	}
	if len(batch) != 2 {
		t.Fatalf("expected 2 items, got %d", len(batch))
	}
	if batch[0].FilePath != "a.go" || batch[1].FilePath != "b.go" {
		t.Fatalf("unexpected order: %#v", batch)
	}
}

func TestMemoryQueue_FullQueueDrops(t *testing.T) {
	q := NewMemoryQueue(1)
	t.Cleanup(func() { _ = q.Close() })

	if got := q.Enqueue(ports.WriteRequest{Operation: ports.WriteOperationDeleteFile, FilePath: "a.go"}); got != ports.EnqueueAccepted {
		t.Fatalf("expected enqueue accepted, got %s", got)
	}
	if got := q.Enqueue(ports.WriteRequest{Operation: ports.WriteOperationDeleteFile, FilePath: "b.go"}); got != ports.EnqueueDropped {
		t.Fatalf("expected enqueue dropped, got %s", got)
	}
}

func TestMemoryQueue_CloseReturnsEOFWhenDrained(t *testing.T) {
	q := NewMemoryQueue(1)
	if got := q.Enqueue(ports.WriteRequest{Operation: ports.WriteOperationDeleteFile, FilePath: "a.go"}); got != ports.EnqueueAccepted {
		t.Fatalf("expected enqueue accepted, got %s", got)
	}
	if err := q.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	batch, err := q.DequeueBatch(context.Background(), 2, 0)
	if len(batch) != 1 {
		t.Fatalf("expected 1 item after close, got %d", len(batch))
	}
	if err == nil {
		t.Fatalf("expected io.EOF with final drained batch")
	}
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}

	batch, err = q.DequeueBatch(context.Background(), 1, 0)
	if err != io.EOF {
		t.Fatalf("expected io.EOF on empty closed queue, got %v", err)
	}
	if len(batch) != 0 {
		t.Fatalf("expected 0 items, got %d", len(batch))
	}
}
