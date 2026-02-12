# AI Audit and Hardening Notes

This project has been AI-audited for performance and security risk classes.

Primary report:
- `docs/reviews/performance-security-review-2026-02-12.md`

## Audit Scope

The audit covered runtime paths in `cmd/` and `internal/`, with focus on:
- graph mutation correctness during file updates
- concurrent access safety around graph state
- watcher event handling and callback serialization
- incremental analysis strategy and update cost
- module-resolution filesystem overhead
- input/config validation consistency
- local logging safety in UI mode

## Audit-Driven Code Changes

The following areas were updated as part of audit hardening work:
- graph update replacement semantics to prevent stale per-file contributions
- graph APIs returning snapshots/copies instead of shared mutable maps
- watcher handling for newly created directories and callback serialization
- incremental unresolved-reference recomputation for affected paths
- Go module resolution caching across files
- initial-scan glob validation (fail fast on invalid patterns)
- UI log path/permission hardening and symlink guard
- `grammars_path` directory validation on startup

## Verification Commands

Use these commands to validate current baseline behavior:

```bash
GOCACHE=/tmp/go-build go test ./...
GOCACHE=/tmp/go-build go vet ./...
GOCACHE=/tmp/go-build go test -race ./...
```

## Notes

- This document records audit coverage and hardening focus areas.
- For severity details, line-level evidence, and recommendations, use the review report in `docs/reviews/`.
