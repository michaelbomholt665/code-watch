# MCP POC Runtime

This repository ships a POC MCP runtime that exposes a single tool (`circular`) over stdio. The protocol is intentionally minimal and designed for local agent integration.
When `mcp.openapi_spec_path` or `mcp.openapi_spec_url` is configured, startup loads and validates the OpenAPI document with `kin-openapi`, converts operations to descriptors, and applies the MCP operation allowlist.

Internally, MCP runtime wiring and scan/query/secrets/cycles/watch/output/report operations now run through the `internal/core/ports.AnalysisService` driving ports exposed by `internal/core/app`.
When `mcp.auto_manage_outputs=true`, startup output synchronization also routes through `AnalysisService.SyncOutputs(...)`.
CLI MCP-mode bootstrap now resolves analysis dependencies via an interface-first runtime factory (`internal/ui/cli/runtime_factory.go`) before handing control to `internal/mcp/runtime`.
Parity coverage in `internal/mcp/adapters/adapter_test.go` asserts summary/output contract equivalence between CLI-facing `AnalysisService` calls and MCP adapter calls for the same fixture graph.

## Transport Protocols

This MCP server supports multiple transport protocols for flexibility in different environments.

### 1. Stdio (Default)

- Transport: `stdio`
- Encoding: JSON (one object per line)
- Best for: Local agent integration where the server is spawned as a child process.

### 2. SSE (Server-Sent Events)

- Transport: `http` / `sse`
- Mode: Asynchronous full-duplex communication over HTTP.
- Best for: Remote connections or integration with clients that prefer HTTP.

**SSE Flow:**
1. Client connects to `GET /sse` to establish an event stream.
2. Server responds with a `session_id`.
3. Client sends JSON-RPC requests via `POST /message?session_id=XYZ`.
4. Server sends responses as events through the established SSE stream.

## Protocol Envelopes

Regardless of transport, Circular supports two envelope formats:

### 1. JSON-RPC 2.0 (Recommended)

Follows the standard MCP JSON-RPC 2.0 specification.

### 2. Circular Legacy Envelope (Stdio only)

Used for simple line-based integration.

### Request Envelope (Legacy)

```json
{"id":"1","tool":"circular","args":{"operation":"scan.run","params":{"paths":["./internal","./cmd"]}}}
```

Fields:
- `id` (string, optional): echoed back in the response
- `tool` (string): tool name; defaults to `circular` unless `mcp.exposed_tool_name` is set
- `args` (object): tool arguments
  - `operation` (string): operation identifier
  - `params` (object): operation-specific fields

### Response Envelope

Success:
```json
{"id":"1","ok":true,"result":{"version":"v1","operation":"scan.run","result":{"files_scanned":120,"modules":23,"duration_ms":84}}}
```

Error:
```json
{"id":"1","ok":false,"error":{"code":"invalid_argument","message":"operation is required"}}
```

## Operations

All operations are dispatched through the single tool. Operation IDs are allowlisted via `mcp.operation_allowlist`.

### `scan.run`

Runs a scan against current watch paths (or explicit `paths`).

Params:
- `paths` (`[]string`, optional)
- `config_path` (`string`, optional)
- `project_root` (`string`, optional)

Result:
- `files_scanned` (`int`)
- `modules` (`int`)
- `duration_ms` (`int`)
- `warnings` (`[]string`, optional)

Notes:
- when DB/history is enabled, `scan.run` also captures a project-key-scoped snapshot through the shared `AnalysisService` history use case.
- uses `AnalysisService.RunScan` internally.

### `surgical.context`

Extracts source context around a symbol with high precision.

Params:
- `symbol` (`string`)
- `file` (`string`)

Result:
- `symbol` (`string`)
- `file` (`string`)
- `snippets` (`[]Snippet`)
  - `line` (`int`)
  - `tag` (`string`, optional — e.g. `SYM_DEF`, `REF_CALL`)
  - `confidence` (`float64`, optional)
  - `ancestry` (`string`, optional)
  - `context` (`[]string` — ±5 lines)

Notes:
- Uses `internal/ui/report/surgical` to scan file content; enriched by symbol store tags.

### `overlays.add`

Adds an AI-verified semantic overlay to the persistent store.

Params:
- `symbol` (`string`)
- `file` (`string`, optional)
- `type` (`string` — `EXCLUSION`, `VETTED_USAGE`, `RE-ALIAS`)
- `reason` (`string`)
- `source_hash` (`string`, optional)

Result:
- `id` (`int64`)
- `status` (`string`)
- `message` (`string`)

Notes:
- Requires `mcp.allow_mutations=true`.
- Persists to `semantic_overlays` table in SQLite.

### `overlays.list`

Lists active overlays.

Params:
- `symbol` (`string`, optional)
- `file` (`string`, optional)

Result:
- `overlays` (`[]Overlay`)
- `total` (`int`)

### `secrets.scan`

