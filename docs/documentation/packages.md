# Package Documentation

## `cmd/circular`

### `main.go`

- defines CLI flags and app version
- initializes logger
- loads config with fallback behavior
- runs initial scan + analysis + output
- starts watcher and optional TUI

### `app.go`

Core methods:
- `NewApp(cfg)`
- builds grammar loader, parser, and empty graph
- registers Go/Python extractors
- `InitialScan()`
- scans configured roots
- expands scan roots to Go module root when `go.mod` is discoverable
- parses each eligible file and updates graph
- `ScanDirectories(paths, excludeDirs, excludeFiles)`
- recursive discovery for `.go` and `.py`
- glob-filtered excludes
- `ProcessFile(path)`
- parses one file and computes module name (Go/Python resolver)
- uses cached Go module-root/module-path lookups to avoid repeated `go.mod` traversal per file
- `HandleChanges(paths)`
- incremental update path for watcher callbacks
- detects cycles + incremental unresolved references for affected files/importers
- runs architecture rule validation and complexity hotspot ranking
- writes outputs and prints summary/UI updates
- `GenerateOutputs(cycles, unusedImports, metrics, violations, hotspots)`
- writes DOT and TSV when configured, including additive unused-import and architecture-violation blocks
- `AnalyzeImpact(path)`
- computes direct/transitive importer impact for a file or module
- `RunUI()`
- starts Bubble Tea list view and pushes initial state
- `StartWatcher()`
- creates and starts recursive fsnotify watcher

### `ui.go`

- Bubble Tea model showing:
- cycle findings
- unresolved reference findings
- file/module counts and last update time
- supports filterable list and quit keys (`q`, `ctrl+c`)

## `internal/config`

- TOML-backed config structs
- `Load(path)` decodes config and applies defaults:
- `watch.debounce = 500ms` if unset
- `watch_paths = ['.']` if empty
- `architecture.top_complexity = 5` if unset/invalid
- validates optional architecture layer/rule schema when enabled

## `internal/parser`

- `GrammarLoader`: loads Go and Python tree-sitter languages
- validates configured `grammars_path` as a directory when provided
- `Parser`: dispatches to extractor by extension
- `PythonExtractor`:
- imports (`import`, `from ... import ...`)
- class/function defs
- assignments, loops, calls, local symbol collection
- per-function complexity metrics (branches, params, nesting, LOC, score)
- `GoExtractor`:
- package clause, imports, funcs/methods, type/interface defs
- var/const/short decls, params, range variables, call refs
- per-function/method complexity metrics (branches, params, nesting, LOC, score)

## `internal/graph`

- mutable concurrent graph with RW mutex
- tracks files, modules, definitions, import edges, reverse import edges
- supports add/remove file updates and cycle detection
- `AddFile` replacement removes prior contributions for the same file path to avoid stale edges/definitions
- public accessors return snapshots/copies rather than exposing internal mutable maps
- `InvalidateTransitive(changedFile)` returns importer-chain affected files
- `LayerRuleEngine.Validate(graph)` returns architecture violations with source/target layers
- `AnalyzeImpact(pathOrModule)` returns direct + transitive import impact and exported symbols
- `TopComplexity(n)` returns ranked hotspots from parser complexity metrics

## `internal/resolver`

- unresolved reference finder over graph files
- resolution strategy:
- local symbols
- same-module definitions
- imported module symbols (alias/from-import/module prefix forms)
- stdlib names (embedded lists)
- language builtins
- exclusion prefixes (`exclude.symbols`)
- includes:
- `GoResolver` for module path mapping via `go.mod`
- `PythonResolver` for dotted module mapping and relative import resolution helpers
- supports path-scoped unresolved analysis for incremental watch updates

## `internal/watcher`

- fsnotify wrapper with:
- recursive watch registration
- registration of newly created directories during watch runtime
- directory/file glob exclusions
- default exclusion of `_test.go` and `_test.py`
- debounce buffering before callback
- serialized callback execution to avoid overlapping update handlers

## `internal/output`

- `DOTGenerator`:
- emits styled graph with internal/external module separation
- highlights cycle nodes/edges
- can annotate labels with module complexity hotspot score (`cx=...`)
- `TSVGenerator`:
- emits tabular edge list with source location
- emits additive unused-import and architecture-violation row blocks
