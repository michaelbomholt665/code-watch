# Circular (Code Watch)

Circular is a dependency analysis tool for Go and Python projects. It detects circular imports, unresolved references, unused imports, and architecture rule violations while supporting one-shot scans, watch mode, and MCP operations.

## Project Overview

- **Core Functionality:** Analyzes Go and Python source code to build a dependency graph of modules and detect cycles.
- **High-Fidelity Parsing:** Uses `tree-sitter` for precise extraction of imports, definitions (functions, classes, types), and references.
- **Live Watch Mode:** Monitors the file system for changes and incrementally updates the graph, re-detecting cycles instantly.
- **Multi-Format Output:** Generates DOT/TSV plus Mermaid and PlantUML dependency diagrams, with optional markdown marker injection.

## Architecture

The project follows a layered architecture with internal packages:

- `cmd/circular/`: Entry point.
- `internal/ui/cli/`: CLI/runtime mode handling and optional Bubble Tea UI.
- `internal/core/app/`: End-to-end orchestration across scan/analyze/output/watch flows.
- `internal/core/config/`: TOML config decoding, defaults, and validation.
- `internal/core/watcher/`: File system watch and debounce pipeline.
- `internal/engine/parser/`: Tree-sitter parsing and normalized file extraction.
- `internal/engine/graph/`: Dependency graph state, cycle/trace/impact/metrics logic.
- `internal/engine/resolver/`: Unresolved and unused-import analysis.
- `internal/ui/report/` and `internal/ui/report/formats/`: DOT/TSV/Mermaid/PlantUML output generation and markdown injection.
- `internal/mcp/`: MCP runtime, schemas, validators, adapters, and operation handlers.

## Building and Running

### Prerequisites

- Go 1.24+.
- (Optional) Graphviz for visualizing DOT output.

### Build

```bash
go build -o circular ./cmd/circular
```

### Run

Run a single scan and exit:
```bash
./circular --config data/config/circular.toml --once
```

Run in watch mode:
```bash
./circular --config data/config/circular.toml
```

### Test

```bash
go test ./...
```

## Configuration

The tool is configured via TOML (`data/config/circular.toml` by default).

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
mermaid = "graph.mmd"
plantuml = "graph.puml"

[output.paths]
root = ""
diagrams_dir = "docs/diagrams"

[alerts]
beep = true
terminal = true
```

## Development Conventions

- **Module Name:** The project uses the module name `circular` in `go.mod`.
- **Parsing:** Language extraction logic belongs in `internal/engine/parser/` (including registry, grammar, and extractor modules).
- **Logging:** Uses `slog` for structured logging. Use `-verbose` flag to enable debug logs.
- **Grammars:** `grammars/` contains parser artifacts and manifest metadata used by optional grammar verification.
