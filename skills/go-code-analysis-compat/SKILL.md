---
name: go-code-analysis-compat
description: Build and maintain Go code analysis and code-watching tooling with idiomatic Go conventions, stable graph/report outputs, and strict documentation hygiene. Use when implementing or reviewing Go analyzers, file watchers, dependency graphs, and output generators, especially when developing on Go 1.25.x while enforcing compatibility with Go 1.24.x.
---

# Go Code Analysis Compat

Execute repository changes for Go-based code watching and static analysis systems with strict compatibility and documentation discipline.

## Use This Workflow

1. Define scope: parser, graph, resolver, watcher, output, or CLI.
2. Implement behavior in the owning package without leaking responsibilities.
3. Write or update tests before finalizing code.
4. Update inline code docs and project docs in `docs/documentation/`.
5. Validate on Go 1.25.x and compatibility-check on Go 1.24.x (minimum 1.24).

## Enforce Go Version Policy

- Develop with Go `1.25.x`.
- Keep syntax/APIs compatible with Go `1.24.*`.
- Treat Go `1.24` as the compatibility floor in CI/local checks.

Run:

```bash
go test ./...
GOTOOLCHAIN=go1.24 go test ./...
```

If incompatibility appears, remove post-1.24 usage and prefer stable alternatives.

## Follow Idiomatic Go Conventions

- Keep package APIs minimal and cohesive.
- Use short receivers, explicit error wrapping, and predictable zero values.
- Prefer composition over inheritance-like patterns.
- Keep interfaces small and consumer-owned.
- Make deterministic behavior explicit for analysis outputs.

## Documentation Rules (Required)

Always document written code and keep repository docs current.

- Add or update package-level comments when package behavior changes.
- Add Go doc comments on exported types/functions.
- Add concise comments for non-obvious logic paths.
- Update `docs/documentation/` for CLI flags, config schema, output formats, and architecture changes.
- Keep examples runnable and aligned with real command behavior.

## Output Generation Standards

Generate machine- and human-consumable outputs from one canonical analysis model.

- TSV: stable column order, escaped values, additive schema evolution.
- DOT: deterministic node IDs and edge ordering.
- Mermaid: concise graph syntax suitable for markdown embedding.
- PlantUML: sequence/component views for architecture communication.

For implementation patterns and templates, read:
- `references/output-autogeneration-patterns.md`
- `references/code-watch-analysis-best-practices.md`

## Do and Don't

Do:
- Keep parser/graph/resolver/watcher/output boundaries strict.
- Use table-driven tests for analyzer behavior and edge cases.
- Preserve backward compatibility for generated output files.
- Keep report generation deterministic and test snapshots.
- Update `docs/documentation/` in the same change as behavior updates.

Don't:
- Introduce Go 1.25-only features without compatibility fallback.
- Mix broad refactors with behavior changes in one patch.
- Emit nondeterministic output ordering.
- Change TSV/DOT/Mermaid/PlantUML structures silently.
- Leave new exported APIs undocumented.

## Required Verification

```bash
go test ./...
GOTOOLCHAIN=go1.24 go test ./...
go test ./... -coverprofile=coverage.out
```

If compatibility checks cannot run, report the exact blocker and residual risk.

## Quick Checklist

- Code is idiomatic and package-local.
- Exported APIs are documented.
- `docs/documentation/` is updated when behavior/output changes.
- Output generators produce deterministic TSV/DOT/Mermaid/PlantUML.
- Go 1.24.x compatibility is verified or explicitly marked unverified.
