// # internal/engine/parser/universal.go
package parser

import (
	"regexp"
	"strings"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// UsageTag classifies the semantic role of a symbol occurrence in the AST.
type UsageTag string

const (
	// TagSymDef is the primary definition site (confidence 1.0).
	TagSymDef UsageTag = "SYM_DEF"
	// TagRefCall is a direct invocation (confidence 0.9).
	TagRefCall UsageTag = "REF_CALL"
	// TagRefType is a type reference in a declaration (confidence 0.8).
	TagRefType UsageTag = "REF_TYPE"
	// TagRefSide is a side-effect import / blank identifier (confidence 0.7).
	TagRefSide UsageTag = "REF_SIDE"
	// TagRefDyn is a potential dynamic reference inside a string literal (confidence 0.4).
	TagRefDyn UsageTag = "REF_DYN"
)

// tagConfidence maps each UsageTag to its default confidence score.
var tagConfidence = map[UsageTag]float64{
	TagSymDef:  1.0,
	TagRefCall: 0.9,
	TagRefType: 0.8,
	TagRefSide: 0.7,
	TagRefDyn:  0.4,
}

// TaggedSymbol extends a Reference with structured semantic metadata.
type TaggedSymbol struct {
	Name       string
	RawName    string
	Tag        UsageTag
	Confidence float64
	// Ancestry is the chain of ancestor node kinds leading to this node,
	// e.g. "source_file->function_declaration->block->call_expression".
	Ancestry string
	Location Location
}

// patternTier pairs a compiled regex against a UsageTag.
type patternTier struct {
	re  *regexp.Regexp
	tag UsageTag
}

// universalPatternTiers is the ordered list of node-kind classifiers.
// Evaluated top-to-bottom; first match wins.
var universalPatternTiers = func() []patternTier {
	specs := []struct {
		pattern string
		tag     UsageTag
	}{
		// Definitions — any *_declaration / *_definition, plus common named forms.
		{`(?i)(^|_)(declaration|definition)$`, TagSymDef},
		{`(?i)^(function_item|method_declaration|class_definition|struct_item|enum_item|interface_declaration|trait_item|impl_item)$`, TagSymDef},

		// Type references — type identifiers and qualified/generic forms.
		{`(?i)^(type_identifier|qualified_type|generic_type|scoped_type_identifier|type_spec|type_ref)$`, TagRefType},

		// Direct calls / invocations.
		{`(?i)^(call_expression|method_invocation|function_call|invocation_expression|call_stmt)$`, TagRefCall},

		// String literals — potential dynamic refs.
		{`(?i)^(string_literal|interpreted_string_literal|raw_string_literal|string|string_content)$`, TagRefDyn},

		// Side-effect / blank imports.
		{`(?i)^blank_identifier$`, TagRefSide},
	}

	tiers := make([]patternTier, 0, len(specs))
	for _, s := range specs {
		tiers = append(tiers, patternTier{
			re:  regexp.MustCompile(s.pattern),
			tag: s.tag,
		})
	}
	return tiers
}()

// classifyNodeKind returns the UsageTag for a tree-sitter node kind, or "".
func classifyNodeKind(kind string) (UsageTag, bool) {
	for _, tier := range universalPatternTiers {
		if tier.re.MatchString(kind) {
			return tier.tag, true
		}
	}
	return "", false
}

// UniversalExtractor implements Extractor using regex-driven node classification
// for any language. It walks every node in the AST and emits TaggedSymbols that
// are stored in File.References alongside confidence and ancestry metadata.
type UniversalExtractor struct{}

// NewUniversalExtractor returns an initialised UniversalExtractor.
func NewUniversalExtractor() *UniversalExtractor { return &UniversalExtractor{} }

// Extract walks root, classifying every node and building the File.
// It runs a language-detection + import-extraction pass first so that all
// standard parser fields (Language, PackageName, Imports) are populated for
// any language whose AST follows Go or Python conventions.
func (e *UniversalExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		Language: detectLangFromPath(filePath),
		ParsedAt: time.Now(),
	}
	if root == nil {
		return file, nil
	}

	// Pass 1: extract imports and package/module names language-agnostically.
	walkImports(root, source, file)

	// Pass 2: classify every node for definitions and references.
	ancestry := make([]string, 0, 32)
	walkUniversal(root, source, file, ancestry)
	return file, nil
}

// detectLangFromPath returns a lowercase language ID from a file extension.
func detectLangFromPath(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx < 0 {
		base := path
		if i := strings.LastIndex(path, "/"); i >= 0 {
			base = path[i+1:]
		}
		switch strings.ToLower(base) {
		case "go.mod":
			return "gomod"
		case "go.sum":
			return "gosum"
		}
		return ""
	}
	switch strings.ToLower(path[idx:]) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".java":
		return "java"
	case ".rs":
		return "rust"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	}
	return ""
}

