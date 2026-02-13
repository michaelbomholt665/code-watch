# Output Reference

This document defines the generated output formats for `circular`.

## `dependencies.tsv`

Base dependency rows:

Header:

```text
From\tTo\tFile\tLine\tColumn
```

Each row represents one import edge from module `From` to module `To`.

## Unused Import Rows (`type=unused_import`)

When unused import findings are present, they are appended after a blank line.

Header:

```text
Type\tFile\tLanguage\tModule\tAlias\tItem\tLine\tColumn\tConfidence
```

Row format:

```text
unused_import\t<file>\t<language>\t<module>\t<alias>\t<item>\t<line>\t<column>\t<confidence>
```

Notes:
- `Confidence` is currently `high` (named imports) or `medium` (module-level usage heuristics).
- `Item` is populated for item imports (for example Python `from x import y`).

## `graph.dot`

The DOT output remains backward compatible and now supports additive module metrics in node labels.

When metric data is provided, internal module labels include:

```text
(d=<depth> in=<fan-in> out=<fan-out>)
```

Depth color hints:
- depth `0`: `honeydew`
- depth `1`: `lemonchiffon`
- depth `2+`: `mistyrose`

Cycle edges continue to be highlighted in red with `label="CYCLE"`.
