# Implementation Plan: Configuration Validation Hardening

**ID:** PLAN-002  
**Status:** Draft  
**Target Package:** `internal/core/config`  
**User Context:** Local first, single user (CLI/MCP)

## Overview

Strengthen the system's resilience by adding comprehensive cross-field validation to the configuration layer. This ensures that the user is notified of incompatible or missing settings at startup rather than encountering cryptic errors during execution.

## Current State

`internal/core/config/validator.go` performs basic type and range checks, but lacks complex cross-field validation (e.g., ensuring a path exists if a feature that uses it is enabled).

## Proposed Changes

### 1. Cross-Field Validation Rules
Implement the following rules in the validator:

| Rule | Description |
|------|-------------|
| **Grammar Path** | If any language is enabled, `grammars_path` must exist and be a directory. |
| **Output Conflicts** | Ensure output files (DOT, Mermaid, etc.) don't overwrite each other. |
| **Architecture Layers** | Validate that layer definitions don't have overlapping file patterns. |
| **Watch Path Sanity** | Ensure `watch_paths` are within the project root or accessible. |
| **MCP SSE Requirements** | If SSE transport is enabled, validate required network settings (if any). |

### 2. Validation Error Aggregation
Instead of returning the first error, collect all validation errors and report them as a single multi-error.

### 3. Proactive Path Verification
Add checks to verify that paths provided in the config are:
- Valid path strings
- Accessible with current user permissions
- Not pointing to restricted system directories (unless explicitly allowed)

## Implementation Steps

### Phase 1: Validator Enhancement
1. Update `validator.go` to support rule-based validation.
2. Implement specific check functions for path existence and permissions.
3. Add logic to check for overlapping architecture patterns.

### Phase 2: Refactor Load Workflow
1. Update `loader.go` to call the enhanced validator immediately after parsing.
2. Ensure `circular --config` fails with a clear list of all issues.

### Phase 3: Integration Tests
1. Create a "malformed" TOML file with conflicting architecture rules.
2. Verify the loader rejects it with specific error messages for each conflict.

## Verification Plan

### Automated Tests
- Test cases for each new validation rule in `validator_test.go`.
- Negative tests for overlapping architecture layers.
- Negative tests for missing grammar directories.

### Manual Verification
- Edit `circular.toml` to point `grammars_path` to a non-existent directory and run the app.
- Define two architecture layers with the same `path` and verify the error report.
