package graph

import (
	"circular/internal/engine/parser"
	"testing"
)

func TestDetectCycles_Iterative(t *testing.T) {
	g := NewGraph()

	// Create a simple cycle: A -> B -> C -> A
	g.AddFile(&parser.File{
		Path:   "a.go",
		Module: "modA",
		Imports: []parser.Import{
			{Module: "modB"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "b.go",
		Module: "modB",
		Imports: []parser.Import{
			{Module: "modC"},
		},
	})
	g.AddFile(&parser.File{
		Path:   "c.go",
		Module: "modC",
		Imports: []parser.Import{
			{Module: "modA"},
		},
	})

	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Fatalf("Expected 1 cycle, got %d", len(cycles))
	}

	expected := []string{"modA", "modB", "modC"}
	if len(cycles[0]) != 3 {
		t.Fatalf("Expected cycle length 3, got %d", len(cycles[0]))
	}
	
	// Cycles might start at different points but should have same elements in order
	match := false
	for i := 0; i < 3; i++ {
		allMatch := true
		for j := 0; j < 3; j++ {
			if cycles[0][j] != expected[(i+j)%3] {
				allMatch = false
				break
			}
		}
		if allMatch {
			match = true
			break
		}
	}
	if !match {
		t.Errorf("Unexpected cycle: %v", cycles[0])
	}
}

func TestDetectCycles_Deep(t *testing.T) {
	g := NewGraph()
	count := 5000 // Deep enough to potentially hit recursion limit if recursive
	
	for i := 0; i < count; i++ {
		mod := string(rune('A' + (i % 26))) + string(rune('0' + (i / 26)))
		nextMod := string(rune('A' + ((i+1) % 26))) + string(rune('0' + ((i+1) / 26)))
		
		g.AddFile(&parser.File{
			Path:   mod + ".go",
			Module: mod,
			Imports: []parser.Import{
				{Module: nextMod},
			},
		})
	}
	
	// This would overflow if recursive and limit is small
	cycles := g.DetectCycles()
	// Should find one big cycle if i+1 wraps around to 0, but here it doesn't wrap back to 0.
	// Let's add a wrap back to verify cycle detection in deep graph.
	lastMod := string(rune('A' + ((count-1) % 26))) + string(rune('0' + ((count-1) / 26)))
	g.AddFile(&parser.File{
		Path: "wrap.go",
		Module: lastMod,
		Imports: []parser.Import{
			{Module: "A0"},
		},
	})
	
	cycles = g.DetectCycles()
	if len(cycles) == 0 {
		t.Error("Expected at least one cycle in deep wrap-around graph")
	}
}
