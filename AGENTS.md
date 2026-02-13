# Repository Guidelines

## Project Structure & Module Organization
- `cmd/circular/`: CLI entrypoint, app orchestration, and optional Bubble Tea UI.
- `internal/config/`: TOML config loading and validation.
- `internal/parser/`: Tree-sitter parsing for Go/Python and normalized file models.
- `internal/graph/`: dependency graph state and cycle detection.
- `internal/resolver/`: unresolved reference analysis and stdlib lookups.
- `internal/watcher/`: filesystem watch/debounce pipeline.
- `internal/output/`: DOT/TSV report generation.
- `grammars/`: bundled parser artifacts (`*.so`, node type metadata).
- `docs/documentation/`: architecture, CLI, config, and package-level docs.

## Build, Test, and Development Commands
- `go build -o circular ./cmd/circular`: build the CLI binary.
- `go run ./cmd/circular --once`: run one scan and exit.
- `go run ./cmd/circular --ui`: run in watch mode with terminal UI.
- `go test ./...`: run all unit tests.
- `go test ./... -coverprofile=coverage.out`: regenerate coverage report.

Use `circular.example.toml` as the starting config:
`cp circular.example.toml circular.toml`.

## Coding Style & Naming Conventions
- Follow standard Go formatting: run `gofmt` (or `go fmt ./...`) before committing.
- Keep package boundaries aligned with pipeline stages (parser/graph/resolver/watcher/output).
- Use idiomatic Go naming: exported `PascalCase`, internal `camelCase`, short receiver names.
- Test files should stay next to code as `*_test.go`.

## Testing Guidelines
- Primary framework is Go `testing` with table-driven tests where practical.
- Add tests for each behavioral change, especially parser extraction, graph replacement logic, resolver accuracy, and watcher event handling.
- Run `go test ./...` locally before opening a PR; include coverage output when touching critical analysis paths.

## Commit & Pull Request Guidelines
- This workspace does not include `.git` history, so commit style cannot be inferred from local logs.
- Use clear, imperative commit subjects (example: `resolver: handle aliased Python imports`).
- Keep commits focused; avoid mixing refactors with behavior changes.
- PRs should include: purpose, key design choices, test evidence (`go test ./...` output), and sample output changes (`graph.dot`/`dependencies.tsv`) when relevant.

## Versioning Guidelines
- Use Semantic Versioning: `MAJOR.MINOR.PATCH`.
- Increment `PATCH` for backward-compatible bug fixes and internal improvements that do not change public behavior/contracts.
- Increment `MINOR` for backward-compatible new features (new flags, additive config fields, additive DOT/TSV fields, new analysis modes).
- Increment `MAJOR` for breaking changes:
- CLI flag removals/renames or behavior changes that break existing workflows.
- Config schema changes that require user config updates.
- Output format changes that break existing parsers/integrations.
- Before release/version bump:
- Update `CHANGELOG.md` with user-facing notes.
- Keep docs in `docs/documentation/` aligned with the versioned behavior.
- Keep additive changes preferred; avoid breaking contracts unless major version bump is intentional.

## Security & Configuration Tips
- Do not commit local paths or private project roots in `circular.toml`.
- Keep `exclude` patterns updated to avoid scanning large generated/vendor directories.