// walkImports does a single-level scan of the root AST to extract import
// declarations and package/module names for Go and Python (and common patterns
// for other languages). It does NOT recurse deeply — top-level nodes only.
func walkImports(root *sitter.Node, source []byte, file *File) {
	count := root.ChildCount()
	for i := uint(0); i < count; i++ {
		node := root.Child(i)
		if node == nil {
			continue
		}
		kind := node.Kind()
		switch kind {
		// ── Go ────────────────────────────────────────────────────────────────
		case "package_clause":
			// package <name>
			for j := uint(0); j < node.ChildCount(); j++ {
				ch := node.Child(j)
				if ch != nil && ch.Kind() == "package_identifier" {
					file.PackageName = nodeText(ch, source)
				}
			}

		case "import_declaration":
			// import "pkg" OR import ( "pkg1" "pkg2" )
			extractGoImportDecl(node, source, file)

		// ── Python ────────────────────────────────────────────────────────────
		case "import_statement":
			// import os  |  import sys as system
			extractPyImportStatement(node, source, file)

		case "import_from_statement":
			// from auth.utils import login  |  from . import mod
			extractPyFromImportStatement(node, source, file)
		}
	}
}

// extractGoImportDecl handles Go's import_declaration node (both single and
// parenthesised forms).
func extractGoImportDecl(node *sitter.Node, source []byte, file *File) {
	for i := uint(0); i < node.ChildCount(); i++ {
		ch := node.Child(i)
		if ch == nil {
			continue
		}
		switch ch.Kind() {
		case "import_spec":
			addGoImportSpec(ch, source, file)
		case "import_spec_list":
			// import ( ... ) — iterate the list
			for j := uint(0); j < ch.ChildCount(); j++ {
				spec := ch.Child(j)
				if spec != nil && spec.Kind() == "import_spec" {
					addGoImportSpec(spec, source, file)
				}
			}
		}
	}
}

// addGoImportSpec resolves a single Go import_spec child into an Import.
func addGoImportSpec(spec *sitter.Node, source []byte, file *File) {
	var alias, module string
	for i := uint(0); i < spec.ChildCount(); i++ {
		ch := spec.Child(i)
		if ch == nil {
			continue
		}
		switch ch.Kind() {
		case "package_identifier", "blank_identifier", "dot":
			alias = nodeText(ch, source)
		case "interpreted_string_literal", "raw_string_literal":
			module = strings.Trim(nodeText(ch, source), `"`+"`")
		}
	}
	if module == "" {
		return
	}
	line := int(spec.StartPosition().Row) + 1
	file.Imports = append(file.Imports, Import{
		Module:   module,
		Alias:    alias,
		Location: Location{File: file.Path, Line: line},
	})
}

// extractPyImportStatement handles Python's "import os" / "import sys as s".
func extractPyImportStatement(node *sitter.Node, source []byte, file *File) {
	line := int(node.StartPosition().Row) + 1
	// Children: "import" keyword, then dotted_name or aliased_import nodes.
	for i := uint(0); i < node.ChildCount(); i++ {
		ch := node.Child(i)
		if ch == nil {
			continue
		}
		switch ch.Kind() {
		case "dotted_name":
			module := nodeText(ch, source)
			file.Imports = append(file.Imports, Import{
				Module:   module,
				Location: Location{File: file.Path, Line: line},
			})
		case "aliased_import":
			// dotted_name "as" identifier
			module, alias := extractPyAliasedImport(ch, source)
			if module != "" {
				file.Imports = append(file.Imports, Import{
					Module:   module,
					Alias:    alias,
					Location: Location{File: file.Path, Line: line},
				})
			}
		}
	}
}

// extractPyFromImportStatement handles "from auth.utils import login as auth_login".
func extractPyFromImportStatement(node *sitter.Node, source []byte, file *File) {
	line := int(node.StartPosition().Row) + 1
	var module string
	// The first dotted_name / relative_import is the "from" target.
	for i := uint(0); i < node.ChildCount(); i++ {
		ch := node.Child(i)
		if ch == nil {
			continue
		}
		switch ch.Kind() {
		case "dotted_name":
			if module == "" {
				module = nodeText(ch, source)
				file.Imports = append(file.Imports, Import{
					Module:   module,
					Location: Location{File: file.Path, Line: line},
				})
			}
		case "relative_import":
			if module == "" {
				module = nodeText(ch, source) // e.g. "." or "..parent"
				file.Imports = append(file.Imports, Import{
					Module:   module,
					Location: Location{File: file.Path, Line: line},
				})
			}
		}
	}
}

