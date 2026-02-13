# Architecture

## Runtime Pipeline

`cmd/circular/main.go` is a thin entrypoint that calls `internal/cliapp.Run`.

`internal/cliapp.Run` pipeline:
1. parse flags/options
2. optionally print version and exit
3. configure logging
4. load config (with default-path fallback)
5. apply mode constraints (`--trace`/`--impact` conflict and arg checks)
6. normalize `grammars_path`
7. build `internal/app.App`
8. run `InitialScan`
9. optionally run single-command mode (`--trace` or `--impact`) and exit
10. run analyses, generate outputs, and print summary
11. if watch mode: start watcher and process updates (with optional UI)

## Core App Responsibilities

`internal/app.App` owns orchestration between parser, graph, resolver, output, and watcher.

Main responsibilities:
- scan directories and parse supported source files
- map files to module names (Go and Python)
- mutate graph on initial and incremental updates
- run analyses:
- cycle detection
- unresolved references
- unused imports
- module metrics
- complexity hotspots
- architecture rule violations
- impact analysis
- emit DOT/TSV outputs
- publish update payloads for UI mode

## Data Flow

1. `ScanDirectories` discovers registry-enabled files by extension/filename routes (respecting excludes).
2. `ProcessFile` parses AST and normalizes a `parser.File`.
3. `Graph.AddFile` replaces prior file contributions to prevent stale edges/definitions.
4. Analyses run against graph snapshots.
5. Output generators serialize graph + findings.
6. In watch mode, changed paths trigger `HandleChanges`, which reprocesses affected files and importer chains.

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

Watcher (`internal/watcher`) behavior:
- recursive directory watch registration
- dynamic registration of newly created directories
- event filtering with directory/file globs
- default ignore for `_test.go` and `_test.py`
- debounced path batching
- serialized callback execution

Update behavior (`internal/app.HandleChanges`):
- invalidates transitive importer chain for changed files
- removes deleted files from graph and incremental caches
- reprocesses changed files
- recomputes analysis outputs for affected set
- emits UI update payload + optional beep

## Boundaries

- `cmd/circular`: process entrypoint only
- `internal/cliapp`: flags, mode decisions, logging, UI runtime wiring
- `internal/app`: orchestration and workflow state
- `internal/parser`: AST extraction to normalized file model
- `internal/graph`: dependency state + graph algorithms
- `internal/resolver`: unresolved/unused heuristics
- `internal/watcher`: fsnotify + debounce
- `internal/output`: DOT/TSV rendering
