package parser

import (
	"circular/internal/engine/parser/registry"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// DynamicExtractor uses configuration to extract symbols from an AST.
type DynamicExtractor struct {
	Config registry.DynamicExtractorConfig
}

func NewDynamicExtractor(cfg registry.DynamicExtractorConfig) *DynamicExtractor {
	return &DynamicExtractor{Config: cfg}
}

func (e *DynamicExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	
	// Create handlers based on config
	handlers := make(map[string]NodeHandler)
	
	if e.Config.NamespaceNode != "" {
		handlers[e.Config.NamespaceNode] = e.extractNamespace
	}
	
	if e.Config.ImportNode != "" {
		handlers[e.Config.ImportNode] = e.extractImport
	}
	
	for _, nodeKind := range e.Config.DefinitionNodes {
		handlers[nodeKind] = e.extractDefinition
	}
	
	engine := NewExtractorEngine(handlers)
	engine.Walk(ctx, root)

	return file, nil
}

func (e *DynamicExtractor) extractNamespace(ctx *ExtractionContext, node *sitter.Node) bool {
	ctx.File.PackageName = ctx.Text(node)
	return true
}

func (ctx *ExtractionContext) extractImport(node *sitter.Node) bool {
	// Simple generic import extraction: use the node text as module name
	module := ctx.Text(node)
	ctx.File.Imports = append(ctx.File.Imports, Import{
		Module:    module,
		RawImport: module,
		Location:  ctx.Location(node),
	})
	return true
}

func (e *DynamicExtractor) extractImport(ctx *ExtractionContext, node *sitter.Node) bool {
	return ctx.extractImport(node)
}

func (e *DynamicExtractor) extractDefinition(ctx *ExtractionContext, node *sitter.Node) bool {
	// Generic definition extraction
	name := ""
	// Try to find a child named "name" or "identifier"
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" || child.Kind() == "name" {
			name = ctx.Text(child)
			break
		}
	}
	if name == "" {
		name = ctx.Text(node)
	}
	
	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:      name,
		FullName:  name,
		Kind:      KindVariable, // Default kind
		Signature: ctx.Text(node),
		Location:  ctx.Location(node),
	})
	return true
}
