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
func (e *UniversalExtractor) Extract(root *sitter.Node, source []byte, filePath string) (*File, error) {
	file := &File{
		Path:     filePath,
		ParsedAt: time.Now(),
	}
	if root == nil {
		return file, nil
	}

	// ancestry tracks the kinds of ancestor nodes as we descend.
	ancestry := make([]string, 0, 32)
	walkUniversal(root, source, file, ancestry)
	return file, nil
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
			loc := Location{
				File:   file.Path,
				Line:   int(node.StartPosition().Row) + 1,
				Column: int(node.StartPosition().Column) + 1,
			}
			tagged := TaggedSymbol{
				Name:       name,
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
// It tries "identifier" and "name" children first, then falls back to the full
// node text (capped to avoid capturing large expression text).
func extractNodeName(node *sitter.Node, source []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := child.Kind()
		if kind == "identifier" || kind == "name" || kind == "field_identifier" || kind == "type_identifier" {
			text := nodeText(child, source)
			if text != "" {
				return text
			}
		}
	}
	// Fall back to the node's own text for simple leaf nodes.
	if node.ChildCount() == 0 {
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
	default:
		// All reference tags are stored in References.
		file.References = append(file.References, Reference{
			Name:     t.Name,
			Location: t.Location,
			Context:  string(t.Tag) + "|" + t.Ancestry,
		})
	}
}
