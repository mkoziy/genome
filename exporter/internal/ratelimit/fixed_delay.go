package ratelimit

import (
	"context"
	"sync"
	"time"
)

// FixedDelayLimiter enforces a fixed delay between requests.
type FixedDelayLimiter struct {
	delay       time.Duration
	lastRequest time.Time
	mu          sync.Mutex
	config      Config
}

// NewFixedDelayLimiter creates a new fixed delay limiter.
func NewFixedDelayLimiter(cfg Config) *FixedDelayLimiter {
	cfg = applyDefaults(cfg)

	return &FixedDelayLimiter{
		delay:  cfg.FixedDelay,
		config: cfg,
	}
}

// Wait blocks for the fixed delay.
func (fdl *FixedDelayLimiter) Wait(ctx context.Context) error {
	fdl.mu.Lock()
	wait, now := fdl.reserve(time.Now())
	fdl.lastRequest = now.Add(wait)
	fdl.mu.Unlock()

	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	select {
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Allow returns true if no wait is needed.
func (fdl *FixedDelayLimiter) Allow() bool {
	fdl.mu.Lock()
	defer fdl.mu.Unlock()

	wait, now := fdl.reserve(time.Now())
	if wait > 0 {
		return false
	}

	fdl.lastRequest = now
	return true
}

// Reserve returns time to wait.
func (fdl *FixedDelayLimiter) Reserve() time.Duration {
	fdl.mu.Lock()
	defer fdl.mu.Unlock()

	wait, _ := fdl.reserve(time.Now())
	return wait
}

func (fdl *FixedDelayLimiter) reserve(now time.Time) (time.Duration, time.Time) {
	if fdl.lastRequest.IsZero() {
		return 0, now
	}

	elapsed := now.Sub(fdl.lastRequest)
	if elapsed >= fdl.delay {
		return 0, now
	}

	return fdl.delay - elapsed, now
}

// RetryAfter returns exponential backoff duration.
func (fdl *FixedDelayLimiter) RetryAfter(attempt int) time.Duration {
	return CalculateBackoff(attempt, fdl.config)
}

// Reset resets the last request time.
func (fdl *FixedDelayLimiter) Reset() {
	fdl.mu.Lock()
	defer fdl.mu.Unlock()
	fdl.lastRequest = time.Time{}
}
