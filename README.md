# circular

`circular` is a Go-based dependency monitor for Go and Python codebases.

It scans source files, builds a module import graph, then reports:
- circular imports
- unresolved symbol references ("hallucinations")
- unused imports
- dependency depth/fan-in/fan-out metrics
- architecture layer-rule violations
- change impact (direct + transitive importers)
- complexity hotspots

It can run once (`--once`) or watch continuously with optional terminal UI mode (`--ui`).

## Features

- Parses `.go` and `.py` files with Tree-sitter
- Builds and updates a module-level dependency graph
- Detects import cycles across internal modules
- Detects unresolved references using local symbols, imports, stdlib, and builtins
- Detects unused imports and appends findings to TSV output
- Computes module dependency metrics (depth, fan-in, fan-out)
- Traces shortest import chain between modules (`--trace`)
- Analyzes blast radius for a file/module (`--impact`)
- Validates optional architecture layer rules
- Reports top complexity hotspots from parser heuristics
- Emits outputs:
  - Graphviz DOT (`graph.dot` by default)
  - TSV edge list (`dependencies.tsv` by default)
  - Mermaid (`graph.mmd`, optional)
  - PlantUML (`graph.puml`, optional)
  - Marker-based Markdown diagram injection (optional)
- Live filesystem watch mode with debounce
- Optional Bubble Tea terminal UI for live issue monitoring

## Runtime Modes

- default (watch mode): run initial scan, print summary, write outputs, then watch for changes
- `--once`: run initial scan + analysis once, then exit
- `--trace <from-module> <to-module>`: print shortest internal import chain, then exit
- `--impact <file-or-module>`: print direct/transitive importer impact, then exit
- `--ui`: run watch mode with Bubble Tea UI

## Install / Build

Requirements:
- Go `1.24+` supported
- Go `1.23` is not supported by current dependencies
- `go.mod` pins `go 1.24`

Build:

```bash
go build -o circular ./cmd/circular
```

Run tests:

```bash
go test ./...
```

## Versioning

This project follows Semantic Versioning: `MAJOR.MINOR.PATCH`.

- `PATCH`: backward-compatible bug fixes and internal improvements with no public contract change.
- `MINOR`: backward-compatible features (for example new CLI flags, additive config fields, additive output fields).
- `MAJOR`: breaking changes to CLI behavior/flags, config schema, or output formats that downstream tooling relies on.

Before a version bump/release:
- Update root `CHANGELOG.md` with user-facing changes.
- Keep docs in `docs/documentation/` aligned with released behavior.

## Quick Start

1. Copy example config:

```bash
cp circular.example.toml circular.toml
```

2. Update `watch_paths` in `circular.toml` to your project path.

3. Run a one-time scan:

```bash
go run ./cmd/circular --once
```

4. Run in watch mode:

```bash
go run ./cmd/circular
```

5. Run in UI mode:

```bash
go run ./cmd/circular --ui
```

## CLI

Flags:
- `--config` path to TOML config (default `./circular.toml`)
- if default config is missing, it falls back to `./circular.example.toml`
- `--once` run one scan and exit
- `--ui` start terminal UI mode
- `--trace` print shortest import chain from one module to another, then exit
- `--impact` analyze direct/transitive impact for a file path or module, then exit
- `--verbose` enable debug logs
- `--version` print version and exit

Positional arg:
- first positional argument overrides `watch_paths` with a single path
- in trace mode, exactly two positional arguments are required (`<from> <to>`)

Version in source: `1.0.0` (`internal/cliapp/cli.go`).

## Configuration

Primary config file is TOML.

Example (`circular.example.toml`):

```toml
grammars_path = "./grammars"
watch_paths = ["./src"]

[exclude]
dirs = [".git", "node_modules", "vendor", "__pycache__"]
files = ["*.tmp", "*.log"]
symbols = ["self", "ctx", "p", "log", "toml", "sitter", "tea", "fsnotify"]

[watch]
debounce = "1s"

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
beep = true
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

[[architecture.rules]]
name = "core-self-only"
from = "core"
allow = ["core"]
```

