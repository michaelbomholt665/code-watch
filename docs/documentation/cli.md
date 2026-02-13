# CLI Reference

Entrypoint: `cmd/circular/main.go`

## Usage

```bash
circular [flags] [path]
```

## Flags

- `--config string`
- default: `./circular.toml`
- if missing and path is default, app falls back to `./circular.example.toml`
- `--once`
- run initial scan and exit
- `--ui`
- start Bubble Tea UI mode
- logs are redirected to user state log path to avoid TUI corruption
- `--trace`
- trace shortest dependency chain: `circular --trace <from-module> <to-module>`
- requires exactly two module arguments and exits after printing the chain
- `--verbose`
- sets logger level to debug
- `--version`
- prints `circular v1.0.0`

## Positional Argument

If a positional path is provided, it overrides config `watch_paths` with that single path.

Example:

```bash
circular --once ./internal
```

Trace example:

```bash
circular --trace github.com/acme/project/cmd/app github.com/acme/project/internal/store
```

## Runtime Modes

- one-shot mode:
- parse -> build graph -> detect cycles/unresolved/unused-imports -> write outputs -> print summary -> exit
- trace mode (`--trace`):
- parse -> build graph -> print import chain (or not-found error) -> exit
- watch mode (default):
- same initial pass, then watches configured paths for changes and incrementally reprocesses changed files
- UI mode (`--ui`):
- same watch behavior, but summary appears in interactive terminal view

## Logging

- default logs: stdout
- with `--ui`: logs write to `$XDG_STATE_HOME/circular/circular.log` when available, else `~/.local/state/circular/circular.log`
- if no writable log location is available, logging may fall back to stdout
