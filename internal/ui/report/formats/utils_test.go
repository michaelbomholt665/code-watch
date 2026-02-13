package formats

import (
	"circular/internal/engine/graph"
	"circular/internal/engine/parser"
	"testing"
)

func TestModuleLabel(t *testing.T) {
	t.Parallel()

	mod := &graph.Module{
		Files:   []string{"a.go", "b.go"},
		Exports: map[string]*parser.Definition{"A": nil, "B": nil, "C": nil},
	}
	metrics := map[string]graph.ModuleMetrics{
		"app": {Depth: 2, FanIn: 3, FanOut: 4},
	}
	hotspots := map[string]int{
		"app": 7,
	}

	got := moduleLabel("app", mod, metrics, hotspots)
	expected := "app\\n(3 funcs, 2 files)\\n(d=2 in=3 out=4)\\n(cx=7)"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}

	got = moduleLabel("core", mod, nil, nil)
	expected = "core\\n(3 funcs, 2 files)"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestSanitizeID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "Empty", input: "", expected: "m"},
		{name: "Alpha", input: "foo", expected: "foo"},
		{name: "DigitsFirst", input: "1mod", expected: "m_1mod"},
		{name: "Symbols", input: "a/b:c", expected: "a_b_c"},
		{name: "OnlySymbols", input: "!!", expected: "__"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeID(tc.input); got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestMakeIDs(t *testing.T) {
	t.Parallel()

	names := []string{"a-b", "a_b", "c"}
	got := makeIDs(names)
	if got["a-b"] != "a_b" {
		t.Fatalf("expected a-b to map to a_b, got %q", got["a-b"])
	}
	if got["a_b"] != "a_b_2" {
		t.Fatalf("expected a_b to map to a_b_2, got %q", got["a_b"])
	}
	if got["c"] != "c" {
		t.Fatalf("expected c to map to c, got %q", got["c"])
	}
}

func TestEscapeLabel(t *testing.T) {
	t.Parallel()

	got := escapeLabel("a\"b\"c")
	if got != "a'b'c" {
		t.Fatalf("expected %q, got %q", "a'b'c", got)
	}
}