## Outputs

- DOT graph (default `graph.dot`) for visual inspection in Graphviz tools
- TSV import edges (default `dependencies.tsv`) with columns:
  - `From`, `To`, `File`, `Line`, `Column`
- Mermaid graph (optional `graph.mmd`) using `flowchart LR`
- PlantUML graph (optional `graph.puml`) using component/package view
- Optional markdown diagram injection via `[[output.update_markdown]]` markers:
  - `<!-- circular:<marker>:start -->`
  - `<!-- circular:<marker>:end -->`
- TSV unused import rows appended when findings exist:
  - `Type`, `File`, `Language`, `Module`, `Alias`, `Item`, `Line`, `Column`, `Confidence`
- TSV architecture violation rows appended when findings exist:
  - `Type`, `Rule`, `FromModule`, `FromLayer`, `ToModule`, `ToLayer`, `File`, `Line`, `Column`
- row ordering in DOT/TSV is map-iteration based and not guaranteed stable
- DOT module labels may include metrics:
  - `d=<depth> in=<fan-in> out=<fan-out>`
  - `cx=<top-complexity-score-in-module>`

## Dependency Diagrams

Mermaid:

<!-- circular:deps-mermaid:start -->
```mermaid
%%{init: {'theme': 'base', 'themeVariables': {'textColor': '#000000', 'primaryTextColor': '#000000', 'lineColor': '#333333'}, 'flowchart': {'nodeSpacing': 80, 'rankSpacing': 110, 'curve': 'basis'}}}%%
flowchart LR
  circular_cmd_circular["circular/cmd/circular\n(0 funcs, 1 files)\n(d=6 in=0 out=1)"]
  circular_internal_app["circular/internal/app\n(34 funcs, 3 files)\n(d=4 in=1 out=8)"]
  circular_internal_cliapp["circular/internal/cliapp\n(20 funcs, 8 files)\n(d=5 in=1 out=8)"]
  circular_internal_config["circular/internal/config\n(18 funcs, 2 files)\n(d=0 in=2 out=0)\n(cx=101)"]
  circular_internal_graph["circular/internal/graph\n(49 funcs, 6 files)\n(d=1 in=5 out=1)"]
  circular_internal_history["circular/internal/history\n(21 funcs, 7 files)\n(d=0 in=4 out=0)"]
  circular_internal_output["circular/internal/output\n(32 funcs, 8 files)\n(d=3 in=2 out=4)\n(cx=123)"]
  circular_internal_parser["circular/internal/parser\n(21 funcs, 7 files)\n(d=0 in=6 out=0)\n(cx=80)"]
  circular_internal_query["circular/internal/query\n(14 funcs, 3 files)\n(d=2 in=2 out=3)"]
  circular_internal_resolver["circular/internal/resolver\n(25 funcs, 6 files)\n(d=2 in=3 out=2)"]
  circular_internal_watcher["circular/internal/watcher\n(7 funcs, 2 files)\n(d=0 in=1 out=0)"]
  __external_aggregate__["External/Stdlib\n(31 modules)"]

  classDef internalNode fill:#f7fbff,stroke:#4d6480,stroke-width:1px,color:#000000;
  class circular_cmd_circular,circular_internal_app,circular_internal_cliapp,circular_internal_config,circular_internal_graph,circular_internal_history,circular_internal_output,circular_internal_parser,circular_internal_query,circular_internal_resolver,circular_internal_watcher internalNode;
  classDef externalNode fill:#efefef,stroke:#808080,stroke-dasharray:4 3,color:#000000;
  class __external_aggregate__ externalNode;
  classDef hotspotNode stroke:#8a4f00,stroke-width:2px,color:#000000;
  class circular_internal_config,circular_internal_output,circular_internal_parser hotspotNode;

  circular_cmd_circular --> circular_internal_cliapp
  circular_internal_app --> circular_internal_config
  circular_internal_app --> circular_internal_graph
  circular_internal_app --> circular_internal_history
  circular_internal_app --> circular_internal_output
  circular_internal_app --> circular_internal_parser
  circular_internal_app --> circular_internal_query
  circular_internal_app --> circular_internal_resolver
  circular_internal_app --> circular_internal_watcher
  circular_internal_cliapp --> circular_internal_app
  circular_internal_cliapp --> circular_internal_config
  circular_internal_cliapp --> circular_internal_graph
  circular_internal_cliapp --> circular_internal_history
  circular_internal_cliapp --> circular_internal_output
  circular_internal_cliapp --> circular_internal_parser
  circular_internal_cliapp --> circular_internal_query
  circular_internal_cliapp --> circular_internal_resolver
  circular_internal_graph --> circular_internal_parser
  circular_internal_output --> circular_internal_graph
  circular_internal_output --> circular_internal_history
  circular_internal_output --> circular_internal_parser
  circular_internal_output --> circular_internal_resolver
  circular_internal_query --> circular_internal_graph
  circular_internal_query --> circular_internal_history
  circular_internal_query --> circular_internal_parser
  circular_internal_resolver --> circular_internal_graph
  circular_internal_resolver --> circular_internal_parser
  circular_cmd_circular -->|ext:1| __external_aggregate__
  circular_internal_app -->|ext:12| __external_aggregate__
  circular_internal_cliapp -->|ext:13| __external_aggregate__
  circular_internal_config -->|ext:7| __external_aggregate__
  circular_internal_graph -->|ext:8| __external_aggregate__
  circular_internal_history -->|ext:13| __external_aggregate__
  circular_internal_output -->|ext:9| __external_aggregate__
  circular_internal_parser -->|ext:11| __external_aggregate__
  circular_internal_query -->|ext:6| __external_aggregate__
  circular_internal_resolver -->|ext:7| __external_aggregate__
  circular_internal_watcher -->|ext:10| __external_aggregate__

  linkStyle 27,28,29,30,31,32,33,34,35,36,37 stroke:#777777,stroke-dasharray:4 3;

  subgraph legend_info["Legend"]
    legend_metrics["Node line 1: module\nline 2: funcs/files\n(d=depth in=fan-in out=fan-out)\n(cx=complexity hotspot score)"]
    legend_edges["Edge labels: CYCLE=import cycle, VIOLATION=architecture rule violation, ext:N=external dependency count"]
  end
  classDef legendNode fill:#fff8dc,stroke:#b8a24c,stroke-width:1px,color:#000000;
  class legend_metrics,legend_edges legendNode;
```
<!-- circular:deps-mermaid:end -->

