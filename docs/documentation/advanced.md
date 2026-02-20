# Advanced Mode

This document covers the implemented advanced feature set from `docs/plans/high-complexity-feature-plan.md` (`T1` through `T5`).

## Capability Summary

- SQLite-backed history persistence at `data/database/history.db`
- SQLite-backed persistent resolver symbol index (`symbols` table) stored in the same DB
- versioned schema bootstrap/migrations (`internal/data/history/schema.go`)
- lock-aware write/read retry policy for transient SQLite contention
- trend reports with configurable moving window (`--history-window`)
- additive trend dimensions:
  - module growth (`delta_modules`, `module_growth_pct`)
  - fan-in/fan-out drift (`delta_avg_fan_in`, `delta_avg_fan_out`)
- shared query service surfaced in CLI (`--query-*`) and TUI
- expanded TUI module explorer with:
  - module detail drill-down (`enter`)
  - dependency cursor (`j`/`k`)
  - trend overlay (`t`)
  - jump-to-source action via `$EDITOR` (`o`)
- history benchmarks and integration tests for advanced paths

## Advanced Architecture Refinements (Current Slice)

- All phases are complete from `docs/plans/advanced-architecture-refinements.md`.
- Implemented:
- persistent symbol table with incremental file-level upsert/delete/prune (`internal/engine/graph/symbol_store.go`) and app-driven updates during initial/watch scans (`internal/core/app/app.go`)
- resolver integration with SQLite-backed symbol lookups and in-memory fallback (`internal/engine/resolver/resolver.go`)
- incremental line-range secret detection for changed hunks (`internal/engine/secrets/detector.go`, `internal/engine/secrets/adapter.go`)
- entropy checks gated to high-risk file extensions to reduce scan noise/cost on general source files
- explicit bridge mapping support via `.circular-bridge.toml` (`internal/engine/resolver/bridge.go` + app resolver wiring)
- read-only CQL support for advanced module queries (`internal/data/query/cql.go`, `internal/data/query/service.go`)
- runtime Tree-sitter grammar loading via `dlopen` (Linux/macOS) for custom language support without recompilation

## Dynamic Grammar Support

Circular supports runtime loading of Tree-sitter grammars. This allows adding support for new languages by providing a shared object (`.so`) file and mapping AST nodes to universal dependency concepts.

### Configuration

Add a `[[dynamic_grammars]]` block to your `circular.toml`:

```toml
[[dynamic_grammars]]
name = "kotlin"
library = "./grammars/kotlin/kotlin.so"
extensions = [".kt"]
namespace_node = "package_header"
import_node = "import_header"
definition_nodes = ["class_declaration", "function_declaration"]
```

- **library**: Path to the shared object (`.so`) file.
- **namespace_node**: AST node kind representing the package or namespace declaration.
- **import_node**: AST node kind representing import statements.
- **definition_nodes**: List of AST node kinds representing symbol definitions (e.g., classes, functions, interfaces).

Circular uses a generic `DynamicExtractor` to traverse the AST and extract dependencies based on these configured node kinds.

## CLI Enablement

### Record history and print trend summary

```bash
circular --once --history
```

### Configure trend moving window

```bash
circular --once --history --history-window 72h
```

### Filter historical window

```bash
circular --once --history --since 2026-02-01
circular --once --history --since 2026-02-01T09:00:00Z
```

Accepted `--since` formats:
- `YYYY-MM-DD`
- RFC3339 timestamp

### Export trend reports

```bash
circular --once --history --history-tsv out/trends.tsv --history-json out/trends.json
```

### Query service CLI surface

```bash
circular --query-modules --query-filter app/
circular --query-module app/core
circular --query-trace app/api:app/storage --query-limit 6
circular --history --query-trends --since 2026-02-01 --query-limit 20
```

## Snapshot Schema

Each history snapshot includes:
- scan timestamp
- optional git commit hash/timestamp
- module/file/cycle/unresolved/unused-import/violation/hotspot counters
- average and max fan-in/fan-out metrics

Duplicate rows (same project key + timestamp + commit hash) are upserted.

## TUI Explorer Flows

- `tab`: switch between Issues and Modules panels
- `enter`: open selected module details
- `esc`: close module details
- `j` / `k`: move highlighted dependency edge
- `t`: toggle trend overlay
- `o`: open selected source location in `$EDITOR`

## Benchmarks and Guardrails

History subsystem benchmarks:
- `BenchmarkStore_SaveSnapshot`
- `BenchmarkStore_LoadSnapshots`

Run them with:

```bash
go test ./internal/data/history -bench .
```
