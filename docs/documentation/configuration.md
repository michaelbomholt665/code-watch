# Configuration Reference

Config is decoded from TOML in `internal/core/config`.

## Full Schema

```toml
version = 2

grammars_path = "./grammars"
watch_paths = ["."]

[grammar_verification]
enabled = true

[paths]
project_root = ""
config_dir = "data/config"
state_dir = "data/state"
cache_dir = "data/cache"
database_dir = "data/database"

[config]
active_file = "circular.toml"
includes = []

[db]
enabled = true
driver = "sqlite"
path = "history.db"
busy_timeout = "5s"
project_mode = "multi"

[projects]
active = ""
registry_file = "projects.toml"

[[projects.entries]]
name = "default"
root = "."
db_namespace = "default"

[mcp]
enabled = false
mode = "embedded"
transport = "stdio"
address = "127.0.0.1:8765"
config_path = ""

[languages]
# Optional per-language overrides.
# Defaults: go=true, python=true, others=false.

# [languages.javascript]
# enabled = true
# extensions = [".js", ".cjs", ".mjs"]
# filenames = []

[exclude]
dirs = [".git", "node_modules", "vendor"]
files = ["*.tmp", "*.log"]
# Add entries here to suppress known-safe, project-specific unresolved references.
symbols = ["self", "ctx", "p"]
# Add entries here to suppress noisy unused-import detections for known-safe imports.
imports = ["fmt", "sort", "strings"]

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
```

## Field Semantics

- `version` (`int`)
- supported values: `1`, `2`
- `paths.*` (`string`)
- centralized relative-path roots; non-absolute values resolve under detected project root
- `config.active_file` (`string`)
- active config file name under `paths.config_dir` when using layered workflows
- `config.includes` (`[]string`)
- reserved additive overlays
- `db.enabled` (`bool`)
- enables/disables history store opening even when `--history` is set
- `db.driver` (`string`)
- currently only `sqlite` is supported
- `db.path` (`string`)
- DB file path; relative values resolve under `paths.database_dir`
- `db.busy_timeout` (`duration`)
- default `5s`
- `db.project_mode` (`string`)
- `single` or `multi`
- `projects.active` (`string`)
- explicit active project name when `projects.entries` is present
- `projects.registry_file` (`string`)
- optional registry source under `paths.config_dir` (`data/config/projects.toml` by default)
- `projects.entries` (`[]table`)
- per-project `name`, `root`, and optional `db_namespace`
- `mcp.enabled` (`bool`)
- additive MCP config contract, runtime wiring still disabled
- `mcp.mode` (`string`)
- `embedded` or `server`
- `mcp.transport` (`string`)
- `stdio` or `http` (reserved)
- `mcp.address` (`string`)
- required when `mcp.transport=http`
- `mcp.config_path` (`string`)
- optional path resolved under `paths.config_dir`
- `grammars_path` (`string`)
- normalized to absolute path at runtime relative to resolved project root
- `grammar_verification.enabled` (`bool`, default `true`)
- verifies enabled language grammar artifacts against `grammars/manifest.toml` at startup
- set to `false` to skip startup checksum/AIB verification
- `languages` (`map[string]table`)
- optional per-language rollout controls
- `languages.<id>.enabled` (`bool`)
- enables/disables a language in parse/scan/watch routing
- parser extraction is profile-driven for enabled non-Go/Python languages (`javascript`, `typescript`, `tsx`, `java`, `rust`, `html`, `css`, `gomod`, `gosum`)
- resolver heuristics currently include language-specific stdlib/module policy for:
- `go`, `python`, `javascript`/`typescript`/`tsx`, `java`, `rust`
- `languages.<id>.extensions` (`[]string`)
- override extension ownership for a language
- `languages.<id>.filenames` (`[]string`)
- optional exact-file routing (for example `go.mod`, `go.sum`)
- `watch_paths` (`[]string`)
- defaults to `["."]`
- `exclude.symbols` (`[]string`)
- symbol names to suppress unresolved reference findings for known-safe locals or framework names
- `exclude.imports` (`[]string`)
- import module paths or reference base names to suppress unused-import findings
- keep lists minimal and prefer project-specific overrides when embedding MCP configs
- `watch.debounce` (`duration`)
- defaults to `500ms`
- `output.*`, `alerts.*`, `architecture.*`
- unchanged semantics from prior versions

## Defaults and Discovery

- CLI default config path is `./data/config/circular.toml`
- default discovery order:
  - `./data/config/circular.toml`
  - `./circular.toml` (legacy fallback, emits deprecation warning)
  - `./data/config/circular.example.toml`
  - `./circular.example.toml`
- custom `--config <path>` is strict (no fallback chain)
- default DB path resolves to `data/database/history.db`

## Validation Rules

Config load fails when:
- `version` is outside supported range
- `db.driver != sqlite`
- `db.project_mode` is not `single|multi`
- `projects.active` references a missing project
- duplicate `projects.entries.name` values exist
- MCP mode/transport combinations are invalid
- output markdown targets are malformed
- architecture rules violate layer/rule constraints
- `languages.*.extensions` or `languages.*.filenames` include empty values

## Migration Notes

- Existing v1-style configs still load without immediate edits.
- Root-level `./circular.toml` remains supported as a deprecated fallback during transition.
- History storage now defaults to `data/database/history.db`; legacy `.circular/history.db` is no longer the default.
