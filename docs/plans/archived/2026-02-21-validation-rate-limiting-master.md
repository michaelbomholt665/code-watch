# Implementation Plan: Validation & Rate Limiting

## 1. Input & Config Validation Tasks

### Phase 1: Advanced Config Validator
- [ ] **Task 1.1: Cross-Field Rules** - Implement validation for relationships between config sections (e.g., `mcp.transport` vs `mcp.address`) in `internal/core/config/validator.go`.
- [ ] **Task 1.2: File System Pre-checks** - Verify permissions/existence of critical paths (grammars, database) in `internal/core/config/validator.go`.
- [ ] **Task 1.3: --check CLI Flag** - Implement a dry-run config validation mode in `internal/ui/cli/cli.go`.

### Phase 2: Enhanced MCP Input Guard
- [ ] **Task 2.1: Path Sanitizer** - Implement `../../` style escape prevention in `internal/mcp/validate/args.go`.
- [ ] **Task 2.2: Comprehensive Error Details** - Return specific field errors to MCP clients in `internal/mcp/validate/args.go`.
- [ ] **Task 2.3: JSON Schema Integration** - Use JSON schema for tool argument validation in `internal/mcp/schema/schema.go`.

### Phase 3: Developer Experience (DX)
- [ ] **Task 3.1: Error Colorization** - Use `lipgloss` for pretty terminal error output in `internal/ui/cli/runtime.go`.
- [ ] **Task 3.2: Config Migration Guide** - Suggest fixes for old config keys in `internal/core/config/loader.go`.

## 2. Rate Limiting Tasks

### Phase 4: Core Rate Limiter Logic
- [ ] **Task 4.1: Limiter implementation** - Token bucket logic in `internal/shared/util/limiter.go`.
- [ ] **Task 4.2: Registry for Limiters** - Manage client limiters in `internal/shared/util/limiter_registry.go`.
- [ ] **Task 4.3: Unit Tests** - Verify logic in `internal/shared/util/limiter_test.go`.

### Phase 5: SSE Integration
- [ ] **Task 5.1: IP Detection Helper** - Extract client IP in `internal/shared/util/net.go`.
- [ ] **Task 5.2: HTTP Middleware** - Intercept requests and check limits in `internal/mcp/transport/sse.go`.
- [ ] **Task 5.3: SSE Error Handling** - Return 429 Too Many Requests in `internal/mcp/transport/sse.go`.

### Phase 6: Stdio & Operation Throttling
- [ ] **Task 6.1: Stdio Limiter** - Protect against local process spam in `internal/mcp/transport/stdio.go`.
- [ ] **Task 6.2: Weighted Operations** - Assign costs to different operations in `internal/mcp/validate/args.go`.

### Phase 7: Configuration & Tuning
- [ ] **Task 7.1: Config Schema** - Add `[mcp.rate_limit]` section in `internal/core/config/config.go`.
- [ ] **Task 7.2: Defaults & Validation** - Sanity checks for rate limit parameters in `internal/core/config/loader.go`.

## 3. Documentation & Finalization
- [ ] **Task 8.1: Documentation Update** - Update `docs/documentation/`.
- [ ] **Task 8.2: README & CHANGELOG** - Update `README.md` and `CHANGELOG.md`.
