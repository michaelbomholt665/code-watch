# Changelog

All notable changes to this project will be documented in this file.

## 2026-02-13

### Added
- `resolver:` Added unused import detection and TSV `unused_import` output rows in `dependencies.tsv`.
- `graph:` Added module dependency metrics (`depth`, `fan-in`, `fan-out`) and DOT metric annotations.
- `cli:` Added `--trace <from> <to>` mode to print shortest import chains.
- `cli:` Added `--impact <file-or-module>` mode to report direct/transitive import impact and exported symbols.
- `architecture:` Added optional layer/rule validation engine with violation reporting.
- `graph:` Added complexity hotspot ranking from parser function metrics.
- `output:` Added additive TSV `architecture_violation` rows and DOT `cx=<score>` module annotations.
- `docs:` Added `docs/documentation/output.md` with DOT/TSV output contracts.
- `skills:` Added `changelog-maintainer` skill for maintaining root `CHANGELOG.md`.
- `runtime:` Added `internal/cliapp` and `internal/app` packages to separate CLI/runtime concerns from analysis orchestration.
- `watcher:` Added handling for newly created directories (recursive registration + enqueue of existing files) and rename-event processing.
- `config:` Added strict architecture config validation for duplicate/overlapping layer paths, duplicate rules, unknown layer references, and multiple rules per `from` layer.
- `output:` Added Mermaid graph generation (`output.mermaid`) with cycle/violation/external edge styling and optional architecture layer subgraphs.
- `output:` Added PlantUML graph generation (`output.plantuml`) with component/package rendering and cycle/violation edge annotations.
- `output:` Added marker-based Markdown diagram injection via `[[output.update_markdown]]` and `<!-- circular:<marker>:start/end -->` blocks.
- `config:` Added `output.paths.root` and `output.paths.diagrams_dir` for root-aware output path resolution.
- `history:` Added `internal/history` phase-1 snapshot persistence (`.circular/history.jsonl`) and optional git commit metadata capture.
- `cli:` Added `--history` and `--since` flags for opt-in historical trend analysis.
- `output:` Added trend report exporters `--history-tsv` and `--history-json` with new `internal/output` renderers.
- `query:` Added `internal/query` shared read service with deterministic module listing, module details, dependency traces, and history trend slices.
- `ui:` Added module explorer panel in terminal UI (`tab` to switch between issues and modules).
- `history:` Added SQLite-backed snapshot storage at `.circular/history.db` with schema migrations, version drift checks, and lock-retry policy.
- `cli:` Added `--history-window` for configurable trend moving windows.
- `cli:` Added query-service command flags `--query-modules`, `--query-filter`, `--query-module`, `--query-trace`, `--query-trends`, and `--query-limit`.
- `ui:` Added module detail drill-down, dependency cursor navigation, trend overlay toggle, and `$EDITOR` source-jump action.
- `history:` Added benchmark coverage (`BenchmarkStore_SaveSnapshot`, `BenchmarkStore_LoadSnapshots`) for persistence performance guardrails.
- `config:` Added versioned schema blocks for `version`, `[paths]`, `[config]`, `[db]`, `[projects]`, and `[mcp]` with validation/defaulting in `internal/config`.
- `config:` Added centralized runtime path resolver (`internal/config/paths.go`) for project root, config/state/cache/database directories, DB path, MCP config path, and output root.
- `config:` Added project registry and active-project resolution helpers in `internal/config/projects.go`.
- `history:` Added project-key-aware snapshot model/storage/query semantics and schema migration to schema version `2`.
- `config:` Added canonical templates `data/config/circular.example.toml` and `data/config/projects.example.toml`.
- `parser:` Added grammar provenance manifest support (`grammars/manifest.toml`) with checksum and AIB version verification helpers.
- `cli:` Added `--verify-grammars` mode to validate enabled-language grammar artifacts and exit.
- `parser:` Added language registry model (`internal/parser/language_registry.go`) with default rollout states and override merge/validation.
- `config:` Added `[grammar_verification]` and `[languages.<id>]` configuration sections for verification control and language routing overrides.
- `resolver:` Added language-specific module-name heuristics for `javascript`/`typescript`/`tsx`, `java`, and `rust`.
- `resolver:` Added language-scoped stdlib catalogs in `internal/resolver/stdlib/{javascript,java,rust}.txt` and wired them into unresolved reference checks.
- `parser:` Added consolidated profile-driven extractor registry in `internal/parser/profile_extractors.go` and default extractor auto-registration for all enabled languages.
- `parser:` Added multi-language parser matrix coverage for `javascript`, `typescript`, `tsx`, `java`, `rust`, `html`, `css`, `gomod`, and `gosum`.
- `config:` Added committed runtime defaults at `data/config/circular.toml` and `data/config/projects.toml` so the default config path works out of the box.
- `mcp:` Added MCP runtime bootstrap scaffolding and project-context resolution for config-driven MCP startup.
- `config:` Added MCP POC configuration fields for server metadata, tool exposure policy, response limits, and auto-sync controls.
- `config:` Added per-project `config_file` support and enforced unique `db_namespace` values for SQLite isolation.
- `config:` Added MCP OpenAPI spec source keys (`mcp.openapi_spec_path`, `mcp.openapi_spec_url`) with mutual exclusivity validation and path resolution.
- `mcp:` Added stdio JSON request/response protocol with single-tool operation dispatch (`circular`) and allowlist enforcement.
- `mcp:` Added adapter + handler packages for scan/query/graph/system operations with bounded outputs.
- `docs:` Added `docs/documentation/mcp.md` covering MCP protocol, operations, and examples.

