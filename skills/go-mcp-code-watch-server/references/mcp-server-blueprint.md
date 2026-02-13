# Go MCP Server Blueprint for Code Watch

## Suggested Package Layout

- `cmd/codewatch-mcp/main.go`: server entrypoint.
- `internal/mcp/server.go`: transport/runtime and registration.
- `internal/mcp/tools.go`: tool handlers and schemas.
- `internal/mcp/resources.go`: resource handlers.
- `internal/mcp/prompts.go`: prompt templates.
- `internal/mcp/adapter.go`: calls into app/parser/graph/resolver/output.

## Adapter Pattern

Use adapter methods that map MCP inputs to existing app capabilities:

```go
type Adapter interface {
    ScanOnce(ctx context.Context, req ScanRequest) (ScanSummary, error)
    DetectCycles(ctx context.Context, req ScanRequest) (CyclesReport, error)
    FindUnresolved(ctx context.Context, req ScanRequest) (UnresolvedReport, error)
    TraceImportChain(ctx context.Context, req TraceRequest) (TraceReport, error)
    GenerateReports(ctx context.Context, req ReportRequest) (ReportBundle, error)
}
```

Keep MCP-specific types near handlers, not inside core analysis packages.

## Transport Guidance

- Prefer STDIO for local agent integration.
- Add HTTP/stream transport only when remote access is required.
- Provide graceful shutdown and context cancellation.

## Testing

- Unit-test each handler with adapter mocks.
- Integration-test one full request per tool.
- Snapshot-test response payload shape and ordering.
