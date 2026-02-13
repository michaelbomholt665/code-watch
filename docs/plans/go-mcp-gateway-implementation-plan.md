# Go MCP Gateway Implementation Plan
## Proof of Concept: Circular Integration with API-as-Tools Pattern

**Target:** ~1000 lines of boilerplate across 6-8 files  
**Library:** `github.com/mark3labs/mcp-go` (most complete Go MCP SDK)  
**Pattern:** Single meta-tool that dynamically exposes operations

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
- `ScanResult` - Structured output
- `handleScan()` - Operation handler

---

### File 6: `circular/op_trace.go` (~80 lines)

**Purpose:** Trace operation handler

```go
package circular

type TraceParams struct {
    FromModule string `json:"from" description:"Source module name"`
    ToModule   string `json:"to" description:"Target module name"`
}

type TraceResult struct {
    Found bool     `json:"found"`
    Chain []string `json:"chain,omitempty"`
}

func (a *Adapter) handleTrace(params map[string]interface{}) (interface{}, error) {
    var p TraceParams
    if err := mapToStruct(params, &p); err != nil {
        return nil, err
    }
    
    // Get graph
    g, err := a.GetGraph()
    if err != nil {
        return nil, err
    }
    
    // Find shortest path
    chain, found := g.FindShortestPath(p.FromModule, p.ToModule)
    
    return &TraceResult{
        Found: found,
        Chain: chain,
    }, nil
}
```

---

### File 7: `circular/op_impact.go` (~80 lines)

**Purpose:** Impact analysis operation

```go
package circular

type ImpactParams struct {
    Target string `json:"target" description:"File path or module name"`
}

type ImpactResult struct {
    Target            string   `json:"target"`
    DirectImporters   []string `json:"direct_importers"`
    TransitiveImporters []string `json:"transitive_importers"`
    TotalImpact       int      `json:"total_impact"`
}

func (a *Adapter) handleImpact(params map[string]interface{}) (interface{}, error) {
    var p ImpactParams
    if err := mapToStruct(params, &p); err != nil {
        return nil, err
    }
    
    // Get graph
    g, err := a.GetGraph()
    if err != nil {
        return nil, err
    }
    
    // Calculate blast radius
    direct, transitive := g.GetImporters(p.Target)
    
    return &ImpactResult{
        Target:              p.Target,
        DirectImporters:     direct,
        TransitiveImporters: transitive,
        TotalImpact:         len(direct) + len(transitive),
    }, nil
}
```

---

### File 8: `circular/op_metrics.go` (~80 lines)

**Purpose:** Dependency metrics operation

```go
package circular

type MetricsParams struct {
    Module string `json:"module,omitempty" description:"Specific module (optional, default: all)"`
}

type MetricsResult struct {
    Modules []ModuleMetrics `json:"modules"`
}

type ModuleMetrics struct {
    Name       string `json:"name"`
    Depth      int    `json:"depth"`
    FanIn      int    `json:"fan_in"`
    FanOut     int    `json:"fan_out"`
    Complexity int    `json:"complexity"`
    FileCount  int    `json:"file_count"`
    FuncCount  int    `json:"func_count"`
}

func (a *Adapter) handleMetrics(params map[string]interface{}) (interface{}, error) {
    var p MetricsParams
    if err := mapToStruct(params, &p); err != nil {
        return nil, err
    }
    
    // Get graph
    g, err := a.GetGraph()
    if err != nil {
        return nil, err
    }
    
    // Calculate metrics
    var metrics []ModuleMetrics
    
    if p.Module != "" {
        // Single module
        m := g.GetModuleMetrics(p.Module)
        metrics = []ModuleMetrics{m}
    } else {
        // All modules
        metrics = g.GetAllModuleMetrics()
    }
    
    return &MetricsResult{
        Modules: metrics,
    }, nil
}
```

---

### File 9: `circular/op_violations.go` (~80 lines)

