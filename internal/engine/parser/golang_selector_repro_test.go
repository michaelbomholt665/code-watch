package parser

import (
	"testing"
	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

func TestGoExtractor_SelectorRepro(t *testing.T) {
	source := `
package adapters
import (
	"circular/internal/engine/secrets"
)
func MyFunc() {
	_ = secrets.MaskValue("val")
}
`
	parser := sitter.NewParser()
	parser.SetLanguage(sitter.NewLanguage(tree_sitter_go.Language()))
	tree := parser.Parse([]byte(source), nil)
	
	e := &GoExtractor{}
	file, err := e.Extract(tree.RootNode(), []byte(source), "adapter.go")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	foundSecrets := false
	for _, ref := range file.References {
		if ref.Name == "secrets" {
			foundSecrets = true
			break
		}
	}

	if !foundSecrets {
		t.Errorf("Reference 'secrets' not found")
		for _, ref := range file.References {
			t.Logf("Found reference: %s", ref.Name)
		}
	}
}
