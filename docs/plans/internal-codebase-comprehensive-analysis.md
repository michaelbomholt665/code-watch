# Code-Watch Internal Codebase Comprehensive Analysis Report

**Generated:** 2026-02-20  
**Scope:** `internal/` directory  
**Architecture:** Hexagonal Architecture (Ports & Adapters)

---

## Improvements Since Previous Analysis

Comparing with [`internal-codebase-analysis-report.md`](internal-codebase-analysis-report.md), significant improvements have been made:

### ✅ MAJOR: God Object Decomposition

| File | Previous Size | Current Size | Reduction |
|------|--------------|--------------|-----------|
| [`app.go`](internal/core/app/app.go:1) | 38,518 chars | 4,896 chars | **87%** |
| [`config.go`](internal/core/config/config.go:1) | 27,693 chars | 8,662 chars | **69%** |

**New Decomposed Files in `internal/core/app/`:**
- [`analyzer.go`](internal/core/app/analyzer.go:1) (6,949 chars) - Analysis operations
- [`output.go`](internal/core/app/output.go:1) (7,762 chars) - Output generation
- [`scanner.go`](internal/core/app/scanner.go:1) (4,777 chars) - File scanning
- [`presentation_service.go`](internal/core/app/presentation_service.go:1) (6,525 chars) - Presentation logic
- [`reporting.go`](internal/core/app/reporting.go:1) (2,830 chars) - Report generation
- [`caches.go`](internal/core/app/caches.go:1) (1,426 chars) - Cache management
- [`gomod.go`](internal/core/app/gomod.go:1) (1,626 chars) - Go module handling
- [`impact_report.go`](internal/core/app/impact_report.go:1) (1,053 chars) - Impact analysis
- [`output_targets.go`](internal/core/app/output_targets.go:1) (1,409 chars) - Output target resolution
- [`symbol_store.go`](internal/core/app/symbol_store.go:1) (1,280 chars) - Symbol store integration
- [`watch.go`](internal/core/app/watch.go:1) (439 chars) - Watch mode
- [`content_cache.go`](internal/core/app/content_cache.go:1) (587 chars) - File content caching

### ✅ NEW: LRU Cache Infrastructure

- [`lru.go`](internal/engine/graph/lru.go:1) (2,947 chars) - Generic thread-safe LRU cache
- [`lru_test.go`](internal/engine/graph/lru_test.go:1) (4,452 chars) - Comprehensive tests
- **Status:** Implemented but NOT YET integrated into Graph/App

### ✅ NEW: SQLite Symbol Store

- [`symbol_store.go`](internal/engine/graph/symbol_store.go:1) (19,188 chars) - SQLite-backed symbol persistence
- [`symbol_store_test.go`](internal/engine/graph/symbol_store_test.go:1) (3,862 chars) - Tests
- Enables persistence for large codebases, reduces memory pressure

### ✅ NEW: MCP Overlays Tool

- [`overlays/handler.go`](internal/mcp/tools/overlays/handler.go:1) (6,968 chars) - AI-verified annotations
- [`overlays/handler_test.go`](internal/mcp/tools/overlays/handler_test.go:1) (3,086 chars) - Tests
- Supports `EXCLUSION`, `VETTED_USAGE`, `RE-ALIAS` overlay types
- Allows AI agents to persist verification decisions

### ✅ NEW: Surgical Report Generation

- [`surgical.go`](internal/ui/report/surgical.go:1) - Precise symbol usage context
- [`surgical_test.go`](internal/ui/report/surgical_test.go:1) - Tests
- Returns snippets with surrounding context (±5 lines)
- Supports semantic tagging for richer results

### ✅ NEW: Graph Infrastructure

- [`storage.go`](internal/engine/graph/storage.go:1) (1,965 chars) - `NodeStorage` interface for persistence
- [`writer.go`](internal/engine/graph/writer.go:1) (4,774 chars) - Graph output utilities
- [`importance.go`](internal/engine/graph/importance.go:1) (1,251 chars) - Module importance scoring

---

## Executive Summary

The `internal/` directory implements a sophisticated **Code Dependency Monitor** called "Circular" using hexagonal architecture. The system parses source code in multiple languages, builds dependency graphs, detects cycles, identifies unresolved references ("hallucinations"), finds unused imports, detects secrets, and provides both CLI and MCP (Model Context Protocol) interfaces for AI assistant integration.

