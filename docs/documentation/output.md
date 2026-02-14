# Output Reference

`circular` can emit DOT, TSV, Mermaid, and PlantUML outputs via `internal/ui/report`.

## Diagram Scope

Mermaid and PlantUML support two output views:
- default dependency view (module-level graph)
- architecture view (layer-level graph) when `output.diagrams.architecture=true`

Implemented overlays:
- cycle highlighting
- architecture-violation highlighting
- optional architecture-layer grouping when `[architecture].enabled=true`

Planned (not yet implemented):
- dedicated component diagram mode
- dedicated flow/call diagram mode

Roadmap source:
- `docs/plans/diagram-expansion-plan.md`

## `dependencies.tsv`

Base dependency block header:

```text
From\tTo\tFile\tLine\tColumn
```

Each row is one import edge:
- `From`: source module
- `To`: imported module
- `File`: source file that contributed the edge
- `Line`, `Column`: import location

## Appended Unused-Import Block

Appended only when findings exist, separated by a blank line.

Header:

```text
Type\tFile\tLanguage\tModule\tAlias\tItem\tLine\tColumn\tConfidence
```

Row prefix is always:

```text
unused_import
```

`Confidence` values currently emitted:
- `high` for item imports (`from x import y`)
- `medium` for module-level alias/name heuristics

## Appended Architecture-Violation Block

Appended only when findings exist, separated by a blank line.

Header:

```text
Type\tRule\tFromModule\tFromLayer\tToModule\tToLayer\tFile\tLine\tColumn
```

Row prefix is always:

```text
architecture_violation
```

## `graph.dot`

DOT graph properties:
- left-to-right layout (`rankdir=LR`)
- internal modules grouped in `cluster_internal`
- external/stdlib modules rendered separately
- internal internal edges: green
- edges to external modules: dashed gray
- cycle edges: red with `label="CYCLE"`

Node labels include:
- module name
- function/export count and file count
- optional metrics annotation: `(d=<depth> in=<fan-in> out=<fan-out>)`
- optional complexity annotation: `(cx=<module-max-hotspot-score>)`

Complexity interpretation guideline:
- `cx >= 80` usually indicates high complexity worth review
- `cx >= 100` is typically very high complexity and a strong refactor candidate
- treat `cx` as a heuristic score, not a hard correctness signal

Depth hint colors:
- depth `0`: `honeydew`
- depth `1`: `lemonchiffon`
- depth `2+`: `mistyrose`

## `graph.mmd` (Mermaid)

Mermaid graph properties:
- diagram type is `flowchart LR`
- includes an init block with increased spacing (`nodeSpacing=80`, `rankSpacing=110`) and smoothed edges (`curve=basis`)
- modules render as nodes with module/function/file summaries
- cycle edges are labeled `CYCLE` and styled red via deterministic `linkStyle` indices
- architecture violations are labeled `VIOLATION` and styled with brown dashed `linkStyle`
- external edges are styled as dashed gray links
- nodes are classified with style groups (`internalNode`, `externalNode`, `cycleNode`, `hotspotNode`)
- when external module count exceeds `10`, external nodes are automatically collapsed into one `External/Stdlib (N modules)` node with `ext:<count>` edge labels
- when architecture config is enabled, modules are grouped by layer in Mermaid subgraphs
- includes a `Legend` subgraph explaining node fields (`funcs/files`, `d`, `in`, `out`, `cx`) and edge labels
- recommended config:
- set `output.mermaid = "graph.mmd"` and keep `output.paths.diagrams_dir = "docs/diagrams"` to write to `<root>/docs/diagrams/graph.mmd`

## `graph.puml` (PlantUML)

PlantUML graph properties:
- output is wrapped with `@startuml` / `@enduml`
- line routing is orthogonal (`skinparam linetype ortho`)
- increased spacing is applied (`skinparam nodesep 80`, `skinparam ranksep 100`)
- modules render as `component` nodes with module/function/file summaries
- cycle edges include `: CYCLE` labels and red thick arrows
- architecture violation edges include `: VIOLATION` labels and brown dashed arrows
- edges to external modules use gray dashed arrows
- when external module count exceeds `10`, external nodes are automatically collapsed into one `External/Stdlib (N modules)` node with `ext:<count>` edge labels
- when architecture config is enabled, modules are grouped in `package` blocks per layer
- output includes a right-side legend that explains node fields (`d`, `in`, `out`, `cx`) and edge semantics
- recommended config:
- set `output.plantuml = "graph.puml"` and keep `output.paths.diagrams_dir = "docs/diagrams"` to write to `<root>/docs/diagrams/graph.puml`

## Markdown Injection

Optional markdown injection is configured via `[[output.update_markdown]]`.

For each entry:
- `file` points to the markdown target file
- `marker` identifies replacement block markers
- `format` selects `mermaid` or `plantuml`

Markers must exist exactly once in the target file:

```text
<!-- circular:<marker>:start -->
...replaced content...
<!-- circular:<marker>:end -->
```

Injected content is a fenced code block:
- Mermaid format: ```` ```mermaid ```` ... ```` ``` ````
- PlantUML format: ```` ```plantuml ```` ... ```` ``` ````

## Ordering and Stability

- output schemas are additive and backward-compatible
- DOT/TSV ordering follows existing map-based traversal and may vary
- Mermaid/PlantUML nodes and edges are sorted for stable diff output