// extractPyAliasedImport returns (module, alias) from an aliased_import node.
func extractPyAliasedImport(node *sitter.Node, source []byte) (module, alias string) {
	for i := uint(0); i < node.ChildCount(); i++ {
		ch := node.Child(i)
		if ch == nil {
			continue
		}
		switch ch.Kind() {
		case "dotted_name":
			if module == "" {
				module = nodeText(ch, source)
			}
		case "identifier":
			alias = nodeText(ch, source)
		}
	}
	return
}

// walkUniversal performs a depth-first traversal of the AST, classifying each
// node and accumulating tagged symbols into file.References and file.Definitions.
func walkUniversal(node *sitter.Node, source []byte, file *File, ancestry []string) {
	if node == nil {
		return
	}

	kind := node.Kind()
	ancestryPath := strings.Join(ancestry, "->")

	// Classify the current node.
	if tag, ok := classifyNodeKind(kind); ok {
		confidence := tagConfidence[tag]
		name := extractNodeName(node, source)
		if name != "" {
			rawName := nodeText(node, source)
			if len(rawName) > 256 {
				rawName = name // cap excessively large expressions
			}
			loc := Location{
				File:   file.Path,
				Line:   int(node.StartPosition().Row) + 1,
				Column: int(node.StartPosition().Column) + 1,
			}
			tagged := TaggedSymbol{
				Name:       name,
				RawName:    rawName,
				Tag:        tag,
				Confidence: confidence,
				Ancestry:   ancestryPath,
				Location:   loc,
			}
			applyTaggedSymbol(file, tagged)
		}
	}

	// Push this node kind onto the ancestry stack for children.
	nextAncestry := append(ancestry, kind) //nolint:gocritic // intentional append
	for i := uint(0); i < node.ChildCount(); i++ {
		walkUniversal(node.Child(i), source, file, nextAncestry)
	}
}

// extractNodeName attempts to get the symbolic name from a node.
// It prioritises specific named fields ("function", "name", "type"),
// then common child node kinds, and finally falls back to simple leaf text.
func extractNodeName(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}

	// For function calls, extract the 'function' child.
	if fn := node.ChildByFieldName("function"); fn != nil {
		if text := nodeText(fn, source); text != "" && len(text) <= 128 {
			return text
		}
	}

	// For other declarations having a 'name' or 'type' field.
	if name := node.ChildByFieldName("name"); name != nil {
		if text := nodeText(name, source); text != "" && len(text) <= 128 {
			return text
		}
	}
	if typ := node.ChildByFieldName("type"); typ != nil {
		if text := nodeText(typ, source); text != "" && len(text) <= 128 {
			return text
		}
	}

	// Search direct children for typical identifier kinds.
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		kind := child.Kind()
		switch kind {
		case "identifier", "name", "field_identifier", "type_identifier", "property_identifier":
			if text := nodeText(child, source); text != "" && len(text) <= 128 {
				return text
			}
		}
	}

	// Fall back to the node's own text if it's a simple leaf node
	// or specific expressions like member_expression or scoped_identifier.
	kind := node.Kind()
	if node.ChildCount() == 0 || kind == "member_expression" || kind == "scoped_identifier" {
		text := nodeText(node, source)
		if len(text) <= 128 {
			return text
		}
	}
	return ""
}

// nodeText returns the source bytes spanned by a node as a trimmed string.
func nodeText(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	start := node.StartByte()
	end := node.EndByte()
	if start >= end || end > uint(len(source)) {
		return ""
	}
	return strings.TrimSpace(string(source[start:end]))
}

// applyTaggedSymbol distributes a TaggedSymbol into the appropriate File slice.
func applyTaggedSymbol(file *File, t TaggedSymbol) {
	switch t.Tag {
	case TagSymDef:
		file.Definitions = append(file.Definitions, Definition{
			Name:     t.Name,
			FullName: t.Name,
			Kind:     KindFunction, // default; language-specific extractors refine this
			Location: t.Location,
			// Store ancestry in Scope for now (visible in SymbolRecord).
			Scope: t.Ancestry,
		})
	case TagRefCall:
		// Base context with tag + ancestry
		context := string(t.Tag) + "|" + t.Ancestry

		// Map known service/ffi/process calls using the extractor common logic
		semanticCtx := callReferenceContext(file.Language, t.RawName)
		if semanticCtx == RefContextDefault {
			semanticCtx = callReferenceContext(file.Language, t.Name)
		}
		if semanticCtx != RefContextDefault {
			context = semanticCtx
		}

		file.References = append(file.References, Reference{
			Name:     t.Name,
			Location: t.Location,
			Context:  context,
		})
	default:
		// All reference tags are stored in References.
		file.References = append(file.References, Reference{
			Name:     t.Name,
			Location: t.Location,
			Context:  string(t.Tag) + "|" + t.Ancestry,
		})
	}
}
