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
- `--verbose`
- sets slog level to debug
- `--version`
- prints `circular v1.0.0`

## Positional Arguments

- in normal/watch/once mode, first positional argument overrides `watch_paths` with one path
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
- once mode: run analyses/output generation and exit
- default watch mode: start watcher and process incremental updates forever
- UI mode: same watch pipeline, plus interactive issue view

## Logging

- default output: stdout
- with `--ui`:
- `$XDG_STATE_HOME/circular/circular.log` when `XDG_STATE_HOME` is set
- otherwise `~/.local/state/circular/circular.log`
- symlink log paths are refused
- if log-file setup fails, runtime warns to stderr and can fall back to stdout logging