### Key Capabilities at a Glance

| Capability | Status | Implementation Quality |
|------------|--------|----------------------|
| Multi-language parsing | ✅ Mature | Tree-sitter based, 9+ languages |
| Dependency graph analysis | ✅ Mature | Thread-safe, cycle detection |
| Cross-language resolution | ✅ Good | Probabilistic + explicit bridges |
| Secret detection | ✅ Good | Pattern + entropy based |
| MCP integration | ✅ Mature | STDIO transport, 10+ tools |
| History/Trends | ✅ Good | SQLite-backed |
| Output formats | ✅ Mature | DOT, Mermaid, PlantUML, SARIF, TSV |
| Architecture validation | ✅ Good | Layer rules engine |

---

## Architecture Overview

```mermaid
graph TB
    subgraph Core [internal/core - Application Layer]
        ports[ports/ports.go<br/>Interface Definitions]
        app[app/*.go<br/>Application Service]
        config[config/*.go<br/>Configuration]
        watcher[watcher/*.go<br/>File Watcher]
    end
    
    subgraph Engine [internal/engine - Domain Layer]
        parser[parser/*.go<br/>Tree-Sitter Parsing]
        graph[graph/*.go<br/>Dependency Graph]
        resolver[resolver/*.go<br/>Symbol Resolution]
        secrets[secrets/*.go<br/>Secret Detection]
    end
    
    subgraph Data [internal/data - Infrastructure Layer]
        history[history/*.go<br/>SQLite History]
        query[query/*.go<br/>Query Service]
    end
    
    subgraph MCP [internal/mcp - Interface Layer]
        runtime[runtime/*.go<br/>MCP Server]
        tools[tools/*/<br/>MCP Tools]
        transport[transport/*.go<br/>STDIO Transport]
    end
    
    subgraph UI [internal/ui - Interface Layer]
        cli[cli/*.go<br/>CLI Runtime]
        report[report/*.go<br/>Report Generation]
    end
    
    ports --> app
    app --> parser
    app --> graph
    app --> resolver
    app --> secrets
    app --> history
    app --> query
    app --> cli
    app --> runtime
    runtime --> tools
    runtime --> transport
    cli --> report
```

---

## Detailed Component Analysis

### 1. Core Layer (`internal/core/`)

#### [`ports/ports.go`](internal/core/ports/ports.go:1)

**Purpose:** Defines all interfaces (Ports) following hexagonal architecture principles.

**Key Interfaces:**

| Interface | Methods | Purpose |
|-----------|---------|---------|
| [`CodeParser`](internal/core/ports/ports.go:14) | 5 | Parse source files, check language support |
| [`SecretScanner`](internal/core/ports/ports.go:24) | 1 | Detect secrets in file content |
| [`HistoryStore`](internal/core/ports/ports.go:41) | 2 | Persist and load analysis snapshots |
| [`QueryService`](internal/core/ports/ports.go:102) | 4 | Read-only dependency queries |
| [`WatchService`](internal/core/ports/ports.go:119) | 3 | File watching lifecycle |
| [`AnalysisService`](internal/core/ports/ports.go:126) | 10 | Main driving port for scan/query operations |

**Strengths:**
- Clean separation between driving and driven ports
- Well-defined DTOs for requests/responses (`ScanRequest`, `SyncOutputsRequest`, etc.)
- Context-aware interfaces for cancellation support
- Comprehensive [`SummarySnapshot`](internal/core/ports/ports.go:83) for state capture

**Improvements Needed:**
- Missing documentation for interface contracts (preconditions/postconditions)
- No error type definitions for domain-specific errors
- [`SummarySnapshot`](internal/core/ports/ports.go:83) mixes concerns (cycles, secrets, violations) - could be decomposed
- No versioning strategy for interfaces

---

#### [`app/app.go`](internal/core/app/app.go:1)

**Purpose:** Central application orchestrator managing parsing, graph building, and analysis.

**Struct Fields:**
```go
type App struct {
    Config        *config.Config
    codeParser    ports.CodeParser
    Graph         *graph.Graph
    secretScanner ports.SecretScanner
    symbolStore   *graph.SQLiteSymbolStore
    archEngine    *graph.LayerRuleEngine
    goModCache    map[string]goModuleCacheEntry
    IncludeTests  bool
    // ... synchronization primitives
}
```

