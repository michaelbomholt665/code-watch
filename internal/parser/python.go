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

	e.walk(root, source, file)

	return file, nil
}

func (e *PythonExtractor) walk(node *sitter.Node, source []byte, file *File) {
	nodeKind := node.Kind()

	switch nodeKind {
	case "import_statement":
		e.extractImport(node, source, file)
	case "import_from_statement":
		e.extractFromImport(node, source, file)
	case "function_definition":
		e.extractFunction(node, source, file)
	case "class_definition":
		e.extractClass(node, source, file)
	case "assignment", "augmented_assignment":
		e.extractAssignment(node, source, file)
	case "for_statement":
		e.extractFor(node, source, file)
	case "call":
		e.extractCall(node, source, file)
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		e.walk(child, source, file)
	}
}

func (e *PythonExtractor) extractImport(node *sitter.Node, source []byte, file *File) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "dotted_name" || child.Kind() == "identifier" {
			module := e.getText(child, source)
			file.Imports = append(file.Imports, Import{
				Module:    module,
				RawImport: module,
				Location:  e.getLocation(child, file.Path),
			})
		} else if child.Kind() == "aliased_import" {
			var module, alias string
			for j := uint(0); j < child.ChildCount(); j++ {
				sub := child.Child(j)
				if sub.Kind() == "dotted_name" || sub.Kind() == "identifier" {
					if module == "" {
						module = e.getText(sub, source)
					} else {
						alias = e.getText(sub, source)
					}
				}
			}
			file.Imports = append(file.Imports, Import{
				Module:    module,
				RawImport: module,
				Alias:     alias,
				Location:  e.getLocation(child, file.Path),
			})
		}
	}
}

func (e *PythonExtractor) extractFromImport(node *sitter.Node, source []byte, file *File) {
	var module string
	var items []string
	isRelative := false

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "relative_import":
			isRelative = true
			relText := e.getText(child, source)
			module = strings.TrimLeft(relText, ".")

		case "dotted_name", "identifier":
			if !isRelative {
				module = e.getText(child, source)
			}

		case "import_list", "aliased_import":
			e.collectItems(child, source, &items)
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
				items = append(items, e.getText(child, source))
			}
		}
	}

	file.Imports = append(file.Imports, Import{
		Module:     module,
		RawImport:  module,
		Items:      items,
		IsRelative: isRelative,
		Location:   e.getLocation(node, file.Path),
	})
}

func (e *PythonExtractor) collectItems(node *sitter.Node, source []byte, items *[]string) {
	kind := node.Kind()
	if kind == "identifier" || kind == "dotted_name" {
		*items = append(*items, e.getText(node, source))
		return
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		e.collectItems(node.Child(i), source, items)
	}
}

func (e *PythonExtractor) extractFunction(node *sitter.Node, source []byte, file *File) {
	name := e.getChildText(node, "identifier", source)
	if name == "" {
		return
	}

	params := node.ChildByFieldName("parameters")
	if params != nil {
		e.collectLocalSymbols(params, source, file)
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
	file.Definitions = append(file.Definitions, Definition{
		Name:            name,
		FullName:        file.Module + "." + name,
		Kind:            KindFunction,
		Exported:        exported,
		ParameterCount:  paramCount,
		BranchCount:     branches,
		NestingDepth:    nesting,
		LOC:             loc,
		ComplexityScore: score,
		Location:        e.getLocation(node, file.Path),
	})
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

func (e *PythonExtractor) extractAssignment(node *sitter.Node, source []byte, file *File) {
	left := node.ChildByFieldName("left")
	if left != nil {
		e.collectLocalSymbols(left, source, file)
	}
}

func (e *PythonExtractor) extractFor(node *sitter.Node, source []byte, file *File) {
	left := node.ChildByFieldName("left")
	if left != nil {
		e.collectLocalSymbols(left, source, file)
	}
}

func (e *PythonExtractor) collectLocalSymbols(node *sitter.Node, source []byte, file *File) {
	if node.Kind() == "identifier" {
		file.LocalSymbols = append(file.LocalSymbols, e.getText(node, source))
		return
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		e.collectLocalSymbols(node.Child(i), source, file)
	}
}

func (e *PythonExtractor) extractClass(node *sitter.Node, source []byte, file *File) {
	name := e.getChildText(node, "identifier", source)
	if name == "" {
		return
	}

	exported := !strings.HasPrefix(name, "_")
	file.Definitions = append(file.Definitions, Definition{
		Name:     name,
		FullName: file.Module + "." + name,
		Kind:     KindClass,
		Exported: exported,
		Location: e.getLocation(node, file.Path),
	})
}

func (e *PythonExtractor) extractCall(node *sitter.Node, source []byte, file *File) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "attribute" || child.Kind() == "identifier" {
			name := e.getText(child, source)
			file.References = append(file.References, Reference{
				Name:     name,
				Location: e.getLocation(child, file.Path),
			})
		}
	}
}

func (e *PythonExtractor) getChildText(node *sitter.Node, kind string, source []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return e.getText(child, source)
		}
	}
	return ""
}

func (e *PythonExtractor) getLocation(node *sitter.Node, filePath string) Location {
	return Location{
		File:   filePath,
		Line:   int(node.StartPosition().Row) + 1,
		Column: int(node.StartPosition().Column) + 1,
	}
}

func (e *PythonExtractor) getText(node *sitter.Node, source []byte) string {
	return string(source[node.StartByte():node.EndByte()])
}
