// # internal/engine/graph/lru_test.go
package graph

import (
	"fmt"
	"sync"
	"testing"
)

func TestLRUCache_GetPut(t *testing.T) {
	c := NewLRUCache[string, int](3)

	// Empty cache miss
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected miss on empty cache")
	}

	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	if c.Len() != 3 {
		t.Fatalf("expected len 3, got %d", c.Len())
	}

	for k, want := range map[string]int{"a": 1, "b": 2, "c": 3} {
		v, ok := c.Get(k)
		if !ok {
			t.Fatalf("expected hit for %q", k)
		}
		if v != want {
			t.Fatalf("key %q: want %d got %d", k, want, v)
		}
	}
}

func TestLRUCache_EvictLRU(t *testing.T) {
	// Capacity 2 — inserting a third entry should evict the LRU.
	c := NewLRUCache[string, int](2)

	c.Put("a", 1)
	c.Put("b", 2)

	// Access "a" so that "b" becomes the LRU.
	c.Get("a")

	// Insert "c" — "b" (LRU) should be evicted.
	c.Put("c", 3)

	if c.Len() != 2 {
		t.Fatalf("expected len 2, got %d", c.Len())
	}
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected 'a' to still be present")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("expected 'c' to be present")
	}
}

func TestLRUCache_UpdateExisting(t *testing.T) {
	c := NewLRUCache[string, int](3)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3)

	// Update "a" — should not change length, and "a" becomes MRU.
	c.Put("a", 99)

	if c.Len() != 3 {
		t.Fatalf("expected len 3 after update, got %d", c.Len())
	}
	v, ok := c.Get("a")
	if !ok || v != 99 {
		t.Fatalf("expected updated value 99, got %d (ok=%v)", v, ok)
	}

	// Insert "d" — "b" should be evicted (LRU after "a" was refreshed).
	c.Put("d", 4)
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected 'b' to be evicted")
	}
}

func TestLRUCache_ExplicitEvict(t *testing.T) {
	c := NewLRUCache[string, int](5)
	c.Put("a", 1)
	c.Put("b", 2)

	c.Evict("a")
	if c.Len() != 1 {
		t.Fatalf("expected len 1 after evict, got %d", c.Len())
	}
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected 'a' to be gone after explicit evict")
	}

	// Evicting non-existent key is a no-op.
	c.Evict("nonexistent")
	if c.Len() != 1 {
		t.Fatalf("expected len still 1, got %d", c.Len())
	}
}

func TestLRUCache_Clear(t *testing.T) {
	c := NewLRUCache[string, int](5)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Clear()

	if c.Len() != 0 {
		t.Fatalf("expected len 0 after clear, got %d", c.Len())
	}
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected empty cache after clear")
	}
}

func TestLRUCache_CapacityOne(t *testing.T) {
	c := NewLRUCache[string, int](1)
	c.Put("a", 1)
	c.Put("b", 2)

	if c.Len() != 1 {
		t.Fatalf("expected len 1, got %d", c.Len())
	}
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected 'a' evicted")
	}
	if v, ok := c.Get("b"); !ok || v != 2 {
		t.Fatalf("expected 'b'=2, got %d (ok=%v)", v, ok)
	}
}

func TestLRUCache_Cap(t *testing.T) {
	c := NewLRUCache[string, int](42)
	if c.Cap() != 42 {
		t.Fatalf("expected capacity 42, got %d", c.Cap())
	}
}

func TestLRUCache_NonPositiveCapacity(t *testing.T) {
	// Negative and zero capacities are normalized to 1.
	for _, cap := range []int{0, -1, -100} {
		c := NewLRUCache[string, int](cap)
		if c.Cap() != 1 {
			t.Errorf("capacity %d: expected normalised cap=1, got %d", cap, c.Cap())
		}
	}
}

func TestLRUCache_ConcurrentAccess(t *testing.T) {
	const workers = 20
	const ops = 100
	c := NewLRUCache[int, int](50)

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				key := (id*ops + i) % 80
				c.Put(key, key*2)
				c.Get(key)
				if key%10 == 0 {
					c.Evict(key)
				}
			}
		}(w)
	}
	wg.Wait()
	// Cache must be internally consistent — len <= capacity.
	if c.Len() > c.Cap() {
		t.Fatalf("len %d exceeds capacity %d after concurrent use", c.Len(), c.Cap())
	}
}

// TestLRUCache_ModuleValues verifies the cache works with *Module values,
// mirroring its intended use inside the graph package.
func TestLRUCache_ModuleValues(t *testing.T) {
	c := NewLRUCache[string, *Module](4)

	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("mod%d", i)
		c.Put(name, &Module{Name: name})
	}

	mod, ok := c.Get("mod0")
	if !ok || mod.Name != "mod0" {
		t.Fatalf("expected mod0, got ok=%v name=%q", ok, mod.Name)
	}

	// Adding a 5th entry evicts the LRU (mod1, since mod0 was just accessed).
	c.Put("mod4", &Module{Name: "mod4"})
	if _, ok := c.Get("mod1"); ok {
		t.Fatal("expected mod1 to be evicted")
	}
}
