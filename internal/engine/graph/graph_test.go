// # internal/graph/graph_test.go
package graph

import (
	"circular/internal/engine/parser"
	"errors"
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

	if g.FileCount() != 1 {
		t.Errorf("Expected 1 file, got %d", g.FileCount())
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
	if g.FileCount() != 0 {
		t.Errorf("Expected 0 files, got %d", g.FileCount())
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

func TestGraph_ComputeModuleMetrics(t *testing.T) {
	g := NewGraph()

	// A -> B -> C -> A (cycle), D -> B, E isolated
	g.AddFile(&parser.File{Path: "a.go", Module: "A", Imports: []parser.Import{{Module: "B"}}})
	g.AddFile(&parser.File{Path: "b.go", Module: "B", Imports: []parser.Import{{Module: "C"}}})
	g.AddFile(&parser.File{Path: "c.go", Module: "C", Imports: []parser.Import{{Module: "A"}}})
	g.AddFile(&parser.File{Path: "d.go", Module: "D", Imports: []parser.Import{{Module: "B"}}})
	g.AddFile(&parser.File{Path: "e.go", Module: "E"})

	metrics := g.ComputeModuleMetrics()

	if len(metrics) != 5 {
		t.Fatalf("Expected metrics for 5 modules, got %d", len(metrics))
	}

	// Cycle members should have same depth and fan-out 1.
	depthA := metrics["A"].Depth
	if metrics["B"].Depth != depthA || metrics["C"].Depth != depthA {
		t.Fatalf("Expected equal cycle depths, got A=%d B=%d C=%d", metrics["A"].Depth, metrics["B"].Depth, metrics["C"].Depth)
	}
	if metrics["A"].FanOut != 1 || metrics["B"].FanOut != 1 || metrics["C"].FanOut != 1 {
		t.Fatalf("Expected fan-out 1 for cycle nodes, got A=%d B=%d C=%d", metrics["A"].FanOut, metrics["B"].FanOut, metrics["C"].FanOut)
	}
	if metrics["B"].FanIn != 2 {
		t.Fatalf("Expected B fan-in 2 (A and D), got %d", metrics["B"].FanIn)
	}

	// D depends on the cycle and should be one layer deeper.
	if metrics["D"].Depth != depthA+1 {
		t.Fatalf("Expected D depth %d, got %d", depthA+1, metrics["D"].Depth)
	}
	if metrics["D"].FanIn != 0 || metrics["D"].FanOut != 1 {
		t.Fatalf("Expected D fan-in 0 / fan-out 1, got in=%d out=%d", metrics["D"].FanIn, metrics["D"].FanOut)
	}

	// Isolated module should be a leaf.
	if metrics["E"].Depth != 0 || metrics["E"].FanIn != 0 || metrics["E"].FanOut != 0 {
		t.Fatalf("Expected E as isolated leaf, got depth=%d in=%d out=%d", metrics["E"].Depth, metrics["E"].FanIn, metrics["E"].FanOut)
	}
}

func TestGraph_FindImportChain(t *testing.T) {
	g := NewGraph()

	// A -> B -> D
	// A -> C -> D
	// E isolated
	g.AddFile(&parser.File{Path: "a.go", Module: "A", Imports: []parser.Import{{Module: "B"}, {Module: "C"}}})
	g.AddFile(&parser.File{Path: "b.go", Module: "B", Imports: []parser.Import{{Module: "D"}}})
	g.AddFile(&parser.File{Path: "c.go", Module: "C", Imports: []parser.Import{{Module: "D"}}})
	g.AddFile(&parser.File{Path: "d.go", Module: "D"})
	g.AddFile(&parser.File{Path: "e.go", Module: "E"})

	tests := []struct {
		name   string
		from   string
		to     string
		ok     bool
		expect []string
	}{
		{
			name:   "shortest path found",
			from:   "A",
			to:     "D",
			ok:     true,
			expect: []string{"A", "B", "D"},
		},
		{
			name:   "same module",
			from:   "A",
			to:     "A",
			ok:     true,
			expect: []string{"A"},
		},
		{
			name: "no path",
			from: "D",
			to:   "A",
			ok:   false,
		},
		{
			name: "missing source module",
			from: "missing",
			to:   "A",
			ok:   false,
		},
		{
			name: "missing target module",
			from: "A",
			to:   "missing",
			ok:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, ok := g.FindImportChain(tc.from, tc.to)
			if ok != tc.ok {
				t.Fatalf("expected ok=%v, got %v (path=%v)", tc.ok, ok, path)
			}

			if !tc.ok {
				return
			}

			if len(path) != len(tc.expect) {
				t.Fatalf("expected path len %d, got %d: %v", len(tc.expect), len(path), path)
			}
			for i := range tc.expect {
				if path[i] != tc.expect[i] {
					t.Fatalf("expected path %v, got %v", tc.expect, path)
				}
			}
		})
	}
}

func TestLayerRuleEngine_Validate(t *testing.T) {
	g := NewGraph()

	g.AddFile(&parser.File{Path: "internal/api/a.go", Module: "internal/api", Imports: []parser.Import{{Module: "internal/core"}}})
	g.AddFile(&parser.File{Path: "internal/core/b.go", Module: "internal/core", Imports: []parser.Import{{Module: "internal/api"}}})

	engine := NewLayerRuleEngine(ArchitectureModel{
		Enabled: true,
		Layers: []ArchitectureLayer{
			{Name: "api", Paths: []string{"internal/api"}},
			{Name: "core", Paths: []string{"internal/core"}},
		},
		Rules: []ArchitectureRule{
			{Name: "api-to-core-only", From: "api", Allow: []string{"core"}},
			{Name: "core-to-core-only", From: "core", Allow: []string{"core"}},
		},
	})

	violations := engine.Validate(g)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	v := violations[0]
	if v.RuleName != "core-to-core-only" || v.FromLayer != "core" || v.ToLayer != "api" {
		t.Fatalf("unexpected violation: %+v", v)
	}
}

func TestGraph_AnalyzeImpact(t *testing.T) {
	g := NewGraph()

	g.AddFile(&parser.File{Path: "a.go", Module: "A", Imports: []parser.Import{{Module: "B"}}})
	g.AddFile(&parser.File{Path: "b.go", Module: "B", Definitions: []parser.Definition{{Name: "Run", Exported: true}}})
	g.AddFile(&parser.File{Path: "c.go", Module: "C", Imports: []parser.Import{{Module: "A"}}})
	g.AddFile(&parser.File{Path: "d.go", Module: "D", Imports: []parser.Import{{Module: "A"}}})

	report, err := g.AnalyzeImpact("b.go")
	if err != nil {
		t.Fatalf("AnalyzeImpact returned error: %v", err)
	}
	if report.TargetModule != "B" {
		t.Fatalf("expected target module B, got %s", report.TargetModule)
	}
	if len(report.DirectImporters) != 1 || report.DirectImporters[0] != "A" {
		t.Fatalf("unexpected direct importers: %v", report.DirectImporters)
	}
	if len(report.TransitiveImporters) != 2 {
		t.Fatalf("unexpected transitive importers: %v", report.TransitiveImporters)
	}
	if len(report.ExternallyUsedSymbols) != 1 || report.ExternallyUsedSymbols[0] != "Run" {
		t.Fatalf("unexpected externally used symbols: %v", report.ExternallyUsedSymbols)
	}
}

func TestGraph_AnalyzeImpact_TargetNotFound(t *testing.T) {
	g := NewGraph()

	_, err := g.AnalyzeImpact("missing.go")
	if err == nil {
		t.Fatal("expected error for missing impact target")
	}
	if !errors.Is(err, ErrImpactTargetNotFound) {
		t.Fatalf("expected errors.Is(err, ErrImpactTargetNotFound) to be true, got err=%v", err)
	}
}

func TestGraph_TopComplexity(t *testing.T) {
	g := NewGraph()

	g.AddFile(&parser.File{
		Path:   "mod/main.go",
		Module: "mod",
		Definitions: []parser.Definition{
			{Name: "Small", Kind: parser.KindFunction, ComplexityScore: 2},
			{Name: "Large", Kind: parser.KindFunction, ComplexityScore: 9, BranchCount: 3, ParameterCount: 2, NestingDepth: 2, LOC: 30},
		},
	})

	top := g.TopComplexity(1)
	if len(top) != 1 {
		t.Fatalf("expected 1 hotspot, got %d", len(top))
	}
	if top[0].Definition != "Large" {
		t.Fatalf("expected Large as top hotspot, got %s", top[0].Definition)
	}
}

func TestGraph_TopComplexity_IgnoresZeroLOC(t *testing.T) {
	g := NewGraph()

	g.AddFile(&parser.File{
		Path:   "mod/main.go",
		Module: "mod",
		Definitions: []parser.Definition{
			{Name: "Ghost", Kind: parser.KindFunction, ComplexityScore: 20},
			{Name: "Real", Kind: parser.KindFunction, ComplexityScore: 8, LOC: 20},
		},
	})

	top := g.TopComplexity(5)
	if len(top) != 1 {
		t.Fatalf("expected 1 hotspot, got %d", len(top))
	}
	if top[0].Definition != "Real" {
		t.Fatalf("expected Real hotspot, got %s", top[0].Definition)
	}
}

func TestGraph_AddFile_PrefersHigherQualityDefinition(t *testing.T) {
	g := NewGraph()

	g.AddFile(&parser.File{
		Path:   "mod/one.go",
		Module: "mod",
		Definitions: []parser.Definition{
			{Name: "Run", Kind: parser.KindFunction, BranchCount: 3, ParameterCount: 2, NestingDepth: 2, LOC: 40},
		},
	})
	g.AddFile(&parser.File{
		Path:   "mod/two.go",
		Module: "mod",
		Definitions: []parser.Definition{
			{Name: "Run", Kind: parser.KindFunction},
		},
	})

	defs, ok := g.GetDefinitions("mod")
	if !ok {
		t.Fatal("expected module definitions")
	}
	def, ok := defs["Run"]
	if !ok {
		t.Fatal("expected Run definition")
	}
	if def.LOC != 40 || def.BranchCount != 3 || def.ParameterCount != 2 || def.NestingDepth != 2 {
		t.Fatalf("expected high-quality metrics to be retained, got %+v", *def)
	}
}

func TestGraph_BuildUniversalSymbolTable(t *testing.T) {
	g := NewGraph()

	g.AddFile(&parser.File{
		Path:     "svc.py",
		Language: "python",
		Module:   "svc",
		Definitions: []parser.Definition{
			{
				Name:       "GreeterServicer",
				FullName:   "svc.GreeterServicer",
				Kind:       parser.KindClass,
				Exported:   true,
				Visibility: "public",
				Scope:      "global",
				Signature:  "class GreeterServicer",
				TypeHint:   "class",
				Decorators: []string{"grpc.service"},
			},
		},
	})

	table := g.BuildUniversalSymbolTable()
	if table == nil {
		t.Fatal("expected non-nil symbol table")
	}

	records := table.Lookup("GreeterServicer")
	if len(records) != 1 {
		t.Fatalf("expected one direct lookup result, got %d", len(records))
	}
	if !records[0].IsService {
		t.Fatal("expected service record classification")
	}

	service := table.LookupService("GreeterClient")
	if len(service) == 0 {
		t.Fatal("expected service-key lookup result for GreeterClient")
	}
}
