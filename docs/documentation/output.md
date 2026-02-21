# Output Reference

`circular` can emit DOT, TSV, Mermaid, PlantUML, and Markdown outputs via `internal/ui/report`.

## Diagram Scope

Mermaid and PlantUML support four output views:
- default dependency view (module-level graph)
- architecture view (layer-level graph) when `output.diagrams.architecture=true`
- component view (module internals and symbol-reference overlays) when `output.diagrams.component=true`
- flow view (bounded traversal from configured entry points) when `output.diagrams.flow=true`

Implemented overlays:
- cycle highlighting
- architecture-violation highlighting (layer + package rules)
- optional architecture-layer grouping when `[architecture].enabled=true`

Roadmap source:
- `docs/plans/diagram-expansion-plan.md`

Mode constraints:
- when all of `output.diagrams.architecture`, `output.diagrams.component`, `output.diagrams.flow` are `false`, default dependency view is generated
- when one mode is enabled, the configured output path is used as-is (for example `graph.mmd`)
- when multiple modes are enabled, mode-suffixed files are generated from the configured base path (`graph-dependency.mmd`, `graph-architecture.mmd`, `graph-component.mmd`, `graph-flow.mmd`)
- Mermaid is enabled by default; PlantUML requires explicit enablement via `output.formats.plantuml=true`
- path resolution is separator-aware: values containing `/` or `\` are treated as relative paths under output root; filename-only values resolve under `output.paths.diagrams_dir`

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

## Appended Architecture-Rule Violation Block

Appended only when findings exist, separated by a blank line.

Header:

```text
Type\tRule\tModule\tViolation\tTarget\tDetail\tFile\tLine\tColumn\tLimit\tActual
```

Row prefix is always:

```text
architecture_rule_violation
```

## Appended Secret-Finding Block

Appended only when findings exist, separated by a blank line.

Header:

```text
Type\tKind\tSeverity\tValue\tEntropy\tConfidence\tFile\tLine\tColumn
```

Row prefix is always:

```text
secret
```

`Value` is masked (for example `AKIA...CDEF`) and raw values are not written to TSV output.

## SARIF v2.1.0 Output

SARIF (Static Analysis Results Interchange Format) output is generated via `internal/ui/report/formats/sarif.go` and enabled by setting `output.sarif` in config or passing `--sarif <path>` on the CLI.

### Enabling SARIF

Via config:
```toml
[output]
sarif = "results/circular.sarif.json"
```

Via CLI flag (overrides config):
```bash
./circular --once --sarif results/circular.sarif.json
```

### SARIF Structure

The report follows the SARIF v2.1.0 schema and contains a single `run` with four rule classes:

| Rule ID | Name | Severity | Triggers on |
| :--- | :--- | :--- | :--- |
| `CIRC001` | `CircularDependency` | `error` | Circular import cycle detected |
| `CIRC002` | `PotentialSecret` | `warning` / `error` | Secret or high-entropy token found |
| `CIRC003` | `ArchitectureViolation` | `warning` | Layer-rule violation |
| `CIRC004` | `ArchitectureRuleViolation` | `warning` | Package-rule violation |

Severity mapping for `CIRC002`:
- `critical`, `high` → SARIF `error`
- `medium` → SARIF `warning`
- `low` or unknown → SARIF `note`

### Path Handling

All `artifactLocation.uri` values are **relative to the project root** and use forward slashes. The `uriBaseId` is set to `%SRCROOT%` so GitHub Code Scanning resolves file anchors correctly.

```json
{
  "$schema": "https://schemastore.azurewebsites.net/schemas/json/sarif-2.1.0-rtm.5.json",
  "version": "2.1.0",
  "runs": [{
    "tool": { "driver": { "name": "circular", "version": "1.0.0", "rules": [...] } },
    "results": [{
      "ruleId": "CIRC001",
      "level": "error",
      "message": { "text": "Circular dependency: a → b → a" },
      "locations": [{
        "physicalLocation": {
          "artifactLocation": { "uri": "internal/core/app.go", "uriBaseId": "%SRCROOT%" }
        }
      }]
    }]
  }]
}
```

### GitHub Actions Integration

Upload SARIF results to GitHub Code Scanning using the official action:

```yaml
- name: Run circular
  run: ./circular --once --sarif results/circular.sarif.json

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results/circular.sarif.json
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

## Component View Notes

When `output.diagrams.component=true`:
- module-to-module edges include `deps:N` and `refs:M` labels
- `refs:M` counts matched symbol references from source module references to target module definitions
- with `output.diagrams.component_config.show_internal=true`, definition-level symbol nodes are included and previewed as `sym:a,b,c` on edges
- architecture layer grouping still applies when `[architecture].enabled=true`

## Flow View Notes

When `output.diagrams.flow=true`:
- traversal starts from configured `output.diagrams.flow_config.entry_points`
- entry points can match either module names or file paths (absolute or project-relative suffix)
- `max_depth` bounds traversal breadth (`step:N` node annotation shows shortest hop distance from nearest entry module)
- if no configured entry point resolves, roots (modules with zero fan-in) are used as fallback

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

## `analysis-report.md` (Markdown Report)

Markdown report generation is enabled when `output.markdown` is set (or via CLI `--report-md`).

Report characteristics:
- YAML frontmatter with `project`, `generated_at`, and `version`
- executive summary table (modules/files/cycles/violations/hotspots/probable-bridges/unresolved/unused)
- detailed sections:
- circular imports (with impact/severity)
- architecture violations
- complexity hotspots
- probable bridge references
- unresolved references
- unused imports
- TSV probable-bridge appendix rows when findings exist:
- `Type`, `File`, `Reference`, `Line`, `Column`, `Confidence`, `Score`, `Reasons`
- optional Mermaid dependency diagram embedding when `output.report.include_mermaid=true`
- configurable presentation:
- `output.report.verbosity` = `summary|standard|detailed`
- `output.report.table_of_contents` controls TOC output
- `output.report.collapsible_sections` controls `<details>` wrappers for long sections

## Ordering and Stability

- output schemas are additive and backward-compatible
- DOT/TSV ordering follows existing map-based traversal and may vary
- Mermaid/PlantUML nodes and edges are sorted for stable diff output

## Advanced Visualizations (Internal)

The reporting engine includes generators for advanced visualization types (currently internal-only, pending CLI wiring):

### Interactive Treemap (`html_interactive.go`)
Generates a self-contained HTML report with a D3.js zoomable treemap.
- **Size**: Number of source files in module
- **Color**: Complexity hotspot score (Blue → Red)
- **Interactivity**: Zoomable headers, tooltips with detailed metrics

### Sequence Diagrams (`sequence.go`)
Generates Mermaid `sequenceDiagram` output by tracing symbol references.
- **Trace**: Breadth-first traversal of cross-module calls
- **Depth**: Configurable max-depth
- **Output**: Participant lines and interaction arrows

### C4-Style Architecture (`mermaid.go`)
An aggregated view of the architecture graph.
- **Clustering**: Modules grouped by defined architecture layers
- **Edges**: Aggregated into single weighted arrows (`deps:N`) between clusters
- **Violations**: Highlighted in bright red
