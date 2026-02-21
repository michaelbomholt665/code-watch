# Architecture

## Runtime Pipeline

`cmd/circular/main.go` is a thin entrypoint that calls `internal/ui/cli.Run`.

`internal/ui/cli.Run` pipeline:
1. parse flags/options
2. optionally print version and exit
3. configure logging
4. load config (with default-path fallback)
5. apply mode constraints (`--trace`/`--impact` conflict and arg checks)
6. normalize `grammars_path`
7. initialize `ports.AnalysisService` through `internal/ui/cli` runtime factory wiring (`runtime_factory.go`)
8. initialize `graph.SQLiteSymbolStore` (Schema v4) for persistent index/overlay storage
9. run initial scan through `AnalysisService.RunScan(...)`
10. optionally run single-command mode (`--trace` or `--impact`) and exit
11. collect summary state + generate outputs through `AnalysisService` (`SummarySnapshot`, `SyncOutputs`) and print summary
11. if watch mode: start watcher through `AnalysisService.WatchService()` and process updates (with optional UI)

## MCP Runtime Pipeline

When `[mcp].enabled=true`, CLI hands off to the MCP runtime after the initial scan:
1. resolve active project context and MCP paths
2. open history store (when DB is enabled)
3. optionally load + convert OpenAPI operation descriptors (`internal/mcp/openapi`) and apply operation allowlist
4. build MCP runtime (`internal/mcp/runtime`)
5. register the single `circular` tool with operation dispatch
6. enter stdio JSON request loop

Each MCP request is validated, allowlisted, and dispatched to tool handlers through `internal/mcp/adapters`, which routes scan/query/watch/output/report operations via `internal/core/ports.AnalysisService`.

## Core App Responsibilities

`internal/core/app.App` owns orchestration between parser, graph, resolver, output, and watcher.

Main responsibilities:
- scan directories and parse supported source files
- map files to module names (Go and Python)
- mutate graph on initial and incremental updates
- run analyses:
- cycle detection
- unresolved references
- unused imports
- secret detection (pattern + entropy/context heuristics)
- module metrics
- complexity hotspots
- architecture layer and package rule violations
- impact analysis
- emit DOT/TSV/Mermaid/PlantUML/Markdown outputs
- publish update payloads for UI mode
- normalize path comparisons and prefix checks through shared utilities to keep module/path matching separator-agnostic across Linux/macOS/Windows
- delegates app-scoped helper logic (diagram mode selection, output path routing, secret masking/detection helpers) to `internal/core/app/helpers`

## Data Flow

1. `ScanDirectories` discovers registry-enabled files by extension/filename routes (respecting excludes).
2. `ProcessFile` parses AST and normalizes a `parser.File`.
3. Parser extraction enriches definitions with visibility/scope/signature/type/decorator metadata and tags known bridge-call reference contexts (`ffi_bridge`, `process_bridge`, `service_bridge`) across core language profiles.
4. When `[secrets].enabled=true`, `ProcessFile` runs secret detection and attaches findings to `parser.File.Secrets`.
5. In watch mode, secret scanning computes changed line ranges and uses line-range detection when supported; full scan fallback is used when line counts shift.
6. Entropy checks are gated to high-risk file extensions (`.env`, `.json`, `.key`, `.pem`, `.p12`, `.pfx`, `.crt`, `.cer`, `.yaml`, `.yml`, `.toml`, `.ini`, `.conf`, `.properties`).
7. `Graph.AddFile` replaces prior file contributions to prevent stale edges/definitions.
8. When DB is enabled, app-level processing updates SQLite symbol rows incrementally per file (upsert/delete/prune), and resolver queries persisted symbols for exact + explicit bridge + probabilistic multi-pass matching (with in-memory fallback).
9. Optional explicit bridge mappings from `.circular-bridge.toml` are loaded from watch roots and applied as a high-priority deterministic resolver pass.
10. Query services can execute read-only CQL (`SELECT modules WHERE ...`) over module summaries and graph metrics.
11. Analyses run against graph snapshots.
12. Output generators serialize graph + findings.
13. In watch mode, changed paths trigger `HandleChanges`, which reprocesses affected files and importer chains.

## Module Naming

- Go files:
- nearest `go.mod` is discovered
- module path is derived from `module` directive + relative package directory
- lookup results are cached in app state for incremental updates
- Python files:
- module path derived from path relative to selected watch root
- leading directories without `__init__.py` are trimmed
- `__init__.py` maps to containing package module

## Watch/Incremental Behavior

Watcher (`internal/core/watcher`) behavior:
- recursive directory watch registration
- dynamic registration of newly created directories
- event filtering with directory/file globs
- default ignore for `_test.go` and `_test.py`
- debounced path batching
- serialized callback execution

Update behavior (`internal/core/app.HandleChanges`):
- invalidates transitive importer chain for changed files
- removes deleted files from graph and incremental caches
- reprocesses changed files
- recomputes analysis outputs for affected set
- emits UI update payload + optional beep

