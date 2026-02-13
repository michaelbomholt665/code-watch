# Code Watching and Analysis Best Practices

## Pipeline Boundaries

- Parser: syntax extraction only.
- Graph: dependency state and traversal.
- Resolver: symbol/reference interpretation.
- Watcher: filesystem events, debounce, batching.
- Output: rendering only.

Do not cross these boundaries except through explicit data structures.

## Watcher Reliability

- Debounce bursty events and deduplicate paths.
- Normalize paths before enqueueing work.
- Keep watcher loop non-blocking; run heavy work in worker paths.
- Handle editor save patterns (atomic rename, temp files).

## Incremental Analysis

- Remove stale file contributions before re-adding parsed results.
- Track dirty sets explicitly.
- Invalidate transitive dependents deterministically.

## Accuracy and False Positives

- Prefer conservative unresolved-symbol reporting.
- Exclude stdlib and known builtins explicitly.
- Treat language-specific quirks in dedicated modules.

## Performance

- Cache expensive derived indexes.
- Recompute only changed subgraphs when possible.
- Measure before optimizing; add targeted benchmarks.

## Documentation and Operations

- Keep CLI/config/output docs current in `docs/documentation/`.
- Document output schema changes and migration notes.
- Include examples for TSV, DOT, Mermaid, and PlantUML usage.
