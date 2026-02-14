# CLI Reference

Entrypoint: `cmd/circular/main.go` delegates to `internal/ui/cli.Run(...)`.

## Usage

```bash
circular [flags] [path]
```

## Flags

- `--config string`
- default: `./data/config/circular.toml`
- default discovery order:
  - `./data/config/circular.toml`
  - `./circular.toml` (deprecated fallback with warning)
  - `./data/config/circular.example.toml`
  - `./circular.example.toml`
- `--once`
- run initial scan and exit
- `--ui`
- run watch mode with Bubble Tea UI
- redirects logs to a state log file to avoid corrupting UI rendering
- `--trace`
- usage: `circular --trace <from-module> <to-module>`
- requires exactly two positional module arguments
- cannot be combined with `--impact`
- `--impact string`
- usage: `circular --impact <file-path-or-module>`
- prints direct importers, transitive importers, and externally used exported symbols
- cannot be combined with `--trace`
- `--verify-grammars`
- verifies enabled-language grammar artifacts against `grammars/manifest.toml` and exits
- language enablement is controlled by `[languages.<id>]` in config (defaults keep only `go`/`python` enabled)
- cannot be combined with `--trace`, `--impact`, or `--query-*`
- `--include-tests`
- include test files in analysis (`_test.go`, `test_*.py`)
- default behavior excludes test files
- `--history`
- enables local history snapshot capture and trend reporting
- writes snapshots to configured `db.path` (default resolved path: `data/database/history.db`)
- snapshots are isolated by active project key
- `--since string`
- optional history lower-bound filter used with `--history`
- accepted formats: RFC3339 or `YYYY-MM-DD`
- `--history-window string`
- moving-window duration used for trend moving averages and drift calculations
- default: `24h`
- `--history-tsv string`
- optional path for trend TSV export
- requires `--history`
- `--history-json string`
- optional path for trend JSON export
- requires `--history`
- `--query-modules`
- list modules through the shared query service
- use `--query-filter` for substring filtering
- `--query-filter string`
- optional substring filter for `--query-modules`
- `--query-module string`
- print details for one module via query service
- `--query-trace string`
- print dependency trace via query service
- format: `<from-module>:<to-module>`
- `--query-trends`
- print history trend slices from query service
- requires `--history`
- `--query-limit int`
- optional row/depth limit for query modes
- `--verbose`
- sets slog level to debug
- `--version`
- prints `circular v1.0.0`

## Positional Arguments

- in normal/watch/once/query/history mode, first positional argument overrides `watch_paths` with one path
- in trace mode, positional args are consumed as `<from> <to>`

## MCP Mode

- MCP startup is config-driven via `[mcp].enabled = true`
- MCP mode cannot be combined with `--once`, `--ui`, `--trace`, `--impact`, `--query-*`, `--history`, `--verify-grammars`, or positional path arguments
- MCP startup runs an initial scan and can auto-write outputs/config when `mcp.auto_manage_outputs` or `mcp.auto_sync_config` are enabled
- OpenAPI conversion (when enabled) reads `mcp.openapi_spec_path` or `mcp.openapi_spec_url` (mutually exclusive)
- MCP runtime uses a stdio JSON request/response loop (one JSON object per line)
- MCP tool protocol and examples live in `docs/documentation/mcp.md`

## Execution Order

For all modes except `--version`, runtime performs:
1. parse flags
2. discover/load config
3. resolve runtime paths and active project
4. normalize `grammars_path` to absolute path when relative
5. initialize app
6. run initial scan

Then mode-specific behavior:
- verify mode: run grammar manifest verification and exit
- trace mode: run shortest-chain query and exit
- impact mode: run impact analysis and exit
- query modes: run query-service read operation and exit
- history mode: append a project-scoped snapshot and print trend summary (plus optional TSV/JSON exports)
- once mode: run analyses/output generation and exit
- default watch mode: start watcher and process incremental updates forever
- UI mode: same watch pipeline, plus interactive issue/module explorer

## Logging

- default output: stdout
- with `--ui`:
- `$XDG_STATE_HOME/circular/circular.log` when `XDG_STATE_HOME` is set
- otherwise `~/.local/state/circular/circular.log`
- symlink log paths are refused
- if log-file setup fails, runtime warns to stderr and can fall back to stdout logging
