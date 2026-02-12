# Performance and Security Review

Date: 2026-02-12
Reviewer: Codex
Scope: `cmd/` and `internal/`

## Executive Summary

The codebase is clean and test-covered, but there are two high-severity issues that can cause correctness drift and runtime instability under sustained watch workloads:

- graph state is not rebuilt safely on file updates, which can leave stale imports/definitions and cause unbounded metadata growth
- internal graph maps are returned by reference and read outside lock windows, creating a real concurrent map read/write risk

Performance is acceptable for small projects, but incremental mode currently does full-graph unresolved analysis and repeatedly re-discovers module roots per file, which will degrade noticeably at scale.

## Methodology

- Manual static review of runtime paths in `cmd/circular` and all `internal/*` packages
- Concurrency and mutation-path inspection focused on watch/update behavior
- `go test ./...` (pass)
- `go vet ./...` (pass)
- `go test -race ./...` could not be validated in this environment due package resolution failure in race mode

## Findings (Ordered by Severity)

### 1. HIGH: Stale graph edges/definitions on file update (correctness + memory growth)

Evidence:
- `internal/graph/graph.go:52` adds/overwrites new file data, but does not remove old per-file contributions when the same file path is updated
- imports are only additive in `AddFile` (`internal/graph/graph.go:93`)
- definitions are only overwritten by matching names (`internal/graph/graph.go:81`), so removed symbols can persist

Impact:
- false cycle detections and false unresolved/resolved outcomes after edits
- stale module definitions/imports accumulate over time, increasing memory usage and analysis cost
- long-running watch sessions can drift further from source-of-truth

Recommendation:
- before applying updated file data, remove prior file contributions (or rebuild module-level aggregates)
- maintain per-file indexes (defs/imports) so replacement is deterministic and O(changed file)

---

### 2. HIGH: Unsafe map/pointer exposure from `Graph` creates race/panic risk

Evidence:
- `Graph.Modules()` returns the internal map directly (`internal/graph/graph.go:197`)
- returned map is used after lock release in multiple places, including `len(a.Graph.Modules())` (`cmd/circular/app.go:186`, `cmd/circular/app.go:192`, `cmd/circular/app.go:275`) and iteration in DOT generation (`internal/output/dot.go:43`, `internal/output/dot.go:50`)

Impact:
- concurrent map read/write panics under overlapping reads/writes
- undefined behavior/data races when watcher updates and render/report paths interleave

Recommendation:
- replace `Modules()` with a safe snapshot API (`ModuleCount()`, `ModulesSnapshot()` deep copy)
- avoid returning internal pointers/maps from synchronized structures
- similarly review `GetModule`/`GetFile` pointer escape patterns (`internal/graph/graph.go:190`, `internal/graph/graph.go:224`)

---

### 3. MEDIUM: `Watcher` can invoke `onChange` concurrently

Evidence:
- debounce timer callback calls `flushChanges()` via `time.AfterFunc` (`internal/watcher/watcher.go:125`)
- `flushChanges()` calls `w.onChange(paths)` without callback serialization (`internal/watcher/watcher.go:139`)

Impact:
- overlapping `HandleChanges` executions are possible during bursty edits
- amplifies race risk and output thrash
- can reorder update application under heavy event streams

Recommendation:
- serialize callbacks through a single worker goroutine/channel
- maintain one stable timer + event queue instead of replacing `AfterFunc` instances

---

### 4. MEDIUM: Per-file Go module root discovery is expensive

Evidence:
- each Go file processing creates resolver and walks parents for `go.mod` (`cmd/circular/app.go:153`, `internal/resolver/go_resolver.go:21`)
- each discovery reparses `go.mod` (`internal/resolver/go_resolver.go:39`)

Impact:
- avoidable filesystem/stat/read overhead proportional to number of Go files
- significant startup and incremental latency in large monorepos

