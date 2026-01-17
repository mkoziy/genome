package ratelimit

import (
	"context"
	"time"
)

// Limiter defines the rate limiting interface.
type Limiter interface {
	Wait(ctx context.Context) error
	Allow() bool
	Reserve() time.Duration
	RetryAfter(attempt int) time.Duration
	Reset()
}

// Strategy defines the rate limiting strategy.
type Strategy string

const (
	StrategyTokenBucket Strategy = "token_bucket"
	StrategyFixedWindow Strategy = "fixed_window"
	StrategyFixedDelay  Strategy = "fixed_delay"
)

// NewLimiter creates a rate limiter based on config.
func NewLimiter(cfg Config) Limiter {
	cfg = applyDefaults(cfg)
	switch cfg.Strategy {
	case StrategyFixedWindow:
		return NewFixedWindow(cfg)
	case StrategyFixedDelay:
		return NewFixedDelayLimiter(cfg)
	default:
		return NewTokenBucket(cfg)
	}
}
