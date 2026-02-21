# Implementation Plan: Parser Pool Resource Tracking

**ID:** PLAN-004  
**Status:** Draft  
**Target Package:** `internal/engine/parser`  
**User Context:** Local first, single user (CLI/MCP stability)

## Overview

Improve the reliability and debuggability of the Tree-sitter parser pool. This plan adds tracking for parser allocation, usage, and deallocation to ensure that resources are properly returned to the pool and eventually released when the application shuts down or reloads grammars.

## Current State

The parser pool (`internal/engine/parser/pool.go`) manages a set of `sitter.Parser` instances. While functional, it lacks visibility into leaks (parsers checked out but never returned) and doesn't explicitly track the lifecycle of grammars loaded from `.so` files.

## Proposed Changes

### 1. Parser Checkout Tracking
Add a registry within the pool that tracks which parser is currently "out" and for how long.

```go
type ParserLease struct {
    ID        string
    Parser    *sitter.Parser
    StartTime time.Time
    Language  string
}
```

### 2. Leak Detection
Implement a periodic check (or a check on pool depletion) to log warnings if a parser has been checked out for an unusually long time (e.g., > 1 minute).

### 3. Resource Finalizers
Add finalizers or explicit `Close()` methods to ensure that `dlclose` (if applicable) and parser cleanup are handled gracefully during shutdown.

### 4. Metrics & Logging
Expose internal pool stats via structured logs:
- `pool_size`: Total parsers in the pool.
- `checked_out`: Number of active parsers.
- `wait_count`: Number of times a goroutine had to wait for a parser.

## Implementation Steps

### Phase 1: Pool Instrumentation
1. Update `Pool` struct to include a tracking map for active leases.
2. Add a unique ID to each parser instance.
3. Update `Get()` and `Put()` methods to update the registry.

### Phase 2: Debugging Tools
1. Add a `DumpPoolStats()` function for troubleshooting.
2. Implement the leak detection timer.

### Phase 3: Lifecycle Management
1. Ensure `Pool.Close()` properly destroys all parsers and handles grammar unloading.
2. Integrate with the application's graceful shutdown signal.

## Verification Plan

### Automated Tests
- Unit test for pool checkout/return logic.
- Test simulating a "lost" parser (not calling `Put`) and verifying the leak detector fires.
- Test verifying all parsers are destroyed on `Close()`.

### Manual Verification
- Run a large scan and check logs (with `-verbose`) for pool stats.
- Reload the configuration (triggering a pool refresh) and monitor for file handle or memory leaks.
