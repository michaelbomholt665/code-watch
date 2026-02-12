# Code Dependency Monitor - Implementation Plan v2

## Overview

Build a long-running Go binary that monitors code files, builds a dependency graph, detects circular imports and potential hallucinations (undefined references), and outputs to DOT format for KGraphViewer.

**Target:** Personal tool, Linux only, 100-1000 file codebases

---

## Tech Stack

| Component | Choice | Reason |
|-----------|--------|--------|
| Language | Go 1.23+ | Single binary, efficient, good concurrency |
| Tree-sitter | `github.com/tree-sitter/go-tree-sitter` v0.25+ | Native bindings, purego support |
| Config | TOML (`github.com/BurntSushi/toml`) | Human-readable, standard |
| File watching | `github.com/fsnotify/fsnotify` v1.7+ | Efficient, battle-tested |
| Glob matching | `github.com/gobwas/glob` | Fast, full glob support |
| DOT output | String builder | Simple, works with KGraphViewer |
| Logging | `log/slog` (stdlib) | Structured logging, Go 1.21+ |

---

## Project Structure

```
circular/
├── cmd/
│   └── circular/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── watcher/
│   │   └── watcher.go
│   ├── parser/
│   │   ├── loader.go          # Grammar loading
│   │   ├── types.go           # Data structures
│   │   ├── parser.go          # Generic parser
│   │   ├── python.go          # Python-specific extraction
│   │   └── golang.go          # Go-specific extraction
│   ├── resolver/
│   │   ├── resolver.go        # Module resolution logic
│   │   ├── python_resolver.go # Python import resolution
│   │   ├── go_resolver.go     # Go import resolution
│   │   └── stdlib.go          # Embedded stdlib lists
│   ├── graph/
│   │   ├── graph.go           # Thread-safe graph
│   │   └── detect.go          # Cycle & hallucination detection
│   └── output/
│       ├── dot.go             # DOT generation
│       └── tsv.go             # TSV export
├── grammars/
│   ├── python/
│   │   ├── python.so
│   │   └── node-types.json
│   └── go/
│       ├── go.so
│       └── node-types.json
├── circular.example.toml
├── go.mod
└── go.sum
```

---

## Core Data Structures

### File Representation

```go
type File struct {
    Path         string
    Language     string
    Module       string       // Fully qualified module name
    PackageName  string       // Local package/module name
    Imports      []Import
    Definitions  []Definition
    References   []Reference  // Function/symbol calls
    ParsedAt     time.Time
}

type Import struct {
    Module       string       // Imported module (resolved to FQN)
    RawImport    string       // Original import string
    Alias        string       // Optional alias
    Items        []string     // For "from X import Y, Z"
    IsRelative   bool         // For Python relative imports
    Location     Location
}

type Definition struct {
    Name         string
    FullName     string       // module.function or package.Type
    Kind         DefinitionKind
    Location     Location
    Exported     bool         // Is public/exported?
    Scope        string       // Global, class method, nested function
}

type Reference struct {
    Name         string
    FullName     string       // Resolved if possible
    Location     Location
    Context      string       // Where this reference occurs
    Resolved     bool         // Did we find the definition?
}

type DefinitionKind int
const (
    KindFunction DefinitionKind = iota
    KindClass
    KindMethod
    KindVariable
    KindConstant
    KindType
    KindInterface
)

type Location struct {
    File   string
    Line   int
    Column int
}
```

### Thread-Safe Graph

```go
type Graph struct {
    mu sync.RWMutex
    
    // Core data
    files        map[string]*File            // path -> file
    modules      map[string]*Module          // module name -> module info
    
    // Relationships
    imports      map[string]map[string]*ImportEdge  // from -> to -> edge
    importedBy   map[string]map[string]bool         // to -> from
    
    // Symbol tables (for hallucination detection)
    definitions  map[string]map[string]*Definition  // module -> symbol -> def
    
    // Invalidation tracking
    dirty        map[string]bool             // Files needing re-analysis
}

type Module struct {
    Name         string
    Files        []string        // Paths to files in this module
    Exports      map[string]*Definition
    RootPath     string          // For Go: module root, Python: package root
}

type ImportEdge struct {
    From         string
    To           string
    ImportedBy   string          // File path
    Location     Location
}
```

### Graph Operations (Thread-Safe)

```go
func (g *Graph) AddFile(file *File) {
    g.mu.Lock()
    defer g.mu.Unlock()
    // ... add file, update modules, imports
}

func (g *Graph) RemoveFile(path string) {
    g.mu.Lock()
    defer g.mu.Unlock()
    // ... remove file, clean up orphaned modules
}

func (g *Graph) GetModule(name string) (*Module, bool) {
    g.mu.RLock()
    defer g.mu.RUnlock()
    mod, ok := g.modules[name]
    return mod, ok
}

func (g *Graph) MarkDirty(paths []string) {
    g.mu.Lock()
    defer g.mu.Unlock()
    for _, p := range paths {
        g.dirty[p] = true
    }
}

func (g *Graph) GetDirty() []string {
    g.mu.Lock()
    defer g.mu.Unlock()
    paths := make([]string, 0, len(g.dirty))
    for p := range g.dirty {
        paths = append(paths, p)
        delete(g.dirty, p)
    }
    return paths
}
```

---

## Module Naming Strategy

### Python Module Naming Algorithm

```go
type PythonResolver struct {
    projectRoot string
    packages    map[string]string  // path -> module name
}

// Algorithm:
// 1. Find all __init__.py files → these define packages
// 2. Build package tree
// 3. For each .py file, walk up to find nearest __init__.py
// 4. Construct FQN from project root

func (r *PythonResolver) GetModuleName(filePath string) string {
    // Example:
    // Project root: /home/user/project
    // File: /home/user/project/src/auth/login.py
    // __init__.py at: /home/user/project/src/auth/__init__.py
    // Result: auth.login
    
    rel := relativeTo(r.projectRoot, filePath)
    // src/auth/login.py
    
    parts := strings.Split(rel, "/")
    // ["src", "auth", "login.py"]
    
    // Remove non-package prefixes (dirs without __init__.py)
    packageStart := 0
    for i := 0; i < len(parts)-1; i++ {
        checkPath := filepath.Join(parts[:i+1]..., "__init__.py")
        if !exists(checkPath) {
            packageStart = i + 1
        } else {
            break  // Found first package
        }
    }
    
    parts = parts[packageStart:]
    // ["auth", "login.py"]
    
    // Remove .py extension
    parts[len(parts)-1] = strings.TrimSuffix(parts[len(parts)-1], ".py")
    
    return strings.Join(parts, ".")
    // "auth.login"
}

// Special case: __init__.py
// /project/src/auth/__init__.py → "auth"

func (r *PythonResolver) ResolveImport(fromModule, importStmt string, isRelative bool, relativeLevel int) string {
    // Absolute: "from auth.utils import hash"
    if !isRelative {
        return importStmt
    }
    
    // Relative: "from . import validators"
    // From module: auth.login
    // Result: auth.validators
    
    // Relative: "from .. import config"
    // From module: auth.login
    // Result: config
    
    parts := strings.Split(fromModule, ".")
    if relativeLevel >= len(parts) {
        return importStmt  // Invalid, too many dots
    }
    
    base := strings.Join(parts[:len(parts)-relativeLevel], ".")
    if importStmt == "" {
        return base
    }
    return base + "." + importStmt
}
```

