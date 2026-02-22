# JULES.md - Codebase Review Guide

This document serves as the primary guide for Jules when working on the `circular` codebase.

## Project Overview
`circular` is a Go-based dependency monitor for codebases parsed with Tree-sitter. It detects circular imports, unresolved symbols, and other dependency issues.

## Architecture
The project follows **Hexagonal Architecture**.
- **Ports (`internal/core/ports`)**: Define interfaces for the core domain. These are the source of truth.
- **Application (`internal/core/app`)**: Contains the business logic and orchestrates use cases. It depends *only* on Ports, not on concrete implementations.
- **Adapters**:
    - `internal/engine`: Implementations for parsing, graph analysis, and resolution.
    - `internal/data`: Data persistence (SQLite, etc.).
    - `internal/ui`: User interface (CLI, TUI, reports).
- **Wiring (`cmd/circular`)**: Connects adapters to the core application.

## Review Strategy
Reviews are conducted in focused chunks:

1.  **Core Review**: Focus on `internal/core/ports` and `internal/core/app` to ensure clean separation of concerns and correct business logic.
2.  **Engine Review**: Focus on `internal/engine` (parser, graph, resolver).
3.  **Data & Infrastructure Review**: Focus on `internal/data`.
4.  **UI & CLI Review**: Focus on `internal/ui` and `cmd`.

## Coding Standards
- **Go Style**: Follow standard Go conventions (idiomatic Go).
- **Testing**: Ensure high test coverage, especially for core logic and complex adapters.
- **Documentation**: Code should be self-documenting where possible, with comments for complex logic.

## Tasks Status
- [x] Review Core Domain (`internal/core`) - See `REVIEW.md`
- [x] Review Engine (`internal/engine`) - See `REVIEW.md`
- [x] Review Data (`internal/data`) - See `REVIEW.md`
- [x] Review UI/CLI (`internal/ui`, `cmd`) - See `REVIEW.md`

## Next Steps (Recommendations)
1.  **Fix Race Condition**: Address the race condition in `SyncOutputs` (`internal/core/app/service.go`).
2.  **Implement Parser Pool**: Integrate `ParserPool` into `Parser` (`internal/engine/parser/parser.go`) to fix the unused pool issue.
3.  **Decouple Infrastructure**: Refactor `App` to use `SymbolStore` interface instead of concrete `SQLiteSymbolStore`.
