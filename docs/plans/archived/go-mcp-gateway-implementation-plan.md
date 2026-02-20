# Go MCP Gateway Implementation Plan
## Proof of Concept: Circular Integration with API-as-Tools Pattern

**Target:** ~1000 lines of boilerplate across 6-8 files  
**Library:** `github.com/mark3labs/mcp-go` (most complete Go MCP SDK)  
**Pattern:** Single meta-tool that dynamically exposes operations

**Status: Fully Implemented** (2026-02-20)
**Archived:** 2026-02-20 - MCP server fully implemented in `internal/mcp/` with single-tool pattern.

---

## Mandatory Alignment Addendum (2026-02-13)

This document is retained for reference, but implementation in this repository must follow these mandatory constraints:

- Do not build a gateway service for this POC; implement in-repo MCP runtime only.
- All MCP runtime behavior must be TOML-configured; avoid hardcoded runtime policy values.
- Expand multi-project config and SQLite namespacing (`projects.entries[*].db_namespace`) as first-class requirements.
- Expose exactly one MCP tool (`circular`) to AI clients; all capabilities are operation-routed through it.
- Include automatic MCP-managed file sync for configured outputs (`mermaid`/`plantuml`/`dot`/`tsv`) and project config sync, without AI needing to request each write.
- Keep SoC boundaries with subpackages under `internal/mcp/*` (single primary concern per package).
- Use FastMCP 3 + FastAPI gateway style as the conceptual API-as-tools model reference, but implement conversion/runtime in Go here.
- Use `kin-openapi` for OpenAPI parsing and operation extraction when converting APIs to MCP operations.

Required operation families under single tool:
- analysis/query operations (`scan.run`, `query.*`, `graph.*`)
- system operations (`system.sync_outputs`, `system.sync_config`, `system.select_project`)

---

## Architecture Overview

```
Go MCP Gateway
├── mcp/
│   ├── server.go              # MCP server setup with mark3labs/mcp-go
│   ├── circular_tool.go       # Single "circular" tool registration
│   └── transport.go           # Stdio transport (SSE for later)
├── circular/
│   ├── adapter.go             # Wraps circular's internal APIs
│   ├── operations.go          # Operation registry pattern
│   ├── op_scan.go             # Scan operation handler
│   ├── op_trace.go            # Trace operation handler
│   ├── op_impact.go           # Impact operation handler
│   ├── op_metrics.go          # Metrics operation handler
│   ├── op_violations.go       # Architecture violations handler
│   ├── op_list.go             # Discovery mechanism
│   └── schema_gen.go          # Struct → JSON schema reflection
└── main.go                    # Entry point
```

---

## Core Principles

### 1. API-as-Tools Pattern
- **ONE MCP tool** named "circular"
- Tool accepts `operation` + `params` structure
- Operations are self-describing via `list` operation
- AI discovers capabilities dynamically

### 2. Token Efficiency
- Single tool definition: ~100 tokens in system prompt
- Operations discovered on-demand: ~300 tokens once
- Savings: ~140K tokens over 100-message conversation vs manual tool definitions

### 3. Memory Efficiency (PoC)
- In-memory only for PoC
- Summary stats cached (~50 KB per project)
- Full graph loaded on-demand
- No SQLite/DuckDB/Kuzu needed for PoC
- Production: Lazy-loading with disk-backed storage

### 4. Extensibility
- Adding new operations is trivial
- Struct tags drive schema generation
- Auto-discovery via reflection

---

## Implementation Details

### File 1: `mcp/server.go` (~150 lines)

**Purpose:** Initialize MCP server with mark3labs/mcp-go

```go
package mcp

import (
    "github.com/mark3labs/mcp-go/server"
    "github.com/mark3labs/mcp-go/transport/stdio"
)

type Server struct {
    mcp        *server.MCPServer
    circularTool *CircularTool
}

func NewServer(circularTool *CircularTool) *Server {
    s := &Server{
        circularTool: circularTool,
    }
    
    // Create MCP server
    s.mcp = server.NewMCPServer(
        "circular-gateway",
        "1.0.0",
    )
    
    // Register single circular tool
    s.mcp.AddTool(circularTool.GetToolDefinition())
    
    return s
}

func (s *Server) Start() error {
    // Use stdio transport for PoC
    transport := stdio.NewStdioServerTransport()
    return s.mcp.Connect(transport)
}
```

**Key functions:**
- `NewServer(circularTool)` - Initialize MCP server
- `Start()` - Connect stdio transport
- Register circular tool with server

---

### File 2: `mcp/circular_tool.go` (~100 lines)

**Purpose:** Single MCP tool that routes to operations

