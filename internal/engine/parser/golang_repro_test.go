package parser

import (
	"testing"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func TestGoExtractor_ReferencesRepro(t *testing.T) {
	source := `
package main
import (
	"fmt"
	"lib/math"
	"lib/types"
	_ "lib/db"
	. "lib/dot"
)

var MyVar math.Number = 10
type MyStruct struct {
	Field types.CustomType
}

type MyAlias = MyStruct

func main() {
	fmt.Println("hello")
	_ = MyStruct{
		Field: types.CustomType(1),
	}
	_ = MyAlias{}
}
`
	parser := sitter.NewParser()
	parser.SetLanguage(sitter.NewLanguage(tree_sitter_go.Language()))
	tree := parser.Parse([]byte(source), nil)
	
	e := &GoExtractor{}
	file, err := e.Extract(tree.RootNode(), []byte(source), "test.go")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	hasRef := func(name string) bool {
		for _, ref := range file.References {
			if ref.Name == name {
				return true
			}
		}
		return false
	}

	// fmt should be found
	if !hasRef("fmt") {
		t.Errorf("Reference 'fmt' not found")
	}

	// math should be found (from math.Number)
	if !hasRef("math") {
		t.Errorf("Reference 'math' not found")
	}

	// types should be found (from types.CustomType)
	if !hasRef("types") {
		t.Errorf("Reference 'types' not found")
	}

	// count math and types
	mathCount := 0
	typesCount := 0
	for _, ref := range file.References {
		if ref.Name == "math" {
			mathCount++
		}
		if ref.Name == "types" {
			typesCount++
		}
	}

	if mathCount < 1 {
		t.Errorf("Expected at least 1 math reference, got %d", mathCount)
	}
	if typesCount < 2 {
		t.Errorf("Expected at least 2 types references (struct field and struct literal), got %d", typesCount)
	}

	// db should NOT be found as a reference (it's a side-effect import)
	if hasRef("db") {
		t.Errorf("Reference 'db' found for side-effect import")
	}

	// dot should NOT be found as a reference
	if hasRef("dot") {
		t.Errorf("Reference 'dot' found for dot import")
	}

	// Verify aliases are correctly captured
	foundDB := false
	foundDot := false
	for _, imp := range file.Imports {
		if imp.Module == "lib/db" && imp.Alias == "_" {
			foundDB = true
		}
		if imp.Module == "lib/dot" && imp.Alias == "." {
			foundDot = true
		}
	}
	if !foundDB {
		t.Errorf("Side-effect import alias '_' not captured")
	}
	if !foundDot {
		t.Errorf("Dot import alias '.' not captured")
	}

	// MyAlias should be defined
	foundAlias := false
	for _, def := range file.Definitions {
		if def.Name == "MyAlias" {
			foundAlias = true
			break
		}
	}
	if !foundAlias {
		t.Errorf("Definition 'MyAlias' not found")
	}
}