**Capabilities:**

| Method | Function | Complexity |
|--------|----------|------------|
| [`New()`](internal/core/app/app.go:70) | Factory with default parser | Medium |
| [`NewWithDependencies()`](internal/core/app/app.go:90) | Dependency injection constructor | Medium |
| `InitialScan()` | Scan all configured paths | High |
| `ProcessFile()` | Parse and add file to graph | Medium |
| `HandleChanges()` | React to file system changes | High |
| `AnalyzeHallucinations()` | Find unresolved references | Medium |
| `AnalyzeUnusedImports()` | Detect unused imports | Medium |
| `ArchitectureViolations()` | Check layer rules | Medium |

**Strengths:**
- Comprehensive module resolution for Go and Python
- Transitive invalidation for incremental updates
- Caching strategies (go.mod cache, file content cache)
- Thread-safe with proper RWMutex usage
- Clean dependency injection pattern

**Improvements Needed:**
- **Large File:** Multiple files in app/ package indicate potential God Object pattern
- **Mixed Concerns:** Parsing, graph, secrets, output all in one struct
- **Error Handling:** Many errors logged but not propagated to callers
- **Testability:** Direct filesystem access makes testing difficult
- **No Interface:** App struct is concrete, making mocking difficult

---

#### [`app/service.go`](internal/core/app/service.go:1)

**Purpose:** Implements [`AnalysisService`](internal/core/ports/ports.go:126) port.

**Key Methods:**
- [`RunScan()`](internal/core/app/service.go:33) - Execute scan with path filtering
- [`TraceImportChain()`](internal/core/app/service.go:73) - Find dependency path between modules
- [`AnalyzeImpact()`](internal/core/app/service.go:83) - Analyze change impact
- [`DetectCycles()`](internal/core/app/service.go:93) - Cycle detection with limits
- [`CaptureHistoryTrend()`](internal/core/app/service.go:126) - Save snapshot and compute trends
- [`SyncOutputs()`](internal/core/app/service.go:286) - Generate output files

**Strengths:**
- Clean implementation of port interface
- Context cancellation support throughout
- Proper error wrapping

**Improvements Needed:**
- Duplicate validation logic across methods (`if s.app == nil`)
- No metrics/observability hooks
- Missing rate limiting for expensive operations
- No caching for repeated queries

---

#### [`config/config.go`](internal/core/config/config.go:1)

**Purpose:** Configuration management with TOML parsing.

**Configuration Sections:**

| Section | Fields | Purpose |
|---------|--------|---------|
| [`Paths`](internal/core/config/config.go:38) | 5 | Project root, config/state directories |
| [`Database`](internal/core/config/config.go:51) | 6 | SQLite database settings |
| [`Projects`](internal/core/config/config.go:59) | 3 | Multi-project support |
| [`MCP`](internal/core/config/config.go:72) | 14 | MCP server configuration |
| [`Languages`](internal/core/config/config.go:95) | 3 | Language-specific overrides |
| [`DynamicGrammars`](internal/core/config/config.go:28) | 6 | Runtime grammar loading |
| [`Architecture`](internal/core/config/config.go:171) | 4 | Layer rules for validation |
| [`Secrets`](internal/core/config/config.go:189) | 6 | Secret detection settings |

**Strengths:**
- Comprehensive configuration options
- Support for multiple projects with namespacing
- Dynamic grammar configuration for extensibility
- Architecture layer rules for validation
- Helper methods for default values

**Improvements Needed:**
- No configuration validation on load (validation is separate in [`validator.go`](internal/core/config/validator.go:1))
- Missing environment variable overrides
- No hot-reload for configuration changes
- No configuration migration between versions

---

#### [`config/validator.go`](internal/core/config/validator.go:1)

**Purpose:** Configuration validation with detailed error messages.

**Validation Functions:**

| Function | Validates |
|----------|-----------|
| [`validateVersion()`](internal/core/config/validator.go:12) | Config version (1 or 2) |
| [`validateDatabase()`](internal/core/config/validator.go:22) | SQLite driver, path, project mode |
| [`validateProjects()`](internal/core/config/validator.go:37) | Project entries, namespaces |
| [`validateMCP()`](internal/core/config/validator.go:79) | Mode, transport, allowlist |
| [`validateOutput()`](internal/core/config/validator.go:150) | Output paths, verbosity |
| [`validateSecrets()`](internal/core/config/validator.go:199) | Entropy threshold, patterns |
| [`validateArchitecture()`](internal/core/config/validator.go:231) | Layers, rules, overlaps |
| [`validateResolver()`](internal/core/config/validator.go:352) | Bridge scoring thresholds |

