# Limitations and Known Constraints

## Parsing and Language Coverage

- default runtime coverage is `.go` and `.py`
- additional languages can be enabled via `[languages.<id>]`; profile-driven extractors currently cover `javascript`, `typescript`, `tsx`, `java`, `rust`, `html`, `css`, `gomod`, and `gosum`
- language detection is registry-driven (extensions + optional exact filename routes)
- grammar artifacts are verified via `grammars/manifest.toml` when `grammar_verification.enabled=true`

## Resolver Heuristics

- unresolved-reference detection is heuristic and not compiler/type-checker accurate
- bridge-call contexts (`ffi_bridge`, `process_bridge`, `service_bridge`) reduce false positives but are pattern-driven and can miss custom interop wrappers
- explicit `.circular-bridge.toml` mappings are deterministic but require manual maintenance and can mask real unresolved references if over-broad
- universal symbol-table + probabilistic fallback matching improves cross-language resolution but can still miss highly dynamic dispatch or generated-code contracts
- service contract linking uses naming/decorator/signature heuristics (for example client/server/servicer suffix families), not schema-aware IDL compilation
- imported symbol resolution is best-effort for aliases/module prefixes and language-specific module naming:
- `go` (path base), `python` (dot modules), `javascript`/`typescript`/`tsx` (package/path base), `java` (package class), `rust` (`::` module base)
- `exclude.symbols` can hide false positives and true positives
- stdlib/builtin lists are static snapshots and language-scoped
- enriched definition metadata (signature/type/decorators/scope) is extracted from syntax only; it is not type-checked or runtime-validated

## Secret Detection Heuristics

- secret detection is heuristic and does not guarantee full credential coverage
- watch-mode incremental scanning is line-range based; fallback to full scan occurs when edits change line counts
- entropy checks are limited to high-risk extensions, so entropy-only findings in other file types are intentionally skipped
- entropy and context checks can still produce false positives/false negatives
- secret findings are exposed via MCP (`secrets.scan`, `secrets.list`) and TSV output blocks, but are still heuristic

## Graph Granularity

- dependency graph is module-level, not symbol-level edges
- cycle detection and import-chain tracing operate on module graph only
- unused import detection is reference-name based, not full semantic usage analysis
- `exclude.imports` suppresses by exact module path or import reference base name; broad entries can hide real issues
- unused import detection is intentionally disabled for metadata/markup languages (for example `html`, `css`, `gomod`, `gosum`)

## CQL Scope

- CQL is currently read-only and module-focused (`SELECT modules WHERE ...`)
- supported predicates are limited to module name and summary/metric fields (`fan_in`, `fan_out`, `depth`, counts)
- CQL is currently available through internal query-service APIs and is not yet exposed as a first-class CLI/MCP operation

## Watch Semantics

- behavior depends on fsnotify event delivery semantics per platform/filesystem
- update batches are debounced and serialized, so high-frequency churn can delay analysis visibility
- watcher filtering uses enabled language extension/filename routes plus language-specific test suffixes

## Cross-Platform Status

- path-prefix comparisons and diagram output path detection are separator-agnostic (`/` and `\`)
- broader cross-platform work remains in progress (`docs/plans/cross-platform-compatibility-plan.md`), including grammar artifact parity and multi-OS watcher/CI verification

## Output Ordering

- DOT node/edge order and TSV row order are not guaranteed stable across runs because map iteration order is not deterministic

## Configuration Constraints

- default config discovery starts at `./data/config/circular.toml` and includes legacy root fallbacks during migration
- strict architecture isolation via empty `allow=[]` is not supported; at least one allowed layer is required

## UI/Logging

- in `--ui` mode, logging is redirected to a user state file when possible
- if file logging cannot be established, logging can fall back to stdout, which may affect terminal presentation