**Purpose:** Architecture violations operation

```go
package circular

type ViolationsParams struct {
    RuleSet string `json:"rule_set,omitempty" description:"Specific rule set (optional)"`
}

type ViolationsResult struct {
    Violations []ArchViolation `json:"violations"`
    Summary    ViolationsSummary `json:"summary"`
}

type ViolationsSummary struct {
    TotalViolations int               `json:"total_violations"`
    ByRule          map[string]int    `json:"by_rule"`
}

func (a *Adapter) handleViolations(params map[string]interface{}) (interface{}, error) {
    var p ViolationsParams
    if err := mapToStruct(params, &p); err != nil {
        return nil, err
    }
    
    // Get graph
    g, err := a.GetGraph()
    if err != nil {
        return nil, err
    }
    
    // Check architecture rules
    violations := g.ValidateArchitectureRules()
    
    // Build summary
    byRule := make(map[string]int)
    for _, v := range violations {
        byRule[v.Rule]++
    }
    
    return &ViolationsResult{
        Violations: violations,
        Summary: ViolationsSummary{
            TotalViolations: len(violations),
            ByRule:          byRule,
        },
    }, nil
}
```

---

### File 10: `circular/schema_gen.go` (~80 lines)

**Purpose:** Generate JSON schemas from Go structs

```go
package circular

import (
    "encoding/json"
    "reflect"
)

func generateSchema(v interface{}) map[string]interface{} {
    // Use reflection to generate JSON schema from struct tags
    // Can use library like github.com/invopop/jsonschema
    // Or implement simple version for PoC
    
    t := reflect.TypeOf(v)
    schema := map[string]interface{}{
        "type": "object",
        "properties": make(map[string]interface{}),
    }
    
    properties := schema["properties"].(map[string]interface{})
    
    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        
        // Get JSON tag
        jsonTag := field.Tag.Get("json")
        if jsonTag == "" || jsonTag == "-" {
            continue
        }
        
        // Get description tag
        desc := field.Tag.Get("description")
        
        // Build property schema
        prop := map[string]interface{}{
            "type": getJSONType(field.Type),
        }
        if desc != "" {
            prop["description"] = desc
        }
        
        properties[jsonTag] = prop
    }
    
    return schema
}

func getJSONType(t reflect.Type) string {
    switch t.Kind() {
    case reflect.String:
        return "string"
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return "integer"
    case reflect.Bool:
        return "boolean"
    case reflect.Slice:
        return "array"
    default:
        return "object"
    }
}

// Helper to map generic params to typed struct
func mapToStruct(params map[string]interface{}, target interface{}) error {
    // Convert map to JSON then unmarshal to struct
    data, err := json.Marshal(params)
    if err != nil {
        return err
    }
    return json.Unmarshal(data, target)
}
```

---

### File 11: `main.go` (~50 lines)

**Purpose:** Entry point

