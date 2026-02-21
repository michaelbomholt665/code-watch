# Implementation Plan: Symbol Store Concurrency Improvement

**ID:** PLAN-006  
**Status:** Draft  
**Target Package:** `internal/engine/graph`  
**User Context:** Local first, single user (Responsive UI during background tasks)

## Overview

Improve the concurrency and throughput of the SQLite-backed symbol store. For a single user, this ensures that background tasks (like the file watcher updating symbols) don't block UI operations (like querying for cycles or generating diagrams).

## Current State

The `SymbolStore` in `internal/engine/graph/symbol_store.go` interacts with SQLite. While SQLite handles multiple readers well, writes can cause "Database Locked" errors or block readers if not handled carefully, especially during large initial scans.

## Proposed Changes

### 1. Write Batching (Transactions)
Implement a batching mechanism for symbol insertion. Instead of one-by-one inserts, group updates from a single file or a group of files into a single transaction.

### 2. WAL Mode Hardening
Ensure WAL (Write-Ahead Logging) mode is correctly configured and tune the `busy_timeout` to handle short-lived write locks without crashing.

### 3. Read/Write Splitting
Use separate database connection pools for read operations (queries) and write operations (updates) to maximize concurrency.

### 4. Background Vacuuming
Implement proactive `VACUUM` and `ANALYZE` calls during idle periods (e.g., after a scan completes and the system has been idle for a while) to keep the database fast.

## Implementation Steps

### Phase 1: Batching Logic
1. Introduce `BeginBatch()` and `CommitBatch()` methods to the `SymbolStore` port.
2. Update `AnalysisService` to wrap file processing in batches where appropriate (e.g., during `RunScan`).

### Phase 2: Driver Tuning
1. Refactor connection string handling to include optimal SQLite parameters (`_journal_mode=WAL`, `_busy_timeout=5000`).
2. Implement separate reader/writer connection management.

### Phase 3: Performance Testing
1. Run a scan of 10,000 files and measure the time taken for symbol persistence.
2. Simulate a "Watch Mode" update during a "Diagram Generation" query to ensure no lock contention.

## Verification Plan

### Automated Tests
- Concurrent read/write stress test in `symbol_store_test.go`.
- Test verifying that WAL mode is active.

### Manual Verification
- Start a large scan in one terminal and immediately run a complex CQL query in another (or via MCP). Verify that the query returns results without waiting for the scan to finish.
