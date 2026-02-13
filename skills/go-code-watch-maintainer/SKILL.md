---
name: go-code-watch-maintainer
description: Maintain and extend the Code Dependency Monitor project in Go with parser/graph/resolver/watcher/output boundaries, production-safe refactors, and test-first changes. Use when implementing features, fixing bugs, reviewing architecture, or planning upgrades in this repository, especially when developing on Go 1.25.x while preserving compatibility with Go 1.24.
---

# Go Code Watch Maintainer

Execute changes for this repository with strict package boundaries, reproducible CLI behavior, and Go version compatibility discipline.

## Follow This Workflow

1. Confirm scope and affected pipeline stages (`parser`, `graph`, `resolver`, `watcher`, `output`, `cmd/circular`).
2. Keep behavior changes localized to the owning package.
3. Add or update table-driven tests in `*_test.go` near edited code.
4. Keep App and MCP capability surfaces in sync using a shared capability contract.
5. Validate with the standard test suite and compatibility checks.
6. Update docs/config examples when flags, outputs, or config schema change.

## Enforce Go Version Policy

Develop using Go `1.25.x`, but preserve runtime/build compatibility with `1.24`.

Use this command order:

```bash
go test ./...
GOTOOLCHAIN=go1.24 go test ./...
```

If compatibility fails:
- Remove or replace APIs introduced after Go 1.24.
- Avoid syntax/features unavailable in Go 1.24.
- Keep `go.mod` `go` directive and dependencies aligned with 1.24 compatibility goals.

## Package-Specific Guidance

### `internal/parser`
- Preserve language extraction accuracy before adding new heuristics.
- Normalize symbols and locations consistently across Go/Python extractors.
- Prefer explicit AST-node handling over broad catch-all logic.

### `internal/graph`
- Keep graph operations deterministic and concurrency-safe.
- Protect incremental updates (`AddFile`/`RemoveFile`) from stale edge state.
- Treat cycle detection and transitive invalidation as correctness-critical.

### `internal/resolver`
- Reduce false positives before increasing detection breadth.
- Keep stdlib and local symbol checks explicit and testable.
- Separate language-specific resolution paths cleanly.

### `internal/watcher`
- Preserve debounce guarantees and idempotent change batching.
- Avoid blocking the event loop with heavy analysis work.

### `internal/output`
- Keep DOT and TSV formats stable and additive.
- Prefer backward-compatible field additions over format breaks.

### `cmd/circular`
- Keep CLI modes (`--once`, `--ui`, watch) behavior consistent.
- Fail fast with actionable error messages for bad config/flags.

## App and MCP Sync Guidance

Treat the CLI/app and MCP server as two interfaces over the same core capabilities.

- Define capabilities once in core services (analysis operations), then map both CLI and MCP handlers to those services.
- Keep a capability registry document that maps: core service method -> CLI flag/mode -> MCP tool/resource.
- Require a parity update for every new feature: if a capability is added or changed in app logic, explicitly decide MCP exposure in the same change.
- Version contract payloads for MCP tools when response shape changes; keep additive changes preferred.
- Mark deprecations in both surfaces at the same time and document removal timeline.
- Avoid feature logic in transport layers (`cmd/circular` and MCP handlers should orchestrate, not own analysis behavior).

Suggested parity table (keep in docs and update per feature):

| Capability | Core Service | CLI Surface | MCP Surface | Status |
| --- | --- | --- | --- | --- |
| Cycle detection | graph/resolver pipeline | `--once` summary | `detect_cycles` tool | parity |
| Unresolved refs | resolver | strict/hallucination outputs | `find_unresolved` tool | parity |
| Import tracing | graph traversal | trace mode | `trace_import_chain` tool | parity |

Required parity checks:

- Add tests that assert CLI and MCP produce equivalent results for the same fixture input (allowing presentation differences).
- Add contract tests for MCP request/response schemas and deterministic ordering.
- Add release checklist item: verify capability registry has no `app-only` entries unless explicitly accepted.

## Do and Don't

Do:
- Use table-driven tests for parser/resolver/graph behavior.
- Keep changes small and package-local when possible.
- Add clear benchmarks or complexity notes for expensive traversals.
- Preserve existing config defaults and output compatibility.
- Keep capability parity between CLI/app and MCP tools/resources.
- Run both default and Go 1.24 compatibility test passes.

Don't:
- Introduce cross-package coupling that bypasses pipeline boundaries.
- Mix unrelated refactors with behavior changes in one patch.
- Change DOT/TSV schema silently.
- Ship new core capabilities in app only without explicit MCP parity decision.
- Depend on Go 1.25-only APIs without guarded fallback/removal.
- Skip regression tests for cycle detection, incremental rebuild, or unresolved reference logic.

## Quality Gate

Before finalizing:

```bash
go test ./...
GOTOOLCHAIN=go1.24 go test ./...
go test ./... -coverprofile=coverage.out
```

If a command cannot run (toolchain missing, environment limits), state it explicitly and document remaining risk.

## Review Checklist

- Scope is confined to relevant packages.
- Feature/bug behavior is covered by tests.
- CLI/config/output docs are updated when applicable.
- App capability changes include explicit MCP sync/parity decision.
- Capability registry and MCP contracts are updated when surfaces change.
- Go 1.24 compatibility was verified or clearly called out as unverified.
