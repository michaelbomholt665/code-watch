---
title: Code Analysis Report
project: code-watch
generated_at: 2026-02-20T01:25:18Z
version: 1.0.0
---

# Analysis Report

## Table of Contents
- [Executive Summary](#executive-summary)
- [Circular Imports](#circular-imports)
- [Architecture Violations](#architecture-violations)
- [Complexity Hotspots](#complexity-hotspots)
- [Unresolved References](#unresolved-references)
- [Unused Imports](#unused-imports)

## Executive Summary
| Metric | Value |
| --- | --- |
| Total Modules | 36 |
| Total Files | 118 |
| Circular Imports | 0 |
| Architecture Violations | 0 |
| Complexity Hotspots | 5 |
| Unresolved References | 0 |
| Unused Imports | 27 |

## Circular Imports
No circular imports detected.

## Architecture Violations
No architecture violations detected.

## Complexity Hotspots
| Module | Definition | File | Score | Branches | Params | Nesting | LOC |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `circular/internal/ui/report/formats` | `Generate` | `internal/ui/report/formats/mermaid.go` | 123 | 47 | 3 | 5 | 168 |
| `circular/internal/core/app` | `GenerateOutputs` | `internal/core/app/output.go` | 110 | 39 | 6 | 4 | 181 |
| `circular/internal/ui/report/formats` | `buildComponentDiagramData` | `internal/ui/report/formats/diagram_modes.go` | 109 | 43 | 2 | 4 | 134 |
| `circular/internal/core/config` | `validateArchitecture` | `internal/core/config/validator.go` | 101 | 39 | 1 | 5 | 120 |
| `circular/internal/ui/report/formats` | `Generate` | `internal/ui/report/formats/plantuml.go` | 97 | 36 | 3 | 5 | 123 |

## Unresolved References
No unresolved references detected.

## Unused Imports
<details>
<summary>Unused import details</summary>

| Language | Module | Alias | Item | Confidence | Location |
| --- | --- | --- | --- | --- | --- |
| `go` | `circular/internal/core/app/helpers` | `` | `` | `medium` | `internal/core/app/analyzer.go:4:2` |
| `go` | `circular/internal/core/app/helpers` | `` | `` | `medium` | `internal/core/app/app.go:4:2` |
| `go` | `circular/internal/engine/secrets` | `secretengine` | `` | `medium` | `internal/core/app/app.go:10:2` |
| `go` | `circular/internal/core/config` | `` | `` | `medium` | `internal/core/app/service.go:4:2` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/core/app/service.go:6:2` |
| `go` | `circular/internal/mcp/schema` | `` | `` | `medium` | `internal/mcp/transport/stdio.go:6:2` |
| `go` | `circular/internal/engine/secrets` | `` | `` | `medium` | `internal/ui/report/formats/tsv.go:8:2` |
| `go` | `circular/internal/core/app` | `coreapp` | `` | `medium` | `internal/ui/cli/runtime_factory.go:4:2` |
| `go` | `circular/internal/core/app/helpers` | `` | `` | `medium` | `internal/core/app/output.go:4:2` |
| `go` | `circular/internal/shared/version` | `` | `` | `medium` | `internal/core/app/output.go:7:2` |
| `go` | `circular/internal/core/app` | `coreapp` | `` | `medium` | `internal/ui/cli/runtime.go:4:2` |
| `go` | `circular/internal/mcp/runtime` | `mcpruntime` | `` | `medium` | `internal/ui/cli/runtime.go:9:2` |
| `go` | `os/signal` | `` | `` | `medium` | `internal/ui/cli/runtime.go:17:2` |
| `go` | `syscall` | `` | `` | `medium` | `internal/ui/cli/runtime.go:20:2` |
| `go` | `circular/internal/engine/secrets` | `` | `` | `medium` | `internal/mcp/adapters/adapter.go:7:2` |
| `go` | `circular/internal/data/history` | `` | `` | `medium` | `internal/mcp/runtime/bootstrap.go:5:2` |
| `go` | `circular/internal/mcp/adapters` | `` | `` | `medium` | `internal/mcp/runtime/bootstrap.go:6:2` |
| `go` | `circular/internal/mcp/openapi` | `` | `` | `medium` | `internal/mcp/runtime/bootstrap.go:7:2` |
| `go` | `circular/internal/mcp/registry` | `` | `` | `medium` | `internal/mcp/runtime/bootstrap.go:8:2` |
| `go` | `circular/internal/mcp/tools/graph` | `` | `` | `medium` | `internal/mcp/runtime/server.go:9:2` |
| `go` | `circular/internal/mcp/tools/query` | `` | `` | `medium` | `internal/mcp/runtime/server.go:10:2` |
| `go` | `circular/internal/mcp/tools/report` | `` | `` | `medium` | `internal/mcp/runtime/server.go:11:2` |
| `go` | `circular/internal/mcp/tools/scan` | `` | `` | `medium` | `internal/mcp/runtime/server.go:12:2` |
| `go` | `circular/internal/mcp/tools/secrets` | `` | `` | `medium` | `internal/mcp/runtime/server.go:13:2` |
| `go` | `circular/internal/mcp/tools/system` | `` | `` | `medium` | `internal/mcp/runtime/server.go:14:2` |
| `go` | `circular/internal/mcp/validate` | `` | `` | `medium` | `internal/mcp/runtime/server.go:16:2` |
| `go` | `net/http` | `` | `` | `medium` | `internal/mcp/openapi/loader.go:7:2` |

</details>

