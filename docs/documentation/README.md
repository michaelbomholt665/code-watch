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
- `../plans/cross-platform-compatibility-plan.md`: cross-platform roadmap and per-session "fully implemented" status tracking
- `../plans/hexagonal-architecture-refactor.md`: phased ports-and-adapters migration plan and per-session implementation status
  - session 3 status (2026-02-14): Phase 2/4 started via `AnalysisService` scan/query driving surface; plan not fully implemented
  - session 4 status (2026-02-14): history snapshot/trend orchestration moved behind `AnalysisService`; CLI history and MCP scan paths now consume that use case; plan still not fully implemented
  - session 5 status (2026-02-14): watch lifecycle driving ports added and wired into CLI/TUI and MCP `system.watch`; plan still not fully implemented
  - session 6 status (2026-02-14): MCP output/report orchestration moved behind `AnalysisService` (`SyncOutputs`, `GenerateMarkdownReport`); plan still not fully implemented
  - session 7 status (2026-02-14): CLI `--trace/--impact` and MCP secrets/cycle reads now route through `AnalysisService` (`TraceImportChain`, `AnalyzeImpact`, `DetectCycles`, `ListFiles`); plan still not fully implemented
  - session 8 status (2026-02-14): CLI/MCP startup summary/output orchestration now uses `AnalysisService` (`SummarySnapshot`, `SyncOutputs`) instead of direct graph reads; plan still not fully implemented
  - session 9 status (2026-02-14): MCP runtime wiring and adapter construction now use `AnalysisService` directly (no concrete `*app.App` dependency), and CLI summary rendering now dispatches through `AnalysisService.PrintSummary(...)`; plan still not fully implemented
  - session 10 status (2026-02-14): CLI and MCP startup in `internal/ui/cli/runtime.go` now acquire `AnalysisService` through an interface-first runtime factory (`runtime_factory.go`) instead of direct `*app.App` construction in runtime orchestration; plan still not fully implemented
  - session 11 status (2026-02-14): output/report compatibility helpers were extracted into `internal/core/app/presentation_service.go` and CLI/MCP parity tests now assert equivalent summary/output contracts; plan is fully implemented

## Intended Audience

- engineers extending parser/resolver/graph behavior
- maintainers debugging watch/update behavior
- users integrating outputs into CI or local tooling