**Strengths:**
- Comprehensive validation coverage
- Detailed error messages with field references
- Overlap detection for architecture layer paths
- Regex validation for secret patterns

**Improvements Needed:**
- No warning-level validations (only errors)
- No deprecation warnings for old config patterns

---

### 2. Engine Layer (`internal/engine/`)

#### [`parser/parser.go`](internal/engine/parser/parser.go:1)

**Purpose:** Tree-sitter based source code parsing orchestration.

**Key Components:**
- [`Parser`](internal/engine/parser/parser.go:16) struct with language registry
- [`Extractor`](internal/engine/parser/parser.go:24) interface for language-specific extraction
- [`RawExtractor`](internal/engine/parser/parser.go:28) interface for grammar-less extraction

**Supported Languages:**
| Language | Extractor File | Complexity |
|----------|---------------|------------|
| Go | `profile_extractors.go` | High |
| Python | `profile_extractors.go` | High |
| JavaScript | `profile_extractors.go` | High |
| TypeScript | `profile_extractors.go` | High |
| TSX | `profile_extractors.go` | High |
| Java | `profile_extractors.go` | Medium |
| Rust | `profile_extractors.go` | Medium |
| HTML | `profile_extractors.go` | Low |
| CSS | `profile_extractors.go` | Low |

**Capabilities:**
- Import extraction with aliases and items
- Definition extraction (functions, classes, methods, variables)
- Reference tracking for resolution
- Complexity metrics (branch count, nesting depth, LOC)
- Local symbol tracking for scope-aware resolution

**Strengths:**
- Tree-sitter provides fast, incremental parsing capability
- Language-specific extractors with consistent interface
- Dynamic grammar support for extensibility
- Complexity metrics for hotspot detection
- Test file detection

**Improvements Needed:**
- **Large Extractor File:** [`profile_extractors.go`](internal/engine/parser/profile_extractors.go:1) is 24,191 chars - needs decomposition
- **Missing Languages:** No C/C++, C#, Ruby, PHP, Kotlin, Swift extractors
- **Error Context:** Parse errors lack source context (line/column)
- **No Incremental Parsing:** Re-parses entire file on change despite tree-sitter support
- **Grammar Pool:** Parser pool exists but underutilized

---

#### [`parser/types.go`](internal/engine/parser/types.go:1)

**Purpose:** Data structures for parsed artifacts.

**Key Types:**

| Type | Fields | Purpose |
|------|--------|---------|
| [`File`](internal/engine/parser/types.go:8) | 10 | Parsed file representation |
| [`Import`](internal/engine/parser/types.go:22) | 8 | Import statement with metadata |
| [`Definition`](internal/engine/parser/types.go:32) | 14 | Symbol definition with complexity |
| [`Reference`](internal/engine/parser/types.go:51) | 6 | Symbol reference with resolution status |
| [`Secret`](internal/engine/parser/types.go:59) | 6 | Detected secret with severity |

**Strengths:**
- Comprehensive metadata for each artifact
- Complexity metrics embedded in definitions
- Location tracking for all artifacts

**Improvements Needed:**
- No generic/parameter support in definitions
- No type signature storage for functions
- No call graph information in references

---

#### [`graph/graph.go`](internal/engine/graph/graph.go:1)

**Purpose:** Dependency graph construction and analysis.

**Key Structures:**
```go
type Graph struct {
    mu sync.RWMutex
    files       map[string]*parser.File
    modules     map[string]*Module
    imports     map[string]map[string]*ImportEdge
    importedBy  map[string]map[string]bool
    definitions map[string]map[string]*parser.Definition
    dirty       map[string]bool
}
```

**Capabilities:**

| Method | Function | Algorithm |
|--------|----------|-----------|
| `AddFile()` | Add/update file in graph | Incremental |
| `RemoveFile()` | Remove file and cleanup | Incremental |
| `DetectCycles()` | Find circular dependencies | DFS + Tarjan SCC |
| `FindImportChain()` | Find path between modules | BFS |
| `ComputeModuleMetrics()` | Calculate fan-in/fan-out | Graph traversal |
| `TopComplexity()` | Find complexity hotspots | Sort by score |

