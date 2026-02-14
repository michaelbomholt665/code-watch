package parser

import (
	"strings"
	"time"
	"unicode"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type GoExtractor struct{}

func (e *GoExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: "go",
		ParsedAt: time.Now(),
	}

	ctx := &ExtractionContext{Source: source, File: file}
	engine := NewExtractorEngine(map[string]NodeHandler{
		"package_clause":        e.extractPackage,
		"import_declaration":    e.extractImports,
		"function_declaration":  e.extractFunction,
		"type_declaration":      e.extractType,
		"short_var_declaration": e.extractVarDecl,
		"var_declaration":       e.extractVarDecl,
		"const_declaration":     e.extractVarDecl,
		"parameter_declaration": e.extractParam,
		"method_declaration":    e.extractMethod,
		"range_clause":          e.extractRange,
		"identifier":            e.captureLocal,
		"selector_expression":   e.extractReference,
		"type_identifier":       e.captureLocal,
		"qualified_type":        e.extractReference,
		"field_identifier":      e.captureLocal,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *GoExtractor) captureLocal(ctx *ExtractionContext, node *sitter.Node) bool {
	name := ctx.Text(node)
	if name == "" || name == "_" || name == "." {
		return true
	}

	// Check if this identifier is actually a package reference
	for _, imp := range ctx.File.Imports {
		if imp.Alias == name || ModuleReferenceBase("go", imp.Module) == name {
			ctx.File.References = append(ctx.File.References, Reference{
				Name:     name,
				Location: ctx.Location(node),
			})
			// Do not return true here, we also want to add it to LocalSymbols
			// just in case, or let other handlers see it.
			// Actually, if it's a package name, it's NOT a local symbol.
			return true
		}
	}

	ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, name)
	return true
}

func (e *GoExtractor) extractPackage(ctx *ExtractionContext, node *sitter.Node) bool {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "package_identifier" {
			ctx.File.PackageName = ctx.Text(child)
		}
	}
	return true
}

func (e *GoExtractor) extractImports(ctx *ExtractionContext, node *sitter.Node) bool {
	e.walkImports(ctx, node)
	return true
}

func (e *GoExtractor) walkImports(ctx *ExtractionContext, node *sitter.Node) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "import_spec" {
			var alias, path string

			// Sitter-go often has 'alias' field or just identifier before the string
			for j := uint(0); j < child.ChildCount(); j++ {
				spec := child.Child(j)
				kind := spec.Kind()

				if kind == "package_identifier" || kind == "_" || kind == "." {
					alias = ctx.Text(spec)
				} else if kind == "interpreted_string_literal" || kind == "raw_string_literal" {
					path = strings.Trim(ctx.Text(spec), "\"`")
				}
			}

			if path != "" {
				ctx.File.Imports = append(ctx.File.Imports, Import{
					Module:    path,
					RawImport: path,
					Alias:     alias,
					Location:  ctx.Location(child),
				})
			}
		} else {
			e.walkImports(ctx, child)
		}
	}
}

func (e *GoExtractor) extractFunction(ctx *ExtractionContext, node *sitter.Node) bool {
	e.extractCallable(ctx, node, KindFunction)
	return false // Continue walking into body
}

func (e *GoExtractor) extractMethod(ctx *ExtractionContext, node *sitter.Node) bool {
	receiver := node.ChildByFieldName("receiver")
	if receiver != nil {
		e.extractParam(ctx, receiver)
	}
	e.extractCallable(ctx, node, KindMethod)
	return false // Continue walking into body
}

func (e *GoExtractor) extractCallable(ctx *ExtractionContext, node *sitter.Node, kind DefinitionKind) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := ctx.Text(nameNode)
	if name == "" {
		return
	}

	ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, name)

	// Capture types in parameters and results
	params := node.ChildByFieldName("parameters")
	if params != nil {
		e.extractSignatureTypes(ctx, params)
	}
	results := node.ChildByFieldName("result")
	if results != nil {
		e.extractSignatureTypes(ctx, results)
	}

	exported := len(name) > 0 && unicode.IsUpper(rune(name[0]))
	visibility := "private"
	if exported {
		visibility = "public"
	}
	paramCount := e.countGoParameters(node.ChildByFieldName("parameters"))
	branches, nesting := e.computeGoComplexity(node.ChildByFieldName("body"), 0)
	loc := int(node.EndPosition().Row-node.StartPosition().Row) + 1
	if loc < 1 {
		loc = 1
	}
	score := (branches * 2) + (nesting * 2) + paramCount + (loc / 10)
	if score == 0 {
		score = 1
	}
	fullName := name
	if ctx.File.Module != "" {
		fullName = ctx.File.Module + "." + name
	}
	paramsText := ""
	if params != nil {
		paramsText = ctx.Text(params)
	}
	resultText := ""
	if results != nil {
		resultText = ctx.Text(results)
	}
	signature := name + paramsText
	if resultText != "" {
		signature += " " + resultText
	}
	scope := "global"
	typeHint := "function"
	if kind == KindMethod {
		scope = "method"
		typeHint = "method"
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:            name,
		FullName:        fullName,
		Kind:            kind,
		Exported:        exported,
		Visibility:      visibility,
		Scope:           scope,
		Signature:       signature,
		TypeHint:        typeHint,
		ParameterCount:  paramCount,
		BranchCount:     branches,
		NestingDepth:    nesting,
		LOC:             loc,
		ComplexityScore: score,
		Location:        ctx.Location(node),
	})
}

