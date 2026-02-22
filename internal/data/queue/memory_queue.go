package queue

import (
	"circular/internal/core/ports"
	"context"
	"io"
	"sync"
	"time"
)

var _ ports.WriteQueuePort = (*MemoryQueue)(nil)

type MemoryQueue struct {
	ch     chan ports.WriteRequest
	mu     sync.RWMutex
	closed bool
}

func NewMemoryQueue(capacity int) *MemoryQueue {
	if capacity <= 0 {
		capacity = 1
	}
	return &MemoryQueue{ch: make(chan ports.WriteRequest, capacity)}
}

func (q *MemoryQueue) Enqueue(req ports.WriteRequest) ports.EnqueueResult {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.closed {
		return ports.EnqueueDropped
	}
	select {
	case q.ch <- req:
		return ports.EnqueueAccepted
	default:
		return ports.EnqueueDropped
	}
}

func (q *MemoryQueue) DequeueBatch(ctx context.Context, maxItems int, wait time.Duration) ([]ports.WriteRequest, error) {
	if maxItems <= 0 {
		maxItems = 1
	}
	batch := make([]ports.WriteRequest, 0, maxItems)

	var timer <-chan time.Time
	if wait > 0 {
		t := time.NewTimer(wait)
		defer t.Stop()
		timer = t.C
	}

	select {
	case req, ok := <-q.ch:
		if !ok {
			return nil, io.EOF
		}
		batch = append(batch, req)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer:
		return nil, nil
	default:
		if wait <= 0 {
			return nil, nil
		}
		select {
		case req, ok := <-q.ch:
			if !ok {
				return nil, io.EOF
			}
			batch = append(batch, req)
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer:
			return nil, nil
		}
	}

	for len(batch) < maxItems {
		select {
		case req, ok := <-q.ch:
			if !ok {
				return batch, io.EOF
			}
			batch = append(batch, req)
		default:
			return batch, nil
		}
	}

	return batch, nil
}

func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return nil
	}
	q.closed = true
	close(q.ch)
	return nil
}

func (q *MemoryQueue) Len() int {
	if q == nil {
		return 0
	}
	return len(q.ch)
}
