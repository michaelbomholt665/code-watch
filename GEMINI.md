# Circular (Code Watch)

Circular is a dependency analysis tool for Go and Python projects. It detects circular imports, unresolved references, unused imports, and architecture rule violations while supporting one-shot scans, watch mode, and MCP operations.

## Project Overview

- **Core Functionality:** Analyzes Go and Python source code to build a dependency graph of modules and detect cycles.
- **High-Fidelity Parsing:** Uses `tree-sitter` for precise extraction of imports, definitions (functions, classes, types), and references.
- **Persistent Symbol Table:** Uses a SQLite-backed store for incremental symbol lookup and cross-language resolution.
- **Live Watch Mode:** Monitors the file system for changes and incrementally updates the graph, re-detecting cycles instantly.
- **Incremental Secret Scanning:** Performs hunk-based secret detection on changed lines to optimize performance.
- **Circular Query Language (CQL):** Provides a read-only query engine for advanced architectural analysis.
- **Multi-Format Output:** Generates DOT/TSV plus Mermaid and PlantUML dependency diagrams, with optional markdown marker injection.

## Architecture

The project follows a **Hexagonal Architecture** (Ports and Adapters) to ensure maintainability, testability, and scalability:

- **Core Domain (`internal/core/`)**
    - `app/`: Pure business logic orchestrating scan and analysis workflows via `AnalysisService`.
    - `ports/`: Defines all system boundaries (interfaces) for both driving (CLI, MCP) and driven (Parser, Storage) adapters.
    - `config/`: Application configuration and validation.
- **Adapters (`internal/engine/`, `internal/data/`, `internal/ui/`, `internal/mcp/`)**
    - `engine/parser/`: Adapts Tree-sitter to the `CodeParser` port.
    - `engine/graph/`: Core dependency graph and symbol store implementation.
    - `engine/resolver/`: Logic for reference resolution and cross-language bridges (`.circular-bridge.toml`).
    - `engine/secrets/`: Incremental secret scanning logic.
    - `data/history/`: Adapts SQLite to the `HistoryStore` port (Schema v3+).
    - `data/query/`: Implements the CQL parser and execution engine.
    - `mcp/`: Driving adapter providing an agentic interface to the `AnalysisService`.
    - `ui/cli/` & `ui/report/`: Driving adapters for terminal and file-based presentation.

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

### Explicit Bridges

For ambiguous cross-language dependencies, create a `.circular-bridge.toml` in the project root:

```toml
[[bridges]]
from = "go:internal/mcp/runtime"
to = "python:circular_mcp.server"
reason = "JSON-RPC over Stdio"
```

## Development Conventions

- **Module Name:** The project uses the module name `circular` in `go.mod`.
- **Incrementalism:** Prioritize incremental updates. Use `UpsertFile` and `DeleteFile` in the symbol store; avoid full-repo rescans where diff-based analysis is possible.
- **Parsing:** Language extraction logic belongs in `internal/engine/parser/` (including registry, grammar, and extractor modules).
- **CQL:** Queries must remain read-only. Complex graph traversals should be implemented in `internal/data/query/`.
- **Logging:** Uses `slog` for structured logging. Use `-verbose` flag to enable debug logs.
- **Grammars:** `grammars/` contains parser artifacts and manifest metadata used by optional grammar verification.
- **Secrets:** Entropy analysis is restricted to high-risk extensions (e.g., `.env`, `.key`, `.json`) to minimize false positives.
