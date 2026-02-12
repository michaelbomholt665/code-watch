# Configuration Reference

Config is loaded from TOML by `internal/config`.

## Full Schema

```toml
grammars_path = "./grammars"
watch_paths = ["."]

[exclude]
dirs = [".git", "node_modules", "vendor"]
files = ["*.tmp", "*.log"]
symbols = ["self", "ctx", "p"]

[watch]
debounce = "500ms"

[output]
dot = "graph.dot"
tsv = "dependencies.tsv"

[alerts]
beep = false
terminal = true
```

## Field Details

- `grammars_path` (`string`)
- path configured for grammar assets
- parser uses compiled Go bindings for grammars; this path is validated as a directory when provided, but grammars are not dynamically loaded from it
- `watch_paths` (`[]string`)
- root directories to recursively scan and watch
- defaults to `['.']` when omitted
- `exclude.dirs` (`[]string`)
- glob patterns (matched against directory base name) skipped during scan and watch registration
- `exclude.files` (`[]string`)
- glob patterns (matched against file base name) skipped during scan and watch events
- invalid exclude glob patterns fail startup/scan initialization
- `exclude.symbols` (`[]string`)
- reference prefixes ignored by unresolved-symbol resolver
- useful for local context aliases (for example `ctx`, `self`, logger vars)
- `watch.debounce` (`duration`)
- debounce window for grouped file change handling
- defaults to `500ms`
- `output.dot` (`string`)
- path for Graphviz DOT output
- empty string disables DOT emission
- `output.tsv` (`string`)
- path for TSV edge list output
- empty string disables TSV emission
- `alerts.beep` (`bool`)
- emits terminal bell when cycles or unresolved references are present in update
- `alerts.terminal` (`bool`)
- enables/disables printed summary output

## Defaults and Fallbacks

- missing config path causes startup failure, except default path behavior:
- if `./circular.toml` is missing, app attempts `./circular.example.toml`
- if `watch_paths` is empty, it becomes `['.']`
- if `watch.debounce` is zero, it becomes `500ms`
