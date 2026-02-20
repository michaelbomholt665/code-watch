package resolver

import (
	"circular/internal/engine/parser"
	"circular/internal/engine/graph"
	"testing"
)

func TestFindUnusedImports_Repro(t *testing.T) {
	tests := []struct {
		name     string
		file     *parser.File
		expected int // number of unused imports expected (should be 0 for these cases)
	}{
		{
			name: "Side-effect import",
			file: &parser.File{
				Path:     "side_effect.go",
				Language: "go",
				Imports: []parser.Import{
					{Module: "lib/db", Alias: "_"},
				},
				References: []parser.Reference{},
			},
			expected: 0,
		},
		{
			name: "Aliased import usage",
			file: &parser.File{
				Path:     "alias.go",
				Language: "go",
				Imports: []parser.Import{
					{Module: "lib/math", Alias: "m"},
				},
				References: []parser.Reference{
					{Name: "m.Abs"},
				},
			},
			expected: 0,
		},
		{
			name: "Type-only usage in var decl",
			file: &parser.File{
				Path:     "type_use.go",
				Language: "go",
				Imports: []parser.Import{
					{Module: "lib/types", Alias: ""},
				},
				References: []parser.Reference{
					{Name: "types.MyType"},
				},
			},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.NewGraph()
			g.AddFile(tc.file)
			r := NewResolver(g, nil, nil)
			unused := r.FindUnusedImports([]string{tc.file.Path})
			if len(unused) != tc.expected {
				t.Errorf("Expected %d unused imports, got %d", tc.expected, len(unused))
				for _, u := range unused {
					t.Errorf("  Unused: %s", u.Module)
				}
			}
		})
	}
}
