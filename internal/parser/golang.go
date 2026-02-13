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

	e.walk(root, source, file)

	return file, nil
}

func (e *GoExtractor) walk(node *sitter.Node, source []byte, file *File) {
	nodeKind := node.Kind()

	switch nodeKind {
	case "package_clause":
		e.extractPackage(node, source, file)
	case "import_declaration":
		e.extractImports(node, source, file)
	case "function_declaration":
		e.extractFunction(node, source, file)
	case "type_declaration":
		e.extractType(node, source, file)
	case "short_var_declaration", "var_declaration", "const_declaration":
		e.extractVarDecl(node, source, file)
	case "parameter_declaration":
		e.extractParam(node, source, file)
	case "method_declaration":
		e.extractMethod(node, source, file)
	case "range_clause":
		e.extractRange(node, source, file)
	case "call_expression":
		e.extractCall(node, source, file)
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		e.walk(child, source, file)
	}
}

func (e *GoExtractor) extractPackage(node *sitter.Node, source []byte, file *File) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "package_identifier" {
			file.PackageName = e.getText(child, source)
		}
	}
}

func (e *GoExtractor) extractImports(node *sitter.Node, source []byte, file *File) {
	e.walkImports(node, source, file)
}

func (e *GoExtractor) walkImports(node *sitter.Node, source []byte, file *File) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "import_spec" {
			var alias, path string

			for j := uint(0); j < child.ChildCount(); j++ {
				spec := child.Child(j)

				if spec.Kind() == "package_identifier" {
					alias = e.getText(spec, source)
				} else if spec.Kind() == "interpreted_string_literal" {
					path = strings.Trim(e.getText(spec, source), "\"")
				}
			}

			if path != "" {
				file.Imports = append(file.Imports, Import{
					Module:    path,
					RawImport: path,
					Alias:     alias,
					Location: Location{
						File:   file.Path,
						Line:   int(child.StartPosition().Row) + 1,
						Column: int(child.StartPosition().Column) + 1,
					},
				})
			}
		} else {
			e.walkImports(child, source, file)
		}
	}
}

func (e *GoExtractor) extractFunction(node *sitter.Node, source []byte, file *File) {
	e.extractCallable(node, source, file, KindFunction)
}

func (e *GoExtractor) extractMethod(node *sitter.Node, source []byte, file *File) {
	receiver := node.ChildByFieldName("receiver")
	if receiver != nil {
		e.extractParam(receiver, source, file)
	}
	e.extractCallable(node, source, file, KindMethod)
}

func (e *GoExtractor) extractCallable(node *sitter.Node, source []byte, file *File, kind DefinitionKind) {
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	name := e.getText(nameNode, source)
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
	if file.Module != "" {
		fullName = file.Module + "." + name
	}

	file.Definitions = append(file.Definitions, Definition{
		Name:            name,
		FullName:        fullName,
		Kind:            kind,
		Exported:        exported,
		ParameterCount:  paramCount,
		BranchCount:     branches,
		NestingDepth:    nesting,
		LOC:             loc,
		ComplexityScore: score,
		Location: Location{
			File:   file.Path,
			Line:   int(node.StartPosition().Row) + 1,
			Column: int(node.StartPosition().Column) + 1,
		},
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

func (e *GoExtractor) extractType(node *sitter.Node, source []byte, file *File) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_spec" {
			e.extractTypeSpec(child, source, file)
		}
	}
}

func (e *GoExtractor) extractTypeSpec(node *sitter.Node, source []byte, file *File) {
	var name string
	var kind DefinitionKind = KindType

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "type_identifier" {
			name = e.getText(child, source)
		} else if child.Kind() == "interface_type" {
			kind = KindInterface
		}
	}

	if name == "" {
		return
	}

	exported := len(name) > 0 && unicode.IsUpper(rune(name[0]))

	file.Definitions = append(file.Definitions, Definition{
		Name:     name,
		FullName: file.Module + "." + name,
		Kind:     kind,
		Exported: exported,
		Location: Location{
			File:   file.Path,
			Line:   int(node.StartPosition().Row) + 1,
			Column: int(node.StartPosition().Column) + 1,
		},
	})
}

func (e *GoExtractor) extractVarDecl(node *sitter.Node, source []byte, file *File) {
	var findIdentifiers func(*sitter.Node)
	findIdentifiers = func(n *sitter.Node) {
		if n.Kind() == "identifier" {
			file.LocalSymbols = append(file.LocalSymbols, e.getText(n, source))
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			findIdentifiers(n.Child(i))
		}
	}

	if node.Kind() == "short_var_declaration" {
		left := node.ChildByFieldName("left")
		if left != nil {
			findIdentifiers(left)
		}
	} else {
		findIdentifiers(node)
	}
}

func (e *GoExtractor) extractParam(node *sitter.Node, source []byte, file *File) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			file.LocalSymbols = append(file.LocalSymbols, e.getText(child, source))
		}
	}
}

func (e *GoExtractor) extractRange(node *sitter.Node, source []byte, file *File) {
	left := node.ChildByFieldName("left")
	if left != nil {
		var findIdentifiers func(*sitter.Node)
		findIdentifiers = func(n *sitter.Node) {
			if n.Kind() == "identifier" {
				file.LocalSymbols = append(file.LocalSymbols, e.getText(n, source))
			}
			for i := uint(0); i < n.ChildCount(); i++ {
				findIdentifiers(n.Child(i))
			}
		}
		findIdentifiers(left)
	}
}

func (e *GoExtractor) extractCall(node *sitter.Node, source []byte, file *File) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "identifier" || child.Kind() == "selector_expression" {
			name := e.getText(child, source)

			file.References = append(file.References, Reference{
				Name: name,
				Location: Location{
					File:   file.Path,
					Line:   int(node.StartPosition().Row) + 1,
					Column: int(node.StartPosition().Column) + 1,
				},
			})
		}
	}
}

func (e *GoExtractor) getText(node *sitter.Node, source []byte) string {
	return string(source[node.StartByte():node.EndByte()])
}