### Go Module Naming Algorithm

```go
type GoResolver struct {
    goModPath   string
    moduleName  string    // From go.mod: module github.com/user/proj
    moduleRoot  string    // Directory containing go.mod
}

func (r *GoResolver) FindGoMod(startPath string) error {
    // Walk up from startPath until we find go.mod
    current := startPath
    for {
        modPath := filepath.Join(current, "go.mod")
        if exists(modPath) {
            r.goModPath = modPath
            r.moduleRoot = current
            return r.parseGoMod()
        }
        
        parent := filepath.Dir(current)
        if parent == current {
            return errors.New("no go.mod found")
        }
        current = parent
    }
}

func (r *GoResolver) parseGoMod() error {
    // Parse go.mod for module name
    // module github.com/user/project
    data, _ := os.ReadFile(r.goModPath)
    
    re := regexp.MustCompile(`module\s+(\S+)`)
    matches := re.FindSubmatch(data)
    if len(matches) > 1 {
        r.moduleName = string(matches[1])
    }
    return nil
}

func (r *GoResolver) GetModuleName(filePath string) string {
    // Example:
    // Module: github.com/user/project
    // Root: /home/user/project
    // File: /home/user/project/internal/auth/login.go
    // Result: github.com/user/project/internal/auth
    
    rel := relativeTo(r.moduleRoot, filePath)
    // internal/auth/login.go
    
    dir := filepath.Dir(rel)
    // internal/auth
    
    if dir == "." {
        return r.moduleName
    }
    
    return r.moduleName + "/" + dir
    // github.com/user/project/internal/auth
}

func (r *GoResolver) ResolveImport(importPath string) string {
    // Go imports are already fully qualified
    // "fmt" → stdlib
    // "github.com/user/project/internal/auth" → internal
    // "github.com/other/lib" → external
    
    return importPath
}

func (r *GoResolver) IsStdlib(importPath string) bool {
    // Check against embedded stdlib list
    return stdlibModules[importPath]
}
```

---

## Symbol Resolution Strategy

### Phase 1: Build Symbol Tables

```go
func (g *Graph) BuildSymbolTables() {
    g.mu.Lock()
    defer g.mu.Unlock()
    
    g.definitions = make(map[string]map[string]*Definition)
    
    for _, file := range g.files {
        if g.definitions[file.Module] == nil {
            g.definitions[file.Module] = make(map[string]*Definition)
        }
        
        for i := range file.Definitions {
            def := &file.Definitions[i]
            if def.Exported {
                g.definitions[file.Module][def.Name] = def
            }
        }
    }
}
```

### Phase 2: Resolve References

```go
type ReferenceResolver struct {
    graph   *Graph
    stdlib  map[string]bool
}

func (r *ReferenceResolver) Resolve(file *File) []UnresolvedReference {
    unresolved := []UnresolvedReference{}
    
    for _, ref := range file.References {
        if r.resolveReference(file, &ref) {
            continue
        }
        unresolved = append(unresolved, UnresolvedReference{
            Reference: ref,
            File:      file.Path,
        })
    }
    
    return unresolved
}

func (r *ReferenceResolver) resolveReference(file *File, ref *Reference) bool {
    // 1. Check local module
    if r.checkModule(file.Module, ref.Name) {
        return true
    }
    
    // 2. Check imported modules
    for _, imp := range file.Imports {
        if imp.Alias != "" {
            // Handle: import numpy as np → np.array
            if strings.HasPrefix(ref.Name, imp.Alias+".") {
                symbolName := strings.TrimPrefix(ref.Name, imp.Alias+".")
                if r.checkModule(imp.Module, symbolName) {
                    return true
                }
            }
        }
        
        // Handle: from auth import login → login()
        if len(imp.Items) > 0 {
            for _, item := range imp.Items {
                if ref.Name == item || strings.HasPrefix(ref.Name, item+".") {
                    if r.checkModule(imp.Module, item) {
                        return true
                    }
                }
            }
        }
        
        // Handle: import auth → auth.login()
        if strings.HasPrefix(ref.Name, imp.Module+".") {
            symbolName := strings.TrimPrefix(ref.Name, imp.Module+".")
            if r.checkModule(imp.Module, symbolName) {
                return true
            }
        }
    }
    
    // 3. Check stdlib
    if r.stdlib[ref.Name] || r.isStdlibCall(ref.Name) {
        return true
    }
    
    // 4. Check builtins (for Python: print, len, etc.)
    if r.isBuiltin(ref.Name) {
        return true
    }
    
    return false
}

func (r *ReferenceResolver) checkModule(moduleName, symbolName string) bool {
    r.graph.mu.RLock()
    defer r.graph.mu.RUnlock()
    
    symbols, ok := r.graph.definitions[moduleName]
    if !ok {
        return false
    }
    
    // Direct match
    if _, ok := symbols[symbolName]; ok {
        return true
    }
    
    // Nested: Class.method or package.Type
    for fullName := range symbols {
        if strings.HasPrefix(fullName, symbolName+".") ||
           strings.HasSuffix(fullName, "."+symbolName) {
            return true
        }
    }
    
    return false
}

func (r *ReferenceResolver) isStdlibCall(name string) bool {
    parts := strings.Split(name, ".")
    if len(parts) == 0 {
        return false
    }
    return r.stdlib[parts[0]]
}
```

### Dependency Invalidation

```go
func (g *Graph) InvalidateTransitive(changedFile string) []string {
    g.mu.RLock()
    defer g.mu.RUnlock()
    
    // Get module of changed file
    file := g.files[changedFile]
    if file == nil {
        return nil
    }
    
    // Find all files that import this module
    toRecheck := []string{changedFile}
    seen := map[string]bool{changedFile: true}
    
    queue := []string{file.Module}
    for len(queue) > 0 {
        mod := queue[0]
        queue = queue[1:]
        
        // Who imports this module?
        for importer := range g.importedBy[mod] {
            if seen[importer] {
                continue
            }
            seen[importer] = true
            
            // Add all files in importer module
            if importerMod, ok := g.modules[importer]; ok {
                toRecheck = append(toRecheck, importerMod.Files...)
                queue = append(queue, importer)
            }
        }
    }
    
    return toRecheck
}
```

