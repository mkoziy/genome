package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucketAllowAndRefill(t *testing.T) {
	cfg := Config{RequestsPerSec: 5, Burst: 5}
	tb := NewTokenBucket(cfg)

	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Fatalf("expected token available at %d", i)
		}
	}
	if tb.Allow() {
		t.Fatalf("expected no token after burst")
	}

	time.Sleep(250 * time.Millisecond)
	if !tb.Allow() {
		t.Fatalf("expected token after partial refill")
	}
}

func TestTokenBucketWaitRespectsContext(t *testing.T) {
	cfg := Config{RequestsPerSec: 1, Burst: 1}
	tb := NewTokenBucket(cfg)

	// consume initial token
	if !tb.Allow() {
		t.Fatalf("expected first token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := tb.Wait(ctx); err == nil {
		t.Fatalf("expected timeout")
	}
}

func TestFixedWindow(t *testing.T) {
	fw := NewFixedWindow(Config{RequestsPerSec: 2})
	if !fw.Allow() || !fw.Allow() {
		t.Fatalf("expected first two to pass")
	}
	if fw.Allow() {
		t.Fatalf("expected third to be blocked")
	}

	time.Sleep(time.Second)
	if !fw.Allow() {
		t.Fatalf("expected allow after window reset")
	}
}

func TestFixedDelay(t *testing.T) {
	delay := 50 * time.Millisecond
	fdl := NewFixedDelayLimiter(Config{FixedDelay: delay})

	if !fdl.Allow() {
		t.Fatalf("expected first allow")
	}

	wait := fdl.Reserve()
	if wait <= 0 {
		t.Fatalf("expected reserve to request wait, got %v", wait)
	}

	if wait < delay/2 {
		t.Fatalf("expected wait close to delay; got %v", wait)
	}
}

func TestCalculateBackoffBounds(t *testing.T) {
	cfg := Config{InitialBackoff: time.Second, MaxBackoff: 10 * time.Second, BackoffMultiplier: 2, MaxRetries: 5}

	for attempt := 1; attempt <= 5; attempt++ {
		d := CalculateBackoff(attempt, cfg)
		if d <= 0 {
			t.Fatalf("backoff should be positive")
		}
		if d > cfg.MaxBackoff {
			t.Fatalf("backoff should cap at max")
		}
	}

	if d := CalculateBackoff(10, cfg); d != cfg.MaxBackoff {
		t.Fatalf("expected max backoff when attempts exceed max retries")
	}
}

func TestConfigLoader(t *testing.T) {
	yamlData := []byte(`rate_limits:
  clinvar:
    strategy: token_bucket
    requests_per_second: 3
    burst: 5
    max_retries: 5
    initial_backoff: 1s
    max_backoff: 60s
    backoff_multiplier: 2
`)

	cfgs, err := LoadSourceConfigs(yamlData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clinvar, err := cfgs.Get("clinvar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if clinvar.RequestsPerSec != 3 {
		t.Fatalf("expected requests_per_second=3, got %v", clinvar.RequestsPerSec)
	}
}
