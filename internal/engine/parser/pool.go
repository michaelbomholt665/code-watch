// # internal/engine/parser/pool.go
package parser

import (
	"sync"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ParserPool recycles tree-sitter parser instances to avoid the per-file
// allocation overhead of sitter.NewParser() / parser.Close().
//
// Each pool is tied to a single tree-sitter language grammar. For multi-
// language workloads, create one ParserPool per language and obtain the pool
// from the per-language registry.
//
// Usage (inside Extractor.Extract or Parser.ParseFile):
//
//	sp := pool.Get()
//	defer pool.Put(sp)
//	tree := sp.Parse(source, nil)
//
// Concurrency: safe for use by multiple goroutines simultaneously.
type ParserPool struct {
	lang *sitter.Language
	pool sync.Pool
}

// NewParserPool creates a pool for the given language grammar.
// The language must remain valid for the lifetime of the pool.
func NewParserPool(lang *sitter.Language) *ParserPool {
	p := &ParserPool{lang: lang}
	p.pool = sync.Pool{
		New: func() any {
			sp := sitter.NewParser()
			sp.SetLanguage(lang)
			return sp
		},
	}
	return p
}

// Get retrieves a parser from the pool, or allocates a new one if the pool is
// empty. The returned parser is already configured for the pool's language.
func (p *ParserPool) Get() *sitter.Parser {
	sp := p.pool.Get().(*sitter.Parser)
	// Ensure the language is set in case the parser was Reset() externally.
	sp.SetLanguage(p.lang)
	return sp
}

// Put returns a parser to the pool for reuse. The parser is reset before
// being stored so that no references to previous parse trees are retained.
// Callers must not use sp after calling Put.
func (p *ParserPool) Put(sp *sitter.Parser) {
	if sp == nil {
		return
	}
	sp.Reset()
	p.pool.Put(sp)
}
