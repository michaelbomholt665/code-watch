# Diagram Expansion Plan

## Goal
Expand diagram support beyond the current module dependency graph so `circular` can generate dedicated architecture, component, and flow views in both Mermaid and PlantUML.

## Current State (Implemented)
- Dependency graph outputs exist in all formats:
- DOT (`output.dot`)
- TSV (`output.tsv`)
- Mermaid (`output.mermaid`)
- PlantUML (`output.plantuml`)
- Mermaid/PlantUML currently render a module dependency view (with cycle and architecture-violation overlays).
- Optional markdown injection is supported via `[[output.update_markdown]]`.
- MCP supports report refresh through `graph.sync_diagrams`.

## Gap Summary
- No dedicated architecture-diagram mode (separate from module graph overlays).
- No component-diagram mode (module internals and symbol-level relationships).
- No flow/call-sequence diagram mode.
- No config block yet for per-diagram-type toggles (`[output.diagrams.*]`).

## Scope and Ownership
- Config schema and validation: `internal/core/config`
- Orchestration and output wiring: `internal/core/app`
- Diagram data shaping: `internal/engine/graph`
- Format emitters: `internal/ui/report/formats`
- MCP operation surface updates (if needed): `internal/mcp/*`
- Docs and examples: `README.md`, `docs/documentation/*`, `data/config/circular.example.toml`

## Proposed Config (Phase 2)
```toml
[output.diagrams]
architecture = true
component = false
flow = false

[output.diagrams.flow]
entry_points = ["cmd/circular/main.go"]
max_depth = 8

[output.diagrams.component]
show_internal = false
```

Notes:
- This block is planned and not implemented yet.
- Existing `output.mermaid` and `output.plantuml` remain backward-compatible.

## Implementation Phases

| Phase | Focus | Status |
| :--- | :--- | :--- |
| P0 | Baseline dependency diagrams, markdown injection, MCP sync operation | Done |
| P1 | Add `output.diagrams` config schema/defaults/validation | Planned |
| P2 | Dedicated architecture diagram generation | Planned |
| P3 | Dedicated component diagram generation | Planned |
| P4 | Flow/call diagram generation with entry-point controls | Planned |

## Task Checklist
- [x] T0: Baseline Mermaid/PlantUML dependency outputs with architecture overlays.
- [x] T1: Markdown marker injection flow for Mermaid/PlantUML diagrams.
- [x] T2: MCP sync operation for output regeneration (`graph.sync_diagrams`).
- [ ] T3: Add `output.diagrams` structs/defaults/validation in `internal/core/config`.
- [ ] T4: Add diagram mode selection/plumbing in `internal/core/app`.
- [ ] T5: Add architecture-view emitters in `internal/ui/report/formats/mermaid.go` and `internal/ui/report/formats/plantuml.go`.
- [ ] T6: Add component-view emitters using parser definitions and graph relationships.
- [ ] T7: Add flow-view emitters using call/reference traversal with bounded depth.
- [ ] T8: Add/extend tests for config validation, generator output, and backward compatibility.
- [ ] T9: Update docs and config examples after each phase lands.

## Constraints
- Keep existing output keys and default behavior backward-compatible.
- Avoid expensive full-project flow rendering by default; require bounded entry points.
- Keep deterministic output ordering for Mermaid/PlantUML where practical.
- Ensure watch/UI/MCP paths remain responsive when advanced diagram generation is enabled.

## Documentation Sync Policy
For each merged phase:
1. Update `README.md` feature and config notes.
2. Update `docs/documentation/configuration.md` and `docs/documentation/output.md`.
3. Update MCP docs when operation payloads or semantics change (`docs/documentation/mcp.md`).
4. Keep this plan status table/checklist accurate.
