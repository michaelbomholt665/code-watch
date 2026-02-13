# Advanced Mode

This document covers the implemented advanced feature set from `docs/plans/high-complexity-feature-plan.md` (`T1` through `T5`).

## Capability Summary

- SQLite-backed history persistence at `data/database/history.db`
- versioned schema bootstrap/migrations (`internal/history/schema.go`)
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
go test ./internal/history -bench .
```