---

## Embedded Standard Library Lists

### Python Stdlib

```go
// internal/resolver/stdlib.go

//go:embed stdlib/python.txt
var pythonStdlibData string

var pythonStdlib = map[string]bool{}

func init() {
    for _, line := range strings.Split(pythonStdlibData, "\n") {
        line = strings.TrimSpace(line)
        if line != "" && !strings.HasPrefix(line, "#") {
            pythonStdlib[line] = true
        }
    }
}

// stdlib/python.txt (embed this file)
/*
abc
aifc
argparse
array
ast
asyncio
atexit
base64
bdb
binascii
bisect
builtins
bz2
calendar
cgi
cgitb
chunk
cmath
cmd
code
codecs
collections
colorsys
compileall
concurrent
configparser
contextlib
copy
copyreg
crypt
csv
ctypes
curses
dataclasses
datetime
dbm
decimal
difflib
dis
distutils
doctest
email
encodings
enum
errno
faulthandler
fcntl
filecmp
fileinput
fnmatch
fractions
ftplib
functools
gc
getopt
getpass
gettext
glob
graphlib
grp
gzip
hashlib
heapq
hmac
html
http
imaplib
imghdr
imp
importlib
inspect
io
ipaddress
itertools
json
keyword
lib2to3
linecache
locale
logging
lzma
mailbox
mailcap
marshal
math
mimetypes
mmap
modulefinder
multiprocessing
netrc
nis
nntplib
numbers
operator
optparse
os
ossaudiodev
parser
pathlib
pdb
pickle
pickletools
pipes
pkgutil
platform
plistlib
poplib
posix
posixpath
pprint
profile
pstats
pty
pwd
py_compile
pyclbr
pydoc
queue
quopri
random
re
readline
reprlib
resource
rlcompleter
runpy
sched
secrets
select
selectors
shelve
shlex
shutil
signal
site
smtpd
smtplib
sndhdr
socket
socketserver
spwd
sqlite3
ssl
stat
statistics
string
stringprep
struct
subprocess
sunau
symtable
sys
sysconfig
syslog
tabnanny
tarfile
tempfile
termios
test
textwrap
threading
time
timeit
tkinter
token
tokenize
tomllib
trace
traceback
tracemalloc
tty
turtle
turtledemo
types
typing
unicodedata
unittest
urllib
uu
uuid
venv
warnings
wave
weakref
webbrowser
winreg
winsound
wsgiref
xdrlib
xml
xmlrpc
zipapp
zipfile
zipimport
zlib
*/
```

### Go Stdlib

```go
// Generate with: go list std

//go:embed stdlib/go.txt
var goStdlibData string

var goStdlib = map[string]bool{}

func init() {
    for _, line := range strings.Split(goStdlibData, "\n") {
        line = strings.TrimSpace(line)
        if line != "" && !strings.HasPrefix(line, "#") {
            goStdlib[line] = true
        }
    }
}

// stdlib/go.txt (generated via go list std)
/*
archive/tar
archive/zip
bufio
bytes
cmp
compress/bzip2
compress/flate
compress/gzip
compress/lzw
compress/zlib
container/heap
container/list
container/ring
context
crypto
crypto/aes
crypto/cipher
crypto/des
crypto/dsa
crypto/ecdh
crypto/ecdsa
crypto/ed25519
crypto/elliptic
crypto/hmac
crypto/md5
crypto/rand
crypto/rc4
crypto/rsa
crypto/sha1
crypto/sha256
crypto/sha512
crypto/subtle
crypto/tls
crypto/x509
crypto/x509/pkix
database/sql
database/sql/driver
debug/buildinfo
debug/dwarf
debug/elf
debug/gosym
debug/macho
debug/pe
debug/plan9obj
embed
encoding
encoding/ascii85
encoding/asn1
encoding/base32
encoding/base64
encoding/binary
encoding/csv
encoding/gob
encoding/hex
encoding/json
encoding/pem
encoding/xml
errors
expvar
flag
fmt
go/ast
go/build
go/build/constraint
go/constant
go/doc
go/doc/comment
go/format
go/importer
go/parser
go/printer
go/scanner
go/token
go/types
go/version
hash
hash/adler32
hash/crc32
hash/crc64
hash/fnv
hash/maphash
html
html/template
image
image/color
image/color/palette
image/draw
image/gif
image/jpeg
image/png
index/suffixarray
io
io/fs
io/ioutil
log
log/slog
log/syslog
maps
math
math/big
math/bits
math/cmplx
math/rand
math/rand/v2
mime
mime/multipart
mime/quotedprintable
net
net/http
net/http/cgi
net/http/cookiejar
net/http/fcgi
net/http/httptest
net/http/httptrace
net/http/httputil
net/http/pprof
net/mail
net/netip
net/rpc
net/rpc/jsonrpc
net/smtp
net/textproto
net/url
os
os/exec
os/signal
os/user
path
path/filepath
plugin
reflect
regexp
regexp/syntax
runtime
runtime/cgo
runtime/coverage
runtime/debug
runtime/metrics
runtime/pprof
runtime/trace
slices
sort
strconv
strings
sync
sync/atomic
syscall
testing
testing/fstest
testing/iotest
testing/quick
testing/slogtest
text/scanner
text/tabwriter
text/template
text/template/parse
time
time/tzdata
unicode
unicode/utf16
unicode/utf8
unsafe
*/
```

### Python Builtins

```go
var pythonBuiltins = map[string]bool{
    "abs": true, "aiter": true, "all": true, "anext": true, "any": true,
    "ascii": true, "bin": true, "bool": true, "breakpoint": true, "bytearray": true,
    "bytes": true, "callable": true, "chr": true, "classmethod": true, "compile": true,
    "complex": true, "delattr": true, "dict": true, "dir": true, "divmod": true,
    "enumerate": true, "eval": true, "exec": true, "filter": true, "float": true,
    "format": true, "frozenset": true, "getattr": true, "globals": true, "hasattr": true,
    "hash": true, "help": true, "hex": true, "id": true, "input": true,
    "int": true, "isinstance": true, "issubclass": true, "iter": true, "len": true,
    "list": true, "locals": true, "map": true, "max": true, "memoryview": true,
    "min": true, "next": true, "object": true, "oct": true, "open": true,
    "ord": true, "pow": true, "print": true, "property": true, "range": true,
    "repr": true, "reversed": true, "round": true, "set": true, "setattr": true,
    "slice": true, "sorted": true, "staticmethod": true, "str": true, "sum": true,
    "super": true, "tuple": true, "type": true, "vars": true, "zip": true,
    "__import__": true,
}
```

