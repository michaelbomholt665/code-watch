# Package Documentation

## `cmd/circular`

- `main.go` only calls `cli.Run(os.Args[1:])`

## `internal/ui/cli`

- parses CLI flags/options (`cli.go`)
- applies mode constraints and config fallback (`runtime.go`)
- initializes analysis runtime dependencies through an interface-first factory (`runtime_factory.go`)
- exposes query-service command surface (`--query-*`) for module/details/trace/trend reads
- routes query command execution through the `internal/core/ports.AnalysisService` driving port
- routes summary-state and output orchestration through `AnalysisService` (`SummarySnapshot`, `SyncOutputs`) instead of direct graph reads in runtime flow
- configures slog targets/levels (`runtime.go`)
- runs Bubble Tea UI loop and update plumbing (`run_ui.go`, `ui.go`)
- starts watch mode via `AnalysisService.WatchService()` and subscribes UI updates through the same driving surface
- provides issue + module-explorer UI panels backed by query read models
- supports module drill-down, trend overlays, and source-jump actions (`ui_actions.go`, `ui_panels.go`)

## `internal/mcp/runtime`

- MCP runtime bootstrap entrypoint and project context resolution
- derives active project, config sync targets, and runtime metadata for MCP startup
- routes `system.watch` through the watch driving port instead of direct watcher startup calls
- depends on `ports.AnalysisService`/`ports.WatchService` rather than concrete `*app.App` runtime wiring

## `internal/mcp/registry`

- tool handler registry with deterministic registration order

## `internal/mcp/transport`

- transport adapters (stdio stub for POC)

## `internal/mcp/contracts`

- MCP tool request/response DTOs and error codes

## `internal/mcp/schema`

- tool schema definitions (single `circular` tool)

## `internal/mcp/validate`

- tool argument validation and normalization

## `internal/mcp/adapters`

- bridges MCP tool inputs to `internal/core/app` and `internal/core/ports.AnalysisService`
- keeps domain calls centralized for scan/query/history/graph/report operations
- routes cycle/secret reads and output/report operations through `AnalysisService` driving methods
- includes parity coverage in `adapter_test.go` asserting CLI-facing `AnalysisService` and MCP adapter summary/output contract equivalence for shared fixture state

## `internal/mcp/tools/scan`

- scan-related handlers (`scan.run`)

## `internal/mcp/tools/query`

- query handlers for modules, module details, trace, and trends

## `internal/mcp/tools/graph`

- graph handlers for cycle detection

## `internal/mcp/tools/system`

- handlers for output/config sync and project selection

## `internal/mcp/tools/report`

- handlers for markdown report generation (`report.generate_markdown`)

## `internal/core/app`

- central orchestrator over parser/graph/resolver/output/watcher
- builds parser from language registry + config overrides
- enforces optional grammar artifact verification at startup
- registers available extractors for enabled languages
- performs initial scan and incremental change handling
- maintains incremental caches for unresolved refs and unused imports
- runs optional secret detection and publishes aggregate secret counts in UI update payloads
- updates persisted resolver symbols incrementally per file (`UpsertFile`, `DeleteFile`, `PruneToPaths`) when DB is enabled
- computes metrics/hotspots/architecture violations
- supports trace and impact commands
- writes DOT/TSV/Mermaid/PlantUML/Markdown outputs
- supports dependency injection for core parsing/secret-scan collaborators via `NewWithDependencies(...)`
- uses port contracts (`CodeParser`, `SecretScanner`, `HistoryStore`) for injected infrastructure dependencies
- provides `NewAnalysisService(...)`/`(*App).AnalysisService()` as a compatibility-preserving service extraction surface for scan/query/history/watch plus trace/impact/cycle/file-list/summary use cases
- uses `presentation_service.go` as a focused collaborator for summary rendering and markdown report generation, keeping `App` methods as compatibility wrappers

## `internal/core/app/helpers`

- app-scoped helpers for diagram-mode selection, output-path resolution, secret masking/incremental detection helpers, and metric leader formatting
- keeps orchestration logic slim by consolidating low-level helpers used across `app`, `presentation_service`, and output wiring

## `internal/core/ports`

- defines focused infrastructure ports used by the core orchestration layer
- includes driven ports (`CodeParser`, `SecretScanner`, `HistoryStore`) and driving ports (`AnalysisService`, `QueryService`, `WatchService`)
- includes optional incremental secret-scanning extension contracts (`LineRange`, `IncrementalSecretScanner`)
- includes history-trend request/result contracts to drive snapshot capture and trend generation from adapters
- includes `WatchUpdate` contracts to drive live watch-mode update payloads through ports
- includes output/report request/result contracts for driving-adapter orchestration (`SyncOutputs*`, `MarkdownReport*`)
- includes trace/impact/cycle/file-list/summary methods on `AnalysisService` to reduce direct adapter/runtime coupling to `App` internals
- includes `SummaryPrintRequest` and `AnalysisService.PrintSummary(...)` so CLI summary rendering can route through driving ports
- serves as the phase-1 baseline plus phase-2/4 kickoff contracts for the hexagonal architecture migration plan

## `internal/data/history`

- local SQLite snapshot persistence with schema migration/version checks
- schema now also includes resolver symbol-index storage (`symbols` table) used by analysis flows
- optional git metadata enrichment for snapshots
- deterministic trend report generation (deltas + moving averages + module growth and fan-in/fan-out drift)
- `Adapter` bridges `Store` into `internal/core/ports.HistoryStore`