Recommendation:
- cache module root/module name by directory prefix
- compute once per scan root and reuse for files in subtree

---

### 5. MEDIUM: Full unresolved-reference scan runs on every change batch

Evidence:
- each change batch calls `AnalyzeHallucinations()` (`cmd/circular/app.go:179`)
- resolver traverses all files and all references (`internal/resolver/resolver.go:29`)
- graph has invalidation helpers (`internal/graph/graph.go:256`, `internal/graph/detect.go:46`) that are not used in app flow

Impact:
- O(total references) per update, even for a single-file change
- watch responsiveness drops as codebase grows

Recommendation:
- use dirty/transitive invalidation to re-evaluate affected files/modules only
- maintain cached unresolved set and update incrementally

---

### 6. MEDIUM: New directories are not added to watcher set after startup

Evidence:
- recursive watch registration only occurs in initial `Watch()` path (`internal/watcher/watcher.go:60`, `internal/watcher/watcher.go:71`)
- create events are scheduled for analysis but newly created directories are not registered with `fsWatcher.Add`

Impact:
- files created inside newly created directories may not be observed
- watch coverage gaps lead to stale graph state

Recommendation:
- on `Create` event, `os.Stat` path; if it is a directory and not excluded, recursively add watchers

---

### 7. LOW: Glob compile errors are silently ignored during initial scan

Evidence:
- `ScanDirectories` ignores `glob.Compile` errors (`cmd/circular/app.go:89`, `cmd/circular/app.go:95`)
- watcher constructor does fail-fast on invalid globs (`internal/watcher/watcher.go:41`, `internal/watcher/watcher.go:49`)

Impact:
- inconsistent behavior between initial scan and watch mode
- malformed patterns can unintentionally widen scan scope (performance and data exposure concern)

Recommendation:
- fail fast on invalid globs in scan path as well, mirroring watcher behavior

---

### 8. LOW: UI log file creation is predictable and symlink-prone in privileged contexts

Evidence:
- UI mode opens `circular.log` in CWD with `0644` and follows default `OpenFile` semantics (`cmd/circular/main.go:40`)

Impact:
- if executed with elevated privileges in untrusted directories, an attacker-controlled symlink could redirect writes
- practical risk is low for normal local developer usage

Recommendation:
- prefer per-user state dir (`$XDG_STATE_HOME`/home), restrictive mode `0600`
- optionally use symlink-safe open strategy where platform allows

---

### 9. LOW: `grammars_path` is accepted but not used by loader

Evidence:
- loader signature accepts path but does not use it (`internal/parser/loader.go:14`)

Impact:
- configuration may imply controls that do not exist
- minor operational confusion; not directly exploitable

Recommendation:
- either implement filesystem grammar loading or remove/deprecate config field

## Positive Observations

- no dynamic command execution from parsed source
- no network-facing runtime surface in core analysis path
- output generation is deterministic and simple
- baseline test suite across all major subsystems is present and passing

## Remediation Roadmap

### Phase 1 (High priority, safety/correctness)

1. Make graph access APIs snapshot-based (no internal map/pointer escape)
2. Fix update semantics so `AddFile` replacement removes stale defs/imports
3. Serialize watcher callback execution to one update pipeline

### Phase 2 (Performance)

1. Cache Go/Python module resolution artifacts across files
2. Switch unresolved analysis to dirty/transitive incremental mode
3. Add watcher support for newly created directories

### Phase 3 (Hardening and consistency)

1. Fail fast on invalid exclude globs in initial scan
2. Harden UI log path/permissions
3. Align `grammars_path` behavior with implementation or remove it

## Suggested Validation After Fixes

- add race-enabled CI (`go test -race ./...`) once environment supports race builds
- add stress test for burst file events and concurrent output generation
- add regression tests for file-update replacement semantics (remove symbol/import, ensure graph cleanup)
- benchmark watch-update latency on medium/large repos before and after incremental resolver changes