```go
package mcp

import (
    "encoding/json"
    "github.com/mark3labs/mcp-go/mcp"
)

type CircularTool struct {
    registry *circular.OperationRegistry
}

func NewCircularTool(registry *circular.OperationRegistry) *CircularTool {
    return &CircularTool{
        registry: registry,
    }
}

func (ct *CircularTool) GetToolDefinition() mcp.Tool {
    return mcp.Tool{
        Name: "circular",
        Description: "Analyze codebase dependencies, cycles, complexity, and architecture violations",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "operation": map[string]interface{}{
                    "type": "string",
                    "enum": ct.registry.GetOperationNames(),
                    "description": "Operation to perform. Use 'list' to discover available operations",
                },
                "params": map[string]interface{}{
                    "type": "object",
                    "description": "Operation-specific parameters",
                },
            },
            "required": []string{"operation"},
        },
    }
}

func (ct *CircularTool) Execute(arguments map[string]interface{}) (interface{}, error) {
    // Extract operation name
    opName, ok := arguments["operation"].(string)
    if !ok {
        return nil, fmt.Errorf("missing or invalid 'operation' field")
    }
    
    // Extract params (optional)
    params := make(map[string]interface{})
    if p, ok := arguments["params"].(map[string]interface{}); ok {
        params = p
    }
    
    // Execute operation via registry
    return ct.registry.Execute(opName, params)
}
```

**Key functions:**
- `GetToolDefinition()` - Returns MCP tool schema
- `Execute(arguments)` - Routes to appropriate operation handler

---

### File 3: `circular/operations.go` (~150 lines)

**Purpose:** Operation registry pattern

```go
package circular

import (
    "fmt"
)

type Operation struct {
    Name        string
    Description string
    Handler     func(params map[string]interface{}) (interface{}, error)
    ParamsType  interface{} // For schema generation
}

type OperationRegistry struct {
    operations map[string]Operation
    adapter    *Adapter
}

func NewOperationRegistry(adapter *Adapter) *OperationRegistry {
    r := &OperationRegistry{
        operations: make(map[string]Operation),
        adapter:    adapter,
    }
    
    // Register all operations
    r.registerCoreOperations()
    
    return r
}

func (r *OperationRegistry) registerCoreOperations() {
    r.Register(Operation{
        Name:        "scan",
        Description: "Run full analysis on project",
        Handler:     r.adapter.handleScan,
        ParamsType:  ScanParams{},
    })
    
    r.Register(Operation{
        Name:        "trace",
        Description: "Find import chain between two modules",
        Handler:     r.adapter.handleTrace,
        ParamsType:  TraceParams{},
    })
    
    r.Register(Operation{
        Name:        "impact",
        Description: "Analyze blast radius for file/module",
        Handler:     r.adapter.handleImpact,
        ParamsType:  ImpactParams{},
    })
    
    r.Register(Operation{
        Name:        "metrics",
        Description: "Get dependency metrics",
        Handler:     r.adapter.handleMetrics,
        ParamsType:  MetricsParams{},
    })
    
    r.Register(Operation{
        Name:        "violations",
        Description: "Check architecture layer-rule violations",
        Handler:     r.adapter.handleViolations,
        ParamsType:  ViolationsParams{},
    })
    
    r.Register(Operation{
        Name:        "list",
        Description: "List all available operations",
        Handler:     r.handleList,
        ParamsType:  nil,
    })
}

func (r *OperationRegistry) Register(op Operation) {
    r.operations[op.Name] = op
}

func (r *OperationRegistry) Execute(opName string, params map[string]interface{}) (interface{}, error) {
    op, exists := r.operations[opName]
    if !exists {
        return nil, fmt.Errorf("unknown operation: %s. Use 'list' to see available operations", opName)
    }
    
    return op.Handler(params)
}

func (r *OperationRegistry) GetOperationNames() []string {
    names := make([]string, 0, len(r.operations))
    for name := range r.operations {
        names = append(names, name)
    }
    return names
}

func (r *OperationRegistry) handleList(params map[string]interface{}) (interface{}, error) {
    ops := make([]map[string]interface{}, 0, len(r.operations))
    
    for _, op := range r.operations {
        opInfo := map[string]interface{}{
            "name":        op.Name,
            "description": op.Description,
        }
        
        // Add schema info if params type exists
        if op.ParamsType != nil {
            opInfo["schema"] = generateSchema(op.ParamsType)
        }
        
        ops = append(ops, opInfo)
    }
    
    return map[string]interface{}{
        "operations": ops,
    }, nil
}
```

**Key functions:**
- `NewOperationRegistry(adapter)` - Initialize and register all operations
- `Register(operation)` - Add operation to registry
- `Execute(name, params)` - Route to handler
- `handleList()` - Return operation catalog for AI discovery

---

