// # internal/engine/graph/writer.go
package graph

import (
	"circular/internal/engine/parser"
	"fmt"
	"sync"
	"time"
)

// BatchWriterConfig controls the flush thresholds for the BatchWriter.
type BatchWriterConfig struct {
	// BatchSize is the number of file updates that trigger an automatic flush.
	// Defaults to 50 when zero or negative.
	BatchSize int
	// FlushInterval is the maximum time to wait before flushing pending changes.
	// Defaults to 1s when zero or negative.
	FlushInterval time.Duration
}

func (c BatchWriterConfig) batchSize() int {
	if c.BatchSize <= 0 {
		return 50
	}
	return c.BatchSize
}

func (c BatchWriterConfig) flushInterval() time.Duration {
	if c.FlushInterval <= 0 {
		return time.Second
	}
	return c.FlushInterval
}

// BatchWriter accumulates file updates and writes them to the symbol store in
// atomic transactions via a single-writer goroutine, preventing SQLITE_BUSY
// contention during high-throughput ingestion.
type BatchWriter struct {
	store *SQLiteSymbolStore
	cfg   BatchWriterConfig

	ch      chan *parser.File
	flushCh chan chan error // manual flush request + result
	done    chan struct{}
	wg      sync.WaitGroup
}

// NewBatchWriter creates a BatchWriter and starts its internal goroutine.
// Callers must call Close() to drain remaining work and stop the goroutine.
func NewBatchWriter(store *SQLiteSymbolStore, cfg BatchWriterConfig) *BatchWriter {
	w := &BatchWriter{
		store:   store,
		cfg:     cfg,
		ch:      make(chan *parser.File, cfg.batchSize()*2),
		flushCh: make(chan chan error, 1),
		done:    make(chan struct{}),
	}
	w.wg.Add(1)
	go w.run()
	return w
}

// Submit enqueues a file for the next flush cycle. It is non-blocking; if the
// internal channel is full, the file is written directly to the store.
func (w *BatchWriter) Submit(f *parser.File) {
	if f == nil {
		return
	}
	select {
	case w.ch <- f:
	default:
		// Channel full — fall back to a direct synchronous write.
		_ = w.store.UpsertFile(f)
	}
}

// Flush forces an immediate write of all pending files and waits until the
// flush is complete.
func (w *BatchWriter) Flush() error {
	result := make(chan error, 1)
	select {
	case w.flushCh <- result:
	case <-w.done:
		return nil
	}
	return <-result
}

// Close flushes remaining pending files and stops the internal goroutine.
func (w *BatchWriter) Close() error {
	close(w.done)
	w.wg.Wait()
	// Drain any files that arrived after done was closed.
	return w.drainChannel()
}

// run is the single-writer goroutine.
func (w *BatchWriter) run() {
	defer w.wg.Done()

	batch := make([]*parser.File, 0, w.cfg.batchSize())
	ticker := time.NewTicker(w.cfg.flushInterval())
	defer ticker.Stop()

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := w.writeBatch(batch)
		batch = batch[:0]
		return err
	}

	for {
		select {
		case f := <-w.ch:
			batch = append(batch, f)
			if len(batch) >= w.cfg.batchSize() {
				drainPending(&batch, w.ch)
				_ = flush()
				ticker.Reset(w.cfg.flushInterval())
			}

		case result := <-w.flushCh:
			// Drain any items that arrived in the channel before the flush
			// signal so that Submit()→Flush() sequences are always coherent.
			drainPending(&batch, w.ch)
			result <- flush()

		case <-ticker.C:
			drainPending(&batch, w.ch)
			_ = flush()

		case <-w.done:
			// Drain the channel before exit.
			for {
				select {
				case f := <-w.ch:
					batch = append(batch, f)
				default:
					_ = flush()
					return
				}
			}
		}
	}
}

// writeBatch wraps a slice of file upserts inside a single transaction.
func (w *BatchWriter) writeBatch(files []*parser.File) error {
	if len(files) == 0 {
		return nil
	}
	tx, err := w.store.db.Begin()
	if err != nil {
		return fmt.Errorf("batch writer begin tx: %w", err)
	}
	for _, f := range files {
		if err := upsertFileRows(tx, w.store.projectKey, f); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("batch writer upsert %q: %w", f.Path, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("batch writer commit tx: %w", err)
	}
	return nil
}

// drainChannel writes any files remaining in the buffered channel directly.
func (w *BatchWriter) drainChannel() error {
	var files []*parser.File
	for {
		select {
		case f := <-w.ch:
			files = append(files, f)
		default:
			return w.writeBatch(files)
		}
	}
}

// drainPending non-blockingly moves all queued files from ch into batch.
// This must be called before flushing to ensure Submit→Flush sequences are
// coherent: a file submitted just before Flush() is called may still be
// sitting in the channel when the flushCh signal is received.
func drainPending(batch *[]*parser.File, ch <-chan *parser.File) {
	for {
		select {
		case f := <-ch:
			*batch = append(*batch, f)
		default:
			return
		}
	}
}
