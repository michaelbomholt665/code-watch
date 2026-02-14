# Repository Guidelines

## Project Structure & Hexagonal Architecture
This project follows **Hexagonal Architecture**. All core logic must remain decoupled from infrastructure.
- **`internal/core/ports/`**: **The Source of Truth.** Define all interfaces (Ports) here.
- **`internal/core/app/`**: Orchestrates use cases using Ports. **DO NOT** import concrete adapters here.
- **`internal/engine/`, `internal/data/`, `internal/ui/`**: Implement adapters that fulfill the Ports.
- **`cmd/circular/`**: Wire up concrete adapters to the core service.

## Coding Style & Naming Conventions
- Follow standard Go formatting: run `gofmt` (or `go fmt ./...`) before committing.
- **Strict Boundaries**: Never pass infrastructure-specific types (e.g., `sitter.Node`, `sql.Rows`) into the core domain. Use normalized DTOs defined in `parser` or `graph`.
- **Dependency Injection**: Always use constructor injection to pass Port implementations into the Service layer.
- Use idiomatic Go naming: exported `PascalCase`, internal `camelCase`, short receiver names.
- `go build -o circular ./cmd/circular`: build the CLI binary.
- `go run ./cmd/circular --once`: run one scan and exit.
- `go run ./cmd/circular --ui`: run in watch mode with terminal UI.
- `go test ./...`: run all unit tests.
- `go test ./... -coverprofile=coverage.out`: regenerate coverage report.
- `scripts/codex-task "<prompt>"`: auto-route Codex by task (`go-dev`/`review`/`pm`) with persona + skill tag + model profile.
- `scripts/codex-task --mode go-dev "<prompt>"`: force Go implementation mode.
- `scripts/codex-task --mode review "<prompt>"`: force defect/code-review mode.
- `scripts/codex-task --mode pm "<prompt>"`: force project-planning mode.
- `scripts/codex-task --show-route "<prompt>"`: execute and print selected mode/profile/skill/persona.
- `scripts/codex-task --dry-run "<prompt>"`: print selected mode/profile/skill/persona without executing Codex.
- `make task PROMPT="<prompt>"`: auto-route with wrapper via Makefile.
- `make task pm PROMPT="<prompt>"`: run PM mode without explicit wrapper flags.
- `make task-go PROMPT="<prompt>"`, `make task-review PROMPT="<prompt>"`, `make task-pm PROMPT="<prompt>"`: short mode-specific wrappers.

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

## Codex Profiles and Personas
- Project-local Codex settings live in `.codex/config.toml`.
- Default profile is token-optimized (`gpt-5.2-codex`, low verbosity).
- Available profiles:
- `quick`: lowest cost/latency (`gpt-5-codex-mini`) for lightweight tasks.
- `build`: balanced coding profile for implementation/refactors.
- `review`: higher reasoning for defect-oriented reviews.
- `deep`: higher-capability non-max model for complex planning/architecture work.
- Persona prompt templates used by `scripts/codex-task`:
- `.codex/persona-go-coder.md`
- `.codex/persona-code-reviewer.md`
- `.codex/persona-project-manager.md`

## Strict Task Routing Policy (Mandatory)
- For all non-trivial tasks in this repository, use `scripts/codex-task` routing semantics as the source of truth for mode/profile/skill/persona selection.
- Prefer normal execution with visible routing: `scripts/codex-task --show-route "<prompt>"`.
- Use `--dry-run` only when debugging routing behavior or wrapper changes.
- Mirror that routing in execution:
- `go-dev` -> `build` profile + `$go-code-watch-maintainer` + `.codex/persona-go-coder.md`
- `review` -> `review` profile + `$go-defect-reviewer` + `.codex/persona-code-reviewer.md`
- `pm` -> `deep` profile + `$project-planner` + `.codex/persona-project-manager.md`
- If explicit user input conflicts with inferred routing, follow explicit user mode/skill.
- In status updates and final responses, state which mode/skill/persona was applied.

## Fallback Policy (If Wrapper Is Unavailable)
- If `scripts/codex-task` cannot be executed, manually replicate routing:
- infer or honor explicit mode (`go-dev`/`review`/`pm`)
- select the corresponding skill tag
- load the corresponding persona file from `.codex/`
- apply persona guidance as a first-class instruction for the turn
- For planning tasks, prefer embedding persona constraints into `project-planner` task framing so plan structure and tone remain consistent.
- If neither wrapper nor persona file is available, state the gap clearly and proceed with best-effort skill-only execution.

## Skills
A skill is a reusable workflow defined by `SKILL.md` under global Codex skills (`/home/michael/.codex/skills`).

### Skill Selection Rules
- If the user explicitly names a skill (for example `$go-defect-reviewer`), use it.
- If task intent clearly matches a skill description below, use that skill automatically.
- If multiple skills apply, choose the minimal set that fully covers the request.
- Prefer repository-specific Go skills for this project before generic skills.

### Available Global Skills
- `go-code-watch-maintainer`: Maintain/extend this Code Dependency Monitor with safe package-boundary changes, tests, and Go 1.24.9 compatibility.
- `go-code-analysis-compat`: Implement/review analyzers, watchers, dependency graphs, and output generation with compatibility/documentation rigor.
- `go-defect-reviewer`: Perform defect-risk reviews (error handling, crashes, resolver correctness, perf/resource leaks).
- `go-mcp-code-watch-server`: Build/extend Go MCP endpoints that expose scan/cycle/unresolved/report capabilities safely and with typed contracts.
- `changelog-maintainer`: Keep root `CHANGELOG.md` accurate and user-facing for added/changed/fixed/removed/docs updates.
- `project-planner`: Produce detailed implementation/migration plans with file inventories, tables, Mermaid/PlantUML diagrams, and task checkboxes.
- `fastmcp-builder`: Design/build/review FastMCP 3.x servers/clients, decorators, transport/runtime wiring, visibility, lifecycle, and integration.
- `database-expert`: Support database and data-engineering tasks (LadybugDB, DuckDB, PostgreSQL, Arrow, Polars, NumPy/SciPy, UUIDv7).
- `skill-creator`: Create or improve skills with clear workflows and compact, high-value context.
- `skill-installer`: List/install curated or GitHub-hosted skills into `$CODEX_HOME/skills`.

### Skill Routing Hints
- Implementation or bugfix request in repo: use `go-code-watch-maintainer`.
- Explicit code review/audit/reliability request: use `go-defect-reviewer`.
- Cross-cutting compatibility/output-contract work: use `go-code-analysis-compat`.
- MCP server/tooling exposure request: use `go-mcp-code-watch-server` or `fastmcp-builder` (Go vs Python/FastMCP).
- Planning/migration roadmap request: use `project-planner`.
- Release/version bump with user-visible changes: add `changelog-maintainer`.
