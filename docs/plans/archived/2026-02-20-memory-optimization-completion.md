# Memory Optimization Completion Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Integrate the existing `LRUCache` into `Graph` and `App` to manage memory usage for large codebases.

**Architecture:**
Circular currently keeps all parsed files and file contents in memory. This plan integrates the `LRUCache` into the `Graph` and `App` structs. `Graph` will cache `parser.File` objects, and `App` will cache raw file contents. This allows the system to evict less-frequently used data under memory pressure.

**Tech Stack:** Go 1.24+, SQLite

---

### Task 1: Update Graph to use LRUCache for Files

**Files:**
- Modify: `internal/engine/graph/graph.go`

**Step 1: Update `Graph` struct and `NewGraph` factory**

Replace `files map[string]*parser.File` with `fileCache *LRUCache[string, *parser.File]`.

```go
type Graph struct {
	mu sync.RWMutex

	// Core data
	fileCache *LRUCache[string, *parser.File] // path -> file
    // ...
}

func NewGraph(capacity int) *Graph {
	return &Graph{
		fileCache: NewLRUCache[string, *parser.File](capacity),
		// ...
	}
}
```

**Step 2: Update `AddFile`, `RemoveFile`, `GetFile`, and `FileCount`**

Update these methods to use `g.fileCache.Put`, `g.fileCache.Evict`, `g.fileCache.Get`, and `g.fileCache.Len`.

**Step 3: Verify tests pass**

Run: `go test ./internal/engine/graph/...`

**Step 4: Commit**

```bash
git add internal/engine/graph/graph.go
git commit -m "feat(graph): use LRUCache for parsed files"
```

---

### Task 2: Update App to use LRUCache for File Contents

**Files:**
- Modify: `internal/core/app/app.go`
- Modify: `internal/core/app/content_cache.go`

**Step 1: Update `App` struct to use `LRUCache`**

```go
type App struct {
    // ...
	fileContentMu sync.RWMutex
	fileContents  *graph.LRUCache[string, []byte]
}
```

**Step 2: Update `NewWithDependencies` to initialize `fileContents` cache**

**Step 3: Update `contentForPath`, `cacheContent`, and `dropContent`**

Update these methods in `internal/core/app/content_cache.go` to use the new cache.

**Step 4: Verify tests pass**

Run: `go test ./internal/core/app/...`

**Step 5: Commit**

```bash
git add internal/core/app/app.go internal/core/app/content_cache.go
git commit -m "feat(app): use LRUCache for file contents"
```

---

### Task 3: Support File Reloading in Graph (Optional/Advanced)

**Files:**
- Modify: `internal/engine/graph/graph.go`
- Modify: `internal/engine/graph/symbol_store.go`
- Modify: `internal/core/app/app.go`

**Step 1: Add `FileBlob` support to `SQLiteSymbolStore`**

Add a table `file_blobs` to store the JSON-encoded `parser.File`.

**Step 2: Update `Graph` to support a `FileLoader` interface**

```go
type FileLoader interface {
    LoadFile(path string) (*parser.File, error)
}
```

**Step 3: Update `GetFile` in `Graph` to attempt reload on cache miss**

**Step 4: Commit**

```bash
git add internal/engine/graph/graph.go internal/engine/graph/symbol_store.go internal/core/app/app.go
git commit -m "feat(graph): support reloading files from symbol store"
```

---

### Task 4: Configuration and Final Integration

**Files:**
- Modify: `internal/core/config/config.go`
- Modify: `internal/core/app/app.go`

**Step 1: Add cache capacity settings to `config.go`**

**Step 2: Initialize caches in `app.go` with configured capacities**

**Step 3: Verify end-to-end**

Run a full scan on a large project (if available) or use a test fixture.

**Step 4: Commit**

```bash
git add internal/core/config/config.go internal/core/app/app.go
git commit -m "feat(config): add configurable cache capacities"
```