PlantUML:

<!-- circular:deps-plantuml:start -->
```plantuml
@startuml
skinparam componentStyle rectangle
skinparam packageStyle rectangle
skinparam linetype ortho
skinparam nodesep 80
skinparam ranksep 100
left to right direction

component "circular/cmd/circular\n(0 funcs, 1 files)\n(d=6 in=0 out=1)" as circular_cmd_circular
component "circular/internal/app\n(34 funcs, 3 files)\n(d=4 in=1 out=8)" as circular_internal_app
component "circular/internal/cliapp\n(20 funcs, 8 files)\n(d=5 in=1 out=8)" as circular_internal_cliapp
component "circular/internal/config\n(18 funcs, 2 files)\n(d=0 in=2 out=0)\n(cx=101)" as circular_internal_config
component "circular/internal/graph\n(49 funcs, 6 files)\n(d=1 in=5 out=1)" as circular_internal_graph
component "circular/internal/history\n(21 funcs, 7 files)\n(d=0 in=4 out=0)" as circular_internal_history
component "circular/internal/output\n(32 funcs, 8 files)\n(d=3 in=2 out=4)\n(cx=123)" as circular_internal_output
component "circular/internal/parser\n(21 funcs, 7 files)\n(d=0 in=6 out=0)\n(cx=80)" as circular_internal_parser
component "circular/internal/query\n(14 funcs, 3 files)\n(d=2 in=2 out=3)" as circular_internal_query
component "circular/internal/resolver\n(25 funcs, 6 files)\n(d=2 in=3 out=2)" as circular_internal_resolver
component "circular/internal/watcher\n(7 funcs, 2 files)\n(d=0 in=1 out=0)" as circular_internal_watcher
component "External/Stdlib\n(31 modules)" as __external_aggregate__ #DDDDDD

circular_cmd_circular --> circular_internal_cliapp
circular_internal_app --> circular_internal_config
circular_internal_app --> circular_internal_graph
circular_internal_app --> circular_internal_history
circular_internal_app --> circular_internal_output
circular_internal_app --> circular_internal_parser
circular_internal_app --> circular_internal_query
circular_internal_app --> circular_internal_resolver
circular_internal_app --> circular_internal_watcher
circular_internal_cliapp --> circular_internal_app
circular_internal_cliapp --> circular_internal_config
circular_internal_cliapp --> circular_internal_graph
circular_internal_cliapp --> circular_internal_history
circular_internal_cliapp --> circular_internal_output
circular_internal_cliapp --> circular_internal_parser
circular_internal_cliapp --> circular_internal_query
circular_internal_cliapp --> circular_internal_resolver
circular_internal_graph --> circular_internal_parser
circular_internal_output --> circular_internal_graph
circular_internal_output --> circular_internal_history
circular_internal_output --> circular_internal_parser
circular_internal_output --> circular_internal_resolver
circular_internal_query --> circular_internal_graph
circular_internal_query --> circular_internal_history
circular_internal_query --> circular_internal_parser
circular_internal_resolver --> circular_internal_graph
circular_internal_resolver --> circular_internal_parser
circular_cmd_circular -[#777777,dashed]-> __external_aggregate__ : ext:1
circular_internal_app -[#777777,dashed]-> __external_aggregate__ : ext:12
circular_internal_cliapp -[#777777,dashed]-> __external_aggregate__ : ext:13
circular_internal_config -[#777777,dashed]-> __external_aggregate__ : ext:7
circular_internal_graph -[#777777,dashed]-> __external_aggregate__ : ext:8
circular_internal_history -[#777777,dashed]-> __external_aggregate__ : ext:13
circular_internal_output -[#777777,dashed]-> __external_aggregate__ : ext:9
circular_internal_parser -[#777777,dashed]-> __external_aggregate__ : ext:11
circular_internal_query -[#777777,dashed]-> __external_aggregate__ : ext:6
circular_internal_resolver -[#777777,dashed]-> __external_aggregate__ : ext:7
circular_internal_watcher -[#777777,dashed]-> __external_aggregate__ : ext:10

legend right
|= Item |= Meaning |
|Node line 1|Module name|
|Node line 2|Function/export count and file count|
|d|Dependency depth|
|in|Fan-in (number of internal modules importing this module)|
|out|Fan-out (number of internal modules this module imports)|
|cx|Top complexity hotspot score in the module|
|<color:#DDDDDD>Component</color>|External module|
|ext:N|Count of external dependencies from that module (aggregated mode)|
endlegend

@enduml
```
<!-- circular:deps-plantuml:end -->

## Documentation

Full documentation is in `docs/documentation/`:
- `docs/documentation/README.md`
- `docs/documentation/cli.md`
- `docs/documentation/configuration.md`
- `docs/documentation/output.md`
- `docs/documentation/architecture.md`
- `docs/documentation/packages.md`
- `docs/documentation/limitations.md`
- `docs/documentation/ai-audit.md`

AI audit reports are in `docs/reviews/`:
- `docs/reviews/performance-security-review-2026-02-12.md`

## Project Layout

- `cmd/circular/` CLI app, orchestration, TUI
- `internal/config/` TOML config loading
- `internal/parser/` Tree-sitter parsing and extractors
- `internal/graph/` dependency graph + cycle detection
- `internal/resolver/` unresolved reference detection
- `internal/watcher/` fsnotify watch + debounce
- `internal/output/` DOT/TSV/Mermaid/PlantUML generators + markdown injection
