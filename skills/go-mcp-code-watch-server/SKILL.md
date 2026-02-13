---
name: go-mcp-code-watch-server
description: Design and implement a Go MCP server that exposes the Code Dependency Monitor capabilities (scan, cycles, unresolved refs, dependency traces, and report generation) to AI agents through safe, typed tools/resources/prompts. Use when building or updating MCP endpoints, transport/runtime wiring, schemas, and tests for this repository.
---

# Go MCP Code Watch Server

Build a production-grade Go MCP server that exposes this project's analysis pipeline to AI agents.

## Outcome

Expose repository capabilities as MCP primitives:
- Tools for active analysis actions.
- Resources for read-only state and generated artifacts.
- Prompts for guided analysis workflows.

Keep contracts typed, deterministic, and safe.

## Recommended Capability Map

Implement these MCP tools first:
- `scan_once(paths, config_path?) -> scan_summary`
- `detect_cycles(paths, config_path?) -> cycles_report`
- `find_unresolved(paths, config_path?) -> unresolved_report`
- `trace_import_chain(from_module, to_module, config_path?) -> chain_report`
- `generate_reports(paths, formats=["tsv","dot","mermaid","plantuml"]) -> report_bundle`

Implement these resources:
- `codewatch://config/current`
- `codewatch://analysis/last-summary`
- `codewatch://analysis/last-cycles`
- `codewatch://reports/{format}`

Implement prompt templates:
- `explain-cycles`
- `plan-refactor-from-hotspots`
- `summarize-dependency-risk`

## Build Workflow

1. Define tool/resource/prompt contracts and JSON-schema-compatible request/response structs.
2. Reuse existing package boundaries (`internal/parser`, `internal/graph`, `internal/resolver`, `internal/output`, `cmd/circular` orchestration).
3. Build a thin MCP adapter layer that calls existing app services.
4. Enforce safe defaults (read-only operations by default; explicit allowlist for write actions).
5. Add unit tests for handlers and integration smoke tests for MCP roundtrips.
6. Update docs in `docs/documentation/` for new MCP surfaces and examples.

## Go and Compatibility Rules

- Develop with Go `1.25.x`.
- Keep compatibility with Go `1.24.x` (floor `1.24`).
- Avoid new APIs/syntax not available in Go 1.24.x.

Verify:

```bash
go test ./...
GOTOOLCHAIN=go1.24 go test ./...
```

## Interface and Contract Rules

- Use explicit request/response types per tool.
- Return structured errors with stable codes.
- Keep outputs deterministic (sorted modules/edges/findings).
- Version contracts when changing payload shape.

Read references when implementing details:
- `references/mcp-server-blueprint.md`
- `references/code-watch-capability-contracts.md`
- `references/code-watch-mcp-best-practices.md`

## Documentation Rules

Always update docs in the same change:
- `docs/documentation/cli.md` when CLI behavior changes.
- `docs/documentation/output.md` for TSV/DOT/Mermaid/PlantUML schemas.
- `docs/documentation/architecture.md` for MCP adapter architecture.
- Add `docs/documentation/mcp.md` for tool/resource/prompt usage and examples.

## Do and Don't

Do:
- Keep MCP handlers thin and delegate core logic to existing packages.
- Keep tool calls bounded and cancellable.
- Enforce path/config validation and sandbox-safe defaults.
- Test contract serialization and deterministic output ordering.

Don't:
- Re-implement parser/graph/resolver logic inside MCP handlers.
- Expose arbitrary file-write or shell-exec tools by default.
- Return unstructured text when structured payloads are expected.
- Break existing output formats without explicit versioning.

## Completion Checklist

- MCP tools/resources/prompts expose core code-watch capabilities.
- Contracts are typed, documented, and tested.
- Deterministic TSV/DOT/Mermaid/PlantUML generation is validated.
- `docs/documentation/` is fully updated.
- Go 1.24.x compatibility check is completed or explicitly marked blocked.