## Boundaries

- `cmd/circular`: process entrypoint only
- `internal/ui/cli`: flags, mode decisions, logging, UI runtime wiring
- `internal/core/app`: orchestration and workflow state
- `internal/engine/parser`: AST extraction to normalized file model, using a `sync.Pool`-backed parser to recycle tree-sitter instances.
- `internal/engine/graph`: dependency state + graph algorithms, including a thread-safe generic LRU cache for in-memory nodes.
- `internal/engine/resolver`: unresolved/unused heuristics
- resolver includes bridge-call heuristics, explicit `.circular-bridge.toml` mappings (`internal/engine/resolver/bridge.go`), SQLite-backed symbol lookup (`internal/engine/graph/symbol_store.go`), and probabilistic cross-language matching so common interop references and service contracts are treated as expected links
- `internal/core/watcher`: fsnotify + debounce
- `internal/ui/report`: output rendering (DOT/TSV/Mermaid/PlantUML/Markdown)
- `internal/mcp/runtime`: MCP startup, allowlist enforcement, stdio dispatch loop
- `internal/mcp/adapters`: AnalysisService/query bridge for MCP tool handlers
- `internal/mcp/tools/overlays`: handler for `add_overlay`/`list_overlays` operations
- `internal/mcp/tools/*`: operation handlers for scan/query/graph/system/report operations

## Persistent Symbol Store (Schema v4)

The symbol store (`internal/engine/graph/symbol_store.go`) is a SQLite-based index that persists:
1. **Symbols**: `symbols` table (canonical/service-key indexed).
   - Schema v4 adds `usage_tag` (e.g. `SYM_DEF`, `REF_CALL`), `confidence` (0.4-1.0), and `ancestry` (structural path).
2. **Overlays**: `semantic_overlays` table for AI-verified annotations (`VETTED_USAGE`, `EXCLUSION`, `RE-ALIAS`).
   - Stored with `source_hash` for staleness detection.
   - Operated via `internal/mcp/tools/overlays`.

## Universal Parser

`internal/engine/parser/universal.go` provides a fallback extractor for any Tree-sitter supported language.
- Walks every AST node.
- Uses regex-based routing to classify nodes into `UsageTag` categories.
- Captures structural ancestry paths (e.g., `process->handler->catch_block`).

## Surgical API

`internal/ui/report/surgical.go` supports high-precision source retrieval for AI analysis:
- `GetSymbolContext`: extracts ±5 lines of context around symbol occurrences.
- Enriches snippets with `Tag`, `Confidence`, and `Ancestry` metadata from the symbol store/parser.

## Hexagonal Refactor (Implemented)

The codebase is transitioning to a ports-and-adapters model tracked in `docs/plans/hexagonal-architecture-refactor.md`.

Current implementation baseline:
- `internal/core/ports` defines infrastructure-facing ports (`CodeParser`, `SecretScanner`, `HistoryStore`).
- `internal/core/ports` also defines driving-port contracts (`AnalysisService`, `QueryService`, `WatchService`, `ScanRequest`, `ScanResult`, `WatchUpdate`, `SummarySnapshot`, `SummaryPrintRequest`, `SyncOutputsRequest`, `SyncOutputsResult`, `MarkdownReportRequest`, `MarkdownReportResult`) plus additional `AnalysisService` trace/impact/graph-read/summary methods (`TraceImportChain`, `AnalyzeImpact`, `DetectCycles`, `ListFiles`, `SummarySnapshot`, `PrintSummary`).
- `internal/engine/parser/adapter.go` provides a parser adapter that satisfies `CodeParser`.
- `internal/engine/secrets/adapter.go` provides a secret-scanner adapter that satisfies `SecretScanner`.
- `internal/data/history/adapter.go` provides a history adapter that satisfies `HistoryStore`.
- `internal/core/app.NewWithDependencies(...)` supports constructor injection of `CodeParser` and optional `SecretScanner`.
- `internal/core/app.New(...)` remains backward compatible and wires parser/secret adapters without exposing concrete parser/detector internals to the scan pipeline.
- `internal/core/app/service.go` provides `NewAnalysisService(...)` and `(*App).AnalysisService()` to expose scan/query/history/watch/output/report/summary use cases without breaking existing `App` consumers.
- `internal/core/app/presentation_service.go` now owns summary rendering and markdown report generation orchestration shared by the service and compatibility facade methods.
- CLI query/history/watch/trace/impact/summary flows, TUI update subscriptions, and MCP scan/query/secrets/cycles/watch/output/report operations now use the `AnalysisService` surface.
- CLI runtime startup now resolves `AnalysisService` through an interface-first factory (`analysisFactory`) so runtime orchestration no longer constructs concrete `*App` directly.
- MCP runtime dependency wiring now accepts `AnalysisService` directly rather than a concrete `*App`.
- Cross-surface parity coverage now includes summary/output contract equivalence checks between CLI-facing `AnalysisService` and MCP adapters.
