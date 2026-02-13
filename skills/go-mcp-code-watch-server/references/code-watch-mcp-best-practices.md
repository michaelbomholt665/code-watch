# Code Watch MCP Best Practices

## Security and Safety

- Keep server read-only by default.
- Validate and normalize all filesystem paths.
- Reject paths outside allowed roots unless explicitly configured.
- Bound long-running analyses with context deadlines.

## Reliability

- Use deterministic sorting before returning lists.
- Keep response payloads compact and structured.
- Prefer stable field names and additive evolution.
- Include `version` in top-level response envelopes for future migrations.

## Performance

- Reuse parser/graph instances where safe.
- Cache immutable reference data (stdlib tables, config defaults).
- Avoid repeated full rescans when incremental scope is available.

## Documentation

- Document every tool/resource/prompt in `docs/documentation/mcp.md`.
- Include request/response examples for each tool.
- Cross-link CLI and output docs when behaviors overlap.
