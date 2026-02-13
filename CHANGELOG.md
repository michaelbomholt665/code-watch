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

### Changed
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

### Fixed
- `compatibility:` Restored `GOTOOLCHAIN=go1.24 go test ./...` compatibility by aligning the module Go directive.
- `watcher:` Serialized debounced callbacks to avoid overlapping update handlers during bursty filesystem activity.

### Removed
- None.

### Docs
- Documented trace-mode CLI behavior and runtime mode flow in `docs/documentation/cli.md`.
- Updated docs index in `docs/documentation/README.md` to include output reference docs.
- Updated architecture/configuration/output/package docs and root `README.md` for new medium-tier features.
- Documented output ordering caveats (map-iteration order is not guaranteed stable for DOT/TSV row ordering).
- Documented Mermaid/PlantUML output configuration and markdown marker usage in `docs/documentation/configuration.md` and `docs/documentation/output.md`.
- Updated `circular.example.toml` and `README.md` output examples to include Mermaid/PlantUML and markdown injection configuration.
- Clarified output metric interpretation in `docs/documentation/output.md`, including practical `cx` severity guidance (`>=80` high, `>=100` very high).
