# Package Documentation

## `cmd/circular`

- `main.go` only calls `cli.Run(os.Args[1:])`

## `internal/ui/cli`

- parses CLI flags/options (`cli.go`)
- applies mode constraints and config fallback (`runtime.go`)
- exposes query-service command surface (`--query-*`) for module/details/trace/trend reads
- configures slog targets/levels (`runtime.go`)
- runs Bubble Tea UI loop and update plumbing (`run_ui.go`, `ui.go`)
- provides issue + module-explorer UI panels backed by query read models
- supports module drill-down, trend overlays, and source-jump actions (`ui_actions.go`, `ui_panels.go`)

## `internal/mcp/runtime`

- MCP runtime bootstrap entrypoint and project context resolution
- derives active project, config sync targets, and runtime metadata for MCP startup

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

- bridges MCP tool inputs to `internal/core/app` and `internal/data/query`
- keeps domain calls centralized for scan/query/graph operations

## `internal/mcp/tools/scan`

- scan-related handlers (`scan.run`)

## `internal/mcp/tools/query`

- query handlers for modules, module details, trace, and trends

## `internal/mcp/tools/graph`

- graph handlers for cycle detection

## `internal/mcp/tools/system`

- handlers for output/config sync and project selection

## `internal/core/app`

- central orchestrator over parser/graph/resolver/output/watcher
- builds parser from language registry + config overrides
- enforces optional grammar artifact verification at startup
- registers available extractors for enabled languages
- performs initial scan and incremental change handling
- maintains incremental caches for unresolved refs and unused imports
- computes metrics/hotspots/architecture violations
- supports trace and impact commands
- writes DOT/TSV outputs

## `internal/data/history`

- local SQLite snapshot persistence with schema migration/version checks
- optional git metadata enrichment for snapshots
- deterministic trend report generation (deltas + moving averages + module growth and fan-in/fan-out drift)

## `internal/data/query`

- shared read/query service over graph/history state
- deterministic module listing, module details, dependency trace, and trend slices
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
- language registry supports additive rollout (`go`/`python` default enabled; additional grammars default disabled)
- `Parser.RegisterDefaultExtractors()` wires language extractors from registry-enabled languages
- profile-driven extractor module (`profile_extractors.go`) covers `javascript`, `typescript`, `tsx`, `java`, `rust`, `html`, `css`, `gomod`, and `gosum`
- Go extractor collects:
- package/imports
- definitions (functions, methods, types, interfaces)
- local symbols and call references
- complexity metrics per callable
- Python extractor collects:
- imports/from-imports
- definitions (functions, classes)
- local symbols and call references
- complexity metrics per callable
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

## `internal/engine/resolver`

- unresolved-reference detection with heuristics:
- local symbols
- same-module definitions
- imported-module symbols/aliases/items
- language-scoped stdlib names (`go`, `python`, `javascript`/`typescript`/`tsx`, `java`, `rust`)
- language builtins
- user exclusion prefixes
- path-scoped unresolved analysis for incremental updates
- unused-import detection with confidence levels
- unused-import checks disabled for unsupported languages to avoid noisy output

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

- concrete format generators: `DOTGenerator`, `TSVGenerator`, `MermaidGenerator`, `PlantUMLGenerator`
