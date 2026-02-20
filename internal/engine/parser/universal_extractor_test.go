// # internal/engine/parser/universal_extractor_test.go
package parser

import (
	"strings"
	"testing"
)

// TestClassifyNodeKind ensures all five tag types are reachable via the regex tiers.
func TestClassifyNodeKind(t *testing.T) {
	cases := []struct {
		kind    string
		wantTag UsageTag
	}{
		{"function_declaration", TagSymDef},
		{"method_declaration", TagSymDef},
		{"class_definition", TagSymDef},
		{"struct_item", TagSymDef},
		{"call_expression", TagRefCall},
		{"function_call", TagRefCall},
		{"method_invocation", TagRefCall},
		{"type_identifier", TagRefType},
		{"qualified_type", TagRefType},
		{"generic_type", TagRefType},
		{"string_literal", TagRefDyn},
		{"interpreted_string_literal", TagRefDyn},
		{"blank_identifier", TagRefSide},
	}

	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			got, ok := classifyNodeKind(tc.kind)
			if !ok {
				t.Fatalf("classifyNodeKind(%q): not matched, expected %s", tc.kind, tc.wantTag)
			}
			if got != tc.wantTag {
				t.Fatalf("classifyNodeKind(%q) = %s, want %s", tc.kind, got, tc.wantTag)
			}
		})
	}
}

// TestTagConfidence verifies all tags have valid non-zero confidence.
func TestTagConfidence(t *testing.T) {
	tags := []UsageTag{TagSymDef, TagRefCall, TagRefType, TagRefSide, TagRefDyn}
	for _, tag := range tags {
		conf, ok := tagConfidence[tag]
		if !ok {
			t.Errorf("no confidence entry for tag %s", tag)
			continue
		}
		if conf <= 0 || conf > 1.0 {
			t.Errorf("tag %s has out-of-range confidence %f", tag, conf)
		}
	}
}

// TestApplyTaggedSymbol_Definitions checks that SYM_DEF tags go to Definitions.
func TestApplyTaggedSymbol_Definitions(t *testing.T) {
	file := &File{Path: "test.go"}
	applyTaggedSymbol(file, TaggedSymbol{
		Name:       "MyFunc",
		Tag:        TagSymDef,
		Confidence: 1.0,
		Ancestry:   "source_file->function_declaration",
		Location:   Location{File: "test.go", Line: 10},
	})
	if len(file.Definitions) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(file.Definitions))
	}
	if file.Definitions[0].Name != "MyFunc" {
		t.Errorf("got name %q, want MyFunc", file.Definitions[0].Name)
	}
	if !strings.Contains(file.Definitions[0].Scope, "function_declaration") {
		t.Errorf("expected ancestry in Scope, got %q", file.Definitions[0].Scope)
	}
}

// TestApplyTaggedSymbol_References checks that non-SYM_DEF tags go to References.
func TestApplyTaggedSymbol_References(t *testing.T) {
	file := &File{Path: "test.go"}
	for _, tag := range []UsageTag{TagRefCall, TagRefType, TagRefDyn, TagRefSide} {
		applyTaggedSymbol(file, TaggedSymbol{
			Name:       "sym",
			Tag:        tag,
			Confidence: tagConfidence[tag],
			Ancestry:   "root->block",
			Location:   Location{File: "test.go", Line: 5},
		})
	}
	if len(file.References) != 4 {
		t.Fatalf("expected 4 references, got %d", len(file.References))
	}
}

// TestExtractNodeName_FallbackToLeaf ensures leaf-node text is returned when no
// named child exists.
func TestExtractNodeName_FallbackToLeaf(t *testing.T) {
	// We don't have a live Tree-sitter tree here, so we test nodeText directly.
	src := []byte("hello world")
	// nodeText with nil node should return "".
	if got := nodeText(nil, src); got != "" {
		t.Errorf("expected empty for nil node, got %q", got)
	}
}