## `internal/data/query`

- shared read/query service over graph/history state
- deterministic module listing, module details, dependency trace, and trend slices
- includes read-only CQL parsing/execution (`cql.go`, `Service.ExecuteCQL(...)`) for advanced module filtering
- context-aware APIs for cancellation-safe calls

## `internal/core/config`

- TOML decode into config structs
- applies defaults:
- `watch_paths=["."]` when empty
- `watch.debounce=500ms` when zero
- `architecture.top_complexity=5` when `<=0`
- validates architecture layer/rule schema when enabled

## `internal/engine/parser`

- `GrammarLoader` creates enabled runtime languages and validates manifest-bound grammar artifacts
- `Parser` routes by registry-defined extensions + filename matchers
- `Adapter` bridges `Parser` into the `internal/core/ports.CodeParser` contract
- language registry supports additive rollout (`go`/`python` default enabled; additional grammars default disabled)
- `Parser.RegisterDefaultExtractors()` wires language extractors from registry-enabled languages
- profile-driven extractor module (`profile_extractors.go`) covers `javascript`, `typescript`, `tsx`, `java`, `rust`, `html`, `css`, `gomod`, and `gosum`
- Go extractor collects:
- package/imports
- definitions (functions, methods, types, interfaces)
- definition metadata: visibility, scope, lightweight signature, type hints
- local symbols and call references
- complexity metrics per callable
- Python extractor collects:
- imports/from-imports
- definitions (functions, classes)
- definition metadata: visibility, scope, decorators, lightweight signature, type hints
- local symbols and call references
- bridge-call reference context tags (`ffi_bridge`, `process_bridge`, `service_bridge`)
- complexity metrics per callable
- JS/TS/Java/Rust profile extractors also populate definition metadata parity fields (`Visibility`, `Scope`, `Signature`, `TypeHint`) for cross-language resolver matching
- `gomod` and `gosum` use raw-text extractors (no runtime tree-sitter binding required)

## `internal/engine/parser/registry`

- owns `LanguageSpec` defaults and override-merging validation
- enforces deterministic extension/filename ownership for enabled languages

## `internal/engine/parser/grammar`

- loads and validates `grammars/manifest.toml`
- verifies enabled-language grammar artifacts (checksums + required manifest coverage)

## `internal/engine/parser/extractors`

- provides wrapper constructors for built-in extractor registrations

## `internal/engine/graph`

- stores files/modules/import edges/reverse edges/definitions
- `AddFile` replacement semantics remove old file contributions first
- exposes defensive-copy getters for graph snapshots
- algorithms:
- cycle detection
- shortest import chain
- transitive invalidation for incremental updates
- module metrics (depth, fan-in, fan-out)
- complexity hotspot ranking
- architecture rule validation
- impact analysis (direct + transitive importers)
- SQLite symbol-store adapter (`symbol_store.go`) for persisted cross-language resolver lookups and incremental symbol row pruning by file path

## `internal/engine/resolver`

- unresolved-reference detection with heuristics:
- local symbols
- same-module definitions
- imported-module symbols/aliases/items
- language-scoped stdlib names (`go`, `python`, `javascript`/`typescript`/`tsx`, `java`, `rust`)
- language builtins
- bridge-call contexts from parser (`ffi_bridge`, `process_bridge`, `service_bridge`) to suppress common polyglot interop false positives
- explicit bridge mappings from `.circular-bridge.toml` (`bridge.go`) for deterministic cross-language reference resolution
- universal symbol-table second pass over graph definitions for cross-language candidate matching
- prefers SQLite-backed symbol lookup via `graph.SQLiteSymbolStore` when DB is enabled by app config; falls back to in-memory universal symbol table when unavailable
- probabilistic scoring for ambiguous symbols (exact-first, scored fallback, ambiguity guardrails)
- framework-aware service linking heuristics for common client/server naming families (for example gRPC/Thrift-style symbols)
- user exclusion prefixes
- path-scoped unresolved analysis for incremental updates
- unused-import detection with confidence levels
- unused-import checks disabled for unsupported languages to avoid noisy output

## `internal/engine/secrets`

- secret detector for hardcoded credential heuristics
- combines built-in regex signatures, custom regex patterns, context-sensitive assignment checks, entropy scoring, and line-range incremental scan support
- gates entropy scoring to high-risk extension set to reduce false positives/cost on general source files
- returns location-scoped findings attached to `parser.File.Secrets`
- `Adapter` bridges `Detector` into `internal/core/ports.SecretScanner` and `IncrementalSecretScanner`

## `internal/engine/resolver/drivers`

- language-specific module-name and import-resolution drivers (`go`, `python`, `javascript`, `java`, `rust`)

## `internal/core/watcher`

- wraps fsnotify with:
- recursive watch registration
- create-time directory expansion
- glob-based filtering plus language-aware file-name/extension filtering
- debounce batching
- serialized callback execution

## `internal/ui/report`

- `DOTGenerator` emits enriched module graph (cycle/metrics/hotspot annotations)
- `TSVGenerator` emits:
- dependency edges
- optional unused-import section
- optional architecture-violation section
- trend renderers emit additive advanced outputs:
- `RenderTrendTSV(...)`
- `RenderTrendJSON(...)`

## `internal/ui/report/formats`

- concrete format generators: `DOTGenerator`, `TSVGenerator`, `MermaidGenerator`, `PlantUMLGenerator`, `MarkdownGenerator`
