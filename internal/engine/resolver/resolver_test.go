// # internal/resolver/resolver_test.go
package resolver

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"os"
	"path/filepath"
	"testing"
)

func TestPythonResolver_GetModuleName(t *testing.T) {
	root, _ := os.MkdirTemp("", "pyproj")
	defer os.RemoveAll(root)

	// Create structure:
	// root/src/auth/__init__.py
	// root/src/auth/utils.py
	// root/src/app.py
	src := filepath.Join(root, "src")
	auth := filepath.Join(src, "auth")
	os.MkdirAll(auth, 0755)
	os.WriteFile(filepath.Join(auth, "__init__.py"), []byte(""), 0644)

	r := NewPythonResolver(root)

	tests := []struct {
		path     string
		expected string
	}{
		{filepath.Join(auth, "utils.py"), "auth.utils"},
		{filepath.Join(auth, "__init__.py"), "auth"},
		{filepath.Join(src, "app.py"), "app"},
	}

	for _, tt := range tests {
		got := r.GetModuleName(tt.path)
		if got != tt.expected {
			t.Errorf("GetModuleName(%s) = %s, expected %s", tt.path, got, tt.expected)
		}
	}
}

func TestGoResolver_GetModuleName(t *testing.T) {
	root, _ := os.MkdirTemp("", "goproj")
	defer os.RemoveAll(root)

	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/test/proj"), 0644)

	r := NewGoResolver()
	err := r.FindGoMod(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		expected string
	}{
		{filepath.Join(root, "main.go"), "github.com/test/proj"},
		{filepath.Join(root, "internal/auth/login.go"), "github.com/test/proj/internal/auth"},
	}

	for _, tt := range tests {
		got := r.GetModuleName(tt.path)
		if got != tt.expected {
			t.Errorf("GetModuleName(%s) = %s, expected %s", tt.path, got, tt.expected)
		}
	}
}

func TestGoResolver_FindGoMod_Failure(t *testing.T) {
	r := NewGoResolver()
	err := r.FindGoMod("/tmp/definitely/not/a/go/project/main.go")
	if err == nil {
		t.Error("Expected error for non-existent go.mod")
	}
}

func TestPythonResolver_ResolveImport(t *testing.T) {
	r := NewPythonResolver("/")

	tests := []struct {
		from          string
		importStmt    string
		isRelative    bool
		relativeLevel int
		expected      string
	}{
		{"auth.login", "utils", false, 0, "utils"},
		{"auth.login", "validators", true, 1, "auth.validators"},
		{"auth.login", "config", true, 2, "config"},
		{"auth.login", "", true, 1, "auth"},
	}

	for _, tt := range tests {
		got := r.ResolveImport(tt.from, tt.importStmt, tt.isRelative, tt.relativeLevel)
		if got != tt.expected {
			t.Errorf("ResolveImport(%s, %s, %v, %d) = %s, expected %s",
				tt.from, tt.importStmt, tt.isRelative, tt.relativeLevel, got, tt.expected)
		}
	}
}

func TestResolver_FindUnresolved(t *testing.T) {
	g := graph.NewGraph()

	// Module A defines FuncA
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Definitions: []parser.Definition{
			{Name: "FuncA", Exported: true},
		},
	})

	// Module B imports modA and calls FuncA (valid) and FuncMissing (invalid)
	g.AddFile(&parser.File{
		Path:     "b.go",
		Language: "go",
		Module:   "modB",
		Imports: []parser.Import{
			{Module: "modA"},
		},
		References: []parser.Reference{
			{Name: "modA.FuncA"},
			{Name: "modA.FuncMissing"},
			{Name: "len"}, // Builtin
		},
	})

	res := NewResolver(g, nil, nil)
	unresolved := res.FindUnresolved()

	if len(unresolved) != 1 {
		t.Fatalf("Expected 1 unresolved reference, got %d", len(unresolved))
	}

	if unresolved[0].Reference.Name != "modA.FuncMissing" {
		t.Errorf("Expected unresolved modA.FuncMissing, got %s", unresolved[0].Reference.Name)
	}
}

