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

[architecture]
enabled = false
top_complexity = 5

[[architecture.layers]]
name = "api"
paths = ["internal/api", "cmd"]

[[architecture.layers]]
name = "core"
paths = ["internal/core", "internal/graph", "internal/parser", "internal/resolver"]

[[architecture.rules]]
name = "api-to-core-only"
from = "api"
allow = ["core"]
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
- when unused imports are detected, the file appends a second TSV block with `Type=unused_import` rows
- `alerts.beep` (`bool`)
- emits terminal bell when cycles, unresolved references, or unused imports are present in update
- `alerts.terminal` (`bool`)
- enables/disables printed summary output
- `architecture.enabled` (`bool`)
- enables architecture layer-rule validation (default disabled)
- `architecture.top_complexity` (`int`)
- number of hotspots printed/exported by complexity ranking (default `5` when unset or `<=0`)
- `architecture.layers` (`[]table`)
- declared named layers and the path/module patterns assigned to each layer
- each entry requires:
- `name` (`string`): unique layer name
- `paths` (`[]string`): one or more literal prefixes or glob patterns
- `architecture.rules` (`[]table`)
- dependency policy per source layer
- each entry requires:
- `name` (`string`): unique rule name
- `from` (`string`): source layer name
- `allow` (`[]string`): allowed target layers for imports originating from `from`

## Defaults and Fallbacks

- missing config path causes startup failure, except default path behavior:
- if `./circular.toml` is missing, app attempts `./circular.example.toml`
- if `watch_paths` is empty, it becomes `['.']`
- if `watch.debounce` is zero, it becomes `500ms`
- if `architecture.top_complexity` is zero/unset, it becomes `5`

## Architecture Validation Rules

When `architecture.enabled=true`, config loading fails fast if:
- no layers are defined
- a layer name is duplicated
- a layer path pattern is duplicated across layers
- literal layer paths overlap (for example `internal` and `internal/api`)
- a rule name is duplicated
- a source layer has multiple rules
- a rule references unknown layers
- a rule has an empty `allow` list
