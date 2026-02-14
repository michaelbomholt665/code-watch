# Documentation Index

This folder contains implementation-level documentation for `circular`.

## Documents

- `cli.md`: CLI flags, mode semantics, argument rules, and logging behavior
- `make.md`: Makefile targets for build/test/watch/UI/history/query workflows
- `configuration.md`: TOML schema, defaults, fallbacks, and architecture validation rules
- `output.md`: DOT and TSV schemas, appended sections, and ordering guarantees
- `advanced.md`: phase-1 historical snapshots and trend-report workflows
- `architecture.md`: scan/update pipeline, data model, and subsystem boundaries
- `packages.md`: package-level ownership and key entrypoints
- `mcp.md`: MCP POC tool protocol, operations, and examples
- `limitations.md`: known behavior constraints and tradeoffs
- `ai-audit.md`: hardening scope and verification baseline
- `../plans/diagram-expansion-plan.md`: planned architecture/component/flow diagram expansion status

## Intended Audience

- engineers extending parser/resolver/graph behavior
- maintainers debugging watch/update behavior
- users integrating outputs into CI or local tooling
