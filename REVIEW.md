# Codebase Review Findings

## 1. Core Domain (`internal/core`)

### Architecture & Design
*   **Strengths**:
    *   The project generally follows Hexagonal Architecture, with clear separation of ports (`internal/core/ports`) and application logic (`internal/core/app`).
    *   Dependency injection is supported via `NewWithDependencies`.
    *   Context usage is consistent.

### Issues Identified

#### 1. Leaking Infrastructure Details into Core
`internal/core/app/app.go` imports `circular/internal/engine/graph` and uses `*graph.SQLiteSymbolStore`.

```go
type App struct {
    // ...
    symbolStore   *graph.SQLiteSymbolStore
    // ...
}
```

**Note**: This concrete dependency is acknowledged as an acceptable deviation for this project's scope (a development tool for a larger system). It serves as a persistent cache/store to optimize memory usage for large codebases.

#### 2. Watcher as an Adapter in Core
`internal/core/watcher/watcher.go` imports `github.com/fsnotify/fsnotify` directly.
**Violation**: This is a concrete adapter implementation residing in `internal/core`. In strict Hexagonal Architecture, core logic should not depend on external drivers like filesystem event watchers.
**Recommendation**: Define a `Watcher` port (interface) in `ports` and move the implementation to `internal/ui/watcher` or `internal/engine/watcher`.

## 2. Engine (`internal/engine`)

### Parser (`internal/engine/parser`)
*   **Issue**: The `Parser` implementation in `parser.go` does **not** seem to use the `ParserPool` defined in `pool.go`.
    ```go
    // parser.go
    func (p *Parser) ParseFile(...) {
        parser := sitter.NewParser() // Allocates new parser every time
        defer parser.Close()
        // ...
    }
    ```
    The `README.md` claims "Uses a `sync.Pool`-backed Tree-sitter parser instance pool". This seems to be false or broken.
    **Resolution**: Fixed by integrating `ParserPool` into `Parser`.

### Graph (`internal/engine/graph`)
*   **Observation**: `SQLiteSymbolStore` is tightly coupled to `parser.File`. While this is acceptable for an internal adapter, it reinforces the need for decoupling via interfaces if we ever want to swap the parser or store.

### Resolver (`internal/engine/resolver`)
*   **Coupling**: `NewResolverWithSQLite` creates a hard dependency on SQLite via `graph`. This should ideally be injected.

## 3. Data (`internal/data`)

### History (`internal/data/history`)
*   **Structure**: `Store` is a concrete struct. It implements `ports.HistoryStore` implicitly (Go structural typing).
*   **Dependency**: Hard dependency on `modernc.org/sqlite`.
*   **Conformance**: It seems to follow the intended design, though strict interface assertions (e.g., `var _ ports.HistoryStore = (*Store)(nil)`) would be beneficial to ensure compliance.

### Config (`data/config`)
*   **Documentation**: `data/config/circular.example.toml` serves as a comprehensive reference for all configuration options.

## 4. UI/CLI (`internal/ui`, `cmd`)

### CLI (`internal/ui/cli`)
*   **Complexity**: `Run` function in `runtime.go` is very large and handles too many responsibilities (config loading, validation, initialization, signal handling, mode dispatch).
*   **Refactoring**: Consider breaking `Run` into smaller, focused functions or a `Runtime` struct that manages the lifecycle.

## 5. Model Context Protocol (`internal/mcp`)
*   **Architecture**: MCP acts as a driving adapter, translating tool calls into `AnalysisService` operations.
*   **Coupling**: `internal/mcp/adapters/adapter.go` depends on the concrete `history.Store` instead of `ports.HistoryStore`.
    ```go
    type Adapter struct {
        history *history.Store
    }
    ```
    This couples the MCP adapter to the specific SQLite history implementation.
*   **Recommendation**: Change `Adapter` to depend on `ports.HistoryStore`.

## 6. Shared (`internal/shared`)
*   **Observability**: `internal/shared/observability` implements Prometheus metrics and OpenTelemetry tracing.
*   **Coupling**: The Core (`internal/core/app`) imports `internal/shared/observability` directly. This couples the core domain to specific observability implementations (Prometheus/OTLP).
*   **Assessment**: While strict Hexagonal Architecture would use an `Observability` port, this is a common pragmatic trade-off. However, it's worth noting as a deviation.

## Summary & Next Steps
The codebase is well-structured and follows the intended architecture mostly. However, there are a few critical issues:
1.  **Performance**: `ParserPool` is unimplemented/unused (Fixed).
2.  **Coupling**: Some infrastructure details (SQLite, fsnotify, Observability) leak into Core.

**Recommended Actions**:
1.  Move `internal/core/watcher` to an adapter layer.
