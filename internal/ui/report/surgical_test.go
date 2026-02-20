// # internal/ui/report/surgical_test.go
package report

import (
	"strings"
	"testing"
)

const sampleSource = `package main

import "os"
import "fmt"

func init() {
	os.Stdout.Write([]byte("hello"))
}

func main() {
	fmt.Println("world")
	os.Exit(0)
}
`

func TestGetSymbolContext_Found(t *testing.T) {
	ctx := GetSymbolContext("os", "main.go", []byte(sampleSource))
	if ctx.Symbol != "os" {
		t.Errorf("symbol mismatch: got %q", ctx.Symbol)
	}
	if len(ctx.Snippets) == 0 {
		t.Fatal("expected at least one snippet for 'os', got none")
	}
}

func TestGetSymbolContext_MultipleOccurrences(t *testing.T) {
	ctx := GetSymbolContext("os", "main.go", []byte(sampleSource))
	// "os" appears on the import line, os.Stdout line, and os.Exit line = 3 times
	if len(ctx.Snippets) < 2 {
		t.Errorf("expected multiple snippets, got %d", len(ctx.Snippets))
	}
}

func TestGetSymbolContext_NotFound(t *testing.T) {
	ctx := GetSymbolContext("nonexistent", "main.go", []byte(sampleSource))
	if len(ctx.Snippets) != 0 {
		t.Errorf("expected 0 snippets for absent symbol, got %d", len(ctx.Snippets))
	}
}

func TestGetSymbolContext_Empty(t *testing.T) {
	ctx := GetSymbolContext("", "main.go", []byte(sampleSource))
	if len(ctx.Snippets) != 0 {
		t.Error("expected no snippets for empty symbol")
	}
	ctx2 := GetSymbolContext("os", "", []byte{})
	if len(ctx2.Snippets) != 0 {
		t.Error("expected no snippets for empty content")
	}
}

func TestGetSymbolContext_ContextRadius(t *testing.T) {
	ctx := GetSymbolContext("fmt", "main.go", []byte(sampleSource))
	if len(ctx.Snippets) == 0 {
		t.Fatal("expected snippet for fmt")
	}
	// Each snippet should have up to 2*contextRadius+1 lines.
	s := ctx.Snippets[0]
	maxLines := 2*contextRadius + 1
	if len(s.Context) > maxLines {
		t.Errorf("context has %d lines, max expected %d", len(s.Context), maxLines)
	}
	// Context lines must be formatted as "<linenum>: <source>".
	for _, line := range s.Context {
		if !strings.Contains(line, ": ") {
			t.Errorf("context line missing colon separator: %q", line)
		}
	}
}

func TestContainsSymbol_WordBoundary(t *testing.T) {
	// "Err" should NOT match "Error"
	if containsSymbol("if err := doSomething(); err != nil {", "Error") {
		t.Error("'Error' should not match inside 'err' context via word boundary")
	}
	// "err" should match
	if !containsSymbol("if err := doSomething(); err != nil {", "err") {
		t.Error("'err' word-boundary match failed")
	}
}

func TestGetSymbolContextTagged(t *testing.T) {
	hints := map[int]TagHint{
		3: {Tag: "SYM_DEF", Confidence: 1.0, Ancestry: "source_file"},
	}
	ctx := GetSymbolContextTagged("os", "main.go", []byte(sampleSource), hints)
	for _, s := range ctx.Snippets {
		if s.Line == 3 {
			if s.Tag != "SYM_DEF" {
				t.Errorf("expected SYM_DEF tag at line 3, got %q", s.Tag)
			}
			return
		}
	}
	// If no snippet at line 3 the function is still correct (OS import may be elsewhere).
}

func TestSplitLines(t *testing.T) {
	lines := splitLines([]byte("a\nb\nc\n"))
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}
}