---

## Tree-Sitter Integration (v0.25+ API)

### Grammar Loading

```go
// internal/parser/loader.go

import (
    sitter "github.com/tree-sitter/go-tree-sitter"
)

type GrammarLoader struct {
    languages map[string]*sitter.Language
}

func NewGrammarLoader(grammarsPath string) (*GrammarLoader, error) {
    gl := &GrammarLoader{
        languages: make(map[string]*sitter.Language),
    }
    
    // Load Python
    pythonLang := sitter.NewLanguage(sitter.Python)
    gl.languages["python"] = pythonLang
    
    // Load Go
    goLang := sitter.NewLanguage(sitter.Go)
    gl.languages["go"] = goLang
    
    return gl, nil
}

// Note: For v0.25+, grammars may be built-in or loaded dynamically
// Check actual API - if external .so loading is needed:

func (gl *GrammarLoader) LoadExternal(langName, soPath string) error {
    // Use purego to load .so if needed
    // This depends on tree-sitter v0.25 API details
    // Consult: https://github.com/tree-sitter/go-tree-sitter
    return nil
}
```

### Parsing Files

```go
// internal/parser/parser.go

type Parser struct {
    loader    *GrammarLoader
    extractors map[string]Extractor  // language -> extractor
}

type Extractor interface {
    Extract(node *sitter.Node, source []byte, filePath string) (*File, error)
}

func NewParser(loader *GrammarLoader) *Parser {
    return &Parser{
        loader: loader,
        extractors: map[string]Extractor{
            "python": &PythonExtractor{},
            "go":     &GoExtractor{},
        },
    }
}

func (p *Parser) ParseFile(path string, content []byte) (*File, error) {
    lang := p.detectLanguage(path)
    if lang == "" {
        return nil, errors.New("unsupported language")
    }
    
    grammar := p.loader.languages[lang]
    if grammar == nil {
        return nil, fmt.Errorf("grammar not loaded: %s", lang)
    }
    
    parser := sitter.NewParser()
    parser.SetLanguage(grammar)
    
    tree := parser.Parse(content, nil)
    if tree == nil {
        return nil, errors.New("parse failed")
    }
    defer tree.Close()
    
    root := tree.RootNode()
    
    extractor := p.extractors[lang]
    if extractor == nil {
        return nil, fmt.Errorf("no extractor for: %s", lang)
    }
    
    return extractor.Extract(root, content, path)
}

func (p *Parser) detectLanguage(path string) string {
    ext := filepath.Ext(path)
    switch ext {
    case ".py":
        return "python"
    case ".go":
        return "go"
    default:
        return ""
    }
}
```

### Python Extractor

```go
// internal/parser/python.go

type PythonExtractor struct{}

func (e *PythonExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
    file := &File{
        Path:     filePath,
        Language: "python",
        ParsedAt: time.Now(),
    }
    
    e.walk(root, source, file)
    
    return file, nil
}

func (e *PythonExtractor) walk(node *sitter.Node, source []byte, file *File) {
    nodeType := node.Type()
    
    switch nodeType {
    case "import_statement":
        e.extractImport(node, source, file)
    case "import_from_statement":
        e.extractFromImport(node, source, file)
    case "function_definition":
        e.extractFunction(node, source, file)
    case "class_definition":
        e.extractClass(node, source, file)
    case "call":
        e.extractCall(node, source, file)
    }
    
    // Recurse
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        e.walk(child, source, file)
    }
}

func (e *PythonExtractor) extractImport(node *sitter.Node, source []byte, file *File) {
    // import module
    // import module as alias
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.Type() == "dotted_name" {
            module := e.getText(child, source)
            
            alias := ""
            // Check for "as" clause
            if i+1 < int(node.ChildCount()) {
                next := node.Child(i + 1)
                if next.Type() == "as" && i+2 < int(node.ChildCount()) {
                    aliasNode := node.Child(i + 2)
                    alias = e.getText(aliasNode, source)
                }
            }
            
            file.Imports = append(file.Imports, Import{
                Module:     module,
                RawImport:  module,
                Alias:      alias,
                IsRelative: false,
                Location: Location{
                    File:   file.Path,
                    Line:   int(node.StartPoint().Row) + 1,
                    Column: int(node.StartPoint().Column) + 1,
                },
            })
        }
    }
}

func (e *PythonExtractor) extractFromImport(node *sitter.Node, source []byte, file *File) {
    // from module import item1, item2
    // from .relative import item
    // from ..parent import item
    
    var module string
    var items []string
    isRelative := false
    relativeLevel := 0
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        
        switch child.Type() {
        case "relative_import":
            isRelative = true
            relText := e.getText(child, source)
            relativeLevel = strings.Count(relText, ".")
            // Get module name after dots
            for j := 0; j < int(child.ChildCount()); j++ {
                subchild := child.Child(j)
                if subchild.Type() == "dotted_name" {
                    module = e.getText(subchild, source)
                }
            }
            
        case "dotted_name":
            if !isRelative {
                module = e.getText(child, source)
            }
            
        case "import_list":
            for j := 0; j < int(child.ChildCount()); j++ {
                item := child.Child(j)
                if item.Type() == "dotted_name" || item.Type() == "identifier" {
                    items = append(items, e.getText(item, source))
                }
            }
        }
    }
    
    file.Imports = append(file.Imports, Import{
        Module:     module,
        RawImport:  module,
        Items:      items,
        IsRelative: isRelative,
        Location: Location{
            File:   file.Path,
            Line:   int(node.StartPoint().Row) + 1,
            Column: int(node.StartPoint().Column) + 1,
        },
    })
}

func (e *PythonExtractor) extractFunction(node *sitter.Node, source []byte, file *File) {
    var name string
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.Type() == "identifier" {
            name = e.getText(child, source)
            break
        }
    }
    
    if name == "" {
        return
    }
    
    exported := !strings.HasPrefix(name, "_")
    
    file.Definitions = append(file.Definitions, Definition{
        Name:     name,
        FullName: file.Module + "." + name,
        Kind:     KindFunction,
        Exported: exported,
        Location: Location{
            File:   file.Path,
            Line:   int(node.StartPoint().Row) + 1,
            Column: int(node.StartPoint().Column) + 1,
        },
    })
}

func (e *PythonExtractor) extractClass(node *sitter.Node, source []byte, file *File) {
    var name string
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.Type() == "identifier" {
            name = e.getText(child, source)
            break
        }
    }
    
    if name == "" {
        return
    }
    
    exported := !strings.HasPrefix(name, "_")
    
    file.Definitions = append(file.Definitions, Definition{
        Name:     name,
        FullName: file.Module + "." + name,
        Kind:     KindClass,
        Exported: exported,
        Location: Location{
            File:   file.Path,
            Line:   int(node.StartPoint().Row) + 1,
            Column: int(node.StartPoint().Column) + 1,
        },
    })
}

func (e *PythonExtractor) extractCall(node *sitter.Node, source []byte, file *File) {
    // Get function being called
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.Type() == "attribute" || child.Type() == "identifier" {
            name := e.getText(child, source)
            
            file.References = append(file.References, Reference{
                Name: name,
                Location: Location{
                    File:   file.Path,
                    Line:   int(node.StartPoint().Row) + 1,
                    Column: int(node.StartPoint().Column) + 1,
                },
            })
        }
    }
}

func (e *PythonExtractor) getText(node *sitter.Node, source []byte) string {
    return string(source[node.StartByte():node.EndByte()])
}
```