### Changed
- `architecture:` Reorganized internal packages into pillar paths: `internal/core/{app,config,watcher}`, `internal/engine/{parser,resolver,graph}`, `internal/data/{history,query}`, and `internal/ui/{cli,report}`.
- `ui:` Renamed UI package identifiers to match directory names (`package cli`, `package report`) while preserving existing CLI behavior and outputs.
- `parser:` Split parser internals into sub-packages `internal/engine/parser/{registry,grammar,extractors}` while preserving `internal/engine/parser` APIs via bridges.
- `resolver:` Split language-specific resolver logic into `internal/engine/resolver/drivers` while preserving `resolver.New*Resolver()` API surface.
- `report:` Split report format generators into `internal/ui/report/formats` while preserving `report.New*Generator()` API surface.
- `runtime:` Lowered module minimum Go version in `go.mod` from `1.25.x` to `1.24`.
- `docs:` Clarified semantic versioning policy in `AGENTS.md` and `README.md`.
- `config:` Extended `circular.toml` schema with `[architecture]`, `[[architecture.layers]]`, and `[[architecture.rules]]`.
- `summary:` Terminal summary now includes architecture violations and top complexity hotspots.
- `cmd:` Reduced `cmd/circular/main.go` to a thin entrypoint shell delegating execution to `internal/cliapp.Run(...)`.
- `logging:` UI-mode logging now writes to state-path files with symlink refusal and tighter file permissions.
- `output:` Refined Mermaid and PlantUML visual styling with explicit cycle/violation labels, edge styling, node classes, and deterministic Mermaid link-style ordering.
- `output:` Relative output paths now resolve from auto-detected project root (`go.mod`/`.git`/`circular.toml`), and filename-only Mermaid/PlantUML paths resolve under `docs/diagrams/` by default.
- `output:` Improved Mermaid/PlantUML readability with increased spacing defaults and expanded legends explaining node metric fields (`d`, `in`, `out`, `cx`) and edge labels.
- `app:` Initial scan root handling now normalizes/deduplicates relative and absolute watch roots to prevent duplicate file ingestion and inflated metrics.
- `runtime:` Single-scan and watch startup flow now optionally records one history snapshot and prints a trend summary when `--history` is enabled.
- `runtime:` UI update pipeline now uses query-service-backed module summaries for explorer rendering.
- `history:` Snapshot/trend models now include fan-in/fan-out aggregate metrics and drift deltas for richer trend reporting.
- `runtime:` History persistence backend changed from JSONL to SQLite while preserving opt-in behavior behind `--history`.
- `cli:` Changed default config path to `./data/config/circular.toml` with explicit fallback chain to legacy root paths.
- `runtime:` Reworked config loading to keep explicit `--config` strict while default discovery uses ordered fallbacks and warns on deprecated `./circular.toml`.
- `runtime:` History DB open path now uses config/path resolver (`db.path` under `paths.database_dir`) instead of hard-coded `.circular/history.db`.
- `runtime:` Query/history flows now scope snapshots to the resolved active project key.
- `output:` Output root resolution in `internal/app` now uses the centralized config path resolver.
- `config:` Updated root `circular.example.toml` as a transitional compatibility template pointing to `data/config`.
- `app:` Startup now builds a parser language registry from config, enforces grammar verification by default, and passes registry-aware filters into scan/watch paths.
- `parser:` File language detection now uses registry-driven extension/filename routing instead of hardcoded `.go`/`.py` switches.
- `watcher:` Event filtering now supports registry-driven extension/filename filters plus configurable test-file suffix sets.
- `resolver:` Unused-import analysis now runs only for supported languages and skips unsupported/metadata languages to reduce false positives.
- `resolver:` Stdlib matching is now scoped per file language instead of a merged cross-language namespace.
- `parser:` Grammar loader now initializes runtime grammars for registry-enabled `css`, `html`, `java`, `javascript`, `rust`, `tsx`, and `typescript`.
- `parser:` `gomod` and `gosum` parsing now uses raw-text extraction paths that do not require runtime tree-sitter bindings.
- `app:` App startup now registers default extractors from the language registry instead of hardcoding Go/Python extractor wiring.
- `config:` Expanded example exclude guidance to cover project-specific symbol/import suppression.
- `runtime:` MCP mode now starts from TOML config and enforces CLI incompatibility checks when enabled.
- `docs:` Updated configuration/CLI/README docs and MCP examples to align with the expanded MCP config contract.
- `docs:` Updated MCP allowlist examples to use operation IDs (`scan.run`, `graph.cycles`, `system.sync_outputs`) with legacy alias mapping.

