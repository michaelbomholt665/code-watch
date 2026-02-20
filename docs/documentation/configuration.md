# Configuration Reference

Config is decoded from TOML in `internal/core/config`.
Loading/defaulting lives in `internal/core/config/loader.go`, and validation rules live in `internal/core/config/validator.go` (with shared helpers in `internal/core/config/helpers/validators.go`).

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
config_file = "circular.toml"

[mcp]
enabled = false
mode = "embedded"
transport = "stdio"
address = "127.0.0.1:8765"
config_path = "circular.toml"
server_name = "circular"
server_version = "1.0.0"
exposed_tool_name = ""
operation_allowlist = ["scan.run", "secrets.scan", "secrets.list", "graph.cycles", "graph.sync_diagrams", "query.modules", "query.module_details", "query.trace", "system.sync_config", "system.generate_config", "system.generate_script", "system.select_project", "system.watch", "query.trends", "report.generate_markdown"]
max_response_items = 500
request_timeout = "30s"
allow_mutations = false
auto_manage_outputs = true
auto_sync_config = true

[languages]
# Optional per-language overrides.
# Defaults: go=true, python=true, others=false.

# [languages.javascript]
# enabled = true
# extensions = [".js", ".cjs", ".mjs"]
# filenames = []

[[dynamic_grammars]]
# Example: Adding Kotlin support via shared object grammar
# name = "kotlin"
# library = "./grammars/kotlin/kotlin.so"
# extensions = [".kt"]
# namespace_node = "package_header"
# import_node = "import_header"
# definition_nodes = ["class_declaration", "function_declaration"]

[exclude]
dirs = [".git", "node_modules", "vendor"]
files = ["*.tmp", "*.log"]
# Add entries here to suppress known-safe, project-specific unresolved references.
symbols = ["self", "ctx", "p"]
# Add entries here to suppress noisy unused-import detections for known-safe imports.
imports = ["fmt", "sort", "strings"]

[watch]
debounce = "500ms"

[secrets]
enabled = false
entropy_threshold = 4.0
min_token_length = 20
# scan_history = 0   # Set to N to scan the last N git commits for deleted secrets

[[secrets.patterns]]
name = "custom-token"
regex = "CTK_[A-Za-z0-9]{20}"
severity = "medium"

[secrets.exclude]
dirs = []
files = []

[output]
dot = "graph.dot"
tsv = "dependencies.tsv"
mermaid = "graph.mmd"
# plantuml = "graph.puml"
markdown = "analysis-report.md"
# sarif = "results/circular.sarif.json"

[output.formats]
mermaid = true
plantuml = false

[output.report]
verbosity = "standard"
table_of_contents = true
collapsible_sections = true
include_mermaid = false

[output.diagrams]
architecture = false
component = false
flow = false

[output.diagrams.flow_config]
entry_points = ["cmd/circular/main.go"]
max_depth = 8

