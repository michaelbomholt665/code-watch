package parser

import (
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type PythonExtractor struct{}

func (e *PythonExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: "python",
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	engine := NewExtractorEngine(map[string]NodeHandler{
		"import_statement":      e.extractImport,
		"import_from_statement": e.extractFromImport,
		"function_definition":   e.extractFunction,
		"class_definition":      e.extractClass,
		"assignment":            e.extractAssignment,
		"augmented_assignment":  e.extractAssignment,
		"for_statement":         e.extractFor,
		"call":                  e.extractCall,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *PythonExtractor) extractImport(ctx *ExtractionContext, node *sitter.Node) bool {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "dotted_name" || child.Kind() == "identifier" {
			module := ctx.Text(child)
			ctx.File.Imports = append(ctx.File.Imports, Import{
				Module:    module,
				RawImport: module,
				Location:  ctx.Location(child),
			})
		} else if child.Kind() == "aliased_import" {
			var module, alias string
			for j := uint(0); j < child.ChildCount(); j++ {
				sub := child.Child(j)
				if sub.Kind() == "dotted_name" || sub.Kind() == "identifier" {
					if module == "" {
						module = ctx.Text(sub)
					} else {
						alias = ctx.Text(sub)
					}
				}
			}
			ctx.File.Imports = append(ctx.File.Imports, Import{
				Module:    module,
				RawImport: module,
				Alias:     alias,
				Location:  ctx.Location(child),
			})
		}
	}
	return true
}

func (e *PythonExtractor) extractFromImport(ctx *ExtractionContext, node *sitter.Node) bool {
	var module string
	var items []string
	isRelative := false

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "relative_import":
			isRelative = true
			relText := ctx.Text(child)
			module = strings.TrimLeft(relText, ".")
		case "dotted_name", "identifier":
			if !isRelative {
				module = ctx.Text(child)
			}
		case "import_list", "aliased_import":
			e.collectItems(ctx, child, &items)
		}
	}

	if len(items) == 0 {
		foundImport := false
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "import" {
				foundImport = true
				continue
			}
			if foundImport && (child.Kind() == "identifier" || child.Kind() == "dotted_name") {
				items = append(items, ctx.Text(child))
			}
		}
	}

	ctx.File.Imports = append(ctx.File.Imports, Import{
		Module:     module,
		RawImport:  module,
		Items:      items,
		IsRelative: isRelative,
		Location:   ctx.Location(node),
	})
	return true
}

func (e *PythonExtractor) collectItems(ctx *ExtractionContext, node *sitter.Node, items *[]string) {
	kind := node.Kind()
	if kind == "identifier" || kind == "dotted_name" {
		*items = append(*items, ctx.Text(node))
		return
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		e.collectItems(ctx, node.Child(i), items)
	}
}

func (e *PythonExtractor) extractFunction(ctx *ExtractionContext, node *sitter.Node) bool {
	name := ctx.ChildText(node, "identifier")
	if name == "" {
		return false
	}

	params := node.ChildByFieldName("parameters")
	if params != nil {
		ctx.AppendLocalIdentifiers(params)
	}

	paramCount := e.countPythonParameters(params)
	branches, nesting := e.computePythonComplexity(node.ChildByFieldName("body"), 0)
	loc := int(node.EndPosition().Row-node.StartPosition().Row) + 1
	if loc < 1 {
		loc = 1
	}
	score := (branches * 2) + (nesting * 2) + paramCount + (loc / 10)
	if score == 0 {
		score = 1
	}

	exported := !strings.HasPrefix(name, "_")
	visibility := "public"
	if !exported {
		visibility = "private"
	}
	fullName := name
	if ctx.File.Module != "" {
		fullName = ctx.File.Module + "." + name
	}
	scope := e.pythonDefinitionScope(node)
	paramsText := "()"
	if params != nil {
		paramsText = ctx.Text(params)
	}
	signature := name + paramsText
	decorators := e.pythonDecorators(ctx, node)
	if scope == "class" {
		scope = "method"
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:            name,
		FullName:        fullName,
		Kind:            KindFunction,
		Exported:        exported,
		Visibility:      visibility,
		Scope:           scope,
		Signature:       signature,
		TypeHint:        "function",
		Decorators:      decorators,
		ParameterCount:  paramCount,
		BranchCount:     branches,
		NestingDepth:    nesting,
		LOC:             loc,
		ComplexityScore: score,
		Location:        ctx.Location(node),
	})
	return false
}

