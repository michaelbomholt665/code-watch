package app

import (
	"circular/internal/core/config"
	"circular/internal/core/ports"
	"circular/internal/data/queue"
	"circular/internal/shared/observability"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (a *App) initWriteQueue() error {
	if a == nil || a.Config == nil || a.symbolStore == nil {
		return nil
	}
	if !a.Config.WriteQueue.QueueEnabled() {
		return nil
	}

	a.writeQueue = queue.NewMemoryQueue(a.Config.WriteQueue.MemoryCapacity)
	if a.Config.WriteQueue.PersistentQueueEnabled() {
		spoolPath := a.resolveWriteSpoolPath(a.Config)
		projectKey := strings.TrimSpace(a.Config.Projects.Active)
		spool, err := queue.OpenSQLiteSpool(spoolPath, projectKey)
		if err != nil {
			return err
		}
		a.writeSpool = spool
	}
	return a.startWriteWorker()
}

func (a *App) resolveWriteSpoolPath(cfg *config.Config) string {
	spoolPath := strings.TrimSpace(cfg.WriteQueue.SpoolPath)
	if spoolPath == "" {
		return spoolPath
	}
	cwd, err := os.Getwd()
	if err == nil {
		if resolved, pathErr := config.ResolvePaths(cfg, cwd); pathErr == nil {
			if !filepath.IsAbs(spoolPath) {
				return filepath.Join(resolved.ProjectRoot, spoolPath)
			}
		}
	}
	return spoolPath
}

func (a *App) startWriteWorker() error {
	if a == nil || a.writeQueue == nil || a.workerCancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.workerCancel = cancel
	a.workerDone = make(chan struct{})
	go a.runWriteWorker(ctx)
	return nil
}

func (a *App) runWriteWorker(ctx context.Context) {
	defer close(a.workerDone)

	batchSize := a.Config.WriteQueue.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}
	flushInterval := a.Config.WriteQueue.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 100 * time.Millisecond
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		memoryBatch, err := a.writeQueue.DequeueBatch(ctx, batchSize, flushInterval)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
			slog.Warn("write queue dequeue failed", "error", err)
			continue
		}
		if errors.Is(err, context.Canceled) {
			return
		}

		requests := make([]ports.WriteRequest, 0, batchSize)
		requests = append(requests, memoryBatch...)

		spooled := make([]ports.SpoolRow, 0)
		if len(requests) < batchSize && a.writeSpool != nil {
			rows, spoolErr := a.writeSpool.DequeueBatch(ctx, batchSize-len(requests))
			if spoolErr != nil {
				slog.Warn("write spool dequeue failed", "error", spoolErr)
			} else {
				for _, row := range rows {
					requests = append(requests, row.Request)
				}
				spooled = rows
			}
		}

		if len(requests) == 0 {
			a.updateQueueMetrics()
			if errors.Is(err, io.EOF) {
				return
			}
			continue
		}

		started := time.Now()
		if applyErr := a.applyWriteBatch(requests); applyErr != nil {
			observability.WriteQueueApplyErrorsTotal.Inc()
			slog.Warn("write worker apply failed", "error", applyErr, "batch_size", len(requests))
			a.handleWriteFailure(spooled, memoryBatch, applyErr)
		} else {
			observability.WriteQueueProcessedTotal.Add(float64(len(requests)))
			if a.writeSpool != nil && len(spooled) > 0 {
				ids := make([]int64, 0, len(spooled))
				for _, row := range spooled {
					ids = append(ids, row.ID)
				}
				if ackErr := a.writeSpool.Ack(ids); ackErr != nil {
					slog.Warn("write spool ack failed", "error", ackErr, "count", len(ids))
				}
			}
			observability.WriteQueueFlushLatencySeconds.Observe(time.Since(started).Seconds())
		}
		a.updateQueueMetrics()
	}
}

func (a *App) handleWriteFailure(spooled []ports.SpoolRow, memoryBatch []ports.WriteRequest, applyErr error) {
	if a == nil || a.writeSpool == nil {
		return
	}
	if len(memoryBatch) > 0 {
		for _, req := range memoryBatch {
			if err := a.writeSpool.Enqueue(req); err != nil {
				slog.Warn("failed to spill memory request to spool", "error", err, "operation", req.Operation)
			} else {
				observability.WriteQueueSpilledTotal.Inc()
			}
		}
	}
	if len(spooled) == 0 {
		return
	}

	maxAttempts := 0
	for _, row := range spooled {
		if row.Attempts > maxAttempts {
			maxAttempts = row.Attempts
		}
	}
	nextAttempt := time.Now().Add(backoffDelay(a.Config.WriteQueue, maxAttempts+1))
	if err := a.writeSpool.Nack(spooled, nextAttempt, applyErr.Error()); err != nil {
		slog.Warn("write spool nack failed", "error", err, "count", len(spooled))
		return
	}
	observability.WriteQueueRetryTotal.Add(float64(len(spooled)))
}

