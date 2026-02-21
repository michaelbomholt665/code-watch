# Implementation Plan: Memory Management Optimization

**ID:** PLAN-003  
**Status:** Draft  
**Target Package:** `internal/engine/graph`, `internal/core/app`  
**User Context:** Local first, single user (Large local projects)

## Overview

Optimize the memory footprint of Circular when analyzing very large repositories. This plan focuses on proactive cache management and memory pressure awareness to prevent the application from consuming excessive RAM on the user's machine.

## Current State

The LRU cache in `internal/engine/graph/lru.go` uses a fixed size. While this prevents unbounded growth, it doesn't respond to external memory pressure from the OS or allow for proactive pruning when the user switches contexts (e.g., switches projects in MCP).

## Proposed Changes

### 1. Memory Pressure Awareness
Integrate with a simple memory monitoring loop (using `runtime.ReadMemStats`) to detect when the heap size exceeds a "high-water mark" defined in configuration or inferred from system totals.

### 2. Proactive Pruning
Implement `Prune()` and `Clear()` methods on the LRU cache and the symbol store.
- **LRU Cache:** Remove oldest N% of entries when under pressure.
- **Symbol Store:** Close SQLite connections or clear in-memory buffers.

### 3. Resource-Aware Scans
In `AnalysisService`, implement a "Batch Mode" for initial scans of huge directories that flushes the cache to disk (SQLite) more frequently.

### 4. Configurable Limits
Add memory-related settings to `circular.toml`:
```toml
[performance]
max_heap_mb = 2048  # Proactively prune if heap > 2GB
cache_size_files = 1000
```

## Implementation Steps

### Phase 1: Cache Enhancements
1. Add `Prune(percentage int)` to `internal/engine/graph/lru.go`.
2. Implement memory monitoring utility in `internal/shared/util/`.

### Phase 2: Service Integration
1. Update `AnalysisService` to monitor memory during long-running scans.
2. Trigger cache pruning if memory limits are reached.
3. Implement `ClearCache()` tool for MCP to allow manual memory reclamation.

### Phase 3: Benchmarking
1. Create a benchmark that simulates a 50,000-file repository.
2. Measure peak memory usage before and after changes.

## Verification Plan

### Automated Tests
- Unit test for `LRUCache.Prune()`.
- Benchmark test simulating high memory pressure and verifying cache reduction.

### Manual Verification
- Run a large scan with `max_heap_mb` set to a low value and monitor memory usage via `top` or `activity monitor`.
- Verify that the app remains responsive and doesn't crash with OOM.
