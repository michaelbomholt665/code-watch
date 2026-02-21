package util

import (
	"sync"
	"time"
)

// LimiterRegistry manages a collection of limiters, typically one per client IP.
type LimiterRegistry struct {
	mu       sync.RWMutex
	limiters map[string]*limiterEntry
	rate     float64
	burst    int
	ttl      time.Duration
}

type limiterEntry struct {
	limiter  *Limiter
	lastUsed time.Time
}

// NewLimiterRegistry creates a new registry.
// rate: tokens per second.
// burst: burst size.
// ttl: how long to keep a limiter in memory after its last use.
func NewLimiterRegistry(r float64, b int, ttl time.Duration) *LimiterRegistry {
	reg := &LimiterRegistry{
		limiters: make(map[string]*limiterEntry),
		rate:     r,
		burst:    b,
		ttl:      ttl,
	}
	go reg.cleanupLoop()
	return reg
}

// Get returns the limiter for the given key (e.g., client IP).
func (r *LimiterRegistry) Get(key string) *Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.limiters[key]
	if !ok {
		entry = &limiterEntry{
			limiter: NewLimiter(r.rate, r.burst),
		}
		r.limiters[key] = entry
	}
	entry.lastUsed = time.Now()
	return entry.limiter
}

func (r *LimiterRegistry) cleanupLoop() {
	ticker := time.NewTicker(r.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		r.cleanup()
	}
}

func (r *LimiterRegistry) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for key, entry := range r.limiters {
		if now.Sub(entry.lastUsed) > r.ttl {
			delete(r.limiters, key)
		}
	}
}
