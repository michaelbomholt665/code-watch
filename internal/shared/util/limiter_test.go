package util

import (
	"context"
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	// 10 tokens per second, burst of 2
	l := NewLimiter(10, 2)

	if !l.Allow(1) {
		t.Error("expected first token to be allowed")
	}
	if !l.Allow(1) {
		t.Error("expected second token to be allowed (burst)")
	}
	if l.Allow(1) {
		t.Error("expected third token to be rejected (burst exhausted)")
	}

	time.Sleep(150 * time.Millisecond)
	if !l.Allow(1) {
		t.Error("expected token to be refilled after wait")
	}
}

func TestLimiterRegistry(t *testing.T) {
	// 100 tokens/sec, burst 10, ttl 100ms
	reg := NewLimiterRegistry(100, 10, 100*time.Millisecond)

	l1 := reg.Get("1.1.1.1")
	l2 := reg.Get("2.2.2.2")

	if l1 == l2 {
		t.Error("expected different limiters for different IPs")
	}

	if reg.Get("1.1.1.1") != l1 {
		t.Error("expected same limiter for same IP")
	}

	time.Sleep(250 * time.Millisecond)
	// Cleanup should have removed the old limiters
	l1_new := reg.Get("1.1.1.1")
	if l1_new == l1 {
		t.Error("expected old limiter to be cleaned up and replaced")
	}
}

func TestLimiter_Wait(t *testing.T) {
	l := NewLimiter(100, 1)
	l.Allow(1) // consume burst

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := l.Wait(ctx, 1)
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if time.Since(start) < 10*time.Millisecond {
		t.Error("Wait returned too early")
	}
}
