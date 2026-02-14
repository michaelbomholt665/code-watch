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
7. build `internal/core/app.App`
8. run `InitialScan`
9. optionally run single-command mode (`--trace` or `--impact`) and exit
10. run analyses, generate outputs, and print summary
11. if watch mode: start watcher and process updates (with optional UI)

## MCP Runtime Pipeline

When `[mcp].enabled=true`, CLI hands off to the MCP runtime after the initial scan:
1. resolve active project context and MCP paths
2. open history store (when DB is enabled)
3. optionally load + convert OpenAPI operation descriptors (`internal/mcp/openapi`) and apply operation allowlist
4. build MCP runtime (`internal/mcp/runtime`)
5. register the single `circular` tool with operation dispatch
6. enter stdio JSON request loop

Each MCP request is validated, allowlisted, and dispatched to tool handlers that call `internal/core/app` or `internal/data/query`.

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
- architecture rule violations
- impact analysis
- emit DOT/TSV/Mermaid/PlantUML/Markdown outputs
- publish update payloads for UI mode

## Data Flow

1. `ScanDirectories` discovers registry-enabled files by extension/filename routes (respecting excludes).
2. `ProcessFile` parses AST and normalizes a `parser.File`.
3. Parser extraction enriches definitions with visibility/scope/signature/type/decorator metadata and tags known bridge-call reference contexts (`ffi_bridge`, `process_bridge`, `service_bridge`) across core language profiles.
4. When `[secrets].enabled=true`, `ProcessFile` runs secret detection and attaches findings to `parser.File.Secrets`.
5. `Graph.AddFile` replaces prior file contributions to prevent stale edges/definitions.
6. Resolver builds a universal symbol table from graph definitions and performs exact + probabilistic multi-pass matching (including service-contract link heuristics).
7. Analyses run against graph snapshots.
8. Output generators serialize graph + findings.
9. In watch mode, changed paths trigger `HandleChanges`, which reprocesses affected files and importer chains.

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
- `internal/engine/parser`: AST extraction to normalized file model
- `internal/engine/graph`: dependency state + graph algorithms
- `internal/engine/resolver`: unresolved/unused heuristics
- resolver includes bridge-call heuristics, a universal symbol-table pass, and probabilistic cross-language matching so common interop references and service contracts are treated as expected links
- `internal/core/watcher`: fsnotify + debounce
- `internal/ui/report`: output rendering (DOT/TSV/Mermaid/PlantUML/Markdown)
- `internal/mcp/runtime`: MCP startup, allowlist enforcement, stdio dispatch loop
- `internal/mcp/adapters`: app/query bridge for MCP tool handlers
- `internal/mcp/tools/*`: operation handlers for scan/query/graph/system/report operations
