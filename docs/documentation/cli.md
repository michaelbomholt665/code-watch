# CLI Reference

Entrypoint: `cmd/circular/main.go` delegates to `internal/cliapp.Run(...)`.

## Usage

```bash
circular [flags] [path]
```

## Flags

- `--config string`
- default: `./circular.toml`
- if default config load fails, runtime retries with `./circular.example.toml`
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
- `--history`
- enables local history snapshot capture and trend reporting
- writes snapshots to `.circular/history.db` (SQLite) in the current working directory
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

## Execution Order

For all modes except `--version`, runtime performs:
1. parse flags
2. load config
3. normalize `grammars_path` to absolute path when relative
4. initialize app
5. run initial scan

Then mode-specific behavior:
- trace mode: run shortest-chain query and exit
- impact mode: run impact analysis and exit
- query modes: run query-service read operation and exit
- history mode: append a snapshot and print trend summary (plus optional TSV/JSON exports)
- once mode: run analyses/output generation and exit
- default watch mode: start watcher and process incremental updates forever
- UI mode: same watch pipeline, plus interactive issue/module explorer
 - UI panels:
 - Issues panel: cycles + unresolved references
 - Module Explorer panel: module summaries + detail drill-down (`tab` to switch, `enter` to drill down)
 - Trend overlay: press `t` in UI
 - Source jump: press `o` in module details (opens `$EDITOR`)

## Logging

- default output: stdout
- with `--ui`:
- `$XDG_STATE_HOME/circular/circular.log` when `XDG_STATE_HOME` is set
- otherwise `~/.local/state/circular/circular.log`
- symlink log paths are refused
- if log-file setup fails, runtime warns to stderr and can fall back to stdout logging