```go
package main

import (
    "log"
    "os"
    
    "yourproject/mcp"
    "yourproject/circular"
)

func main() {
    // Get config path from env or default
    configPath := os.Getenv("CIRCULAR_CONFIG")
    if configPath == "" {
        configPath = "./data/config/circular.toml"
    }
    
    // Initialize circular adapter
    adapter, err := circular.NewAdapter(configPath)
    if err != nil {
        log.Fatalf("Failed to initialize circular adapter: %v", err)
    }
    
    // Create operation registry
    registry := circular.NewOperationRegistry(adapter)
    
    // Create circular tool
    circularTool := mcp.NewCircularTool(registry)
    
    // Create and start MCP server
    server := mcp.NewServer(circularTool)
    
    log.Println("Starting MCP gateway...")
    if err := server.Start(); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

---

## Integration with Circular

### Required Circular Internal Packages

From circular's codebase, you'll need to import:

```go
import (
    "github.com/yourusername/circular/internal/core/app"
    "github.com/yourusername/circular/internal/core/config"
    "github.com/yourusername/circular/internal/engine/graph"
    "github.com/yourusername/circular/internal/engine/parser"
    "github.com/yourusername/circular/internal/engine/resolver"
)
```

### Key Circular APIs to Use

**From `internal/core/app`:**
- `app.New(config)` - Initialize circular app
- `app.RunOnce()` - Execute single scan

**From `internal/engine/graph`:**
- `graph.Graph` - Dependency graph structure
- `graph.FindShortestPath(from, to)` - Trace operation
- `graph.GetImporters(module)` - Impact analysis
- `graph.GetModuleMetrics(name)` - Metrics

**From `internal/engine/resolver`:**
- Unresolved reference detection results

---

## AI Usage Examples

### Example 1: Discovery
```json
{
  "operation": "list"
}
```

**Response:**
```json
{
  "operations": [
    {
      "name": "scan",
      "description": "Run full analysis on project",
      "schema": {
        "type": "object",
        "properties": {
          "project_path": {"type": "string", "description": "..."},
          "scope": {"type": "array", "description": "..."},
          "include_tests": {"type": "boolean", "description": "..."}
        }
      }
    },
    // ... other operations
  ]
}
```

### Example 2: Scan Project
```json
{
  "operation": "scan",
  "params": {
    "project_path": "/home/user/myproject",
    "include_tests": false
  }
}
```

**Response:**
```json
{
  "summary": {
    "total_cycles": 2,
    "total_unresolved": 5,
    "total_unused": 3,
    "total_violations": 0,
    "modules_scanned": 14,
    "files_scanned": 47
  },
  "cycles": [
    {
      "modules": ["api/handlers", "internal/db", "api/handlers"]
    }
  ],
  "unresolved": [
    {
      "file": "api/handlers/users.go",
      "symbol": "ValidateEmail",
      "line": 42
    }
  ]
}
```

### Example 3: Trace Import Chain
```json
{
  "operation": "trace",
  "params": {
    "from": "cmd/circular",
    "to": "internal/engine/graph"
  }
}
```

**Response:**
```json
{
  "found": true,
  "chain": [
    "cmd/circular",
    "internal/ui/cli",
    "internal/core/app",
    "internal/engine/graph"
  ]
}
```

### Example 4: Impact Analysis
```json
{
  "operation": "impact",
  "params": {
    "target": "internal/engine/graph"
  }
}
```

**Response:**
```json
{
  "target": "internal/engine/graph",
  "direct_importers": [
    "internal/core/app",
    "internal/ui/cli",
    "internal/ui/report"
  ],
  "transitive_importers": [
    "cmd/circular",
    "internal/ui/report/formats"
  ],
  "total_impact": 5
}
```

---

## Testing Strategy

### Unit Tests
```go
// circular/operations_test.go
func TestOperationRegistry_Execute(t *testing.T) {
    adapter := &Adapter{} // Mock adapter
    registry := NewOperationRegistry(adapter)
    
    // Test list operation
    result, err := registry.Execute("list", nil)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration Tests
```go
// Test with actual circular analysis
func TestScanOperation_RealProject(t *testing.T) {
    adapter, _ := NewAdapter("./testdata/circular.toml")
    result, err := adapter.handleScan(map[string]interface{}{
        "project_path": "./testdata/sample-project",
    })
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### MCP End-to-End Test
```bash
# Test via stdio
echo '{"operation": "list"}' | go run main.go
```

---

## Production Considerations (Future)

### Multi-Project Support
```go
type ProjectManager struct {
    projects map[string]*Adapter
    mu       sync.RWMutex
}

func (pm *ProjectManager) GetOrCreate(projectPath string) *Adapter {
    // Thread-safe project instance management
}
```

### Watcher Integration
```go
type WatcherBridge struct {
    adapter   *Adapter
    watcher   *watcher.Watcher
    debouncer *Debouncer
}

func (wb *WatcherBridge) Start() {
    // Watch for file changes
    // Trigger incremental scans
    // Update cached graph
}
```

### Notification Queue (Phase-Gated Context)
```go
type NotificationQueue struct {
    pending []CircularUpdate
    mu      sync.Mutex
}

func (nq *NotificationQueue) EnqueueOnChange(update CircularUpdate) {
    // Queue updates for AI context injection
}

func (nq *NotificationQueue) GetPendingForPhase(phase int) []CircularUpdate {
    // Return relevant updates for current workflow phase
}
```

### Multi-Agent Configuration
```toml
# gateway/config.toml
[agents.architect]
scope_strategy = "full"
[agents.architect.injection]
critical = "immediate"
normal = "phase_transition"

[agents.implementer]
scope_strategy = "module"
[agents.implementer.injection]
critical = "tool_response"
normal = "task_completion"
```

### Disk-Backed Storage (DuckDB/Parquet/Kuzu)
```go
type Adapter struct {
    config      *config.Config
    db          *duckdb.DB  // For production scale
    summaryStats CircularSummary  // In-memory (~50 KB)
}

func (a *Adapter) GetGraph() (*graph.Graph, error) {
    // Lazy-load from DuckDB/Parquet
    return a.loadGraphFromDisk()
}
```

---

## Line Count Summary

| File | Lines | Purpose |
|------|-------|---------|
| `mcp/server.go` | ~150 | MCP server setup |
| `mcp/circular_tool.go` | ~100 | Single tool routing |
| `circular/operations.go` | ~150 | Operation registry |
| `circular/adapter.go` | ~100 | Circular API wrapper |
| `circular/op_scan.go` | ~100 | Scan handler |
| `circular/op_trace.go` | ~80 | Trace handler |
| `circular/op_impact.go` | ~80 | Impact handler |
| `circular/op_metrics.go` | ~80 | Metrics handler |
| `circular/op_violations.go` | ~80 | Violations handler |
| `circular/schema_gen.go` | ~80 | Schema generation |
| `main.go` | ~50 | Entry point |
| **Total** | **~1050** | **Well within target** |

---

## Next Steps for Implementation

1. **Set up project structure** with files above
2. **Install dependencies:**
   ```bash
   go get github.com/mark3labs/mcp-go
   ```
3. **Implement core files in order:**
   - Start with `adapter.go` (wraps circular)
   - Then `operations.go` (registry pattern)
   - Then operation handlers (`op_*.go`)
   - Then MCP layer (`server.go`, `circular_tool.go`)
   - Finally `main.go`
4. **Test with circular on small project:**
   ```bash
   go run main.go
   # In another terminal:
   echo '{"operation": "list"}' | nc localhost 5000
   ```
5. **Iterate on operation schemas** based on AI feedback

---

## Key Design Wins

✅ **Token Efficiency:** Single tool vs 6+ manual tools saves ~140K tokens over 100 messages  
✅ **Extensibility:** Adding operations is trivial (1 struct + 1 handler)  
✅ **Memory Efficiency:** In-memory for PoC, lazy-loading ready for production  
✅ **Discovery:** AI learns API dynamically via `list` operation  
✅ **Clean Architecture:** Registry pattern separates concerns  
✅ **Production Ready:** Extends naturally to multi-agent, watcher, disk-backed storage  

---

## Dependencies

```go
// go.mod
module yourproject/circular-gateway

go 1.24

require (
    github.com/mark3labs/mcp-go v0.x.x
    github.com/yourusername/circular v1.0.0
)
```

---

## Configuration

```toml
# gateway/config.toml (for PoC, minimal)
[mcp]
transport = "stdio"

[circular]
config_path = "./data/config/circular.toml"
```

For production, extend with agent configs, project registry, etc.

---

**End of Implementation Plan**

This document provides complete architecture for ~1000-line PoC that extends cleanly to production multi-agent system. Focus on API-as-Tools pattern for maximum flexibility and token efficiency.
