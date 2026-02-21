package graph

import (
	"fmt"
	"testing"
	"circular/internal/engine/parser"
)

func BenchmarkAddFile(b *testing.B) {
	g := NewGraph(1000)
	files := make([]*parser.File, 100)
	for i := 0; i < 100; i++ {
		files[i] = &parser.File{
			Path:     fmt.Sprintf("file%d.go", i),
			Language: "go",
			Imports: []parser.Import{
				{Module: fmt.Sprintf("file%d", (i+1)%100)},
			},
			Definitions: []parser.Definition{
				{Name: fmt.Sprintf("Func%d", i)},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		file := files[i%100]
		g.AddFile(file)
	}
}

func BenchmarkDetectCycles(b *testing.B) {
	g := NewGraph(1000)
	// Create a large graph with a cycle
	for i := 0; i < 500; i++ {
		g.AddFile(&parser.File{
			Path:     fmt.Sprintf("file%d.go", i),
			Language: "go",
			Imports: []parser.Import{
				{Module: fmt.Sprintf("file%d", (i+1)%500)},
			},
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.DetectCycles(100)
	}
}
