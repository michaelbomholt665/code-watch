# Changelog

All notable changes to this project will be documented in this file.

## 2026-02-13

### Added
- `resolver:` Added unused import detection and TSV `unused_import` output rows in `dependencies.tsv`.
- `graph:` Added module dependency metrics (`depth`, `fan-in`, `fan-out`) and DOT metric annotations.
- `cli:` Added `--trace <from> <to>` mode to print shortest import chains.
- `docs:` Added `docs/documentation/output.md` with DOT/TSV output contracts.
- `skills:` Added `changelog-maintainer` skill for maintaining root `CHANGELOG.md`.

### Changed
- `runtime:` Lowered module minimum Go version in `go.mod` from `1.25.7` to `1.24.9`.
- `docs:` Clarified semantic versioning policy in `AGENTS.md` and `README.md`.

### Fixed
- `compatibility:` Restored `GOTOOLCHAIN=go1.24.9 go test ./...` compatibility by aligning the module Go directive.

### Removed
- None.

### Docs
- Documented trace-mode CLI behavior and runtime mode flow in `docs/documentation/cli.md`.
- Updated docs index in `docs/documentation/README.md` to include output reference docs.
