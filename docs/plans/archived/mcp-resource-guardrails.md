# Implementation Plan: MCP Resource Guardrails

**ID:** PLAN-005  
**Status:** Draft  
**Target Package:** `internal/mcp/runtime`, `internal/mcp/validate`  
**User Context:** Local first, single user (Protecting host resources)

## Overview

Implement guardrails for the MCP server to prevent excessive resource consumption. Even for a single user, an automated agent (like an LLM) can inadvertently trigger expensive operations (e.g., recursive scans of a huge `/home` directory) that could freeze the host system.

## Current State

The MCP server has a basic token-bucket rate limiter. However, it doesn't account for "expensive" vs "cheap" operations well, nor does it limit the scale of operations (e.g., maximum depth of a scan).

## Proposed Changes

### 1. Cost-Based Weighting
Refine the `OperationWeight` system to better reflect real-world cost:
- `scan_run`: High weight (depends on directory size)
- `query_modules`: Low weight
- `graph_cycles`: Medium weight

### 2. Request Scale Limits
Add limits to tool arguments in `internal/mcp/validate/args.go`:
- `max_scan_depth`: Prevent infinite recursion in deep structures.
- `max_output_size`: Limit the number of modules returned in a single query.
- `timeout_per_op`: Enforce a hard timeout on individual tool calls.

### 3. State-Aware Guardrails
Prevent concurrent expensive operations. If a `scan_run` is active, other `scan_run` requests should be queued or rejected to avoid CPU contention.

### 4. User Notification
Ensure that when a guardrail is hit, the error message clearly explains *which* limit was exceeded and how to adjust the request (or the config).

## Implementation Steps

### Phase 1: Argument Validation
1. Update `validate/args.go` with `MaxItems`, `MaxDepth`, and `AllowedPaths` checks.
2. Integrate these checks into the tool handlers.

### Phase 2: Refined Rate Limiting
1. Update the token bucket weight for high-cost operations.
2. Implement a simple "Active Operation" lock to prevent multiple concurrent heavy scans.

### Phase 3: Configuration Support
1. Add a `[mcp.guardrails]` section to `circular.toml`.
2. Allow the user to disable or relax these for large legitimate local projects.

## Verification Plan

### Automated Tests
- Unit tests for new validation rules in `args_test.go`.
- Test verifying that concurrent `scan_run` calls return a "Resource Busy" error.

### Manual Verification
- Attempt to run a scan on a forbidden or overly deep directory via MCP and verify rejection.
- Flood the server with `query_modules` calls and verify the rate limiter kicks in.
