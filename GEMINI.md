# Circular (Code Watch)

Circular is a real-time dependency analysis tool for Go and Python projects. It detects circular imports and unresolved references, providing a live feedback loop for developers to maintain a healthy project architecture.

## Project Overview

- **Core Functionality:** Analyzes Go and Python source code to build a dependency graph of modules and detect cycles.
- **High-Fidelity Parsing:** Uses `tree-sitter` for precise extraction of imports, definitions (functions, classes, types), and references.
- **Live Watch Mode:** Monitors the file system for changes and incrementally updates the graph, re-detecting cycles instantly.
- **Multi-Format Output:** Generates DOT (Graphviz) files for visualization and TSV files for programmatic analysis.

## Architecture

The project follows a layered architecture with internal packages:

- `cmd/circular/`: Entry point and application orchestration.
- `internal/parser/`: Tree-sitter powered code parsing for Go and Python.
- `internal/resolver/`: Maps file paths to module names and resolves imports.
- `internal/graph/`: Manages the dependency graph and implements cycle detection (DFS).
- `internal/watcher/`: File system change detection with debouncing.
- `internal/config/`: TOML-based configuration management.
- `internal/output/`: Generators for report formats (DOT, TSV).

## Building and Running

### Prerequisites

- Go 1.25 or later.
- (Optional) Graphviz for visualizing DOT output.

### Build

```bash
go build -o circular ./cmd/circular
```

### Run

Run a single scan and exit:
```bash
./circular -config circular.toml -once
```

Run in watch mode:
```bash
./circular -config circular.toml
```

### Test

```bash
go test ./...
```

## Configuration

The tool is configured via a `circular.toml` file.

```toml
grammars_path = "./grammars"
watch_paths = ["./src"]

[exclude]
dirs = [".git", "node_modules", "vendor", "__pycache__"]
files = ["*.tmp", "*.log"]

[watch]
debounce = "1s"

[output]
dot = "graph.dot"
tsv = "dependencies.tsv"

[alerts]
beep = true
terminal = true
```

## Development Conventions

- **Module Name:** The project uses the module name `circular` in `go.mod`.
- **Parsing:** All language-specific extraction logic should be implemented in `internal/parser/` by implementing the `Extractor` interface.
- **Logging:** Uses `slog` for structured logging. Use `-verbose` flag to enable debug logs.
- **Grammars:** While currently using static Go bindings for Tree-sitter, the `grammars/` directory contains shared libraries for reference or alternative loading strategies.
