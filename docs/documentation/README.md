# Documentation Index

This folder contains implementation-level documentation for `circular`.

## Documents

- `cli.md`: CLI flags, mode semantics, argument rules, and logging behavior
- `make.md`: Makefile targets for build/test/watch/UI/history/query workflows
- `configuration.md`: TOML schema, defaults, fallbacks, architecture validation rules, and secret-detection settings
- `output.md`: DOT/TSV/Mermaid/PlantUML/Markdown report schemas and ordering guarantees
- `advanced.md`: phase-1 historical snapshots and trend-report workflows
- `architecture.md`: scan/update pipeline, data model, and subsystem boundaries
- `packages.md`: package-level ownership and key entrypoints
- `mcp.md`: MCP POC tool protocol, operations, and examples
- `limitations.md`: known behavior constraints and tradeoffs
- `ai-audit.md`: hardening scope and verification baseline
- `../plans/diagram-expansion-plan.md`: planned architecture/component/flow diagram expansion status
- `../plans/cross-language-analysis-optimization-plan.md`: cross-language resolver/parser optimization roadmap and session status (currently marked fully implemented as a heuristic baseline)

## Intended Audience

- engineers extending parser/resolver/graph behavior
- maintainers debugging watch/update behavior
- users integrating outputs into CI or local tooling
