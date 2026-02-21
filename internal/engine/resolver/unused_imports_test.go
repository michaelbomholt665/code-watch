package resolver

import (
	"circular/internal/engine/parser"
	"circular/internal/engine/graph"
	"context"
	"testing"
)

func TestHasSymbolUse_FalsePositives(t *testing.T) {
	refHits := map[string]int{
		"ports.AnalysisService": 1,
		"coreapp.New":           1,
		"slog.Logger":           1,
		"version.Version":       1,
	}

	tests := []struct {
		symbol string
		want   bool
	}{
		{"ports", true},
		{"coreapp", true},
		{"slog", true},
		{"version", true},
		{"fmt", false},
	}

	for _, tc := range tests {
		got := hasSymbolUse(refHits, tc.symbol)
		if got != tc.want {
			t.Errorf("hasSymbolUse(refHits, %q) = %v, want %v", tc.symbol, got, tc.want)
		}
	}
}

func TestFindUnusedImports_InternalPackages(t *testing.T) {
	g := graph.NewGraph()
	
	// Create a file that uses internal packages
	file := &parser.File{
		Path:     "main.go",
		Language: "go",
		Module:   "main",
		Imports: []parser.Import{
			{Module: "circular/internal/core/ports", Alias: ""},
			{Module: "circular/internal/shared/version", Alias: ""},
		},
		References: []parser.Reference{
			{Name: "ports.AnalysisService"},
			{Name: "version.Version"},
		},
	}
	
	// In a real scenario, golang.go would also add the base parts:
	file.References = append(file.References, 
		parser.Reference{Name: "ports"},
		parser.Reference{Name: "version"},
	)
	
	g.AddFile(file)
	
	r := NewResolver(g, nil, nil)
	unused := r.FindUnusedImports(context.Background(), []string{"main.go"})
	
	if len(unused) > 0 {
		for _, u := range unused {
			t.Errorf("Unexpected unused import: %s in %s", u.Module, u.File)
		}
	}
}
