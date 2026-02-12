# circular

`circular` is a Go-based dependency monitor for Go and Python codebases.

It scans source files, builds a module import graph, then reports:
- circular imports
- unresolved symbol references ("hallucinations")

It can run once (`--once`) or watch continuously with optional terminal UI mode (`--ui`).

## Features

- Parses `.go` and `.py` files with Tree-sitter
- Builds and updates a module-level dependency graph
- Detects import cycles across internal modules
- Detects unresolved references using local symbols, imports, stdlib, and builtins
- Emits outputs:
  - Graphviz DOT (`graph.dot` by default)
  - TSV edge list (`dependencies.tsv` by default)
- Live filesystem watch mode with debounce
- Optional Bubble Tea terminal UI for live issue monitoring

## Install / Build

Requirements:
- Go `1.24.2+` supported
- Go `1.23` is not supported by current dependencies
- `go.mod` currently pins `go 1.25.7`

Build:

```bash
go build -o circular ./cmd/circular
```

Run tests:

```bash
go test ./...
```

## Quick Start

1. Copy example config:

```bash
cp circular.example.toml circular.toml
```

2. Update `watch_paths` in `circular.toml` to your project path.

3. Run a one-time scan:

```bash
go run ./cmd/circular --once
```

4. Run in watch mode:

```bash
go run ./cmd/circular
```

5. Run in UI mode:

```bash
go run ./cmd/circular --ui
```

## CLI

Flags:
- `--config` path to TOML config (default `./circular.toml`)
- `--once` run one scan and exit
- `--ui` start terminal UI mode
- `--verbose` enable debug logs
- `--version` print version and exit

Positional arg:
- first positional argument overrides `watch_paths` with a single path

Version in source: `1.0.0` (`cmd/circular/main.go`).

## Configuration

Primary config file is TOML.

Example (`circular.example.toml`):

```toml
grammars_path = "./grammars"
watch_paths = ["./src"]

[exclude]
dirs = [".git", "node_modules", "vendor", "__pycache__"]
files = ["*.tmp", "*.log"]
symbols = ["self", "ctx", "p", "log", "toml", "sitter", "tea", "fsnotify"]

[watch]
debounce = "1s"

[output]
dot = "graph.dot"
tsv = "dependencies.tsv"

[alerts]
beep = true
terminal = true
```

## Outputs

- DOT graph (default `graph.dot`) for visual inspection in Graphviz tools
- TSV import edges (default `dependencies.tsv`) with columns:
  - `From`, `To`, `File`, `Line`, `Column`

## Documentation

Full documentation is in `docs/documentation/`:
- `docs/documentation/README.md`
- `docs/documentation/cli.md`
- `docs/documentation/configuration.md`
- `docs/documentation/architecture.md`
- `docs/documentation/packages.md`
- `docs/documentation/limitations.md`
- `docs/documentation/ai-audit.md`

AI audit reports are in `docs/reviews/`:
- `docs/reviews/performance-security-review-2026-02-12.md`

## Project Layout

- `cmd/circular/` CLI app, orchestration, TUI
- `internal/config/` TOML config loading
- `internal/parser/` Tree-sitter parsing and extractors
- `internal/graph/` dependency graph + cycle detection
- `internal/resolver/` unresolved reference detection
- `internal/watcher/` fsnotify watch + debounce
- `internal/output/` DOT and TSV generators
