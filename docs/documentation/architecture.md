# Architecture

## High-Level Pipeline

1. Load config (`internal/config`).
2. Initialize parser + extractors (`internal/parser`).
3. Recursively scan source files (`.go`, `.py`).
4. Parse each file into a normalized `parser.File` model.
5. Add file into dependency graph (`internal/graph`).
6. Detect cycles (`graph.DetectCycles`).
7. Resolve unresolved references (`internal/resolver`).
8. Generate DOT/TSV outputs (`internal/output`).
9. In watch mode, repeat incrementally on changed files (`internal/watcher`).

Audit note:
- this pipeline has been AI-audited for concurrency/performance/security issue classes; see `docs/documentation/ai-audit.md`

## Main Components

- `App` (`cmd/circular/app.go`)
- orchestration for scan, parse, graph update, analysis, output generation, and watch callbacks
- `Watcher` (`internal/watcher`)
- fsnotify wrapper with debounce and glob-based filtering
- `Graph` (`internal/graph`)
- central module/file/import/definition state
- `Resolver` (`internal/resolver`)
- unresolved reference analysis over collected refs and symbol tables
- output generators (`internal/output`)
- render graph state as DOT and TSV

## Data Model

`parser.File` contains:
- metadata: `Path`, `Language`, `Module`, `PackageName`, `ParsedAt`
- imports: normalized `Import` entries
- definitions: symbols declared in file
- references: callable/symbol uses seen in AST
- local symbols: vars/params/loop vars used to suppress false unresolved findings

`graph.Graph` stores:
- files by path
- modules by name
- import edges (`from -> to`)
- reverse import index (`importedBy`)
- definitions by module for symbol resolution

## Language Handling

- parser detects language by extension:
- `.go` -> Go extractor
- `.py` -> Python extractor
- Go module name resolution uses nearest `go.mod` and relative package directory.
- Python module name resolution trims non-package prefixes (no `__init__.py`) and maps path to dotted module.

## Watch Loop

- watch paths are recursively registered at startup, and newly created directories are added when create events are seen
- events (`write`, `create`, `remove`) are debounced and batched
- each changed path is reprocessed (or removed from graph if deleted)
- unresolved-reference analysis is recomputed incrementally for affected paths/importer chains
- callbacks are serialized to avoid overlapping update pipelines
- fresh outputs are generated after each batch
- optional UI receives update messages (`updateMsg`) with cycle/unresolved counts
