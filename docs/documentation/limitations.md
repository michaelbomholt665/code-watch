# Limitations and Known Constraints

## Parsing and Language Coverage

- only `.go` and `.py` files are parsed
- language detection is extension-based only
- parser uses bundled Tree-sitter Go/Python grammars, not dynamic grammar loading

## Resolver Heuristics

- unresolved-reference detection is heuristic and not compiler/type-checker accurate
- imported symbol resolution is best-effort for aliases, module prefixes, and `from ... import ...`
- `exclude.symbols` can hide false positives and true positives
- stdlib/builtin lists are static snapshots

## Graph Granularity

- dependency graph is module-level, not symbol-level edges
- cycle detection and import-chain tracing operate on module graph only
- unused import detection is reference-name based, not full semantic usage analysis

## Watch Semantics

- behavior depends on fsnotify event delivery semantics per platform/filesystem
- update batches are debounced and serialized, so high-frequency churn can delay analysis visibility
- test files (`*_test.go`, `*_test.py`) are ignored by watcher event processing

## Output Ordering

- DOT node/edge order and TSV row order are not guaranteed stable across runs because map iteration order is not deterministic

## Configuration Constraints

- only default config path (`./circular.toml`) has fallback to `./circular.example.toml`
- strict architecture isolation via empty `allow=[]` is not supported; at least one allowed layer is required

## UI/Logging

- in `--ui` mode, logging is redirected to a user state file when possible
- if file logging cannot be established, logging can fall back to stdout, which may affect terminal presentation
