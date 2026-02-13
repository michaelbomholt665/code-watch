# Makefile Cheat Sheet

This project includes a `Makefile` to run common build, test, and CLI workflows.

## Usage

```bash
make <target> [VAR=value]
```

List all targets:

```bash
make help
```

## Core Targets

- `make build`
- build `./cmd/circular` to local binary (`BINARY`, default `./circular`)
- `make once`
- run a single scan and exit
- `make watch` (or `make run`)
- run default watch mode
- `make ui`
- run watch mode with terminal UI
- `make test`
- run all tests with local Go caches under `.cache/`
- `make test-offline`
- run tests without network module fetch
- `make coverage`
- run tests with `coverage.out`
- `make fmt`
- run `go fmt ./...`
- `make clean`
- remove cache/build artifacts

## Analysis and Query Targets

- `make trace TRACE_FROM=<module> TRACE_TO=<module>`
- `make impact IMPACT=<file-or-module>`
- `make history [SINCE=YYYY-MM-DD] [HISTORY_WINDOW=24h]`
- `make history-export [SINCE=YYYY-MM-DD]`
- `make query-modules [QUERY_FILTER=<substring>] [QUERY_LIMIT=<n>]`
- `make query-module QUERY_MODULE=<module> [QUERY_LIMIT=<n>]`
- `make query-trace QUERY_TRACE=<from:to> [QUERY_LIMIT=<n>]`
- `make query-trends [SINCE=YYYY-MM-DD] [QUERY_LIMIT=<n>]`

## Common Variables

- `CONFIG`
- config path passed to CLI (default `./data/config/circular.toml`)
- `PATH_ARG`
- optional positional path override for `watch_paths`
- `TRACE_FROM`, `TRACE_TO`
- used by `trace`
- `IMPACT`
- used by `impact`
- `SINCE`
- used by history/query trend modes (`RFC3339` or `YYYY-MM-DD`)
- `HISTORY_WINDOW`
- moving average window for history mode (default `24h`)
- `HISTORY_TSV`, `HISTORY_JSON`
- output paths for `history-export`
- `QUERY_FILTER`, `QUERY_MODULE`, `QUERY_TRACE`, `QUERY_LIMIT`
- query mode inputs

## Examples

```bash
make once
make watch PATH_ARG=.
make ui CONFIG=./data/config/circular.example.toml
make trace TRACE_FROM=cmd/circular TRACE_TO=internal/parser
make impact IMPACT=internal/resolver/resolver.go
make history SINCE=2026-02-01 HISTORY_WINDOW=72h
make query-modules QUERY_FILTER=resolver QUERY_LIMIT=20
make query-trace QUERY_TRACE=cmd/circular:internal/parser QUERY_LIMIT=10
```
