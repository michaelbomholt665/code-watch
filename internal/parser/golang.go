// # internal/parser/golang.go
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
	var name string

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			name = e.getText(child, source)
			break
		}
	}

	if name == "" {
		return
	}

	exported := len(name) > 0 && unicode.IsUpper(rune(name[0]))

	file.Definitions = append(file.Definitions, Definition{
		Name:     name,
		FullName: file.Module + "." + name,
		Kind:     KindFunction,
		Exported: exported,
		Location: Location{
			File:   file.Path,
			Line:   int(node.StartPosition().Row) + 1,
			Column: int(node.StartPosition().Column) + 1,
		},
	})
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
	// Simple DFS to find all identifiers on the left side
	var findIdentifiers func(*sitter.Node)
	findIdentifiers = func(n *sitter.Node) {
		if n.Kind() == "identifier" {
			file.LocalSymbols = append(file.LocalSymbols, e.getText(n, source))
		}
		// In short_var_declaration, left is a field. We only want the left side of :=
		// But for now, any identifier in these declaration nodes is likely a local symbol
		for i := uint(0); i < n.ChildCount(); i++ {
			findIdentifiers(n.Child(i))
		}
	}

	// For var_declaration/const_declaration, we want the names being defined
	// For short_var_declaration, we only want the left side.
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

func (e *GoExtractor) extractMethod(node *sitter.Node, source []byte, file *File) {
	// Extract receiver
	receiver := node.ChildByFieldName("receiver")
	if receiver != nil {
		e.extractParam(receiver, source, file)
	}
	e.extractFunction(node, source, file)
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