### Go Extractor

```go
// internal/parser/golang.go

type GoExtractor struct{}

func (e *GoExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
    file := &File{
        Path:     filePath,
        Language: "go",
        ParsedAt: time.Now(),
    }
    
    e.walk(root, source, file)
    
    return file, nil
}

func (e *GoExtractor) walk(node *sitter.Node, source []byte, file *File) {
    nodeType := node.Type()
    
    switch nodeType {
    case "package_clause":
        e.extractPackage(node, source, file)
    case "import_declaration":
        e.extractImports(node, source, file)
    case "function_declaration":
        e.extractFunction(node, source, file)
    case "type_declaration":
        e.extractType(node, source, file)
    case "call_expression":
        e.extractCall(node, source, file)
    }
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        e.walk(child, source, file)
    }
}

func (e *GoExtractor) extractPackage(node *sitter.Node, source []byte, file *File) {
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.Type() == "package_identifier" {
            file.PackageName = e.getText(child, source)
        }
    }
}

func (e *GoExtractor) extractImports(node *sitter.Node, source []byte, file *File) {
    // import "fmt"
    // import alias "github.com/user/pkg"
    // import (
    //     "fmt"
    //     "strings"
    // )
    
    e.walkImports(node, source, file)
}

func (e *GoExtractor) walkImports(node *sitter.Node, source []byte, file *File) {
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        
        if child.Type() == "import_spec" {
            var alias, path string
            
            for j := 0; j < int(child.ChildCount()); j++ {
                spec := child.Child(j)
                
                if spec.Type() == "package_identifier" {
                    alias = e.getText(spec, source)
                } else if spec.Type() == "interpreted_string_literal" {
                    path = strings.Trim(e.getText(spec, source), "\"")
                }
            }
            
            if path != "" {
                file.Imports = append(file.Imports, Import{
                    Module:    path,
                    RawImport: path,
                    Alias:     alias,
                    Location: Location{
                        File:   file.Path,
                        Line:   int(child.StartPoint().Row) + 1,
                        Column: int(child.StartPoint().Column) + 1,
                    },
                })
            }
        } else {
            e.walkImports(child, source, file)
        }
    }
}

func (e *GoExtractor) extractFunction(node *sitter.Node, source []byte, file *File) {
    var name string
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.Type() == "identifier" {
            name = e.getText(child, source)
            break
        }
    }
    
    if name == "" {
        return
    }
    
    // Exported if starts with uppercase
    exported := len(name) > 0 && unicode.IsUpper(rune(name[0]))
    
    file.Definitions = append(file.Definitions, Definition{
        Name:     name,
        FullName: file.Module + "." + name,
        Kind:     KindFunction,
        Exported: exported,
        Location: Location{
            File:   file.Path,
            Line:   int(node.StartPoint().Row) + 1,
            Column: int(node.StartPoint().Column) + 1,
        },
    })
}

func (e *GoExtractor) extractType(node *sitter.Node, source []byte, file *File) {
    // type MyType struct { ... }
    // type MyInterface interface { ... }
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        if child.Type() == "type_spec" {
            e.extractTypeSpec(child, source, file)
        }
    }
}

func (e *GoExtractor) extractTypeSpec(node *sitter.Node, source []byte, file *File) {
    var name string
    var kind DefinitionKind = KindType
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        
        if child.Type() == "type_identifier" {
            name = e.getText(child, source)
        } else if child.Type() == "interface_type" {
            kind = KindInterface
        }
    }
    
    if name == "" {
        return
    }
    
    exported := len(name) > 0 && unicode.IsUpper(rune(name[0]))
    
    file.Definitions = append(file.Definitions, Definition{
        Name:     name,
        FullName: file.Module + "." + name,
        Kind:     kind,
        Exported: exported,
        Location: Location{
            File:   file.Path,
            Line:   int(node.StartPoint().Row) + 1,
            Column: int(node.StartPoint().Column) + 1,
        },
    })
}

func (e *GoExtractor) extractCall(node *sitter.Node, source []byte, file *File) {
    // function()
    // pkg.Function()
    // obj.Method()
    
    for i := 0; i < int(node.ChildCount()); i++ {
        child := node.Child(i)
        
        if child.Type() == "identifier" || child.Type() == "selector_expression" {
            name := e.getText(child, source)
            
            file.References = append(file.References, Reference{
                Name: name,
                Location: Location{
                    File:   file.Path,
                    Line:   int(node.StartPoint().Row) + 1,
                    Column: int(node.StartPoint().Column) + 1,
                },
            })
        }
    }
}

func (e *GoExtractor) getText(node *sitter.Node, source []byte) string {
    return string(source[node.StartByte():node.EndByte()])
}
```

---

## File Watcher with Debouncing