func TestResolver_LocalSymbols(t *testing.T) {
	g := graph.NewGraph()

	g.AddFile(&parser.File{
		Path:         "c.go",
		Language:     "go",
		Module:       "modC",
		LocalSymbols: []string{"p", "ctx"},
		References: []parser.Reference{
			{Name: "p.Register"},
			{Name: "ctx.Done"},
			{Name: "missing.Call"},
		},
	})

	res := NewResolver(g, []string{"log"}, nil)
	unresolved := res.FindUnresolved()

	// Should only have 1 unresolved: missing.Call
	// p.Register and ctx.Done should be ignored as local symbols
	// log.Printf (if added) should be ignored as excluded symbol

	found := false
	for _, u := range unresolved {
		if u.Reference.Name == "missing.Call" {
			found = true
		} else {
			t.Errorf("Symbol %s should have been resolved as local or excluded", u.Reference.Name)
		}
	}

	if !found {
		t.Error("missing.Call should be unresolved")
	}

	if len(unresolved) != 1 {
		t.Errorf("Expected 1 unresolved reference, got %d", len(unresolved))
	}
}

func TestResolver_LocalSymbolsWithIndexAccess(t *testing.T) {
	g := graph.NewGraph()

	g.AddFile(&parser.File{
		Path:         "idx.go",
		Language:     "go",
		Module:       "modIdx",
		LocalSymbols: []string{"items"},
		References: []parser.Reference{
			{Name: "items[i].Field"},
			{Name: "items[i].Other"},
		},
	})

	res := NewResolver(g, nil, nil)
	unresolved := res.FindUnresolved()

	if len(unresolved) != 0 {
		t.Fatalf("expected no unresolved references, got %d", len(unresolved))
	}
}

func TestResolver_FindUnresolvedForPaths(t *testing.T) {
	g := graph.NewGraph()

	g.AddFile(&parser.File{
		Path:     "a.go",
		Language: "go",
		Module:   "modA",
		References: []parser.Reference{
			{Name: "missing.Call"},
		},
	})

	g.AddFile(&parser.File{
		Path:     "b.go",
		Language: "go",
		Module:   "modB",
		References: []parser.Reference{
			{Name: "len"}, // builtin; should resolve
		},
	})

	r := NewResolver(g, nil, nil)
	unresolved := r.FindUnresolvedForPaths([]string{"a.go"})

	if len(unresolved) != 1 {
		t.Fatalf("Expected 1 unresolved reference, got %d", len(unresolved))
	}
	if unresolved[0].File != "a.go" {
		t.Fatalf("Expected unresolved reference from a.go, got %s", unresolved[0].File)
	}
}

func TestResolver_FindUnusedImports(t *testing.T) {
	g := graph.NewGraph()

	g.AddFile(&parser.File{
		Path:     "sample.py",
		Language: "python",
		Module:   "sample",
		Imports: []parser.Import{
			{Module: "auth", Alias: "a", Location: parser.Location{File: "sample.py", Line: 1, Column: 1}},
			{Module: "math", Items: []string{"sqrt", "pow"}, Location: parser.Location{File: "sample.py", Line: 2, Column: 1}},
		},
		References: []parser.Reference{
			{Name: "a.login"},
			{Name: "sqrt"},
		},
	})

	g.AddFile(&parser.File{
		Path:     "sample.go",
		Language: "go",
		Module:   "samplego",
		Imports: []parser.Import{
			{Module: "fmt", Location: parser.Location{File: "sample.go", Line: 1, Column: 1}},
			{Module: "log/slog", Alias: "_", Location: parser.Location{File: "sample.go", Line: 2, Column: 1}},
		},
		References: []parser.Reference{
			{Name: "fmt.Println"},
		},
	})

	r := NewResolver(g, nil, nil)
	unused := r.FindUnusedImports([]string{"sample.py", "sample.go"})
	if len(unused) != 1 {
		t.Fatalf("Expected 1 unused import finding, got %d", len(unused))
	}

	if unused[0].File != "sample.py" {
		t.Fatalf("Expected unused import in sample.py, got %s", unused[0].File)
	}
	if unused[0].Module != "math" || unused[0].Item != "pow" {
		t.Fatalf("Expected unused item math.pow, got module=%s item=%s", unused[0].Module, unused[0].Item)
	}
	if unused[0].Confidence != "high" {
		t.Fatalf("Expected high confidence for from-import item, got %s", unused[0].Confidence)
	}
}