**Strengths:**
- Thread-safe with RWMutex
- Efficient cycle detection using Tarjan's algorithm
- Bidirectional tracking (imports/importedBy)
- Dirty tracking for incremental updates
- Module metrics with importance scoring

**LRU Cache Status:**
The [`LRUCache[K, V]`](internal/engine/graph/lru.go:18) is a complete, thread-safe, generic LRU cache implementation with:
- [`Get()`](internal/engine/graph/lru.go:45) - retrieves and promotes to most-recently used
- [`Put()`](internal/engine/graph/lru.go:60) - inserts with automatic LRU eviction
- [`Evict()`](internal/engine/graph/lru.go:83) - explicit key removal
- [`Clear()`](internal/engine/graph/lru.go:108) - reset cache
- Comprehensive test coverage in [`lru_test.go`](internal/engine/graph/lru_test.go:1)

**⚠️ Critical Gap:** The LRU cache is **implemented but NOT integrated** into the Graph or App. All files are still kept in memory without any eviction strategy. The cache exists as infrastructure but is unused.

**Improvements Needed:**
- **LRU Not Integrated:** [`LRUCache`](internal/engine/graph/lru.go:18) is implemented but NOT used - all files kept in memory without eviction
- **No Graph Persistence:** Rebuilds on restart (symbol store helps but not complete)
- **Cycle Detection:** Reports all cycles without prioritization
- **Missing Algorithms:** No centrality measures, community detection, or impact scoring
- **Clone Operations:** Deep cloning on every read is expensive

---

#### [`graph/symbol_store.go`](internal/engine/graph/symbol_store.go:1)

**Purpose:** SQLite-backed symbol persistence for large codebases.

**Capabilities:**
- Symbol persistence across restarts
- Project-key namespacing
- Incremental updates (upsert/delete)
- WAL mode for concurrency

**Strengths:**
- Reduces memory pressure for large codebases
- Supports incremental updates
- Transaction-based operations

**Improvements Needed:**
- No schema migration strategy
- No connection pooling configuration
- No query optimization for lookups

---

#### [`resolver/resolver.go`](internal/engine/resolver/resolver:1)

**Purpose:** Symbol resolution and reference validation.

**Resolution Pipeline:**
1. Local symbol check (vars, params)
2. Explicit bridge mappings (`.circular-bridge.toml`)
3. Standard library recognition
4. Qualified reference resolution
5. Built-in recognition
6. Probabilistic cross-language resolution

**Capabilities:**

| Feature | Implementation |
|---------|---------------|
| Local resolution | Scope-aware symbol table |
| Stdlib recognition | Per-language stdlib sets |
| Cross-language bridges | Explicit + probabilistic |
| Unused import detection | Reference counting |
| Hallucination detection | Unresolved reference tracking |

**Strengths:**
- Multi-language stdlib knowledge (Go, Python, JS, Java, Rust)
- Probabilistic resolution reduces false positives
- Explicit bridge configuration for FFI/service boundaries
- Incremental analysis support

**Improvements Needed:**
- **False Positives:** Probabilistic matching can be inaccurate
- **No Type Information:** Lacks type-aware resolution
- **Limited Cross-File:** Struggles with dynamic languages
- **Stdlib Coverage:** Incomplete stdlib definitions
- **No Caching:** Re-resolves on every analysis

---

#### [`secrets/detector.go`](internal/engine/secrets/detector.go:1)

**Purpose:** Secret detection in source code.

**Detection Strategies:**

| Strategy | Method | Use Case |
|----------|--------|----------|
| Pattern matching | Regex patterns | Known secret formats |
| Entropy analysis | Shannon entropy | High-randomness strings |
| Context detection | Variable names | `password=`, `api_key=` |

**Built-in Patterns:**
- AWS Access Key ID (`AKIA...`)
- GitHub PATs (classic and fine-grained)
- Stripe Live Secrets
- Slack Tokens
- Private Key Blocks

**Strengths:**
- Multiple detection strategies
- Configurable thresholds
- Incremental scanning via line ranges
- Severity classification
- Custom pattern support

**Improvements Needed:**
- **False Positives:** Entropy-based detection is noisy
- **No Verification:** Can't verify if secrets are active
- **Limited Patterns:** Missing many cloud provider keys (GCP, Azure, etc.)
- **No Baseline:** Can't mark known secrets as acceptable
- **No Secret Rotation:** No support for rotated/revoked secrets

