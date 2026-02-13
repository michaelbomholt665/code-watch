# Output Reference

`circular` can emit DOT and TSV outputs via `internal/output`.

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

Depth hint colors:
- depth `0`: `honeydew`
- depth `1`: `lemonchiffon`
- depth `2+`: `mistyrose`

## Ordering and Stability

- output schemas are additive and backward-compatible
- row and edge ordering is not guaranteed stable because graph data is iterated from maps
