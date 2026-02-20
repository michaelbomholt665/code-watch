# Code-Watch Internal Codebase Analysis Report

**Generated:** 2026-02-20  
**Scope:** `internal/` directory  
**Architecture:** Hexagonal Architecture (Ports & Adapters)

---

## Executive Summary

The `internal/` directory implements a sophisticated **Code Dependency Monitor** called "Circular" using hexagonal architecture. The system parses source code, builds dependency graphs, detects cycles, identifies unresolved references ("hallucinations"), finds unused imports, detects secrets, and provides both CLI and MCP (Model Context Protocol) interfaces for interaction.

---

## Architecture Overview

```mermaid
graph TB
    subgraph Core [internal/core]
        ports[ports/ports.go<br/>Interfaces]
        app[app/*.go<br/>Application Service]
        config[config/*.go<br/>Configuration]
        watcher[watcher/*.go<br/>File Watcher]
    end
    
    subgraph Engine [internal/engine]
        parser[parser/*.go<br/>Tree-Sitter Parsing]
        graph[graph/*.go<br/>Dependency Graph]
        resolver[resolver/*.go<br/>Symbol Resolution]
        secrets[secrets/*.go<br/>Secret Detection]
    end
    
    subgraph Data [internal/data]
        history[history/*.go<br/>SQLite History]
        query[query/*.go<br/>Query Service]
    end
    
    subgraph MCP [internal/mcp]
        runtime[runtime/*.go<br/>MCP Server]
        tools[tools/*/<br/>MCP Tools]
        transport[transport/*.go<br/>STDIO Transport]
    end
    
    subgraph UI [internal/ui]
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

## Capabilities by Component

### 1. Core Layer (`internal/core/`)

#### [`ports/ports.go`](internal/core/ports/ports.go)
**Purpose:** Defines all interfaces (Ports) following hexagonal architecture principles.

**Key Interfaces:**
| Interface | Purpose |
|-----------|---------|
| `CodeParser` | Parse source files, check language support |
| `SecretScanner` | Detect secrets in file content |
| `HistoryStore` | Persist and load analysis snapshots |
| `QueryService` | Read-only dependency queries |
| `WatchService` | File watching lifecycle |
| `AnalysisService` | Main driving port for scan/query operations |

**Strengths:**
- Clean separation between driving and driven ports
- Well-defined DTOs for requests/responses
- Context-aware interfaces for cancellation

**Improvements Needed:**
- Missing documentation for interface contracts
- No error type definitions for domain-specific errors
- `SummarySnapshot` mixes concerns (cycles, secrets, violations)

---

#### [`app/app.go`](internal/core/app/app.go) (38,518 chars)
**Purpose:** Central application orchestrator managing parsing, graph building, and analysis.

**Capabilities:**
- **File Processing:** Parse files, detect language, resolve module names
- **Graph Management:** Build and maintain dependency graph
- **Secret Detection:** Integrate with secret scanner
- **Incremental Updates:** Handle file changes with transitive invalidation
- **Output Generation:** DOT, TSV, Mermaid, PlantUML, Markdown
- **Symbol Persistence:** SQLite-backed symbol store for large codebases

**Key Methods:**
| Method | Function |
|--------|----------|
| `InitialScan()` | Scan all configured paths |
| `ProcessFile()` | Parse and add file to graph |
| `HandleChanges()` | React to file system changes |
| `AnalyzeHallucinations()` | Find unresolved references |
| `AnalyzeUnusedImports()` | Detect unused imports |
| `ArchitectureViolations()` | Check layer rules |

**Strengths:**
- Comprehensive module resolution for Go and Python
- Transitive invalidation for incremental updates
- Caching strategies (go.mod cache, file content cache)
- Thread-safe with proper mutex usage

**Improvements Needed:**
- **Large File Warning:** 38,518 chars indicates potential God Object
- **Mixed Concerns:** Parsing, graph, secrets, output all in one struct
- **Error Handling:** Many errors logged but not propagated
- **Testability:** Direct filesystem access makes testing difficult

---

#### [`app/service.go`](internal/core/app/service.go)
**Purpose:** Implements `AnalysisService` port.

**Capabilities:**
- Run scans with path filtering
- Trace import chains between modules
- Analyze impact of changes
- Detect cycles with limits
- Query service integration
- History trend capture

**Improvements Needed:**
- Duplicate validation logic across methods
- No metrics/observability hooks
- Missing rate limiting for expensive operations

---

#### [`config/config.go`](internal/core/config/config.go) (27,693 chars)
**Purpose:** Configuration management with TOML parsing.

**Configuration Sections:**
| Section | Purpose |
|---------|---------|
| `Paths` | Project root, config/state directories |
| `DB` | SQLite database settings |
| `Projects` | Multi-project support |
| `MCP` | MCP server configuration |
| `Languages` | Language-specific overrides |
| `DynamicGrammars` | Runtime grammar loading |
| `Architecture` | Layer rules for validation |
| `Secrets` | Secret detection settings |

**Strengths:**
- Comprehensive configuration options
- Support for multiple projects
- Dynamic grammar configuration
- Architecture layer rules

**Improvements Needed:**
- No configuration validation on load
- Missing environment variable overrides
- No hot-reload for configuration changes
- Large file suggests need for decomposition

---

#### [`watcher/watcher.go`](internal/core/watcher/watcher.go)
**Purpose:** File system watching with debouncing.

**Capabilities:**
- Recursive directory watching via fsnotify
- Configurable debounce period
- Extension/filename filtering
- Test file exclusion
- Automatic new directory watching

**Strengths:**
- Proper debouncing to batch changes
- Glob-based exclusion patterns
- Thread-safe callback handling

**Improvements Needed:**
- No support for inotify limits configuration
- Missing metrics for watch events
- No graceful shutdown handling

---

### 2. Engine Layer (`internal/engine/`)

#### [`parser/`](internal/engine/parser/)
**Purpose:** Tree-sitter based source code parsing.

**Key Files:**
| File | Purpose |
|------|---------|
| `parser.go` | Parser orchestration |
| `golang.go` | Go language extractor (17,902 chars) |
| `python.go` | Python language extractor (8,883 chars) |
| `profile_extractors.go` | Profile-based extraction (24,456 chars) |
| `loader.go` | Grammar loading |
| `types.go` | Data structures |

**Supported Languages:**
- Go, Python, JavaScript, TypeScript, TSX
- Java, Rust, HTML, CSS

**Capabilities:**
- Import extraction
- Definition extraction (functions, classes, methods, variables)
- Reference tracking
- Complexity metrics (branch count, nesting depth, LOC)
- Local symbol tracking for resolution

**Strengths:**
- Tree-sitter provides fast, incremental parsing
- Language-specific extractors
- Dynamic grammar support
- Complexity metrics for hotspot detection

**Improvements Needed:**
- **Large Extractor Files:** `golang.go` and `profile_extractors.go` need decomposition
- **Missing Languages:** No C/C++, C#, Ruby, PHP extractors
- **Error Context:** Parse errors lack source context
- **No Incremental Parsing:** Re-parses entire file on change

---

#### [`graph/`](internal/engine/graph/)
**Purpose:** Dependency graph construction and analysis.

**Key Files:**
| File | Purpose |
|------|---------|
| `graph.go` | Core graph structure |
| `detect.go` | Cycle detection |
| `architecture.go` | Layer rule validation |
| `metrics.go` | Module metrics (fan-in/fan-out) |
| `impact.go` | Impact analysis |
| `symbol_store.go` | SQLite symbol persistence |
| `symbol_table.go` | In-memory symbol lookup |

**Capabilities:**
- Module/file tracking
- Import edge management
- Cycle detection via DFS
- Import chain finding via BFS
- Transitive invalidation
- Architecture layer validation
- Complexity hotspot identification
- Symbol persistence for large codebases

**Strengths:**
- Thread-safe with RWMutex
- Efficient cycle detection
- SQLite persistence for scalability
- Layer-based architecture validation

**Improvements Needed:**
- **Memory Usage:** All files kept in memory
- **No Graph Persistence:** Rebuilds on restart (symbol store helps but not complete)
- **Cycle Detection:** Reports all cycles, no prioritization
- **Missing Algorithms:** No centrality measures, community detection

---

#### [`resolver/`](internal/engine/resolver/)
**Purpose:** Symbol resolution and reference validation.

**Key Files:**
| File | Purpose |
|------|---------|
| `resolver.go` | Main resolver logic |
| `bridge.go` | Cross-language bridge hints |
| `heuristics.go` | Heuristic resolution |
| `probabilistic.go` | Probabilistic matching |
| `unused_imports.go` | Unused import detection |
| `unresolved.go` | Hallucination detection |
| `stdlib.go` | Standard library knowledge |
| `drivers/*.go` | Language-specific resolvers |

**Capabilities:**
- Local symbol resolution
- Standard library recognition (Go, Python, JS, Java, Rust)
- Cross-language bridge detection (FFI, service calls)
- Probabilistic resolution for ambiguous references
- Unused import detection
- Explicit bridge mappings via `.circular-bridge.toml`

**Strengths:**
- Multi-language stdlib knowledge
- Probabilistic resolution reduces false positives
- Explicit bridge configuration
- Incremental analysis support

**Improvements Needed:**
- **False Positives:** Probabilistic matching can be inaccurate
- **No Type Information:** Lacks type-aware resolution
- **Limited Cross-File:** Struggles with dynamic languages
- **Stdlib Coverage:** Incomplete stdlib definitions

---

#### [`secrets/`](internal/engine/secrets/)
**Purpose:** Secret detection in source code.

**Capabilities:**
- Pattern-based detection (AWS keys, GitHub PATs, Stripe keys, etc.)
- Entropy analysis for high-randomness strings
- Context-aware detection (password=, api_key=, etc.)
- Incremental scanning via line ranges
- Custom pattern configuration

**Built-in Patterns:**
- AWS Access Key ID
- GitHub Personal Access Tokens (classic and fine-grained)
- Stripe Live Secrets
- Slack Tokens
- Private Key Blocks

**Strengths:**
- Multiple detection strategies
- Configurable thresholds
- Incremental scanning support
- Severity classification

**Improvements Needed:**
- **False Positives:** Entropy-based detection is noisy
- **No Verification:** Can't verify if secrets are active
- **Limited Patterns:** Missing many cloud provider keys
- **No Baseline:** Can't mark known secrets as acceptable

---

### 3. Data Layer (`internal/data/`)

#### [`history/`](internal/data/history/)
**Purpose:** Historical snapshot storage for trend analysis.

**Capabilities:**
- SQLite-backed persistence
- Snapshot storage with metrics
- Git commit correlation
- Trend report generation
- Project-key namespacing

**Schema:**
```sql
CREATE TABLE snapshots (
  project_key TEXT,
  schema_version INTEGER,
  ts_utc TEXT,
  commit_hash TEXT,
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
  max_fan_out INTEGER
);
```

**Strengths:**
- WAL mode for concurrency
- Retry logic for busy database
- Trend analysis support

**Improvements Needed:**
- **No Migration Strategy:** Schema changes not handled
- **Limited Metrics:** Missing code churn, complexity trends
- **No Cleanup:** Old snapshots never purged

---

#### [`query/`](internal/data/query/)
**Purpose:** Read-only query service for dependency analysis.

**Capabilities:**
- List modules with filtering
- Module details with dependencies
- Dependency trace between modules
- Trend slice retrieval

**Strengths:**
- Clean API design
- Context-aware for cancellation
- Sorted, consistent output

**Improvements Needed:**
- **Limited Queries:** No full-text search, no impact queries
- **No Pagination:** Large result sets not handled
- **No Caching:** Repeated queries not optimized

---

### 4. MCP Layer (`internal/mcp/`)

#### [`runtime/`](internal/mcp/runtime/)
**Purpose:** MCP server implementation for AI assistant integration.

**Capabilities:**
- STDIO transport
- Tool registration and dispatch
- Project context management
- Operation allowlist for security
- Auto-managed outputs

**Strengths:**
- Clean separation of concerns
- Allowlist for security
- Project context isolation

**Improvements Needed:**
- **Single Transport:** Only STDIO, no HTTP/SSE
- **No Authentication:** No auth mechanism
- **Limited Observability:** No metrics/logging hooks

---

#### [`tools/`](internal/mcp/tools/)
**Purpose:** Individual MCP tool implementations.

**Available Tools:**
| Tool | Purpose |
|------|---------|
| `scan` | Run dependency scan |
| `graph/cycles` | Detect dependency cycles |
| `query/*` | Query modules and dependencies |
| `secrets/*` | Secret detection |
| `report` | Generate reports |
| `system/*` | System information |

**Strengths:**
- Focused, single-responsibility tools
- Consistent error handling
- Test coverage

**Improvements Needed:**
- **Missing Tools:** No impact analysis tool
- **No Batching:** Can't run multiple operations
- **Limited Output Control:** No format selection

---

### 5. UI Layer (`internal/ui/`)

#### [`cli/`](internal/ui/cli/)
**Purpose:** Command-line interface runtime.

**Capabilities:**
- Single scan mode (`--once`)
- Watch mode with TUI (`--ui`)
- MCP server mode
- Markdown report generation
- History trend mode
- Grammar verification

**Strengths:**
- Multiple operation modes
- Signal handling for graceful shutdown
- Mode compatibility validation

**Improvements Needed:**
- **Large Runtime File:** `runtime.go` is 20,797 chars
- **No Progress Indication:** Long scans lack feedback
- **Limited CLI Help:** Could use better documentation

---

#### [`report/`](internal/ui/report/)
**Purpose:** Report generation in multiple formats.

**Supported Formats:**
- DOT (Graphviz)
- Mermaid
- PlantUML
- TSV
- Markdown

**Capabilities:**
- Architecture diagrams
- Component diagrams
- Flow diagrams
- Cycle highlighting
- Complexity hotspots
- Layer visualization

**Strengths:**
- Multiple output formats
- Architecture layer visualization
- Cycle and violation highlighting

**Improvements Needed:**
- **Large Files:** `mermaid.go` is 19,715 chars
- **No Interactive Output:** Static diagrams only
- **Limited Customization:** Hard-coded styling

---

## Cross-Cutting Concerns

### Testing
**Coverage Areas:**
- Parser extractors
- Graph operations
- Resolver logic
- Secret detection
- MCP tools
- CLI runtime

**Gaps:**
- Integration tests limited
- No performance benchmarks (except history)
- No fuzz testing for parsers

### Error Handling
**Current State:**
- Errors often logged but not propagated
- No structured error types
- Missing error context

**Recommendations:**
- Define domain error types
- Use error wrapping consistently
- Add error codes for MCP responses

### Concurrency
**Current State:**
- RWMutex usage throughout
- Sync primitives in App, Graph, Watcher
- SQLite WAL mode for database concurrency

**Gaps:**
- No context timeout propagation
- Potential deadlocks not analyzed
- No rate limiting

### Observability
**Current State:**
- slog for structured logging
- No metrics collection
- No tracing

**Recommendations:**
- Add OpenTelemetry integration
- Expose Prometheus metrics
- Add request tracing for MCP

---

## Priority Improvements

### High Priority

1. **Decompose Large Files**
   - [`app.go`](internal/core/app/app.go) (38,518 chars) → Split into scanner.go, analyzer.go, output.go
   - [`config.go`](internal/core/config/config.go) (27,693 chars) → Split into sections
   - [`profile_extractors.go`](internal/engine/parser/profile_extractors.go) (24,456 chars) → Per-language files
   - [`runtime.go`](internal/ui/cli/runtime.go) (20,797 chars) → Split into modes
   - [`mermaid.go`](internal/ui/report/formats/mermaid.go) (19,715 chars) → Split into components

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
   - Implement file content eviction
   - Add graph pruning for unused modules
   - Consider streaming for large codebases

5. **Enhanced Observability**
   - Add OpenTelemetry tracing
   - Expose Prometheus metrics endpoint
   - Add health check endpoint

6. **Configuration Improvements**
   - Add environment variable support
   - Add configuration validation
   - Add hot-reload support

### Low Priority

7. **Additional Language Support**
   - Add C/C++ extractor
   - Add C# extractor
   - Add Ruby extractor

8. **Graph Algorithms**
   - Add centrality measures
   - Add community detection
   - Add impact scoring

9. **Secret Detection Improvements**
   - Add verification hooks
   - Add baseline support
   - Add more cloud provider patterns

---

## Metrics Summary

| Component | Files | Total Lines (approx) | Test Files |
|-----------|-------|---------------------|------------|
| core/app | 6 | ~65,000 | 3 |
| core/config | 4 | ~35,000 | 2 |
| core/ports | 1 | ~160 | 0 |
| core/watcher | 2 | ~6,000 | 1 |
| engine/parser | 19 | ~90,000 | 7 |
| engine/graph | 8 | ~42,000 | 2 |
| engine/resolver | 13 | ~35,000 | 4 |
| engine/secrets | 4 | ~17,000 | 2 |
| data/history | 7 | ~22,000 | 3 |
| data/query | 4 | ~12,000 | 2 |
| mcp/* | 21 | ~45,000 | 9 |
| ui/cli | 7 | ~42,000 | 3 |
| ui/report | 9 | ~55,000 | 3 |

**Total:** ~466,000 lines across 105 files

---

## Conclusion

The `internal/` codebase demonstrates solid architectural principles with hexagonal design, clean port definitions, and good separation of concerns. The system is feature-rich with multi-language parsing, dependency analysis, secret detection, and MCP integration.

**Key Strengths:**
- Well-defined interfaces following hexagonal architecture
- Comprehensive language support via tree-sitter
- Multiple output formats for visualization
- MCP integration for AI assistant workflows
- SQLite persistence for scalability

**Key Areas for Improvement:**
- File size reduction (several files exceed 20,000 chars)
- Error handling standardization
- Test coverage expansion
- Memory optimization for large codebases
- Observability integration

The codebase is production-ready but would benefit from refactoring to improve maintainability and scalability.
