package parser

import (
	"context"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
)

func FuzzGoParser(f *testing.F) {
	f.Add([]byte(`package main
func main() {
	println("hello")
}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		lang := sitter.NewLanguage(tree_sitter_go.Language())
		parser := sitter.NewParser()
		defer parser.Close()
		parser.SetLanguage(lang)

		tree := parser.Parse(data, nil)
		if tree == nil {
			return
		}
		defer tree.Close()

		extractor := NewUniversalExtractor()
		_, _ = extractor.Extract(tree.RootNode(), data, "test.go")
	})
}

func FuzzPythonParser(f *testing.F) {
	f.Add([]byte(`def main():
    print("hello")
if __name__ == "__main__":
    main()`))
	f.Fuzz(func(t *testing.T, data []byte) {
		lang := sitter.NewLanguage(tree_sitter_python.Language())
		parser := sitter.NewParser()
		defer parser.Close()
		parser.SetLanguage(lang)

		tree := parser.Parse(data, nil)
		if tree == nil {
			return
		}
		defer tree.Close()

		extractor := NewUniversalExtractor()
		_, _ = extractor.Extract(tree.RootNode(), data, "test.py")
	})
}