[output.diagrams.component_config]
show_internal = false

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
- this SQLite file is shared by history snapshots and the resolver `symbols` index
- `db.busy_timeout` (`duration`)
- default `5s`
- `db.project_mode` (`string`)
- `single` or `multi`
- `projects.active` (`string`)
- explicit active project name when `projects.entries` is present
- `projects.registry_file` (`string`)
- optional registry source under `paths.config_dir` (`data/config/projects.toml` by default)
- `projects.entries` (`[]table`)
- per-project `name`, `root`, `db_namespace`, and optional `config_file`
- `db_namespace` must be unique and non-empty after trimming; it scopes SQLite history isolation
- `config_file` resolves under `paths.config_dir` when set and is used for MCP config sync
- `mcp.enabled` (`bool`)
- enables MCP runtime startup and single-tool operation dispatch
- `mcp.mode` (`string`)
- `embedded` or `server`
- `mcp.transport` (`string`)
- `stdio` or `http` (reserved)
- `mcp.address` (`string`)
- required when `mcp.transport=http`
- `mcp.config_path` (`string`)
- config file path resolved under `paths.config_dir` for MCP auto-sync
- defaults to `config.active_file` when empty
- `mcp.openapi_spec_path` (`string`)
- optional OpenAPI spec file path; resolved under `paths.config_dir`
- `mcp.openapi_spec_url` (`string`)
- optional OpenAPI spec URL (http/https); cannot be set with `mcp.openapi_spec_path`
- `mcp.server_name` (`string`)
- required when `mcp.enabled=true`
- `mcp.server_version` (`string`)
- required when `mcp.enabled=true`
- `mcp.exposed_tool_name` (`string`)
- optional single-tool exposure; must not contain whitespace
- `mcp.operation_allowlist` (`[]string`)
- explicit operation allowlist for MCP exposure (examples: `scan.run`, `secrets.scan`, `secrets.list`, `graph.cycles`, `graph.sync_diagrams`, `system.generate_config`, `system.generate_script`, `system.watch`, `report.generate_markdown`)
- required when `mcp.enabled=true` if `mcp.exposed_tool_name` is empty
- legacy aliases accepted at runtime: `scan_once`, `detect_cycles`, `trace_import_chain`, `generate_reports`, `system.sync_outputs`
- `mcp.max_response_items` (`int`)
- maximum list payload items (default `500`)
- `mcp.request_timeout` (`duration`)
- bounds: `1s` to `2m` (default `30s`)
- `mcp.allow_mutations` (`bool`)
- gate for mutation-enabled MCP tools (default `false`)
- `mcp.auto_manage_outputs` (`bool`)
- when `true`, MCP startup auto-writes configured outputs
- `mcp.auto_sync_config` (`bool`)
- when `true`, MCP startup ensures project-local bootstrap artifacts exist:
- `circular.toml` generated from `data/config/circular.example.toml` if missing
- `circular-mcp` helper script generated if missing
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
- resolver also applies a graph-derived universal symbol table and probabilistic second-pass matching for cross-language unresolved-reference reduction
- when `db.enabled=true`, resolver symbol lookups are backed by a persisted SQLite `symbols` table with in-memory fallback on DB errors
- service bridge references (`service_bridge`) are matched against normalized service keys to link likely contract definitions across files/languages
- resolver can also load optional explicit bridge mappings from `.circular-bridge.toml` in watch roots and apply them as a deterministic pre-probabilistic pass
- `resolver.bridge_scoring.confirmed_threshold` (`int`)
- minimum bridge score required to auto-resolve a bridge-like reference (default `8`)
- `resolver.bridge_scoring.probable_threshold` (`int`)
- minimum bridge score required to classify as probable bridge reference (default `5`)
- must be `<= confirmed_threshold`
- `resolver.bridge_scoring.weight_*` (`int`)
- bridge scoring weights for explicit rules, context, import evidence, candidate ambiguity, and conflict penalties
- defaults are provided in `data/config/circular.example.toml`
- `languages.<id>.extensions` (`[]string`)
- override extension ownership for a language
- `languages.<id>.filenames` (`[]string`)
- optional exact-file routing (for example `go.mod`, `go.sum`)
- `dynamic_grammars` (`[]table`)
- runtime Tree-sitter grammar loading via `dlopen`
- `name`: required unique language identifier
- `library`: required path to `.so` (Unix) or `.dll` (Windows) grammar file
- `extensions`: optional list of file extensions
- `filenames`: optional list of exact filenames
- `namespace_node`: required AST node kind for package/namespace extraction
- `import_node`: required AST node kind for import extraction
- `definition_nodes`: required list of AST node kinds for symbol definition extraction
- `watch_paths` (`[]string`)
- defaults to `["."]`
- `exclude.symbols` (`[]string`)
- symbol names to suppress unresolved reference findings for known-safe locals or framework names
- `exclude.imports` (`[]string`)
- import module paths or reference base names to suppress unused-import findings
- keep lists minimal and prefer project-specific overrides when embedding MCP configs
- `watch.debounce` (`duration`)
- defaults to `500ms`
- `secrets.enabled` (`bool`)
- enables/disables hardcoded secret scanning during `ProcessFile`
- watch-mode scans use changed-line incremental detection when possible; full-scan fallback is used when file line counts shift
- `secrets.entropy_threshold` (`float`)
- Shannon entropy threshold used by high-entropy token checks
- entropy checks are applied only to high-risk extension set: `.env`, `.json`, `.key`, `.pem`, `.p12`, `.pfx`, `.crt`, `.cer`, `.yaml`, `.yml`, `.toml`, `.ini`, `.conf`, `.properties`
- bounds: `1.0` to `8.0` (default `4.0`)
- `secrets.min_token_length` (`int`)
- minimum token length for entropy/context candidate strings
- bounds: `8` to `256` (default `20`)
- `secrets.patterns` (`[]table`)
- custom regex rules with `name`, `regex`, optional `severity`
- regex values are compile-validated at startup
- `secrets.exclude.dirs` (`[]string`)
- directory basename globs skipped by secret scanning
- `secrets.exclude.files` (`[]string`)
- file basename globs skipped by secret scanning
- `secrets.scan_history` (`int`, default `0`)
  - when `> 0`, scans the last N git commits for secrets that were added and then deleted
  - requires `git` binary in PATH; silently skipped if git is unavailable
  - can also be set at runtime with `--scan-history N` CLI flag
  - findings are reported to stdout with synthetic paths `git:history:<short-commit>:<file>`
  - capped at `1000` commits to protect large repositories