Runs a scan (full or path-scoped) and returns detected secret findings with masked values.

Params:
- `paths` (`[]string`, optional)

Result:
- `files_scanned` (`int`)
- `secret_count` (`int`)
- `findings` (`[]SecretFinding`, optional)
- `warnings` (`[]string`, optional)

`SecretFinding` fields:
- `kind`, `severity`
- `value_masked` (masked value, never raw secret)
- `entropy`, `confidence`
- `file`, `line`, `column`

### `secrets.list`

Lists currently detected secrets from in-memory graph state.

Params:
- `limit` (`int`, optional)

Result:
- `secret_count` (`int`)
- `findings` (`[]SecretFinding`, optional)

Notes:
- Secret listing is delegated through `AnalysisService.ListFiles(...)` rather than direct adapter access to graph internals.

### `graph.cycles`

Params:
- `limit` (`int`, optional)

Result:
- `cycle_count` (`int`)
- `cycles` (`[][]string`, optional)

Notes:
- Cycle detection is delegated through `AnalysisService.DetectCycles(...)`.

### `query.modules`

Params:
- `filter` (`string`, optional)
- `limit` (`int`, optional)

Result:
- `modules` (`[]ModuleSummary`)

### `query.module_details`

Params:
- `module` (`string`)

Result:
- `module` (`ModuleDetails`)

### `query.trace`

Params:
- `from_module` (`string`)
- `to_module` (`string`)
- `max_depth` (`int`, optional)

Result:
- `found` (`bool`)
- `path` (`[]string`, optional)
- `depth` (`int`, optional)

### `query.trends`

Params:
- `since` (`string`, optional, RFC3339 or YYYY-MM-DD)
- `limit` (`int`, optional)

Result:
- `since` (`string`, optional)
- `until` (`string`, optional)
- `scan_count` (`int`)
- `snapshots` (`[]TrendSnapshot`)

Notes:
- Requires DB/history enabled (`[db].enabled = true`).

### `graph.sync_diagrams`

Writes configured DOT/TSV/Mermaid/PlantUML/Markdown outputs and optional markdown injections.

Params:
- `formats` (`[]string`, optional, values: `dot|tsv|mermaid|plantuml|markdown`)

Result:
- `written` (`[]string`)

Notes:
- Requires `mcp.allow_mutations=true`.
- Legacy operation ID `system.sync_outputs` is accepted and normalized to `graph.sync_diagrams`.
- Output sync orchestration is delegated through `AnalysisService.SyncOutputs(...)`.

### `system.sync_config`

Writes active config to the configured MCP config path.

Result:
- `synced` (`bool`)
- `target` (`string`, optional)

Notes:
- Requires `mcp.allow_mutations=true`.

### `system.generate_config`

Generates a project-local config file from `data/config/circular.example.toml` if the target file does not already exist.

Result:
- `generated` (`bool`)
- `target` (`string`, optional)

Notes:
- Requires `mcp.allow_mutations=true`.
- Idempotent: existing config files are preserved.

### `system.generate_script`

Generates a project-local `circular-mcp` helper script if it does not already exist.

Result:
- `generated` (`bool`)
- `target` (`string`, optional)

Notes:
- Requires `mcp.allow_mutations=true`.
- Idempotent: existing scripts are preserved.
- Generated script mode is executable (`0755`).

### `system.select_project`

Switches active project context.

Params:
- `name` (`string`)

Result:
- `project` (`ProjectSummary`)

Notes:
- Requires `mcp.allow_mutations=true`.
- Project switch updates history namespace; scan/watch roots are not reloaded automatically in this POC.

### `system.watch`

Starts a non-blocking background watcher for configured watch paths.

Result:
- `status` (`string`)
- `already_watching` (`bool`, optional)

Notes:
- Requires `mcp.allow_mutations=true`.
- The runtime prevents multiple watcher instances in the same server session.
- Watch startup is delegated through `AnalysisService.WatchService().Start(...)` instead of direct app watcher orchestration.

### `report.generate_markdown`

Generates a markdown analysis report and optionally writes it to disk.

Params:
- `write_file` (`bool`, optional)
- `path` (`string`, optional)
- `verbosity` (`string`, optional: `summary|standard|detailed`)

Result:
- `markdown` (`string`)
- `path` (`string`, optional)
- `written` (`bool`)

Notes:
- Markdown report generation is delegated through `AnalysisService.GenerateMarkdownReport(...)`.

## Allowlist Notes

`mcp.operation_allowlist` entries should use operation IDs above. Legacy aliases are accepted:
- `scan_once` -> `scan.run`
- `detect_cycles` -> `graph.cycles`
- `trace_import_chain` -> `query.trace`
- `generate_reports` -> `graph.sync_diagrams`
- `system.sync_outputs` -> `graph.sync_diagrams`

When OpenAPI conversion is enabled, allowlist filtering is applied to converted descriptors at startup and startup fails if no operations remain after filtering.

## Error Codes