func (e *GoExtractor) extractSignatureTypes(ctx *ExtractionContext, node *sitter.Node) {
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		nk := n.Kind()
		if nk == "selector_expression" || nk == "qualified_type" {
			e.extractReference(ctx, n)
			return
		}
		if nk == "type_identifier" {
			e.captureLocal(ctx, n)
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(node)
}

func (e *GoExtractor) countGoParameters(params *sitter.Node) int {
	if params == nil {
		return 0
	}
	count := 0
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		switch n.Kind() {
		case "identifier":
			count++
		case "variadic_parameter":
			for i := uint(0); i < n.ChildCount(); i++ {
				child := n.Child(i)
				if child.Kind() == "identifier" {
					count++
				}
			}
			return
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(params)
	return count
}

func (e *GoExtractor) computeGoComplexity(body *sitter.Node, depth int) (branches int, maxDepth int) {
	if body == nil {
		return 0, depth
	}

	maxDepth = depth
	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		childDepth := depth

		switch child.Kind() {
		case "if_statement", "for_statement", "range_clause", "switch_statement", "type_switch_statement", "select_statement", "case_clause", "communication_case":
			branches++
			childDepth = depth + 1
		}

		subBranches, subDepth := e.computeGoComplexity(child, childDepth)
		branches += subBranches
		if subDepth > maxDepth {
			maxDepth = subDepth
		}
	}

	return branches, maxDepth
}

func (e *GoExtractor) extractType(ctx *ExtractionContext, node *sitter.Node) bool {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_spec" {
			e.extractTypeSpec(ctx, child)
			nameNode := child.ChildByFieldName("name")
			if nameNode != nil {
				ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, ctx.Text(nameNode))
			}
		}
	}
	return true
}

func (e *GoExtractor) extractTypeSpec(ctx *ExtractionContext, node *sitter.Node) bool {
	var name string
	kind := KindType

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "type_identifier" {
			name = ctx.Text(child)
		} else if child.Kind() == "interface_type" {
			kind = KindInterface
		}
	}

	if name == "" {
		return false
	}

	exported := len(name) > 0 && unicode.IsUpper(rune(name[0]))
	visibility := "private"
	if exported {
		visibility = "public"
	}
	fullName := name
	if ctx.File.Module != "" {
		fullName = ctx.File.Module + "." + name
	}
	typeNode := node.ChildByFieldName("type")
	typeHint := "type"
	signature := name
	if typeNode != nil {
		typeHint = typeNode.Kind()
		signature += " " + ctx.Text(typeNode)
	}
	if kind == KindInterface {
		typeHint = "interface"
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:       name,
		FullName:   fullName,
		Kind:       kind,
		Exported:   exported,
		Visibility: visibility,
		Scope:      "global",
		Signature:  signature,
		TypeHint:   typeHint,
		Location:   ctx.Location(node),
	})
	ctx.ProcessedChildren = true
	return true
}

func (e *GoExtractor) extractVarDecl(ctx *ExtractionContext, node *sitter.Node) bool {
	if node.Kind() == "short_var_declaration" {
		left := node.ChildByFieldName("left")
		if left != nil {
			ctx.AppendLocalIdentifiers(left)
		}
		return false
	}
	ctx.AppendLocalIdentifiers(node)
	ctx.ProcessedChildren = true
	return true // Processed local identifiers, skip walking deeper if it's just a simple decl
}

func (e *GoExtractor) extractParam(ctx *ExtractionContext, node *sitter.Node) bool {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, ctx.Text(child))
		}
	}
	return true
}

func (e *GoExtractor) extractRange(ctx *ExtractionContext, node *sitter.Node) bool {
	left := node.ChildByFieldName("left")
	if left != nil {
		ctx.AppendLocalIdentifiers(left)
	}
	return false
}

func (e *GoExtractor) extractReference(ctx *ExtractionContext, node *sitter.Node) bool {
	nk := node.Kind()

	// Only care about identifiers, qualified names and types
	if nk != "selector_expression" && nk != "qualified_type" && nk != "identifier" && nk != "type_identifier" {
		return false
	}

	// Skip if inside an import, package clause or index expression
	p := node.Parent()
	for p != nil {
		pk := p.Kind()
		if pk == "import_spec" || pk == "package_clause" || pk == "index_expression" {
			return true
		}
		p = p.Parent()
	}

	name := ctx.Text(node)
	if name == "" || name == "_" || name == "." {
		return true
	}

	// If we are a leaf identifier inside a selector, we let the parent capture the full name.
	if nk == "identifier" || nk == "type_identifier" {
		if parent := node.Parent(); parent != nil {
			pk := parent.Kind()
			if pk == "selector_expression" || pk == "qualified_type" {
				return false
			}
		}
	}

	// Check if this is a known local symbol (variable, parameter, method receiver)
	for _, sym := range ctx.File.LocalSymbols {
		if sym == name {
			return true
		}
	}

	// If it's a selector, we might want to capture just Pkg.Symbol
	if nk == "selector_expression" || nk == "qualified_type" {
		parts := strings.Split(name, ".")
		if len(parts) > 2 {
			name = parts[0] + "." + parts[1]
		}
		ctx.ProcessedChildren = true
	}

	ctx.File.References = append(ctx.File.References, Reference{
		Name:     name,
		Location: ctx.Location(node),
		Context:  callReferenceContext("go", name),
	})

	return true
}
