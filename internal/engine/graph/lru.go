// # internal/engine/graph/lru.go
package graph

import (
	"container/list"
	"sync"
)

// LRUCache is a thread-safe, capacity-bounded Least-Recently-Used cache.
// It uses Go generics so it can hold any key/value pair.
// When the cache is full the least-recently-used entry is evicted automatically.
//
// Usage:
//
//	cache := NewLRUCache[string, *Module](512)
//	cache.Put("modA", mod)
//	if v, ok := cache.Get("modA"); ok { ... }
type LRUCache[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	items    map[K]*list.Element
	order    *list.List // front = most-recently used
}

type lruEntry[K comparable, V any] struct {
	key   K
	value V
}

// NewLRUCache creates a new cache with the given capacity.
// Capacity must be >= 1; values <= 0 are normalised to 1.
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element, capacity),
		order:    list.New(),
	}
}

// Get returns the cached value and true if the key exists, else the zero value
// and false. A hit moves the entry to the front (most-recently used).
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	c.order.MoveToFront(el)
	return el.Value.(*lruEntry[K, V]).value, true
}

// Put inserts or updates a key/value pair. If the cache is at capacity the
// least-recently-used entry is evicted first.
func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		// Update existing entry — move to front.
		c.order.MoveToFront(el)
		el.Value.(*lruEntry[K, V]).value = value
		return
	}

	// Evict LRU entry when at capacity.
	if c.order.Len() >= c.capacity {
		c.evictLeastRecentLocked()
	}

	entry := &lruEntry[K, V]{key: key, value: value}
	el := c.order.PushFront(entry)
	c.items[key] = el
}

// Evict removes a specific key from the cache. It is a no-op if the key does
// not exist.
func (c *LRUCache[K, V]) Evict(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return
	}
	c.order.Remove(el)
	delete(c.items, key)
}

// Peek returns the cached value without moving it to the front.
func (c *LRUCache[K, V]) Peek(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	return el.Value.(*lruEntry[K, V]).value, true
}

// Keys returns all keys currently in the cache.
func (c *LRUCache[K, V]) Keys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := make([]K, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the current number of items in the cache.
func (c *LRUCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// Cap returns the configured maximum capacity.
func (c *LRUCache[K, V]) Cap() int {
	return c.capacity
}

// Clear removes all items from the cache.
func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.order.Init()
	c.items = make(map[K]*list.Element, c.capacity)
}

// evictLeastRecentLocked removes the back (least-recently-used) element.
// Caller must hold c.mu.
func (c *LRUCache[K, V]) evictLeastRecentLocked() {
	back := c.order.Back()
	if back == nil {
		return
	}
	c.order.Remove(back)
	delete(c.items, back.Value.(*lruEntry[K, V]).key)
}
