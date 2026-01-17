package ratelimit

import (
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// CalculateBackoff computes exponential backoff with +/-25% jitter.
func CalculateBackoff(attempt int, cfg Config) time.Duration {
	if attempt <= 0 {
		return 0
	}
	if attempt > cfg.MaxRetries {
		return cfg.MaxBackoff
	}

	base := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt-1))
	if base > float64(cfg.MaxBackoff) {
		base = float64(cfg.MaxBackoff)
	}

	jitter := base * 0.25 * (2*rand.Float64() - 1) // +/-25%
	backoff := base + jitter

	if backoff < 0 {
		backoff = 0
	}
	if backoff > float64(cfg.MaxBackoff) {
		backoff = float64(cfg.MaxBackoff)
	}

	return time.Duration(backoff)
}

// ShouldRetry returns true if attempt is within allowed retries.
func ShouldRetry(attempt int, maxRetries int) bool {
	return attempt <= maxRetries
}
