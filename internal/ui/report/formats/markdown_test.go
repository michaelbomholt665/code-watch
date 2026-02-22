package formats

import (
	"circular/internal/engine/graph"
	"strings"
	"testing"
)

func TestMarkdownGenerator_OmitsComplexitySectionWhenNoHotspots(t *testing.T) {
	gen := NewMarkdownGenerator()
	out, err := gen.Generate(
		MarkdownReportData{},
		MarkdownReportOptions{
			TableOfContents: true,
		},
	)
	if err != nil {
		t.Fatalf("generate markdown: %v", err)
	}
	if strings.Contains(out, "- [Complexity Hotspots](#complexity-hotspots)") {
		t.Fatal("expected complexity hotspot TOC entry to be omitted")
	}
	if strings.Contains(out, "## Complexity Hotspots") {
		t.Fatal("expected complexity hotspot section to be omitted")
	}
}

func TestMarkdownGenerator_IncludesComplexitySectionWhenHotspotsPresent(t *testing.T) {
	gen := NewMarkdownGenerator()
	out, err := gen.Generate(
		MarkdownReportData{
			Hotspots: []graph.ComplexityHotspot{
				{Module: "mod", Definition: "Run", Score: 5, LOC: 20},
			},
		},
		MarkdownReportOptions{
			TableOfContents: true,
		},
	)
	if err != nil {
		t.Fatalf("generate markdown: %v", err)
	}
	if !strings.Contains(out, "## Complexity Hotspots") {
		t.Fatal("expected complexity hotspot section to be included")
	}
}
