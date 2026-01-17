package ratelimit

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

// FixedWindow implements fixed window rate limiting.
type FixedWindow struct {
	limit       int
	window      time.Duration
	count       int
	windowStart time.Time
	mu          sync.Mutex
	config      Config
}

// NewFixedWindow creates a new fixed window limiter.
func NewFixedWindow(cfg Config) *FixedWindow {
	cfg = applyDefaults(cfg)

	return &FixedWindow{
		limit:       int(cfg.RequestsPerSec),
		window:      time.Second,
		windowStart: time.Now(),
		config:      cfg,
	}
}

// Wait blocks until request can proceed.
func (fw *FixedWindow) Wait(ctx context.Context) error {
	for {
		if fw.Allow() {
			return nil
		}

		wait := fw.Reserve()
		if wait <= 0 {
			continue
		}

		// Add jitter to avoid thundering herd
		jitter := time.Duration(rand.Int63n(int64(wait) / 4))
		time.Sleep(wait + jitter)
	}
}

// Allow returns true if request can proceed.
func (fw *FixedWindow) Allow() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.resetWindowIfNeeded()

	if fw.count < fw.limit {
		fw.count++
		return true
	}

	return false
}

// Reserve returns wait time until next available slot.
func (fw *FixedWindow) Reserve() time.Duration {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.resetWindowIfNeeded()

	if fw.count < fw.limit {
		return 0
	}

	elapsed := time.Since(fw.windowStart)
	return fw.window - elapsed
}

// RetryAfter returns exponential backoff duration.
func (fw *FixedWindow) RetryAfter(attempt int) time.Duration {
	return CalculateBackoff(attempt, fw.config)
}

// Reset resets the window.
func (fw *FixedWindow) Reset() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.count = 0
	fw.windowStart = time.Now()
}

func (fw *FixedWindow) resetWindowIfNeeded() {
	now := time.Now()
	if now.Sub(fw.windowStart) >= fw.window {
		fw.count = 0
		fw.windowStart = now
	}
}