### File 4: `circular/adapter.go` (~100 lines)

**Purpose:** Wrapper around circular's internal APIs

```go
package circular

import (
    "github.com/yourusername/circular/internal/core/app"
    "github.com/yourusername/circular/internal/core/config"
    "github.com/yourusername/circular/internal/engine/graph"
    "github.com/yourusername/circular/internal/engine/resolver"
)

type Adapter struct {
    config      *config.Config
    app         *app.App
    currentGraph *graph.Graph
}

func NewAdapter(configPath string) (*Adapter, error) {
    // Load circular config
    cfg, err := config.Load(configPath)
    if err != nil {
        return nil, err
    }
    
    // Initialize circular app
    circularApp, err := app.New(cfg)
    if err != nil {
        return nil, err
    }
    
    return &Adapter{
        config: cfg,
        app:    circularApp,
    }, nil
}

// RunScan executes circular analysis and caches result
func (a *Adapter) RunScan(projectPath string, includeTests bool) (*ScanResult, error) {
    // Update config if projectPath provided
    if projectPath != "" {
        a.config.WatchPaths = []string{projectPath}
    }
    
    // Run circular scan
    g, issues, err := a.app.RunOnce()
    if err != nil {
        return nil, err
    }
    
    // Cache graph
    a.currentGraph = g
    
    // Build result
    return &ScanResult{
        Cycles:      issues.Cycles,
        Unresolved:  issues.Unresolved,
        Unused:      issues.UnusedImports,
        Violations:  issues.ArchViolations,
        Complexity:  issues.ComplexityHotspots,
    }, nil
}

// GetGraph returns cached graph or runs scan
func (a *Adapter) GetGraph() (*graph.Graph, error) {
    if a.currentGraph == nil {
        _, err := a.RunScan("", false)
        if err != nil {
            return nil, err
        }
    }
    return a.currentGraph, nil
}
```

**Key functions:**
- `NewAdapter(configPath)` - Initialize circular wrapper
- `RunScan(path, includeTests)` - Execute analysis
- `GetGraph()` - Cached graph access
- Integration points with circular's internal packages

---

### File 5: `circular/op_scan.go` (~100 lines)

**Purpose:** Scan operation handler

```go
package circular

type ScanParams struct {
    ProjectPath  string   `json:"project_path,omitempty" description:"Project root path (optional if in config)"`
    Scope        []string `json:"scope,omitempty" description:"Filter to specific modules/paths"`
    IncludeTests bool     `json:"include_tests" description:"Analyze test files (default: false)"`
}

type ScanResult struct {
    Summary     ScanSummary        `json:"summary"`
    Cycles      []Cycle            `json:"cycles,omitempty"`
    Unresolved  []UnresolvedRef    `json:"unresolved,omitempty"`
    Unused      []UnusedImport     `json:"unused,omitempty"`
    Violations  []ArchViolation    `json:"violations,omitempty"`
    Complexity  []ComplexityHotspot `json:"complexity,omitempty"`
}

type ScanSummary struct {
    TotalCycles     int    `json:"total_cycles"`
    TotalUnresolved int    `json:"total_unresolved"`
    TotalUnused     int    `json:"total_unused"`
    TotalViolations int    `json:"total_violations"`
    ModulesScanned  int    `json:"modules_scanned"`
    FilesScanned    int    `json:"files_scanned"`
}

type Cycle struct {
    Modules []string `json:"modules"`
}

type UnresolvedRef struct {
    File   string `json:"file"`
    Symbol string `json:"symbol"`
    Line   int    `json:"line"`
}

type UnusedImport struct {
    File   string `json:"file"`
    Import string `json:"import"`
    Line   int    `json:"line"`
}

type ArchViolation struct {
    From string `json:"from"`
    To   string `json:"to"`
    Rule string `json:"rule"`
}

type ComplexityHotspot struct {
    Module     string `json:"module"`
    Complexity int    `json:"complexity"`
}

func (a *Adapter) handleScan(params map[string]interface{}) (interface{}, error) {
    var p ScanParams
    if err := mapToStruct(params, &p); err != nil {
        return nil, err
    }
    
    // Run scan
    result, err := a.RunScan(p.ProjectPath, p.IncludeTests)
    if err != nil {
        return nil, err
    }
    
    // Apply scope filter if provided
    if len(p.Scope) > 0 {
        result = filterByScope(result, p.Scope)
    }
    
    return result, nil
}

// Helper to filter results by scope
func filterByScope(result *ScanResult, scope []string) *ScanResult {
    // Filter cycles, unresolved, etc. to only include items in scope
    // Implementation depends on circular's data structures
    return result
}
```

**Key structures:**
- `ScanParams` - Input parameters with JSON schema tags
