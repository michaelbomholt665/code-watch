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
		"call_expression":       e.extractCall,
	})
	engine.Walk(ctx, root)

	return file, nil
}

func (e *GoExtractor) extractPackage(ctx *ExtractionContext, node *sitter.Node) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "package_identifier" {
			ctx.File.PackageName = ctx.Text(child)
		}
	}
}

func (e *GoExtractor) extractImports(ctx *ExtractionContext, node *sitter.Node) {
	e.walkImports(ctx, node)
}

func (e *GoExtractor) walkImports(ctx *ExtractionContext, node *sitter.Node) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "import_spec" {
			var alias, path string

			for j := uint(0); j < child.ChildCount(); j++ {
				spec := child.Child(j)

				if spec.Kind() == "package_identifier" {
					alias = ctx.Text(spec)
				} else if spec.Kind() == "interpreted_string_literal" {
					path = strings.Trim(ctx.Text(spec), "\"")
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

func (e *GoExtractor) extractFunction(ctx *ExtractionContext, node *sitter.Node) {
	e.extractCallable(ctx, node, KindFunction)
}

func (e *GoExtractor) extractMethod(ctx *ExtractionContext, node *sitter.Node) {
	receiver := node.ChildByFieldName("receiver")
	if receiver != nil {
		e.extractParam(ctx, receiver)
	}
	e.extractCallable(ctx, node, KindMethod)
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

	exported := len(name) > 0 && unicode.IsUpper(rune(name[0]))
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

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:            name,
		FullName:        fullName,
		Kind:            kind,
		Exported:        exported,
		ParameterCount:  paramCount,
		BranchCount:     branches,
		NestingDepth:    nesting,
		LOC:             loc,
		ComplexityScore: score,
		Location:        ctx.Location(node),
	})
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

func (e *GoExtractor) extractType(ctx *ExtractionContext, node *sitter.Node) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_spec" {
			e.extractTypeSpec(ctx, child)
		}
	}
}

func (e *GoExtractor) extractTypeSpec(ctx *ExtractionContext, node *sitter.Node) {
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
		return
	}

	exported := len(name) > 0 && unicode.IsUpper(rune(name[0]))
	fullName := name
	if ctx.File.Module != "" {
		fullName = ctx.File.Module + "." + name
	}

	ctx.File.Definitions = append(ctx.File.Definitions, Definition{
		Name:     name,
		FullName: fullName,
		Kind:     kind,
		Exported: exported,
		Location: ctx.Location(node),
	})
}

func (e *GoExtractor) extractVarDecl(ctx *ExtractionContext, node *sitter.Node) {
	if node.Kind() == "short_var_declaration" {
		left := node.ChildByFieldName("left")
		if left != nil {
			ctx.AppendLocalIdentifiers(left)
		}
		return
	}
	ctx.AppendLocalIdentifiers(node)
}

func (e *GoExtractor) extractParam(ctx *ExtractionContext, node *sitter.Node) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, ctx.Text(child))
		}
	}
}

func (e *GoExtractor) extractRange(ctx *ExtractionContext, node *sitter.Node) {
	left := node.ChildByFieldName("left")
	if left != nil {
		ctx.AppendLocalIdentifiers(left)
	}
}

func (e *GoExtractor) extractCall(ctx *ExtractionContext, node *sitter.Node) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" || child.Kind() == "selector_expression" {
			ctx.File.References = append(ctx.File.References, Reference{
				Name:     ctx.Text(child),
				Location: ctx.Location(node),
			})
		}
	}
}
