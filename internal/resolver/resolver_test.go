// # internal/resolver/resolver_test.go
package resolver

import (
	"circular/internal/graph"
	"circular/internal/parser"
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

	res := NewResolver(g, nil)
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

	res := NewResolver(g, []string{"log"})
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

	r := NewResolver(g, nil)
	unresolved := r.FindUnresolvedForPaths([]string{"a.go"})

	if len(unresolved) != 1 {
		t.Fatalf("Expected 1 unresolved reference, got %d", len(unresolved))
	}
	if unresolved[0].File != "a.go" {
		t.Fatalf("Expected unresolved reference from a.go, got %s", unresolved[0].File)
	}
}