---

### 3. Data Layer (`internal/data/`)

#### [`history/store.go`](internal/data/history/store.go:1)

**Purpose:** Historical snapshot storage for trend analysis.

**Schema:**
```sql
CREATE TABLE snapshots (
  project_key TEXT,
  schema_version INTEGER,
  ts_utc TEXT,
  commit_hash TEXT,
  commit_ts_utc TEXT,
  module_count INTEGER,
  file_count INTEGER,
  cycle_count INTEGER,
  unresolved_count INTEGER,
  unused_import_count INTEGER,
  violation_count INTEGER,
  hotspot_count INTEGER,
  avg_fan_in REAL,
  avg_fan_out REAL,
  max_fan_in INTEGER,
  max_fan_out INTEGER,
  PRIMARY KEY (project_key, ts_utc, commit_hash)
);
```

**Strengths:**
- WAL mode for concurrency
- Retry logic for busy database
- Git commit correlation
- Trend analysis support

**Improvements Needed:**
- **No Migration Strategy:** Schema changes not handled
- **Limited Metrics:** Missing code churn, complexity trends
- **No Cleanup:** Old snapshots never purged
- **No Indexing:** Missing indexes for common queries

---

#### [`query/service.go`](internal/data/query/service.go:1)

**Purpose:** Read-only query service for dependency analysis.

**Capabilities:**

| Method | Function |
|--------|----------|
| [`ListModules()`](internal/data/query/service.go:31) | List modules with filtering |
| [`ModuleDetails()`](internal/data/query/service.go:74) | Get module with dependencies |
| [`DependencyTrace()`](internal/data/query/service.go:141) | Find path between modules |
| [`TrendSlice()`](internal/data/query/service.go:163) | Get historical snapshots |
| [`ExecuteCQL()`](internal/data/query/service.go:192) | Execute CQL queries |

**CQL (Circular Query Language):**
- SQL-like syntax for querying modules
- Supports conditions on: `name`, `fan_in`, `fan_out`, `depth`, `file_count`
- Operators: `=`, `!=`, `>`, `>=`, `<`, `<=`, `contains`

**Strengths:**
- Clean API design
- Context-aware for cancellation
- Sorted, consistent output
- Custom query language (CQL)

**Improvements Needed:**
- **Limited Queries:** No full-text search, no impact queries
- **No Pagination:** Large result sets not handled
- **No Caching:** Repeated queries not optimized
- **CQL Limited:** Only supports `modules` target

---

### 4. MCP Layer (`internal/mcp/`)

#### [`runtime/server.go`](internal/mcp/runtime/server.go:1)

**Purpose:** MCP server implementation for AI assistant integration.

**Server Structure:**
```go
type Server struct {
    cfg       *config.Config
    deps      Dependencies
    project   ProjectContext
    registry  *registry.Registry
    transport transport.Adapter
    adapter   *adapters.Adapter
    watch     ports.WatchService
    history   historyStore
    allowlist OperationAllowlist
    toolName  string
}
```

**Capabilities:**
- STDIO transport for process-based communication
- Tool registration and dispatch
- Project context management
- Operation allowlist for security
- Auto-managed outputs
- Config synchronization

**Strengths:**
- Clean separation of concerns
- Allowlist for security
- Project context isolation
- Graceful shutdown support

**Improvements Needed:**
- **Single Transport:** Only STDIO, no HTTP/SSE
- **No Authentication:** No auth mechanism
- **Limited Observability:** No metrics/logging hooks
- **No Rate Limiting:** Vulnerable to DoS

---

#### [`adapters/adapter.go`](internal/mcp/adapters/adapter.go:1)

**Purpose:** Tool adapter connecting MCP tools to analysis service.

**Exposed Operations:**

| Operation | Input | Output |
|-----------|-------|--------|
| `RunScan` | paths | files_scanned, modules, warnings |
| `ScanSecrets` | paths | files_scanned, secret_count, findings |
| `ListSecrets` | limit | secret_count, findings |
| `Cycles` | limit | cycle_count, cycles |
| `ListModules` | filter, limit | modules |
| `ModuleDetails` | module_name | module details |
| `Trace` | from, to | path |
| `Impact` | path | impact report |
| `SyncOutputs` | formats | written paths |
| `Unresolved` | limit | unresolved references |
| `UnusedImports` | limit | unused imports |

