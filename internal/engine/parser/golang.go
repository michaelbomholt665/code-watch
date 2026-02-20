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
		"func_literal":          e.extractFuncLiteral,
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

				if kind == "package_identifier" || kind == "_" || kind == "." || kind == "blank_identifier" || kind == "dot" {
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

func (e *GoExtractor) extractFuncLiteral(ctx *ExtractionContext, node *sitter.Node) bool {
	// Capture parameters and their names
	params := node.ChildByFieldName("parameters")
	if params != nil {
		// parameter_list -> parameter_declaration
		for i := uint(0); i < params.ChildCount(); i++ {
			child := params.Child(i)
			if child.Kind() == "parameter_declaration" {
				e.extractParam(ctx, child)
			}
		}
	}
	results := node.ChildByFieldName("result")
	if results != nil {
		e.walkForReferences(ctx, results)
	}
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
		e.walkForReferences(ctx, params)
	}
	results := node.ChildByFieldName("result")
	if results != nil {
		e.walkForReferences(ctx, results)
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

	// For methods, try to prepend receiver type to name
	displayName := name
	if kind == KindMethod {
		receiver := node.ChildByFieldName("receiver")
		if receiver != nil {
			// receiver typically looks like (p *Parser) or (p Parser)
			// We want 'Parser'
			recvText := ctx.Text(receiver)
			recvText = strings.Trim(recvText, "()")
			parts := strings.Fields(recvText)
			if len(parts) > 0 {
				typeName := parts[len(parts)-1]
				typeName = strings.TrimLeft(typeName, "*")
				displayName = typeName + "." + name
			}
		}
	}

	fullName := displayName
	if ctx.File.Module != "" {
		fullName = ctx.File.Module + "." + displayName
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

func (e *GoExtractor) walkForReferences(ctx *ExtractionContext, node *sitter.Node) {
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		nk := n.Kind()
		if nk == "func_literal" {
			e.extractFuncLiteral(ctx, n)
		}
		if nk == "selector_expression" || nk == "qualified_type" {
			e.extractReference(ctx, n)
			return
		}
		if nk == "type_identifier" || nk == "identifier" {
			e.extractReference(ctx, n)
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
		if child.Kind() == "type_spec" || child.Kind() == "type_alias" {
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

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		name = ctx.Text(nameNode)
	}

	if name == "" {
		// Fallback to searching for type_identifier or identifier if field name lookup fails
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			ck := child.Kind()
			if ck == "type_identifier" || ck == "identifier" {
				name = ctx.Text(child)
				break
			}
		}
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		if node.Child(i).Kind() == "interface_type" {
			kind = KindInterface
			break
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
		e.walkForReferences(ctx, typeNode)
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
	return false
}

func (e *GoExtractor) extractVarDecl(ctx *ExtractionContext, node *sitter.Node) bool {
	// Check if we are at the top level (package scope)
	isTopLevel := false
	if parent := node.Parent(); parent != nil && parent.Kind() == "source_file" {
		isTopLevel = true
	}

	if isTopLevel {
		e.extractTopLevelVars(ctx, node)
		return true
	}

	// For local declarations, we need to capture local symbols AND extract references
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}

		nk := n.Kind()
		if nk == "var_spec" || nk == "const_spec" || nk == "short_var_declaration" {
			// In var_spec/const_spec, names come before type/values
			// In short_var_declaration, names are in 'left' field
			if nk == "short_var_declaration" {
				left := n.ChildByFieldName("left")
				ctx.AppendLocalIdentifiers(left)
				right := n.ChildByFieldName("right")
				if right != nil {
					e.walkForReferences(ctx, right)
				}
				return
			}

			// For var_spec/const_spec
			hasTypeOrValue := false
			for i := uint(0); i < n.ChildCount(); i++ {
				child := n.Child(i)
				ck := child.Kind()
				if ck == "identifier" && !hasTypeOrValue {
					ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, ctx.Text(child))
				} else if ck != "," && ck != "=" && ck != ":" {
					hasTypeOrValue = true
					e.walkForReferences(ctx, child)
				}
			}
			return
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}

	walk(node)
	ctx.ProcessedChildren = true
	return true
}

func (e *GoExtractor) extractTopLevelVars(ctx *ExtractionContext, node *sitter.Node) {
	kind := KindVariable
	if node.Kind() == "const_declaration" {
		kind = KindConstant
	}

	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Kind() == "import_spec" {
			return
		}

		if n.Kind() == "identifier" {
			// Ensure this identifier is actually a name being defined, not a type or value.
			// In Go tree-sitter, var/const names are children of var_spec/const_spec.
			// We only want them if they are in the 'name' field or similar.
			p := n.Parent()
			isName := false
			if p != nil {
				pk := p.Kind()
				if pk == "var_spec" || pk == "const_spec" {
					// Check if n is among the names
					for i := uint(0); i < p.ChildCount(); i++ {
						child := p.Child(i)
						if child.Kind() == "identifier" {
							if child.Id() == n.Id() {
								isName = true
								break
							}
						} else {
							// Once we hit something else (type or =), names are over
							break
						}
					}
				}
			}

			if isName {
				name := ctx.Text(n)
				if name != "" && name != "_" {
					exported := isExportedName(name)
					visibility := "private"
					if exported {
						visibility = "public"
					}
					fullName := name
					if ctx.File.Module != "" {
						fullName = ctx.File.Module + "." + name
					}

					ctx.File.Definitions = append(ctx.File.Definitions, Definition{
						Name:       name,
						FullName:   fullName,
						Kind:       kind,
						Exported:   exported,
						Visibility: visibility,
						Scope:      "global",
						Location:   ctx.Location(n),
					})
					ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, name)
				}
				return
			}
		}

		// Extract references from types and values
		nk := n.Kind()
		if nk == "type_identifier" || nk == "qualified_type" || nk == "call_expression" || nk == "selector_expression" {
			e.walkForReferences(ctx, n)
			return
		}

		for i := uint(0); i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(node)
}

func (e *GoExtractor) extractParam(ctx *ExtractionContext, node *sitter.Node) bool {
	// parameter_declaration: name (identifier), type (qualified_type, etc.)
	// or just type (for anonymous params)
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, ctx.Text(nameNode))
	} else {
		// Fallback: if it has 2+ identifiers, the first is likely the name
		// unless it's a variadic param or something.
		ids := make([]*sitter.Node, 0)
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "identifier" {
				ids = append(ids, child)
			}
		}
		if len(ids) >= 2 {
			ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, ctx.Text(ids[0]))
		} else if len(ids) == 1 {
			// Could be name OR type. Hard to tell without more context.
			// If the only child is identifier, it might be an anonymous param of type T.
			// But if it's func(m T), 'm' is identifier, 'T' is type_identifier.
			// So if we have 1 identifier and 1 type_identifier, identifier is the name.
			hasType := false
			for i := uint(0); i < node.ChildCount(); i++ {
				child := node.Child(i)
				if child.Kind() == "type_identifier" || child.Kind() == "qualified_type" || child.Kind() == "pointer_type" {
					hasType = true
					break
				}
			}
			if hasType {
				ctx.File.LocalSymbols = append(ctx.File.LocalSymbols, ctx.Text(ids[0]))
			}
		}
	}
	typeNode := node.ChildByFieldName("type")
	if typeNode != nil {
		e.walkForReferences(ctx, typeNode)
	}
	return true
}

func (e *GoExtractor) extractRange(ctx *ExtractionContext, node *sitter.Node) bool {
	left := node.ChildByFieldName("left")
	if left != nil {
		ctx.AppendLocalIdentifiers(left)
	}
	right := node.ChildByFieldName("right")
	if right != nil {
		e.walkForReferences(ctx, right)
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

	// If it's a selector, we want to capture the base part (pkg.Symbol or receiver.Method)
	if nk == "selector_expression" || nk == "qualified_type" {
		parts := strings.Split(name, ".")
		if len(parts) > 1 {
			// Handle func().Method or receiver.Method
			if strings.Contains(parts[0], "(") {
				// It's a call like newSvc(a).Method.
				// We capture 'Method' as the reference.
				name = parts[len(parts)-1]
			} else {
				// Capture the base part too (e.g. pkg) to help unused import detection
				ctx.File.References = append(ctx.File.References, Reference{
					Name:     parts[0],
					Location: ctx.Location(node),
				})

				if len(parts) > 2 {
					// Too deep, like pkg.Sub.Symbol. Just keep first two.
					name = parts[0] + "." + parts[1]
				}
			}
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
