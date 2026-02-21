# Implementation Plan: Error Context Propagation

**ID:** PLAN-001  
**Status:** Draft  
**Target Package:** `internal/core/errors`, `internal/core/app`  
**User Context:** Local first, single user (CLI/MCP)

## Overview

Improve error observability by ensuring that errors bubbling up from deep within the system (parsing, graph operations, resolver) retain critical context such as file paths, operation types, and relevant metadata. This is crucial for a single user to understand why a specific file or module failed to process without digging into logs.

## Current State

Errors are often wrapped using `fmt.Errorf("...: %w", ..., err)` which is good for human reading but can sometimes lose structured context or become deeply nested and repetitive in the UI.

## Proposed Changes

### 1. Enhanced Domain Error Type
Update `internal/core/errors/errors.go` to include context maps.

```go
type DomainError struct {
    Code    ErrorCode
    Message string
    Err     error
    Context map[string]interface{}
}

func (e *DomainError) WithContext(key string, value interface{}) *DomainError {
    if e.Context == nil {
        e.Context = make(map[string]interface{})
    }
    e.Context[key] = value
    return e
}
```

### 2. Standardized Context Keys
Define common keys for error context:
- `path`: File or directory path
- `operation`: The high-level task (e.g., "parse", "resolve", "detect_cycles")
- `language`: If applicable (e.g., "go", "python")
- `symbol`: If applicable (e.g., "MyFunc")

### 3. Systematic Wrapping in Service Layer
Refactor `internal/core/app/service.go` to use structured wrapping.

| Target Function | Context to Add |
|-----------------|----------------|
| `RunScan` | `root_path`, `options` |
| `ProcessFile` | `file_path`, `language` |
| `TraceImportChain` | `from`, `to` |

## Implementation Steps

### Phase 1: Core Error Update
1. Modify `DomainError` struct in `internal/core/errors/`.
2. Implement helper methods for context injection.
3. Update `Error()` method to include context in the string representation if desired.

### Phase 2: Refactor Service Layer
1. Identify all `fmt.Errorf` calls in `internal/core/app/`.
2. Replace with `errors.NewDomainError(...)` or `errors.Wrap(err, ...)` with context.
3. Ensure warnings in `RunScan` include structured information.

### Phase 3: UI Integration
1. Update `internal/ui/cli/` to display context keys if present in the error.
2. Update MCP error responses to include context in the `data` field of the error object.

## Verification Plan

### Automated Tests
- Unit test `DomainError` context methods.
- Integration test for `RunScan` with a malformed file, verifying the error contains the `path` context.

### Manual Verification
- Run `circular` on a project with a read-only file or malformed grammar and check the CLI output for detailed error context.
- Trigger an error via MCP and inspect the JSON-RPC error response.