**Strengths:**
- Thread-safe with mutex protection
- Consistent error handling
- History integration

**Improvements Needed:**
- No request validation
- No response size limits
- No timeout handling per operation

---

### 5. UI Layer (`internal/ui/`)

#### [`cli/runtime.go`](internal/ui/cli/runtime.go:1)

**Purpose:** Command-line interface runtime.

**Operation Modes:**

| Mode | Flag | Description |
|------|------|-------------|
| Single scan | `--once` | Scan and exit |
| Watch mode | `--ui` | Continuous monitoring with TUI |
| MCP server | `--mcp` | Start MCP server |
| Report | `--report-markdown` | Generate markdown report |
| History | `--history` | Show trend analysis |
| Grammar verify | `--verify-grammars` | Verify grammar artifacts |

**Strengths:**
- Multiple operation modes
- Signal handling for graceful shutdown
- Mode compatibility validation
- Logging configuration

**Improvements Needed:**
- **Large File:** `runtime.go` is 23,785 chars - needs decomposition
- **No Progress Indication:** Long scans lack feedback
- **Limited CLI Help:** Could use better documentation
- **No Shell Completion:** Missing auto-completion support

---

#### [`report/formats/mermaid.go`](internal/ui/report/formats/mermaid.go:1)

**Purpose:** Mermaid diagram generation.

**Capabilities:**
- Flowchart diagrams (LR direction)
- Architecture layer subgraphs
- Cycle highlighting
- Complexity hotspot styling
- External module aggregation

**Strengths:**
- Multiple output formats
- Architecture layer visualization
- Cycle and violation highlighting
- Configurable styling

**Improvements Needed:**
- **Large File:** `mermaid.go` is 24,679 chars
- **No Interactive Output:** Static diagrams only
- **Limited Customization:** Hard-coded styling
- **No Sequence Diagrams:** Only flowcharts

---

## Cross-Cutting Concerns

### Testing

**Coverage Areas:**
- Parser extractors (unit tests)
- Graph operations (unit tests)
- Resolver logic (unit tests)
- Secret detection (unit tests)
- MCP tools (unit tests)
- CLI runtime (unit tests)

**Gaps:**
- Integration tests limited
- No performance benchmarks (except history)
- No fuzz testing for parsers
- No end-to-end tests for MCP

### Error Handling

**Current State:**
- Errors often logged but not propagated
- No structured error types
- Missing error context in many places
- Inconsistent error wrapping

**Recommendations:**
- Define domain error types in `internal/core/errors/`
- Use sentinel errors and error wrapping consistently
- Add error codes for MCP responses
- Implement error context with source location

### Concurrency

**Current State:**
- RWMutex usage throughout (App, Graph, Watcher)
- Sync primitives for update handlers
- SQLite WAL mode for database concurrency
- Single connection limit for SQLite

**Gaps:**
- No context timeout propagation in long operations
- Potential deadlocks not analyzed
- No rate limiting for expensive operations
- No concurrent scan support

### Observability

**Current State:**
- `slog` for structured logging
- No metrics collection
- No tracing
- No health check endpoints

**Recommendations:**
- Add OpenTelemetry integration
- Expose Prometheus metrics
- Add request tracing for MCP
- Add health check endpoint for server mode

---

## Priority Improvements

### High Priority

1. **Decompose Large Files**
   - [`profile_extractors.go`](internal/engine/parser/profile_extractors.go:1) (24,191 chars) → Per-language files
   - [`runtime.go`](internal/ui/cli/runtime.go:1) (23,785 chars) → Split into mode files
   - [`mermaid.go`](internal/ui/report/formats/mermaid.go:1) (24,679 chars) → Split into components
   - [`symbol_store.go`](internal/engine/graph/symbol_store.go:1) (19,188 chars) → Split schema/operations

2. **Add Domain Error Types**
   - Define errors in `internal/core/errors/`
   - Use sentinel errors and error wrapping
   - Map to MCP error codes

3. **Improve Test Coverage**
   - Add integration tests
   - Add performance benchmarks
   - Add fuzz tests for parsers

### Medium Priority

4. **Memory Optimization**
   - **Integrate existing LRU cache** - [`LRUCache`](internal/engine/graph/lru.go:18) is implemented with tests but NOT used in Graph or App
   - Add graph pruning for unused modules
   - Consider streaming for large codebases