### Fixed
- `compatibility:` Restored `GOTOOLCHAIN=go1.24 go test ./...` compatibility by aligning the module Go directive.
- `watcher:` Serialized debounced callbacks to avoid overlapping update handlers during bursty filesystem activity.
- `history:` Improved corrupt-database handling with explicit SQLite initialization/ping failures and drift-safe schema validation.
- `resolver:` Corrected unused-import suppression to honor `exclude.imports` (module path/base name matching) instead of symbol exclusions.
- `config:` Corrected TOML key placement for runtime/example configs so top-level `grammars_path` and `watch_paths` load under the intended schema.
- `resolver:` Treated indexed local symbols (for example `items[i].Field`) as local to avoid false unresolved findings.
- `resolver:` Reported qualified unresolved references even when no matching import is present.

### Removed
- `parser:` Removed language-specific extractor files `internal/parser/{javascript,typescript,tsx,java,rust,html,css,gomod,gosum}.go` after profile parity migration.

### Docs
- Updated `README.md` and `docs/documentation/*` package-path references for the new `internal/core|engine|data|ui` layout.
- Updated `docs/plans/internal-refactor-plan.md` checklist to mark completed refactor and verification tasks.
- Updated package documentation for new sub-package locations under parser/resolver/report.
- Documented trace-mode CLI behavior and runtime mode flow in `docs/documentation/cli.md`.
- Updated docs index in `docs/documentation/README.md` to include output reference docs.
- Updated architecture/configuration/output/package docs and root `README.md` for new medium-tier features.
- Documented output ordering caveats (map-iteration order is not guaranteed stable for DOT/TSV row ordering).
- Documented Mermaid/PlantUML output configuration and markdown marker usage in `docs/documentation/configuration.md` and `docs/documentation/output.md`.
- Updated `circular.example.toml` and `README.md` output examples to include Mermaid/PlantUML and markdown injection configuration.
- Clarified output metric interpretation in `docs/documentation/output.md`, including practical `cx` severity guidance (`>=80` high, `>=100` very high).
- Added `docs/documentation/advanced.md` for phase-1 history/trend workflows and updated CLI/package docs for new history flags and outputs.
- Updated advanced roadmap docs with explicit implemented vs pending high-complexity tasks and current T3/T4 partial status.
- Updated `docs/plans/high-complexity-feature-plan.md` with a task-by-task implemented-vs-missing status snapshot.
- Updated advanced docs and CLI/package references for SQLite history backend, query command surface, TUI drill-down flows, and benchmark guidance.
- Updated README quickstart/CLI/config examples to use `data/config/circular.toml` and document default discovery order.
- Updated docs for config schema v2 pathing/project/MCP blocks, DB default path (`data/database/history.db`), and migration behavior.
- Updated `docs/plans/config-expansion-pathing-plan.md` task checklist to mark T1-T8 complete.
- Updated README, CLI/config/package/limitations documentation, and grammar expansion plan status to reflect manifest verification, language registry controls, and `--verify-grammars`.
- Updated grammar expansion plan checklist to mark T6 (resolver/stdlib strategy) complete.
- Updated README/configuration/packages/limitations docs to describe language-scoped resolver policies and unsupported-language unused-import behavior.
- Updated README and docs (`architecture.md`, `cli.md`, `configuration.md`, `packages.md`, `limitations.md`) to reflect profile-driven multi-language extraction and registry-based rollout controls.
- Updated `docs/plans/grammar-expansion-aib14-aib15-plan.md` to mark T5/T7/T8/T9 complete and record the simplified profile implementation approach.
- Corrected README and configuration examples to place `grammars_path`/`watch_paths` at TOML top-level and documented `exclude.imports` for unused-import suppression.
- Documented `--include-tests` default behavior (tests excluded unless flag is enabled).
- Documented example exclude lists for project-specific symbol/import suppression in configuration docs.
- Documented MCP OpenAPI spec source configuration in `README.md` and `docs/documentation/cli.md`.
- Documented MCP stdio protocol and operation surface in `docs/documentation/mcp.md`.
- Documented systemd socket-activation and stdio bridge setup for MCP clients in `docs/documentation/mcp.md`.
