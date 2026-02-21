# Implementation Plan: Watcher Content Hashing

**ID:** PLAN-007  
**Status:** Draft  
**Target Package:** `internal/core/watcher`, `internal/core/app`  
**User Context:** Local first, single user (Efficiency in Watch Mode)

## Overview

Optimize the file watcher by adding content hashing. This prevents the system from re-parsing and re-analyzing files that have had "false" change events (e.g., file metadata updates, IDE "save" events that don't change content, or redundant file system notifications).

## Current State

The watcher in `internal/core/watcher/watcher.go` uses a time-based debounce. When a change is detected, it triggers processing of the file. If multiple events fire for the same file, the debounce catches some, but if an event fires and the content hasn't actually changed, Circular still performs the full parse/resolve cycle.

## Proposed Changes

### 1. File Hash Cache
Maintain a lightweight map of `file_path -> content_hash (SHA-1 or FNV-1a)` for all files currently being watched.

### 2. Pre-Processing Hash Check
In the watcher's flush loop:
1. Read the file content.
2. Compute the hash.
3. Compare with the cached hash.
4. Only trigger `ProcessFile` if the hash has changed.

### 3. Hash Cache Persistence (Optional)
Consider storing these hashes in the SQLite history database so they persist across application restarts, allowing for a "fast start" on subsequent runs.

### 4. Efficient Hashing
Use a fast hashing algorithm like FNV-1a or a non-cryptographic hash (as security is not the primary concern here) to minimize CPU overhead during the check.

## Implementation Steps

### Phase 1: Hash Implementation
1. Add a hashing utility in `internal/shared/util/`.
2. Update the `Watcher` struct to include the `lastHashes` map (protected by a mutex).

### Phase 2: Watcher Logic Update
1. Modify `flushChanges()` to perform the hash check.
2. Ensure the hash cache is updated after successful processing.
3. Handle file deletions by removing entries from the hash cache.

### Phase 3: Integration
1. Update `RunScan` to populate the initial hash cache.

## Verification Plan

### Automated Tests
- Unit test for the hashing logic.
- Watcher test: Trigger two "change" events with identical content and verify `ProcessFile` is only called once.
- Watcher test: Trigger two "change" events with different content and verify `ProcessFile` is called twice.

### Manual Verification
- Run `circular` in watch mode.
- Use `touch` on a file (updates metadata but not content) and verify no re-analysis occurs in logs.
- Edit a file and verify analysis triggers correctly.
