# Configuration Reference

Config is decoded from TOML in `internal/config`.

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
mermaid = "graph.mmd"
plantuml = "graph.puml"

[output.paths]
root = ""
diagrams_dir = "docs/diagrams"

[[output.update_markdown]]
file = "README.md"
marker = "deps-mermaid"
format = "mermaid"

[[output.update_markdown]]
file = "README.md"
marker = "deps-plantuml"
format = "plantuml"

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

## Field Semantics

- `grammars_path` (`string`)
- parser grammars are compiled into the binary
- this path is still accepted by config and normalized to absolute path at runtime
- startup fails only if the path exists and is not a directory
- `watch_paths` (`[]string`)
- root directories scanned recursively and watched in watch mode
- defaults to `["."]` when empty or omitted
- `exclude.dirs` (`[]string`)
- glob patterns matched against directory base name
- applied during initial scan and recursive watcher registration
- invalid glob patterns fail startup
- `exclude.files` (`[]string`)
- glob patterns matched against file base name
- applied during scan and watch events
- invalid glob patterns fail startup
- `exclude.symbols` (`[]string`)
- prefix-style exclusions for unresolved-reference checks
- `watch.debounce` (`duration`)
- debounce window for batching fsnotify events
- defaults to `500ms` when unset
- `output.dot` (`string`)
- DOT output file path; empty disables DOT generation
- `output.tsv` (`string`)
- TSV output file path; empty disables TSV generation
- `output.mermaid` (`string`)
- Mermaid output file path; empty disables Mermaid file generation
- when value is filename-only (for example `graph.mmd`), it resolves under `<output.paths.root>/<output.paths.diagrams_dir>/`
- `output.plantuml` (`string`)
- PlantUML output file path; empty disables PlantUML file generation
- when value is filename-only (for example `graph.puml`), it resolves under `<output.paths.root>/<output.paths.diagrams_dir>/`
- `output.paths.root` (`string`)
- optional root override for relative output paths
- when empty, root is auto-detected by walking up from `watch_paths`/cwd and selecting the first directory containing `go.mod`, `.git`, or `circular.toml`
- `output.paths.diagrams_dir` (`string`)
- default `docs/diagrams`; base directory for filename-only Mermaid/PlantUML paths
- `output.update_markdown` (`[]table`)
- optional marker-based markdown updates; each table requires:
- `file` target markdown file path
- `marker` marker name used with `<!-- circular:<marker>:start -->` and `<!-- circular:<marker>:end -->`
- `format` one of `mermaid` or `plantuml`
- `alerts.beep` (`bool`)
- emits terminal bell on updates containing issues (cycles, unresolved, unused imports, architecture violations)
- `alerts.terminal` (`bool`)
- enables summary printing to terminal
- `architecture.enabled` (`bool`)
- enables layer-rule validation
- `architecture.top_complexity` (`int`)
- number of hotspots used in output/summary
- coerced to `5` when unset or `<= 0`
- `architecture.layers` (`[]table`)
- layer name + one or more path patterns
- `architecture.rules` (`[]table`)
- exactly one rule per `from` layer
- `allow` must contain at least one target layer

## Defaults and Fallbacks

- missing `./circular.toml` triggers fallback load attempt of `./circular.example.toml`
- custom `--config` paths do not fallback
- empty `watch_paths` becomes `["."]`
- zero `watch.debounce` becomes `500ms`
- `architecture.top_complexity <= 0` becomes `5`
- unset Mermaid/PlantUML/markdown output keys keep existing DOT/TSV-only behavior unchanged
- empty `output.paths.diagrams_dir` becomes `docs/diagrams`

## Architecture Validation

When `architecture.enabled=true`, config loading fails if:
- no layers are defined
- layer names are duplicated
- layer path patterns are duplicated across layers
- layer path patterns overlap (literal/literal, wildcard/literal, or wildcard/wildcard overlap checks)
- rule names are duplicated
- multiple rules are defined for one `from` layer
- rules reference unknown layers
- a rule has an empty `allow` list