```go
// internal/watcher/watcher.go

type Watcher struct {
    fsWatcher   *fsnotify.Watcher
    debounce    time.Duration
    excludeDirs []glob.Glob
    excludeFiles []glob.Glob
    onChange    func([]string)
    
    pending     map[string]time.Time
    pendingMu   sync.Mutex
    timer       *time.Timer
}

func NewWatcher(debounce time.Duration, excludeDirs, excludeFiles []string, onChange func([]string)) (*Watcher, error) {
    fsw, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }
    
    w := &Watcher{
        fsWatcher:   fsw,
        debounce:    debounce,
        onChange:    onChange,
        pending:     make(map[string]time.Time),
    }
    
    // Compile globs
    for _, pattern := range excludeDirs {
        g, err := glob.Compile(pattern)
        if err != nil {
            return nil, err
        }
        w.excludeDirs = append(w.excludeDirs, g)
    }
    
    for _, pattern := range excludeFiles {
        g, err := glob.Compile(pattern)
        if err != nil {
            return nil, err
        }
        w.excludeFiles = append(w.excludeFiles, g)
    }
    
    return w, nil
}

func (w *Watcher) Watch(paths []string) error {
    for _, path := range paths {
        if err := w.watchRecursive(path); err != nil {
            return err
        }
    }
    
    go w.run()
    return nil
}

func (w *Watcher) watchRecursive(root string) error {
    return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if info.IsDir() {
            if w.shouldExcludeDir(path) {
                return filepath.SkipDir
            }
            return w.fsWatcher.Add(path)
        }
        
        return nil
    })
}

func (w *Watcher) run() {
    for {
        select {
        case event, ok := <-w.fsWatcher.Events:
            if !ok {
                return
            }
            
            if w.shouldExcludeFile(event.Name) {
                continue
            }
            
            if event.Op&fsnotify.Write == fsnotify.Write ||
               event.Op&fsnotify.Create == fsnotify.Create ||
               event.Op&fsnotify.Remove == fsnotify.Remove {
                w.scheduleChange(event.Name)
            }
            
        case err, ok := <-w.fsWatcher.Errors:
            if !ok {
                return
            }
            slog.Error("watcher error", "error", err)
        }
    }
}

func (w *Watcher) scheduleChange(path string) {
    w.pendingMu.Lock()
    defer w.pendingMu.Unlock()
    
    w.pending[path] = time.Now()
    
    if w.timer != nil {
        w.timer.Stop()
    }
    
    w.timer = time.AfterFunc(w.debounce, func() {
        w.flushChanges()
    })
}

func (w *Watcher) flushChanges() {
    w.pendingMu.Lock()
    paths := make([]string, 0, len(w.pending))
    for path := range w.pending {
        paths = append(paths, path)
    }
    w.pending = make(map[string]time.Time)
    w.pendingMu.Unlock()
    
    if len(paths) > 0 {
        w.onChange(paths)
    }
}

func (w *Watcher) shouldExcludeDir(path string) bool {
    base := filepath.Base(path)
    for _, g := range w.excludeDirs {
        if g.Match(base) {
            return true
        }
    }
    return false
}

func (w *Watcher) shouldExcludeFile(path string) bool {
    base := filepath.Base(path)
    
    // Always exclude test files
    if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.py") {
        return true
    }
    
    for _, g := range w.excludeFiles {
        if g.Match(base) {
            return true
        }
    }
    return false
}

func (w *Watcher) Close() error {
    if w.timer != nil {
        w.timer.Stop()
    }
    return w.fsWatcher.Close()
}
```

---

## DOT Output Generation

```go
// internal/output/dot.go

type DOTGenerator struct {
    graph *graph.Graph
}

func NewDOTGenerator(g *graph.Graph) *DOTGenerator {
    return &DOTGenerator{graph: g}
}

func (d *DOTGenerator) Generate(cycles [][]string) (string, error) {
    var buf strings.Builder
    
    buf.WriteString("digraph dependencies {\n")
    buf.WriteString("  rankdir=LR;\n")
    buf.WriteString("  node [shape=box, style=rounded];\n\n")
    
    // Build cycle edge set for highlighting
    cycleEdges := make(map[string]map[string]bool)
    for _, cycle := range cycles {
        for i := 0; i < len(cycle); i++ {
            from := cycle[i]
            to := cycle[(i+1)%len(cycle)]
            if cycleEdges[from] == nil {
                cycleEdges[from] = make(map[string]bool)
            }
            cycleEdges[from][to] = true
        }
    }
    
    // Nodes
    d.graph.mu.RLock()
    for modName, mod := range d.graph.modules {
        funcCount := len(mod.Exports)
        fileCount := len(mod.Files)
        
        label := fmt.Sprintf("%s\\n(%d funcs, %d files)", 
            modName, funcCount, fileCount)
        
        // Color nodes involved in cycles
        inCycle := false
        for _, cycle := range cycles {
            for _, m := range cycle {
                if m == modName {
                    inCycle = true
                    break
                }
            }
        }
        
        if inCycle {
            buf.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", fillcolor=\"#ffcccc\", style=\"rounded,filled\"];\n",
                modName, label))
        } else {
            buf.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\"];\n",
                modName, label))
        }
    }
    buf.WriteString("\n")
    
    // Edges
    for from, targets := range d.graph.imports {
        for to := range targets {
            isCycle := cycleEdges[from] != nil && cycleEdges[from][to]
            
            if isCycle {
                buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [color=\"#ff0000\", penwidth=2.0, label=\"CYCLE\"];\n",
                    from, to))
            } else {
                buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n",
                    from, to))
            }
        }
    }
    d.graph.mu.RUnlock()
    
    // Legend
    if len(cycles) > 0 {
        buf.WriteString("\n  subgraph cluster_legend {\n")
        buf.WriteString("    label=\"Legend\";\n")
        buf.WriteString("    style=dashed;\n")
        buf.WriteString("    legend_normal [label=\"Normal Import\", shape=plaintext];\n")
        buf.WriteString("    legend_cycle [label=\"Circular Import\", shape=plaintext, fontcolor=\"#ff0000\"];\n")
        buf.WriteString("  }\n")
    }
    
    buf.WriteString("}\n")
    
    return buf.String(), nil
}
```

---

## Main Entry Point