func backoffDelay(cfg config.WriteQueueConfig, attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	delay := cfg.RetryBaseDelay
	if delay <= 0 {
		delay = 500 * time.Millisecond
	}
	maxDelay := cfg.RetryMaxDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}
	for i := 1; i < attempts; i++ {
		delay *= 2
		if delay >= maxDelay {
			return maxDelay
		}
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func (a *App) enqueueSymbolWrite(req ports.WriteRequest) error {
	if a == nil || a.symbolStore == nil {
		return nil
	}
	if !a.Config.WriteQueue.QueueEnabled() || a.writeQueue == nil {
		return a.applyWriteRequest(req)
	}
	if req.ProjectKey == "" {
		req.ProjectKey = strings.TrimSpace(a.Config.Projects.Active)
	}
	result := a.writeQueue.Enqueue(req)
	switch result {
	case ports.EnqueueAccepted:
		observability.WriteQueueEnqueuedTotal.Inc()
		a.updateQueueMetrics()
		return nil
	case ports.EnqueueDropped:
		observability.WriteQueueDroppedTotal.Inc()
		if a.writeSpool != nil {
			if err := a.writeSpool.Enqueue(req); err != nil {
				if a.Config.WriteQueue.SyncFallbackEnabled() {
					return a.applyWriteRequest(req)
				}
				return err
			}
			observability.WriteQueueSpilledTotal.Inc()
			a.updateQueueMetrics()
			return nil
		}
		if a.Config.WriteQueue.SyncFallbackEnabled() {
			return a.applyWriteRequest(req)
		}
		return fmt.Errorf("write queue full and sync fallback disabled")
	default:
		return fmt.Errorf("unknown enqueue result %q", result)
	}
}

func (a *App) applyWriteBatch(batch []ports.WriteRequest) error {
	if a == nil || a.symbolStore == nil {
		return nil
	}
	if len(batch) == 0 {
		return nil
	}

	b, err := a.symbolStore.BeginBatch()
	if err != nil {
		return err
	}
	defer b.Rollback()

	for _, req := range batch {
		switch req.Operation {
		case ports.WriteOperationUpsertFile:
			if err := b.UpsertFile(req.File); err != nil {
				return err
			}
		case ports.WriteOperationDeleteFile:
			if err := b.DeleteFile(req.FilePath); err != nil {
				return err
			}
		case ports.WriteOperationPruneToPaths:
			if err := b.PruneToPaths(req.Paths); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported write operation %q", req.Operation)
		}
	}
	return b.Commit()
}

func (a *App) applyWriteRequest(req ports.WriteRequest) error {
	if a == nil || a.symbolStore == nil {
		return nil
	}
	switch req.Operation {
	case ports.WriteOperationUpsertFile:
		return a.symbolStore.UpsertFile(req.File)
	case ports.WriteOperationDeleteFile:
		return a.symbolStore.DeleteFile(req.FilePath)
	case ports.WriteOperationPruneToPaths:
		return a.symbolStore.PruneToPaths(req.Paths)
	default:
		return fmt.Errorf("unsupported write operation %q", req.Operation)
	}
}

func (a *App) stopWriteWorker(ctx context.Context) error {
	if a == nil {
		return nil
	}
	if a.workerCancel != nil {
		a.workerCancel()
		a.workerCancel = nil
	}
	if a.workerDone != nil {
		select {
		case <-a.workerDone:
		case <-ctx.Done():
			return ctx.Err()
		}
		a.workerDone = nil
	}
	if err := a.drainWriteQueue(ctx); err != nil {
		return err
	}
	if a.writeQueue != nil {
		if err := a.writeQueue.Close(); err != nil {
			return err
		}
		a.writeQueue = nil
	}
	if a.writeSpool != nil {
		if err := a.writeSpool.Close(); err != nil {
			return err
		}
		a.writeSpool = nil
	}
	return nil
}

func (a *App) drainWriteQueue(ctx context.Context) error {
	if a == nil {
		return nil
	}
	batchSize := a.Config.WriteQueue.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}
	for {
		batch := make([]ports.WriteRequest, 0, batchSize)
		if a.writeQueue != nil {
			memBatch, err := a.writeQueue.DequeueBatch(ctx, batchSize, 0)
			if err != nil && !errors.Is(err, io.EOF) {
				return err
			}
			batch = append(batch, memBatch...)
		}
		if len(batch) < batchSize && a.writeSpool != nil {
			rows, err := a.writeSpool.DequeueBatch(ctx, batchSize-len(batch))
			if err != nil {
				return err
			}
			if len(rows) > 0 {
				for _, row := range rows {
					batch = append(batch, row.Request)
				}
				ids := make([]int64, 0, len(rows))
				for _, row := range rows {
					ids = append(ids, row.ID)
				}
				if err := a.applyWriteBatch(batch); err != nil {
					nextAttempt := time.Now().Add(backoffDelay(a.Config.WriteQueue, 1))
					_ = a.writeSpool.Nack(rows, nextAttempt, err.Error())
					return err
				}
				if err := a.writeSpool.Ack(ids); err != nil {
					return err
				}
				continue
			}
		}
		if len(batch) == 0 {
			return nil
		}
		if err := a.applyWriteBatch(batch); err != nil {
			return err
		}
	}
}

func (a *App) updateQueueMetrics() {
	if a == nil {
		return
	}
	if mq, ok := a.writeQueue.(*queue.MemoryQueue); ok {
		observability.WriteQueueDepth.Set(float64(mq.Len()))
	}
	if a.writeSpool != nil {
		if count, err := a.writeSpool.PendingCount(context.Background()); err == nil {
			observability.WriteSpoolDepth.Set(float64(count))
		}
	}
}

func (a *App) Close(ctx context.Context) error {
	if a == nil {
		return nil
	}
	drainTimeout := 10 * time.Second
	if a.Config != nil && a.Config.WriteQueue.ShutdownDrainTimeout > 0 {
		drainTimeout = a.Config.WriteQueue.ShutdownDrainTimeout
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, drainTimeout)
		defer cancel()
	}
	if err := a.stopWriteWorker(ctx); err != nil {
		return err
	}
	if a.symbolStore != nil {
		if err := a.symbolStore.Close(); err != nil {
			return err
		}
		a.symbolStore = nil
	}
	return nil
}
