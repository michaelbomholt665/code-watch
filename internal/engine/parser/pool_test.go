// # internal/engine/parser/pool_test.go
package parser

import (
	"sync"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// goLanguage returns the tree-sitter Go language grammar for test use.
func goLanguage() *sitter.Language {
	return sitter.NewLanguage(tree_sitter_go.Language())
}

func TestParserPool_GetPut(t *testing.T) {
	lang := goLanguage()
	pool := NewParserPool(lang)

	sp := pool.Get()
	if sp == nil {
		t.Fatal("expected non-nil parser from pool")
	}

	// Return to pool — must not panic.
	pool.Put(sp)
}

func TestParserPool_ReusesParsers(t *testing.T) {
	lang := goLanguage()
	pool := NewParserPool(lang)

	sp1 := pool.Get()
	pool.Put(sp1)

	// The sync.Pool may or may not return the exact same pointer (GC can
	// clear it), but it must return a valid, usable parser.
	sp2 := pool.Get()
	if sp2 == nil {
		t.Fatal("expected non-nil parser on second Get")
	}
	pool.Put(sp2)
}

func TestParserPool_PutNil(t *testing.T) {
	lang := goLanguage()
	pool := NewParserPool(lang)

	// Put(nil) must be a no-op — must not panic.
	pool.Put(nil)
}

func TestParserPool_ParsesValidGo(t *testing.T) {
	lang := goLanguage()
	pool := NewParserPool(lang)

	sp := pool.Get()
	defer pool.Put(sp)

	src := []byte("package main\nfunc main() {}\n")
	tree := sp.Parse(src, nil)
	if tree == nil {
		t.Fatal("expected non-nil parse tree for valid Go source")
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil || root.HasError() {
		t.Fatalf("expected error-free root node, got hasError=%v", root.HasError())
	}
}

func TestParserPool_ConcurrentAccess(t *testing.T) {
	lang := goLanguage()
	pool := NewParserPool(lang)

	const goroutines = 20
	const iters = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	src := []byte("package main\nfunc run() {}\n")

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				sp := pool.Get()
				tree := sp.Parse(src, nil)
				if tree == nil {
					t.Errorf("expected non-nil parse tree")
				} else {
					tree.Close()
				}
				pool.Put(sp)
			}
		}()
	}

	wg.Wait()
}

func TestParserPool_LanguageSetAfterReset(t *testing.T) {
	// Verify that Get() re-sets the language after Reset() was called.
	lang := goLanguage()
	pool := NewParserPool(lang)

	sp := pool.Get()
	sp.Reset() // Simulate external reset before Put.
	pool.Put(sp)

	// Next Get() should still return a parser with the language set.
	sp2 := pool.Get()
	defer pool.Put(sp2)

	src := []byte("package main\nfunc ok() {}\n")
	tree := sp2.Parse(src, nil)
	if tree == nil {
		t.Fatal("parser with reset language should still parse correctly after Get")
	}
	defer tree.Close()
}