```go
// cmd/circular/main.go

package main

import (
    "flag"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
    "time"
    
    "circular/internal/config"
    "circular/internal/graph"
    "circular/internal/output"
    "circular/internal/parser"
    "circular/internal/resolver"
    "circular/internal/watcher"
)

var (
    configPath = flag.String("config", "./circular.toml", "Path to config file")
    once       = flag.Bool("once", false, "Run single scan and exit")
    verbose    = flag.Bool("verbose", false, "Enable verbose logging")
    version    = flag.Bool("version", false, "Print version and exit")
)

const VERSION = "1.0.0"

func main() {
    flag.Parse()
    
    if *version {
        fmt.Printf("circular v%s\n", VERSION)
        os.Exit(0)
    }
    
    // Setup logging
    logLevel := slog.LevelInfo
    if *verbose {
        logLevel = slog.LevelDebug
    }
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: logLevel,
    }))
    slog.SetDefault(logger)
    
    // Load config
    cfg, err := config.Load(*configPath)
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }
    
    // Initialize components
    grammarLoader, err := parser.NewGrammarLoader(cfg.GrammarsPath)
    if err != nil {
        slog.Error("failed to load grammars", "error", err)
        os.Exit(1)
    }
    
    parser := parser.NewParser(grammarLoader)
    graph := graph.NewGraph()
    
    // Initial scan
    slog.Info("starting initial scan", "paths", cfg.WatchPaths)
    start := time.Now()
    
    files, err := scanDirectories(cfg.WatchPaths, cfg.Exclude.Dirs, cfg.Exclude.Files)
    if err != nil {
        slog.Error("scan failed", "error", err)
        os.Exit(1)
    }
    
    for _, filePath := range files {
        if err := processFile(filePath, parser, graph, cfg); err != nil {
            slog.Warn("failed to process file", "path", filePath, "error", err)
        }
    }
    
    duration := time.Since(start)
    
    // Analyze
    cycles := graph.DetectCycles()
    hallucinations := analyzeHallucinations(graph, cfg)
    
    // Output
    if err := generateOutputs(graph, cycles, cfg); err != nil {
        slog.Error("failed to generate outputs", "error", err)
        os.Exit(1)
    }
    
    // Print summary
    printSummary(len(files), len(graph.Modules()), duration, cycles, hallucinations)
    
    if *once {
        os.Exit(0)
    }
    
    // Watch mode
    slog.Info("entering watch mode", "debounce", cfg.Watch.Debounce)
    
    w, err := watcher.NewWatcher(
        cfg.Watch.Debounce,
        cfg.Exclude.Dirs,
        cfg.Exclude.Files,
        func(changedPaths []string) {
            handleChanges(changedPaths, parser, graph, cfg)
        },
    )
    if err != nil {
        slog.Error("failed to create watcher", "error", err)
        os.Exit(1)
    }
    defer w.Close()
    
    if err := w.Watch(cfg.WatchPaths); err != nil {
        slog.Error("failed to start watching", "error", err)
        os.Exit(1)
    }
    
    // Block forever
    select {}
}

func processFile(path string, p *parser.Parser, g *graph.Graph, cfg *config.Config) error {
    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    
    file, err := p.ParseFile(path, content)
    if err != nil {
        return err
    }
    
    // Resolve module name
    if file.Language == "python" {
        resolver := resolver.NewPythonResolver(cfg.WatchPaths[0])
        file.Module = resolver.GetModuleName(path)
    } else if file.Language == "go" {
        resolver := resolver.NewGoResolver()
        if err := resolver.FindGoMod(path); err == nil {
            file.Module = resolver.GetModuleName(path)
        }
    }
    
    g.AddFile(file)
    return nil
}

func handleChanges(paths []string, p *parser.Parser, g *graph.Graph, cfg *config.Config) {
    slog.Info("detected changes", "count", len(paths))
    
    start := time.Now()
    
    // Re-process changed files
    for _, path := range paths {
        if _, err := os.Stat(path); os.IsNotExist(err) {
            g.RemoveFile(path)
            continue
        }
        
        if err := processFile(path, p, g, cfg); err != nil {
            slog.Warn("failed to re-process file", "path", path, "error", err)
        }
    }
    
    // Get transitively affected files
    affected := g.InvalidateTransitive(paths[0])
    slog.Debug("transitive invalidation", "affected", len(affected))
    
    // Re-analyze
    cycles := g.DetectCycles()
    hallucinations := analyzeHallucinations(g, cfg)
    
    // Regenerate outputs
    if err := generateOutputs(g, cycles, cfg); err != nil {
        slog.Error("failed to regenerate outputs", "error", err)
    }
    
    duration := time.Since(start)
    
    // Print update
    printSummary(len(paths), len(g.Modules()), duration, cycles, hallucinations)
    
    if cfg.Alerts.Beep && (len(cycles) > 0 || len(hallucinations) > 0) {
        fmt.Print("\a")  // Terminal bell
    }
}

func analyzeHallucinations(g *graph.Graph, cfg *config.Config) []resolver.UnresolvedReference {
    res := resolver.NewResolver(g)
    return res.FindUnresolved()
}

func generateOutputs(g *graph.Graph, cycles [][]string, cfg *config.Config) error {
    // DOT output
    if cfg.Output.DOT != "" {
        dotGen := output.NewDOTGenerator(g)
        dot, err := dotGen.Generate(cycles)
        if err != nil {
            return err
        }
        
        if err := os.WriteFile(cfg.Output.DOT, []byte(dot), 0644); err != nil {
            return err
        }
        slog.Debug("wrote DOT file", "path", cfg.Output.DOT)
    }
    
    // TSV output
    if cfg.Output.TSV != "" {
        tsvGen := output.NewTSVGenerator(g)
        tsv, err := tsvGen.Generate()
        if err != nil {
            return err
        }
        
        if err := os.WriteFile(cfg.Output.TSV, []byte(tsv), 0644); err != nil {
            return err
        }
        slog.Debug("wrote TSV file", "path", cfg.Output.TSV)
    }
    
    return nil
}

func printSummary(fileCount, moduleCount int, duration time.Duration, cycles [][]string, hallucinations []resolver.UnresolvedReference) {
    if !cfg.Alerts.Terminal {
        return
    }
    
    fmt.Println()
    fmt.Println("══════════════════════════════════════════════════════════")
    fmt.Printf("[%s] Scan Complete\n", time.Now().Format("15:04:05"))
    fmt.Println("══════════════════════════════════════════════════════════")
    fmt.Println()
    fmt.Printf("📁 Scanned: %d files (%d modules)\n", fileCount, moduleCount)
    fmt.Printf("⏱️  Duration: %s\n\n", duration.Round(time.Millisecond))
    
    if len(cycles) == 0 {
        fmt.Println("✅ No circular imports detected")
    } else {
        fmt.Printf("🔴 Circular imports detected (%d):\n", len(cycles))
        for i, cycle := range cycles {
            fmt.Printf("   Cycle %d: %s\n", i+1, formatCycle(cycle))
        }
    }
    fmt.Println()
    
    if len(hallucinations) == 0 {
        fmt.Println("✅ No potential hallucinations detected")
    } else {
        fmt.Printf("⚠️  Potential hallucinations (%d):\n", len(hallucinations))
        count := min(10, len(hallucinations))
        for i := 0; i < count; i++ {
            h := hallucinations[i]
            fmt.Printf("   %s:%d - call to undefined '%s'\n",
                h.File, h.Reference.Location.Line, h.Reference.Name)
        }
        if len(hallucinations) > 10 {
            fmt.Printf("   ... and %d more\n", len(hallucinations)-10)
        }
    }
    fmt.Println()
    
    if cfg.Output.DOT != "" {
        fmt.Printf("📊 Output: %s (updated)\n", cfg.Output.DOT)
    }
    fmt.Println()
}

func formatCycle(cycle []string) string {
    if len(cycle) == 0 {
        return ""
    }
    return fmt.Sprintf("%s → %s", strings.Join(cycle, " → "), cycle[0])
}

func scanDirectories(paths, excludeDirs, excludeFiles []string) ([]string, error) {
    var files []string
    
    // Compile exclude patterns
    excludeDirGlobs := make([]glob.Glob, len(excludeDirs))
    for i, pattern := range excludeDirs {
        g, err := glob.Compile(pattern)
        if err != nil {
            return nil, err
        }
        excludeDirGlobs[i] = g
    }
    
    excludeFileGlobs := make([]glob.Glob, len(excludeFiles))
    for i, pattern := range excludeFiles {
        g, err := glob.Compile(pattern)
        if err != nil {
            return nil, err
        }
        excludeFileGlobs[i] = g
    }
    
    for _, root := range paths {
        err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
            if err != nil {
                return err
            }
            
            if info.IsDir() {
                base := filepath.Base(path)
                for _, g := range excludeDirGlobs {
                    if g.Match(base) {
                        return filepath.SkipDir
                    }
                }
                return nil
            }
            
            // Check file extension
            ext := filepath.Ext(path)
            if ext != ".py" && ext != ".go" {
                return nil
            }
            
            // Check exclude patterns
            base := filepath.Base(path)
            
            // Always exclude test files
            if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.py") {
                return nil
            }
            
            for _, g := range excludeFileGlobs {
                if g.Match(base) {
                    return nil
                }
            }
            
            files = append(files, path)
            return nil
        })
        
        if err != nil {
            return nil, err
        }
    }
    
    return files, nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

---

## Updated Dependencies (go.mod)

```go
module circular

