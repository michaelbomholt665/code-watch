// # internal/graph/graph_test.go
package graph

import (
	"circular/internal/parser"
	"testing"
)

func TestGraph_AddRemoveFile(t *testing.T) {
	g := NewGraph()

	f1 := &parser.File{
		Path:   "/path/to/a.go",
		Module: "moduleA",
		Definitions: []parser.Definition{
			{Name: "FuncA", Exported: true},
		},
		Imports: []parser.Import{
			{Module: "moduleB"},
		},
	}

	g.AddFile(f1)

	if len(g.files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(g.files))
	}
	if len(g.modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(g.modules))
	}
	if _, ok := g.imports["moduleA"]["moduleB"]; !ok {
		t.Error("Expected import edge from moduleA to moduleB")
	}
	if !g.importedBy["moduleB"]["moduleA"] {
		t.Error("Expected importedBy entry for moduleB from moduleA")
	}

	g.RemoveFile("/path/to/a.go")
	if len(g.files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(g.files))
	}
	if len(g.modules) != 0 {
		t.Errorf("Expected 0 modules, got %d", len(g.modules))
	}
	if len(g.importedBy["moduleB"]) != 0 {
		t.Error("Expected moduleB importedBy to be empty")
	}
}

func TestGraph_DetectCycles(t *testing.T) {
	g := NewGraph()

	// A -> B -> C -> A
	g.AddFile(&parser.File{
		Path:    "a.go",
		Module:  "A",
		Imports: []parser.Import{{Module: "B"}},
	})
	g.AddFile(&parser.File{
		Path:    "b.go",
		Module:  "B",
		Imports: []parser.Import{{Module: "C"}},
	})
	g.AddFile(&parser.File{
		Path:    "c.go",
		Module:  "C",
		Imports: []parser.Import{{Module: "A"}},
	})

	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Fatalf("Expected 1 cycle, got %d", len(cycles))
	}

	cycle := cycles[0]
	if len(cycle) != 3 {
		t.Errorf("Expected cycle length 3, got %d", len(cycle))
	}

	// Verify cycle content (order might vary but should be A, B, C)
	found := make(map[string]bool)
	for _, m := range cycle {
		found[m] = true
	}
	if !found["A"] || !found["B"] || !found["C"] {
		t.Errorf("Unexpected cycle content: %v", cycle)
	}
}

func TestGraph_InvalidateTransitive(t *testing.T) {
	g := NewGraph()

	// C -> B -> A
	g.AddFile(&parser.File{Path: "a.go", Module: "A"})
	g.AddFile(&parser.File{Path: "b.go", Module: "B", Imports: []parser.Import{{Module: "A"}}})
	g.AddFile(&parser.File{Path: "c.go", Module: "C", Imports: []parser.Import{{Module: "B"}}})

	affected := g.InvalidateTransitive("a.go")
	if len(affected) != 3 {
		t.Errorf("Expected 3 affected files, got %d: %v", len(affected), affected)
	}
}

func TestGraph_Getters(t *testing.T) {
	g := NewGraph()
	f := &parser.File{Path: "test.go", Module: "mod"}
	g.AddFile(f)

	if _, ok := g.GetModule("mod"); !ok {
		t.Error("GetModule failed")
	}
	if len(g.Modules()) != 1 {
		t.Error("Modules failed")
	}
	if _, ok := g.GetFile("test.go"); !ok {
		t.Error("GetFile failed")
	}
	if len(g.GetAllFiles()) != 1 {
		t.Error("GetAllFiles failed")
	}
	if len(g.GetImports()) != 1 {
		t.Error("GetImports failed")
	}
}

func TestGraph_RemoveFile_Incremental(t *testing.T) {
	g := NewGraph()

	f1 := &parser.File{
		Path:        "f1.go",
		Module:      "mod",
		Definitions: []parser.Definition{{Name: "Func1", Exported: true}},
		Imports:     []parser.Import{{Module: "other"}},
	}
	f2 := &parser.File{
		Path:        "f2.go",
		Module:      "mod",
		Definitions: []parser.Definition{{Name: "Func2", Exported: true}},
	}

	g.AddFile(f1)
	g.AddFile(f2)

	if len(g.modules["mod"].Files) != 2 {
		t.Errorf("Expected 2 files in module, got %d", len(g.modules["mod"].Files))
	}

	g.RemoveFile("f1.go")

	mod, ok := g.modules["mod"]
	if !ok {
		t.Fatal("Module 'mod' should still exist")
	}
	if len(mod.Files) != 1 || mod.Files[0] != "f2.go" {
		t.Errorf("Expected only f2.go, got %v", mod.Files)
	}
	if _, ok := mod.Exports["Func1"]; ok {
		t.Error("Func1 should have been removed from exports")
	}
	if _, ok := mod.Exports["Func2"]; !ok {
		t.Error("Func2 should still be in exports")
	}
	if len(g.imports["mod"]) != 0 {
		t.Error("Imports should be empty after removing f1.go")
	}
}

func TestGraph_DirtyTracking(t *testing.T) {
	g := NewGraph()
	g.MarkDirty([]string{"f1.go", "f2.go"})

	dirty := g.GetDirty()
	if len(dirty) != 2 {
		t.Errorf("Expected 2 dirty files, got %d", len(dirty))
	}

	// Should be empty now
	dirty = g.GetDirty()
	if len(dirty) != 0 {
		t.Error("Expected dirty set to be empty after GetDirty")
	}
}

func TestGraph_AddFile_ReplacesExistingContributions(t *testing.T) {
	g := NewGraph()

	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Definitions: []parser.Definition{
			{Name: "OldFunc", Exported: true},
		},
		Imports: []parser.Import{
			{Module: "modB"},
		},
	})

	// Re-add same path with updated definitions/imports.
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Definitions: []parser.Definition{
			{Name: "NewFunc", Exported: true},
		},
		Imports: []parser.Import{
			{Module: "modC"},
		},
	})

	imports := g.GetImports()
	if _, ok := imports["modA"]["modB"]; ok {
		t.Fatal("stale import modA->modB should have been removed")
	}
	if _, ok := imports["modA"]["modC"]; !ok {
		t.Fatal("expected updated import modA->modC")
	}

	defs, ok := g.GetDefinitions("modA")
	if !ok {
		t.Fatal("expected definitions for modA")
	}
	if _, ok := defs["OldFunc"]; ok {
		t.Fatal("stale definition OldFunc should have been removed")
	}
	if _, ok := defs["NewFunc"]; !ok {
		t.Fatal("expected definition NewFunc")
	}
}
