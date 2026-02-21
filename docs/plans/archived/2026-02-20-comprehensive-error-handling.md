# Comprehensive Error Handling Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Standardize error handling across Circular using domain-driven error types and context-aware wrapping.

**Architecture:**
Circular uses hexagonal architecture. This plan introduces a central `internal/core/errors` package to define domain-specific error types. These errors will be used across ports and adapters to ensure consistent error propagation and mapping in driving interfaces (CLI and MCP).

**Tech Stack:** Go 1.24+

---

### Task 1: Create Domain Error Package

**Files:**
- Create: `internal/core/errors/errors.go`

**Step 1: Define base domain error types**

```go
package errors

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeValidationError  ErrorCode = "VALIDATION_ERROR"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeInternal         ErrorCode = "INTERNAL_ERROR"
	CodeNotSupported     ErrorCode = "NOT_SUPPORTED"
	CodePermissionDenied ErrorCode = "PERMISSION_DENIED"
)

type DomainError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

func New(code ErrorCode, msg string) error {
	return &DomainError{Code: code, Message: msg}
}

func Wrap(err error, code ErrorCode, msg string) error {
	return &DomainError{Code: code, Message: msg, Err: err}
}

// IsCode checks if an error has a specific error code.
func IsCode(err error, code ErrorCode) bool {
	var de *DomainError
	if errors.As(err, &de) {
		return de.Code == code
	}
	return false
}
```

**Step 2: Commit**

```bash
git add internal/core/errors/errors.go
git commit -m "feat(core): add domain error package"
```

---

### Task 2: Refactor App Construction Errors

**Files:**
- Modify: `internal/core/app/app.go`

**Step 1: Update New() and NewWithDependencies() to use domain errors**

```go
// In New()
if err != nil {
    return nil, errors.Wrap(err, errors.CodeInternal, "failed to build parser registry")
}

// In NewWithDependencies()
if cfg == nil {
    return nil, errors.New(errors.CodeValidationError, "config must not be nil")
}
```

**Step 2: Verify tests pass**

Run: `go test ./internal/core/app/...`

**Step 3: Commit**

```bash
git add internal/core/app/app.go
git commit -m "refactor(app): use domain errors in app constructor"
```

---

### Task 3: Standardize Parser Errors

**Files:**
- Modify: `internal/engine/parser/parser.go`

**Step 1: Update ParseFile to return wrapped domain errors**

**Step 2: Commit**

```bash
git add internal/engine/parser/parser.go
git commit -m "refactor(parser): use domain errors for parsing failures"
```

---

### Task 4: Map Domain Errors to MCP Responses

**Files:**
- Modify: `internal/mcp/adapters/adapter.go`

**Step 1: Create an error mapper for MCP**

```go
func mapToMCPError(err error) int {
    if errors.IsCode(err, errors.CodeNotFound) {
        return -32001 // Custom MCP error codes
    }
    // ...
    return -32603 // Internal error
}
```

**Step 2: Commit**

```bash
git add internal/mcp/adapters/adapter.go
git commit -m "feat(mcp): map domain errors to MCP error codes"
```