- diagram / report output controls currently supported in schema:
- `output.dot`, `output.tsv`, `output.mermaid`, `output.plantuml`, `output.markdown`, `output.sarif`
- `output.formats.mermaid`, `output.formats.plantuml`
- `output.report.verbosity`, `output.report.table_of_contents`, `output.report.collapsible_sections`, `output.report.include_mermaid`
- `output.diagrams.architecture`, `output.diagrams.component`, `output.diagrams.flow`
- `output.diagrams.flow_config.entry_points`, `output.diagrams.flow_config.max_depth`
- `output.diagrams.component_config.show_internal`
- `output.paths.*`, `output.update_markdown`
- current wiring:
- `output.diagrams.architecture=true` enables dedicated architecture diagrams (layer-level)
- `output.diagrams.component=true` enables component diagrams (module internals + symbol-reference overlays)
- `output.diagrams.flow=true` enables bounded flow diagrams rooted at configured entry points
- multiple diagram modes may be enabled together
- when multiple modes are enabled, output file names are mode-suffixed (`-dependency`, `-architecture`, `-component`, `-flow`)
- Mermaid is enabled by default; PlantUML is disabled by default unless `output.formats.plantuml=true`
- `output.diagrams.component_config.show_internal=true` includes definition-level symbol nodes
- `output.diagrams.flow_config.max_depth` limits traversal depth from entry points
- for `output.mermaid` and `output.plantuml`:
- filename-only values (for example `graph.mmd`) resolve under `output.paths.diagrams_dir`
- values containing `/` or `\` resolve under output root (`output.paths.root` or detected project root)
- `output.*`, `alerts.*`, `architecture.*`
- unchanged semantics from prior versions

## Optional Bridge Mapping File

Resolver overrides can be declared in `.circular-bridge.toml` at any configured watch root.

```toml
[[bridges]]
from = "go:internal/mcp/runtime"
to = "python:circular_mcp.server"
reason = "JSON-RPC over stdio"
references = ["circular_mcp.server.*"]
```

Fields:
- `bridges[].from`: required `<language>:<module>` source endpoint
- `bridges[].to`: required `<language>:<module>` destination endpoint
- `bridges[].reason`: optional rationale
- `bridges[].references`: optional reference patterns (`*` wildcard supported)

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
- duplicate `projects.entries.db_namespace` values exist or `db_namespace` is empty
- MCP mode/transport combinations are invalid
- MCP server metadata is missing when enabled
- MCP tool exposure configuration is missing or duplicated
- resolver bridge scoring thresholds are invalid (`probable_threshold > confirmed_threshold` or non-positive values)
- MCP OpenAPI spec path and URL are both set
- MCP response/timeout limits exceed bounds
- output markdown targets are malformed
- `output.report.verbosity` is not `summary|standard|detailed`
- `output.diagrams.flow_config.max_depth < 1`
- `output.diagrams.flow_config.entry_points` contains empty or duplicate values
- architecture rules violate layer/rule constraints
- `languages.*.extensions` or `languages.*.filenames` include empty values

## Migration Notes

- Existing v1-style configs still load without immediate edits.
- Root-level `./circular.toml` remains supported as a deprecated fallback during transition.
- History storage now defaults to `data/database/history.db`; legacy `.circular/history.db` is no longer the default.