go 1.23

require (
    github.com/BurntSushi/toml v1.4.0
    github.com/fsnotify/fsnotify v1.7.0
    github.com/gobwas/glob v0.2.3
    github.com/tree-sitter/go-tree-sitter v0.25.0
)
```

---

## Configuration Example

```toml
# circular.toml

# Directories to watch (relative or absolute paths)
watch_paths = ["./src", "./internal"]

# Path to compiled grammars
grammars_path = "./grammars"

[exclude]
# Directories to skip
dirs = ["venv", "node_modules", ".git", "__pycache__", "vendor", "dist", "build", ".pytest_cache"]

# File patterns to skip (glob syntax)
# Test files are automatically excluded
files = ["*_gen.go", "*.min.js", "*_pb.go"]

[output]
# DOT file path (for KGraphViewer)
dot = "./code_map.dot"

# TSV file path (human readable)
tsv = "./code_map.tsv"

[alerts]
# Print alerts to terminal
terminal = true

# Beep on issues (terminal bell)
beep = false

[watch]
# Debounce interval for file changes
debounce = "500ms"
```

---

## Memory Optimization Notes

For 100-1000 file codebases:

1. **String Interning:** Consider using a string interner for module names to reduce memory:
```go
type StringInterner struct {
    mu      sync.RWMutex
    strings map[string]string
}

func (si *StringInterner) Intern(s string) string {
    si.mu.RLock()
    if interned, ok := si.strings[s]; ok {
        si.mu.RUnlock()
        return interned
    }
    si.mu.RUnlock()
    
    si.mu.Lock()
    defer si.mu.Unlock()
    
    if interned, ok := si.strings[s]; ok {
        return interned
    }
    
    si.strings[s] = s
    return s
}
```

2. **Node Cleanup:** Ensure tree-sitter trees are properly closed:
```go
defer tree.Close()
```

3. **Incremental Parsing:** When a file changes, reuse previous parse tree:
```go
oldTree := p.trees[path]
newTree := parser.Parse(content, oldTree)
```

4. **Reference Pooling:** Use sync.Pool for temporary allocations:
```go
var nodePool = sync.Pool{
    New: func() interface{} {
        return &[]Reference{}
    },
}
```

---

## Implementation Order

### Phase 1: Foundation
1. `go.mod` + dependencies
2. `internal/config/config.go` - Config parsing
3. `internal/parser/loader.go` - Grammar loading
4. `internal/parser/types.go` - Data structures
5. `internal/parser/parser.go` - Generic parser
6. `internal/parser/python.go` - Python extractor (imports + functions only)
7. `internal/graph/graph.go` - Thread-safe graph
8. `internal/output/dot.go` - DOT generation
9. `cmd/circular/main.go` - Entry point (single scan only)
10. Test with small Python project

### Phase 2: Go Support
1. `internal/parser/golang.go` - Go extractor
2. `internal/resolver/go_resolver.go` - Go module resolution
3. Test with small Go project

### Phase 3: Python Module Resolution
1. `internal/resolver/python_resolver.go` - Python import resolution
2. `internal/resolver/stdlib.go` - Embed stdlib lists
3. Test import resolution

### Phase 4: Watching
1. `internal/watcher/watcher.go` - File watcher + debouncing
2. Update `main.go` for watch mode
3. Test live updates

### Phase 5: Cycle Detection
1. `internal/graph/detect.go` - DFS cycle detection
2. Update DOT output to highlight cycles
3. Test with known circular imports

### Phase 6: Hallucination Detection
1. `internal/resolver/resolver.go` - Symbol resolution
2. Extract function calls in extractors
3. Build symbol tables
4. Cross-reference
5. Test with known undefined calls

### Phase 7: Polish
1. `internal/output/tsv.go` - TSV generation
2. Improve terminal output formatting
3. Add stats (LOC, complexity, etc.)
4. Performance profiling
5. Memory leak testing

---

## Testing Checkpoints

After each phase:
```bash
# Run on a real project
./circular --config ./test-config.toml --once -verbose

# Check DOT output
kgraphviewer code_map.dot

# Watch mode
./circular --config ./test-config.toml

# Make changes to watched files, observe updates
```

---

## Known Limitations

1. **Dynamic Imports:** Cannot resolve `__import__()`, `importlib.import_module()`, or Go plugins
2. **Reflection:** Cannot track reflect-based calls in Go
3. **External Dependencies:** Only tracks local code + stdlib
4. **Build Tags:** Go files with build tags always included
5. **Generated Code:** Should be excluded via config

---

This plan now addresses all the critical issues from the review. Ready to start implementation?