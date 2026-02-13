package parser

import (
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// NodeHandler processes a node for a language-specific extractor.
type NodeHandler func(ctx *ExtractionContext, node *sitter.Node)

// ExtractionContext carries shared state/helpers used by all extractors.
type ExtractionContext struct {
	Source []byte
	File   *File
}

// ExtractorEngine walks the syntax tree and dispatches node handlers by kind.
type ExtractorEngine struct {
	handlers map[string]NodeHandler
}

func NewExtractorEngine(handlers map[string]NodeHandler) *ExtractorEngine {
	return &ExtractorEngine{handlers: handlers}
}

func (e *ExtractorEngine) Walk(ctx *ExtractionContext, node *sitter.Node) {
	if node == nil {
		return
	}

	if handler, ok := e.handlers[node.Kind()]; ok {
		handler(ctx, node)
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		e.Walk(ctx, node.Child(i))
	}
}

func (c *ExtractionContext) Text(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	return string(c.Source[node.StartByte():node.EndByte()])
}

func (c *ExtractionContext) Location(node *sitter.Node) Location {
	return Location{
		File:   c.File.Path,
		Line:   int(node.StartPosition().Row) + 1,
		Column: int(node.StartPosition().Column) + 1,
	}
}

func (c *ExtractionContext) ChildText(node *sitter.Node, kind string) string {
	if node == nil {
		return ""
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return c.Text(child)
		}
	}
	return ""
}

func (c *ExtractionContext) AppendLocalIdentifiers(node *sitter.Node) {
	if node == nil {
		return
	}
	if node.Kind() == "identifier" {
		c.File.LocalSymbols = append(c.File.LocalSymbols, c.Text(node))
		return
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		c.AppendLocalIdentifiers(node.Child(i))
	}
}