5. **Enhanced Observability**
   - Add OpenTelemetry tracing
   - Expose Prometheus metrics endpoint
   - Add health check endpoint

6. **Configuration Improvements**
   - Add environment variable support
   - Add configuration hot-reload
   - Add schema migration for version changes

### Low Priority

7. **Additional Language Support**
   - Add C/C++ extractor
   - Add C# extractor
   - Add Ruby extractor
   - Add PHP extractor

8. **Graph Algorithms**
   - Add centrality measures (PageRank, betweenness)
   - Add community detection
   - Add impact scoring

9. **Secret Detection Improvements**
   - Add verification hooks
   - Add baseline support
   - Add more cloud provider patterns

---

## Code Quality Metrics

| Component | Files | Approx Lines | Test Files | Test Coverage |
|-----------|-------|--------------|------------|---------------|
| core/app | 17 | ~45,000 | 5 | Medium |
| core/config | 6 | ~30,000 | 3 | High |
| core/ports | 1 | ~160 | 0 | N/A |
| core/watcher | 2 | ~6,000 | 1 | Medium |
| engine/parser | 19 | ~90,000 | 8 | Medium |
| engine/graph | 12 | ~55,000 | 5 | Medium |
| engine/resolver | 13 | ~35,000 | 5 | Medium |
| engine/secrets | 4 | ~17,000 | 3 | Medium |
| data/history | 7 | ~22,000 | 4 | High |
| data/query | 4 | ~12,000 | 3 | Medium |
| mcp/* | 21 | ~45,000 | 10 | High |
| ui/cli | 7 | ~42,000 | 3 | Low |
| ui/report | 12 | ~65,000 | 4 | Medium |

**Total:** ~464,000 lines across 125 files

---

## Architectural Patterns Observed

### Positive Patterns

1. **Hexagonal Architecture**
   - Clean port definitions in [`ports/ports.go`](internal/core/ports/ports.go:1)
   - Adapters in engine, data, ui, mcp layers
   - Dependency injection throughout

2. **Repository Pattern**
   - [`HistoryStore`](internal/core/ports/ports.go:41) for snapshot persistence
   - [`SQLiteSymbolStore`](internal/engine/graph/symbol_store.go:17) for symbol storage

3. **Factory Pattern**
   - [`NewParser()`](internal/engine/parser/parser.go:32), [`NewResolver()`](internal/engine/resolver/resolver.go:80), [`NewDetector()`](internal/engine/secrets/detector.go:48)

4. **Strategy Pattern**
   - Language extractors implement [`Extractor`](internal/engine/parser/parser.go:24) interface

### Anti-Patterns Detected

1. **God Object**
   - [`App`](internal/core/app/app.go:37) struct has too many responsibilities
   - Multiple large files indicate concentration of logic

2. **Primitive Obsession**
   - String types used for module names, paths without type safety

3. **Shotgun Surgery**
   - Error handling scattered across files
   - Validation logic duplicated

---

## Conclusion

The `internal/` codebase demonstrates solid architectural principles with hexagonal design, clean port definitions, and good separation of concerns. The system is feature-rich with multi-language parsing, dependency analysis, secret detection, and MCP integration.

### Key Strengths
- Well-defined interfaces following hexagonal architecture
- Comprehensive language support via tree-sitter
- Multiple output formats for visualization
- MCP integration for AI assistant workflows
- SQLite persistence for scalability
- Thread-safe implementations throughout

### Key Areas for Improvement
- File size reduction (several files exceed 20,000 chars)
- Error handling standardization
- Test coverage expansion (especially integration tests)
- Memory optimization for large codebases
- Observability integration

### Production Readiness Assessment

| Aspect | Status | Notes |
|--------|--------|-------|
| Functionality | ✅ Ready | All core features working |
| Performance | ⚠️ Needs work | Memory usage for large codebases |
| Reliability | ✅ Ready | Thread-safe, proper error handling |
| Observability | ❌ Not ready | Missing metrics, tracing |
| Security | ⚠️ Needs work | No auth for MCP, no rate limiting |
| Scalability | ⚠️ Needs work | In-memory graph limits scale |

**Overall:** The codebase is production-ready for small to medium codebases but would benefit from refactoring to improve maintainability and scalability for large-scale deployments.