func (e *PythonExtractor) countPythonParameters(params *sitter.Node) int {
	if params == nil {
		return 0
	}
	count := 0
	for i := uint(0); i < params.ChildCount(); i++ {
		if e.isPythonParameterNode(params.Child(i)) {
			count++
		}
	}
	return count
}

func (e *PythonExtractor) isPythonParameterNode(node *sitter.Node) bool {
	if node == nil {
		return false
	}

	switch node.Kind() {
	case "identifier", "typed_parameter", "default_parameter", "typed_default_parameter", "list_splat_pattern", "dictionary_splat_pattern":
		return true
	case ",", "(", ")", "*", "/":
		return false
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		if e.isPythonParameterNode(node.Child(i)) {
			return true
		}
	}

	kind := node.Kind()
	return strings.HasSuffix(kind, "_parameter") || strings.HasSuffix(kind, "_pattern")
}

func (e *PythonExtractor) computePythonComplexity(node *sitter.Node, depth int) (branches int, maxDepth int) {
	if node == nil {
		return 0, depth
	}

	maxDepth = depth
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		childDepth := depth

		switch child.Kind() {
		case "if_statement", "elif_clause", "for_statement", "while_statement", "try_statement", "except_clause", "with_statement", "match_statement":
			branches++
			childDepth = depth + 1
		}

		subBranches, subDepth := e.computePythonComplexity(child, childDepth)
		branches += subBranches
		if subDepth > maxDepth {
			maxDepth = subDepth
		}
	}

	return branches, maxDepth
}

func (e *PythonExtractor) extractAssignment(ctx *ExtractionContext, node *sitter.Node) bool {
	left := node.ChildByFieldName("left")
	if left != nil {
		ctx.AppendLocalIdentifiers(left)
	}
	return false
}

func (e *PythonExtractor) extractFor(ctx *ExtractionContext, node *sitter.Node) bool {
	left := node.ChildByFieldName("left")
	if left != nil {
		ctx.AppendLocalIdentifiers(left)
	}
	return false
}

func (e *PythonExtractor) extractClass(ctx *ExtractionContext, node *sitter.Node) bool {
	name := ctx.ChildText(node, "identifier")
	if name == "" {
		return false
	}

	exported := !strings.HasPrefix(name, "_")
	visibility := "public"
	if !exported {
		visibility = "private"
	}
	fullName := name
	if ctx.File.Module != "" {
		fullName = ctx.File.Module + "." + name
	}
	scope := e.pythonDefinitionScope(node)
	signature := name
	if superclasses := node.ChildByFieldName("superclasses"); superclasses != nil {
		signature += "(" + ctx.Text(superclasses) + ")"
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:       name,
		FullName:   fullName,
		Kind:       KindClass,
		Exported:   exported,
		Visibility: visibility,
		Scope:      scope,
		Signature:  signature,
		TypeHint:   "class",
		Decorators: e.pythonDecorators(ctx, node),
		Location:   ctx.Location(node),
	})
	return false
}

func (e *PythonExtractor) extractCall(ctx *ExtractionContext, node *sitter.Node) bool {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "attribute" || child.Kind() == "identifier" {
			name := ctx.Text(child)
			ctx.File.References = append(ctx.File.References, Reference{
				Name:     name,
				Location: ctx.Location(child),
				Context:  callReferenceContext("python", name),
			})
		}
	}
	return false
}

func (e *PythonExtractor) pythonDecorators(ctx *ExtractionContext, node *sitter.Node) []string {
	if node == nil {
		return nil
	}
	parent := node.Parent()
	if parent == nil || parent.Kind() != "decorated_definition" {
		return nil
	}

	decorators := make([]string, 0, parent.ChildCount())
	for i := uint(0); i < parent.ChildCount(); i++ {
		child := parent.Child(i)
		if child.Kind() != "decorator" {
			continue
		}
		dec := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ctx.Text(child)), "@"))
		if dec == "" {
			continue
		}
		decorators = append(decorators, dec)
	}
	return decorators
}

func (e *PythonExtractor) pythonDefinitionScope(node *sitter.Node) string {
	scope := "global"
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case "class_definition":
			return "class"
		case "function_definition":
			scope = "nested"
		}
	}
	return scope
}