func TestResolver_FindUnusedImports_ExcludeImports(t *testing.T) {
	g := graph.NewGraph()

	g.AddFile(&parser.File{
		Path:     "sample.go",
		Language: "go",
		Module:   "samplego",
		Imports: []parser.Import{
			{Module: "fmt", Location: parser.Location{File: "sample.go", Line: 1, Column: 1}},
		},
		References: nil,
	})

	r := NewResolver(g, nil, []string{"fmt"})
	unused := r.FindUnusedImports([]string{"sample.go"})
	if len(unused) != 0 {
		t.Fatalf("expected excluded import to be ignored, got %d findings", len(unused))
	}
}

func TestResolver_ImportReferenceNameByLanguage(t *testing.T) {
	tests := []struct {
		name     string
		language string
		imp      parser.Import
		want     string
	}{
		{
			name:     "go module base",
			language: "go",
			imp:      parser.Import{Module: "log/slog"},
			want:     "slog",
		},
		{
			name:     "python module base",
			language: "python",
			imp:      parser.Import{Module: "urllib.request"},
			want:     "request",
		},
		{
			name:     "javascript scoped package base",
			language: "javascript",
			imp:      parser.Import{Module: "@scope/pkg/sub"},
			want:     "sub",
		},
		{
			name:     "java package base",
			language: "java",
			imp:      parser.Import{Module: "java.util"},
			want:     "util",
		},
		{
			name:     "rust path base",
			language: "rust",
			imp:      parser.Import{Module: "std::io"},
			want:     "io",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := importReferenceName(tc.language, tc.imp)
			if got != tc.want {
				t.Fatalf("importReferenceName(%q, %q) = %q, want %q", tc.language, tc.imp.Module, got, tc.want)
			}
		})
	}
}

func TestResolver_FindUnusedImportsUnsupportedLanguage(t *testing.T) {
	g := graph.NewGraph()
	g.AddFile(&parser.File{
		Path:     "page.html",
		Language: "html",
		Module:   "page",
		Imports: []parser.Import{
			{Module: "header", Location: parser.Location{File: "page.html", Line: 1, Column: 1}},
		},
	})

	r := NewResolver(g, nil, nil)
	unused := r.FindUnusedImports([]string{"page.html"})
	if len(unused) != 0 {
		t.Fatalf("expected no unused import findings for unsupported language, got %d", len(unused))
	}
}

func TestResolver_StdlibIsLanguageScoped(t *testing.T) {
	g := graph.NewGraph()

	g.AddFile(&parser.File{
		Path:     "main.js",
		Language: "javascript",
		Module:   "web",
		References: []parser.Reference{
			{Name: "fs.readFile"},
			{Name: "fmt.Println"},
		},
	})

	r := NewResolver(g, nil, nil)
	unresolved := r.FindUnresolvedForPaths([]string{"main.js"})
	if len(unresolved) != 1 {
		t.Fatalf("expected exactly one unresolved reference, got %d", len(unresolved))
	}
	if unresolved[0].Reference.Name != "fmt.Println" {
		t.Fatalf("expected fmt.Println unresolved for javascript file, got %s", unresolved[0].Reference.Name)
	}
}
