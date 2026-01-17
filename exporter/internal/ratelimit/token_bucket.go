package ratelimit

import (
	"context"
	"sync"
	"time"
)

// TokenBucket implements token bucket rate limiting.
type TokenBucket struct {
	rate       float64
	burst      int
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
	config     Config
}

// NewTokenBucket creates a new token bucket limiter.
func NewTokenBucket(cfg Config) *TokenBucket {
	cfg = applyDefaults(cfg)

	return &TokenBucket{
		rate:       cfg.RequestsPerSec,
		burst:      cfg.Burst,
		tokens:     float64(cfg.Burst),
		lastUpdate: time.Now(),
		config:     cfg,
	}
}

// Wait blocks until a token is available or context is canceled.
func (tb *TokenBucket) Wait(ctx context.Context) error {
	tb.mu.Lock()
	tb.refill()

	if tb.tokens >= 1.0 {
		tb.tokens--
		tb.mu.Unlock()
		return nil
	}

	deficit := 1.0 - tb.tokens
	wait := time.Duration(deficit/tb.rate*float64(time.Second)) + time.Nanosecond
	tb.mu.Unlock()

	timer := time.NewTimer(wait)
	select {
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
		tb.mu.Lock()
		tb.refill()
		if tb.tokens >= 1.0 {
			tb.tokens--
		}
		tb.mu.Unlock()
		return nil
	}
}

// Allow returns true if a token is available immediately.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1.0 {
		tb.tokens--
		return true
	}
	return false
}

// Reserve returns the duration to wait for the next token.
func (tb *TokenBucket) Reserve() time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1.0 {
		return 0
	}

	deficit := 1.0 - tb.tokens
	wait := time.Duration(deficit / tb.rate * float64(time.Second))
	return wait
}

// RetryAfter returns exponential backoff duration.
func (tb *TokenBucket) RetryAfter(attempt int) time.Duration {
	return CalculateBackoff(attempt, tb.config)
}

// Reset resets the bucket to full capacity.
func (tb *TokenBucket) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.tokens = float64(tb.burst)
	tb.lastUpdate = time.Now()
}

// refill adds tokens based on elapsed time (call with lock held).
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate)
	if elapsed <= 0 {
		return
	}

	tb.tokens += elapsed.Seconds() * tb.rate
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}
	tb.lastUpdate = now
}
