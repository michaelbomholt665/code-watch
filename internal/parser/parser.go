// # internal/parser/parser.go
package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Parser struct {
	loader     *GrammarLoader
	extractors map[string]Extractor // language -> extractor
}

type Extractor interface {
	Extract(node *sitter.Node, source []byte, filePath string) (*File, error)
}

func NewParser(loader *GrammarLoader) *Parser {
	return &Parser{
		loader: loader,
		extractors: make(map[string]Extractor),
	}
}

func (p *Parser) RegisterExtractor(lang string, e Extractor) {
	p.extractors[lang] = e
}

func (p *Parser) ParseFile(path string, content []byte) (*File, error) {
	lang := p.detectLanguage(path)
	if lang == "" {
		return nil, errors.New("unsupported language")
	}

	grammar := p.loader.languages[lang]
	if grammar == nil {
		return nil, fmt.Errorf("grammar not loaded: %s", lang)
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(grammar)

	tree := parser.Parse(content, nil)
	if tree == nil {
		return nil, errors.New("parse failed")
	}
	defer tree.Close()

	root := tree.RootNode()

	extractor := p.extractors[lang]
	if extractor == nil {
		return nil, fmt.Errorf("no extractor for: %s", lang)
	}

	return extractor.Extract(root, content, path)
}

func (p *Parser) detectLanguage(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".py":
		return "python"
	case ".go":
		return "go"
	default:
		return ""
	}
}
