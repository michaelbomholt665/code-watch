// # internal/parser/loader.go
package parser

import (
	"fmt"
	"os"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

type GrammarLoader struct {
	languages map[string]*sitter.Language
}

func NewGrammarLoader(grammarsPath string) (*GrammarLoader, error) {
	if grammarsPath != "" {
		if info, err := os.Stat(grammarsPath); err == nil && !info.IsDir() {
			return nil, fmt.Errorf("grammars path is not a directory: %s", grammarsPath)
		}
	}

	gl := &GrammarLoader{
		languages: make(map[string]*sitter.Language),
	}

	// Load Python
	pythonLang := sitter.NewLanguage(tree_sitter_python.Language())
	gl.languages["python"] = pythonLang

	// Load Go
	goLang := sitter.NewLanguage(tree_sitter_go.Language())
	gl.languages["go"] = goLang

	return gl, nil
}
