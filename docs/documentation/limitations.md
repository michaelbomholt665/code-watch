# Limitations and Known Constraints

## Parsing and Language Support

- only `.go` and `.py` files are parsed
- language detection is extension-based only
- grammar loader uses embedded tree-sitter bindings; `grammars_path` is validated as a directory but is not used to dynamically load grammars from disk

## Watch Behavior

- watcher recursively registers directories at startup
- directories created later are registered when create events are observed
- watch callback receives changed paths; app reprocesses those paths directly
- event handling remains filesystem-event driven and may still vary by platform/fsnotify backend characteristics

## Resolver Heuristics

- unresolved-reference logic is heuristic, not full type/scope analysis
- method/field chains are treated by name prefix logic
- cross-package and alias resolution is best effort
- `exclude.symbols` can suppress false positives but may hide real misses if over-broad
- watch-mode unresolved analysis is incremental for affected paths; conservative impact expansion may still over-include files

## Graph Granularity

- dependency graph is module-level (not per-symbol)
- cycle detection runs across module import edges only
- unresolved reference checks rely on extracted definitions/references, not full compiler semantic analysis

## UI and Logging

- in `--ui` mode logs are redirected to a user state path (`$XDG_STATE_HOME` or `~/.local/state/circular/circular.log`) when available
- symlink log paths are refused, but fallback-to-stdout behavior can still occur if no log path is writable

## Configuration Fallbacks

- missing `./circular.toml` triggers fallback to `./circular.example.toml`
- other missing config paths fail immediately

## Audit Status

- performance/security issue classes in this codebase were AI-audited; see `docs/documentation/ai-audit.md`
- detailed evidence and severity ranking are in `docs/reviews/performance-security-review-2026-02-12.md`
