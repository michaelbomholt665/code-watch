package util

import (
	"context"
	"golang.org/x/time/rate"
	"time"
)

// Limiter wraps rate.Limiter to provide a simpler interface.
type Limiter struct {
	inner *rate.Limiter
}

// NewLimiter creates a new token bucket limiter.
// r: tokens per second.
// b: burst size.
func NewLimiter(r float64, b int) *Limiter {
	return &Limiter{
		inner: rate.NewLimiter(rate.Limit(r), b),
	}
}

// Allow reports whether an event with weight n may happen at time now.
func (l *Limiter) Allow(n int) bool {
	return l.inner.AllowN(time.Now(), n)
}

// Wait blocks until n tokens are available.
func (l *Limiter) Wait(ctx context.Context, n int) error {
	return l.inner.WaitN(ctx, n)
}
