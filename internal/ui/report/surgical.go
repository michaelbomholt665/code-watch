// # internal/ui/report/surgical.go
package report

import (
	"bytes"
	"strings"
)

const contextRadius = 5 // ±5 lines around each occurrence

// SymbolContext is the response returned by GetSymbolContext.
type SymbolContext struct {
	// Symbol is the queried symbol name.
	Symbol string
	// File is the source file that was searched.
	File string
	// Snippets are each individual usage site found in the file.
	Snippets []Snippet
}

// Snippet represents a single occurrence of a symbol with surrounding context.
type Snippet struct {
	// Line is the 1-indexed line number of the occurrence.
	Line int
	// Tag is the semantic classification (e.g. "REF_CALL", "SYM_DEF").
	// Empty when the match is plain text without semantic context.
	Tag string
	// Confidence is the semantic confidence score (0.0 – 1.0); 0 when unknown.
	Confidence float64
	// Ancestry is the structural path to this occurrence
	// (e.g. "source_file->function_declaration->block").
	Ancestry string
	// Context is the surrounding source lines, centred on the match.
	// Each entry has the format "<linenum>: <source>".
	Context []string
}

// GetSymbolContext scans file content for occurrences of symbol and returns
// each occurrence with ±contextRadius lines of surrounding source.
//
// The search is case-sensitive plain text. For richer semantic results,
// callers should pass TaggedSymbols from the UniversalExtractor as annotations
// (future enhancement: accept []parser.TaggedSymbol overlay).
func GetSymbolContext(symbol, filePath string, content []byte) SymbolContext {
	ctx := SymbolContext{Symbol: symbol, File: filePath}
	if symbol == "" || len(content) == 0 {
		return ctx
	}

	lines := splitLines(content)
	for i, line := range lines {
		if !containsSymbol(line, symbol) {
			continue
		}
		lineNum := i + 1 // convert to 1-indexed
		ctx.Snippets = append(ctx.Snippets, Snippet{
			Line:    lineNum,
			Context: buildContext(lines, i, contextRadius),
		})
	}
	return ctx
}

// GetSymbolContextTagged is an extended variant that annotates Snippets with
// Tag, Confidence, and Ancestry from a pre-computed list of tagged occurrences.
// Occurrences are matched by line number.
func GetSymbolContextTagged(symbol, filePath string, content []byte, tags map[int]TagHint) SymbolContext {
	ctx := GetSymbolContext(symbol, filePath, content)
	for i := range ctx.Snippets {
		if hint, ok := tags[ctx.Snippets[i].Line]; ok {
			ctx.Snippets[i].Tag = hint.Tag
			ctx.Snippets[i].Confidence = hint.Confidence
			ctx.Snippets[i].Ancestry = hint.Ancestry
		}
	}
	return ctx
}

// TagHint carries semantic metadata keyed by line number.
type TagHint struct {
	Tag        string
	Confidence float64
	Ancestry   string
}

// containsSymbol returns true if line contains symbol as a word-boundary match.
// The check is intentionally simple to remain language-agnostic and avoid
// false negatives from partial identifier matches.
func containsSymbol(line, symbol string) bool {
	idx := strings.Index(line, symbol)
	if idx < 0 {
		return false
	}
	// Verify word boundaries so "Err" doesn't match "Error".
	before := idx > 0 && isIdentChar(rune(line[idx-1]))
	after := idx+len(symbol) < len(line) && isIdentChar(rune(line[idx+len(symbol)]))
	return !before && !after
}

func isIdentChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_'
}

// buildContext returns ±radius lines around the hit, formatted as
// "<linenum>: <source>".
func buildContext(lines []string, hitIdx, radius int) []string {
	start := hitIdx - radius
	if start < 0 {
		start = 0
	}
	end := hitIdx + radius + 1
	if end > len(lines) {
		end = len(lines)
	}

	out := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		// Use a fixed-width line number so snippets align nicely.
		out = append(out, formatContextLine(i+1, lines[i]))
	}
	return out
}

// formatContextLine returns a "<linenum>: <source>" string.
func formatContextLine(lineNum int, source string) string {
	var b strings.Builder
	b.Grow(8 + len(source))
	// Write up to 6-digit line numbers, right-aligned.
	lineStr := formatLineNum(lineNum)
	b.WriteString(lineStr)
	b.WriteString(": ")
	b.WriteString(source)
	return b.String()
}

func formatLineNum(n int) string {
	s := strings.Repeat(" ", 6)
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if len(digits) == 0 {
		digits = []byte{'0'}
	}
	pad := 6 - len(digits)
	if pad < 0 {
		pad = 0
	}
	return s[:pad] + string(digits)
}

// splitLines splits content on newlines, preserving empty lines.
func splitLines(content []byte) []string {
	raw := bytes.Split(content, []byte("\n"))
	lines := make([]string, len(raw))
	for i, b := range raw {
		lines[i] = string(b)
	}
	// Trim trailing empty line that Split adds for a final newline.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