- `invalid_argument`
- `not_found`
- `unavailable`
- `internal`

## Quick Example

```bash
cat <<'JSON' | go run ./cmd/circular --config data/config/circular.toml
{"id":"1","tool":"circular","args":{"operation":"query.modules","params":{"limit":5}}}
JSON
```

## Wrapper Script

Use `scripts/circular-mcp` to avoid hand-writing JSON payloads.

Examples:

```bash
scripts/circular-mcp modules --limit 5
scripts/circular-mcp generate-config
scripts/circular-mcp generate-script
scripts/circular-mcp sync-diagrams --format mermaid --format plantuml
scripts/circular-mcp watch
```

## Socket-Activated (systemd --user)

If you want multiple tools to connect without keeping a long-lived stdio process, use a socket-activated user service.
Each connection spawns a new MCP instance over the UNIX socket.

Create the socket unit at `~/.config/systemd/user/circular-mcp.socket`:

```ini
[Unit]
Description=Circular MCP socket

[Socket]
ListenStream=%t/circular-mcp.sock
SocketMode=0600
Accept=yes

[Install]
WantedBy=sockets.target
```

Create the service template at `~/.config/systemd/user/circular-mcp@.service`:

```ini
[Unit]
Description=Circular MCP connection handler

[Service]
Type=simple
WorkingDirectory=/path/to/code-watch
StandardInput=socket
StandardOutput=socket
StandardError=journal
ExecStart=/usr/bin/env bash -lc 'cd /path/to/code-watch && go run ./cmd/circular --config /path/to/code-watch/data/config/circular.toml'
```

Enable the socket:

```bash
systemctl --user daemon-reload
systemctl --user enable --now circular-mcp.socket
```

Use a small stdio-to-socket bridge as the MCP client command (example `~/.local/bin/circular-mcp-client`):

```python
#!/usr/bin/env python3
import os
import selectors
import socket
import sys

runtime_dir = os.environ.get("XDG_RUNTIME_DIR", f"/run/user/{os.getuid()}")
sock_path = os.environ.get("CIRCULAR_MCP_SOCK", os.path.join(runtime_dir, "circular-mcp.sock"))

s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
s.connect(sock_path)
s.setblocking(False)

sel = selectors.DefaultSelector()
sel.register(sys.stdin, selectors.EVENT_READ)
sel.register(s, selectors.EVENT_READ)

while True:
    for key, _ in sel.select():
        if key.fileobj is sys.stdin:
            data = sys.stdin.buffer.read1(4096)
            if not data:
                sys.exit(0)
            s.sendall(data)
        else:
            data = s.recv(4096)
            if not data:
                sys.exit(0)
            sys.stdout.buffer.write(data)
            sys.stdout.buffer.flush()
```

Point your MCP client config at that bridge:

```json
{
  "mcpServers": {
    "circular": {
      "command": "/home/you/.local/bin/circular-mcp-client",
      "args": [],
      "env": {}
    }
  }
}
```

Notes:
- Each connection runs a new process. For faster startup, build a binary and use it in `ExecStart`.
- The socket path is `%t/circular-mcp.sock` (typically `/run/user/<uid>/circular-mcp.sock`).

## Client Config Schemas

Use these client-side configs to connect local tools to this MCP server over stdio.

### VS Code (`.vscode/mcp.json`)

```json
{
  "servers": {
    "circular": {
      "type": "stdio",
      "command": "go",
      "args": ["run", "./cmd/circular", "--config", "data/config/circular.toml"],
      "env": {}
    }
  }
}
```

### Antigravity (`mcp_config.json`)

Antigravity no longer relies on the old MCP enable toggle in recent builds; use **Agent Panel -> ... -> Manage MCP Servers -> View raw config** and edit `mcp_config.json`.

```json
{
  "mcpServers": {
    "circular": {
      "command": "go",
      "args": ["run", "./cmd/circular", "--config", "data/config/circular.toml"],
      "env": {}
    }
  }
}
```

### Gemini CLI (`~/.gemini/settings.json` or `./.gemini/settings.json`)

```json
{
  "mcpServers": {
    "circular": {
      "command": "go",
      "args": ["run", "./cmd/circular", "--config", "data/config/circular.toml"],
      "cwd": ".",
      "env": {}
    }
  }
}
```

### Kilo Code (`~/.kilocode/mcp_settings.json` or `./.kilocode/mcp.json`)

```json
{
  "mcpServers": {
    "circular": {
      "type": "stdio",
      "command": "go",
      "args": ["run", "./cmd/circular", "--config", "data/config/circular.toml"],
      "cwd": ".",
      "env": {},
      "disabled": false
    }
  }
}
```

### Codex

Codex CLI MCP config is **TOML**, not JSON. Configure `~/.codex/config.toml`:

```toml
[mcp_servers.circular]
command = "go"
args = ["run", "./cmd/circular", "--config", "data/config/circular.toml"]
```
