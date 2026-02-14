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
- Dedicated component and flow diagram modes are now implemented for Mermaid and PlantUML.
- Diagram toggles are fully consumed by output generation (`architecture`, `component`, `flow`).
- Remaining follow-up work is quality iteration (richer call resolution heuristics, additional large-project rendering tuning).

Architecture mode status:
- Dedicated architecture diagram generation is now implemented for Mermaid/PlantUML when `output.diagrams.architecture=true`.
- Dedicated component and flow diagram generation is now implemented for Mermaid/PlantUML when enabled.

## Scope and Ownership
- Config schema and validation: `internal/core/config`
- Orchestration and output wiring: `internal/core/app`
- Diagram data shaping: `internal/engine/graph`
- Format emitters: `internal/ui/report/formats`
- MCP operation surface updates (if needed): `internal/mcp/*`
- Docs and examples: `README.md`, `docs/documentation/*`, `data/config/circular.example.toml`

## Config Baseline (P1 Implemented)
```toml
[output.diagrams]
architecture = true
component = false
flow = false

[output.formats]
mermaid = true
plantuml = false

[output.diagrams.flow_config]
entry_points = ["cmd/circular/main.go"]
max_depth = 8

[output.diagrams.component_config]
show_internal = false
```

Notes:
- This block is implemented in config parsing/validation.
- Existing `output.mermaid` and `output.plantuml` remain backward-compatible.
- Multiple enabled diagram modes now emit mode-suffixed files from each configured format path.

## Implementation Phases

| Phase | Focus | Status |
| :--- | :--- | :--- |
| P0 | Baseline dependency diagrams, markdown injection, MCP sync operation | Complete |
| P1 | Add `output.diagrams` config schema/defaults/validation | Complete |
| P2 | Dedicated architecture diagram generation | Complete |
| P3 | Dedicated component diagram generation | Complete |
| P4 | Flow/call diagram generation with entry-point controls | Complete |

## Task Checklist
- [x] T0: Baseline Mermaid/PlantUML dependency outputs with architecture overlays.
- [x] T1: Markdown marker injection flow for Mermaid/PlantUML diagrams.
- [x] T2: MCP sync operation for output regeneration (`graph.sync_diagrams`).
- [x] T3: Add `output.diagrams` structs/defaults/validation in `internal/core/config`.
- [x] T4: Add diagram mode selection/plumbing in `internal/core/app`.
- [x] T5: Add architecture-view emitters in `internal/ui/report/formats/mermaid.go` and `internal/ui/report/formats/plantuml.go`.
- [x] T6: Add component-view emitters using parser definitions and graph relationships.
- [x] T7: Add flow-view emitters using call/reference traversal with bounded depth.
- [x] T8: Add/extend tests for config validation, generator output, and backward compatibility.
- [x] T9: Update docs and config examples after each phase lands.

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
