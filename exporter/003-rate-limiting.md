# 003 - Rate Limiting System

## Feature Overview
Implement a robust rate limiting system to respect API quotas and avoid overwhelming external data sources. The system should support multiple rate limiting strategies and be configurable per data source.

## Goals
- Respect API rate limits for all sources
- Configurable limits per source
- Token bucket algorithm for smooth traffic
- Exponential backoff on errors
- Handle HTTP 429 responses gracefully
- Concurrent-safe operations

## Rate Limit Requirements by Source

### ClinVar (NCBI E-utilities)
- **Rate**: 3 requests/second without API key
- **Rate**: 10 requests/second with API key
- **Strategy**: Token bucket
- **Retry**: Exponential backoff on 429

### dbSNP (NCBI)
- **Rate**: 3 requests/second without API key
- **Rate**: 10 requests/second with API key
- **Strategy**: Token bucket
- **Retry**: Exponential backoff

### OpenSNP
- **Rate**: Unknown, be conservative
- **Strategy**: 1 request/second
- **Retry**: Exponential backoff

### PharmGKB
- **Rate**: Depends on API tier
- **Strategy**: Configurable
- **Retry**: As specified by API

### SNPedia (Web Scraping)
- **Rate**: 1 request every 5 seconds (polite scraping)
- **Strategy**: Fixed delay
- **Retry**: Long backoff on errors

## Package Structure

```
internal/ratelimit/
├── limiter.go          # Main rate limiter interface
├── token_bucket.go     # Token bucket implementation
├── fixed_window.go     # Fixed window implementation
├── exponential.go      # Exponential backoff
├── config.go           # Configuration structs
└── limiter_test.go     # Tests
```

## Implementation

### limiter.go - Interface and Factory

```go
package ratelimit

import (
    "context"
    "time"
)

// Limiter defines the rate limiting interface
type Limiter interface {
    // Wait blocks until the limiter allows an action to proceed
    Wait(ctx context.Context) error
    
    // Allow returns true if an action can proceed now
    Allow() bool
    
    // Reserve reserves a slot and returns time to wait
    Reserve() time.Duration
    
    // RetryAfter returns the duration to wait before retrying after error
    RetryAfter(attempt int) time.Duration
    
    // Reset resets the limiter state
    Reset()
}

// Config holds rate limiter configuration
type Config struct {
    Strategy        Strategy      `yaml:"strategy" json:"strategy"`
    RequestsPerSec  float64       `yaml:"requests_per_second" json:"requests_per_second"`
    Burst           int           `yaml:"burst" json:"burst"`
    FixedDelay      time.Duration `yaml:"fixed_delay" json:"fixed_delay"`
    MaxRetries      int           `yaml:"max_retries" json:"max_retries"`
    InitialBackoff  time.Duration `yaml:"initial_backoff" json:"initial_backoff"`
    MaxBackoff      time.Duration `yaml:"max_backoff" json:"max_backoff"`
    BackoffMultiplier float64     `yaml:"backoff_multiplier" json:"backoff_multiplier"`
}

// Strategy defines the rate limiting strategy
type Strategy string

const (
    TokenBucket  Strategy = "token_bucket"
    FixedWindow  Strategy = "fixed_window"
    FixedDelay   Strategy = "fixed_delay"
)

// NewLimiter creates a new rate limiter based on config
func NewLimiter(cfg Config) Limiter {
    switch cfg.Strategy {
    case TokenBucket:
        return NewTokenBucket(cfg)
    case FixedWindow:
        return NewFixedWindow(cfg)
    case FixedDelay:
        return NewFixedDelayLimiter(cfg)
    default:
        // Default to token bucket
        return NewTokenBucket(cfg)
    }
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
    return Config{
        Strategy:          TokenBucket,
        RequestsPerSec:    3.0,
        Burst:             5,
        MaxRetries:        5,
        InitialBackoff:    1 * time.Second,
        MaxBackoff:        60 * time.Second,
        BackoffMultiplier: 2.0,
    }
}
```

### token_bucket.go - Token Bucket Algorithm

```go
package ratelimit

import (
    "context"
    "sync"
    "time"
)

// TokenBucket implements token bucket rate limiting
type TokenBucket struct {
    rate       float64       // tokens per second
    burst      int           // maximum burst size
    tokens     float64       // current tokens
    lastUpdate time.Time     // last token addition
    mu         sync.Mutex    // protects tokens and lastUpdate
    config     Config
}

// NewTokenBucket creates a new token bucket limiter
func NewTokenBucket(cfg Config) *TokenBucket {
    if cfg.RequestsPerSec <= 0 {
        cfg.RequestsPerSec = 1.0
    }
    if cfg.Burst <= 0 {
        cfg.Burst = int(cfg.RequestsPerSec)
    }
    
    return &TokenBucket{
        rate:       cfg.RequestsPerSec,
        burst:      cfg.Burst,
        tokens:     float64(cfg.Burst),
        lastUpdate: time.Now(),
        config:     cfg,
    }
}

// Wait blocks until a token is available
func (tb *TokenBucket) Wait(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        if tb.Allow() {
            return nil
        }
        
        // Calculate wait time
        waitTime := tb.Reserve()
        if waitTime <= 0 {
            return nil
        }
        
        timer := time.NewTimer(waitTime)
        select {
        case <-ctx.Done():
            timer.Stop()
            return ctx.Err()
        case <-timer.C:
            // Continue loop to check again
        }
    }
}

// Allow returns true if a token is available
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

// Reserve reserves a token and returns wait time
func (tb *TokenBucket) Reserve() time.Duration {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    tb.refill()
    
    if tb.tokens >= 1.0 {
        tb.tokens--
        return 0
    }
    
    // Calculate time until next token
    deficit := 1.0 - tb.tokens
    waitTime := time.Duration(deficit/tb.rate*float64(time.Second))
    return waitTime
}

// refill adds tokens based on elapsed time (must hold lock)
func (tb *TokenBucket) refill() {
    now := time.Now()
    elapsed := now.Sub(tb.lastUpdate)
    
    // Add tokens based on elapsed time
    tokensToAdd := elapsed.Seconds() * tb.rate
    tb.tokens += tokensToAdd
    
    // Cap at burst size
    if tb.tokens > float64(tb.burst) {
        tb.tokens = float64(tb.burst)
    }
    
    tb.lastUpdate = now
}

// RetryAfter returns exponential backoff duration
func (tb *TokenBucket) RetryAfter(attempt int) time.Duration {
    return CalculateBackoff(attempt, tb.config)
}

// Reset resets the token bucket to full capacity
func (tb *TokenBucket) Reset() {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    tb.tokens = float64(tb.burst)
    tb.lastUpdate = time.Now()
}
```

### fixed_window.go - Fixed Window Algorithm

```go
package ratelimit

import (
    "context"
    "sync"
    "time"
)

// FixedWindow implements fixed window rate limiting
type FixedWindow struct {
    limit      int           // max requests per window
    window     time.Duration // window duration
    count      int           // current count
    windowStart time.Time    // start of current window
    mu         sync.Mutex
    config     Config
}

// NewFixedWindow creates a new fixed window limiter
func NewFixedWindow(cfg Config) *FixedWindow {
    if cfg.RequestsPerSec <= 0 {
        cfg.RequestsPerSec = 1.0
    }
    
    window := 1 * time.Second
    limit := int(cfg.RequestsPerSec)
    
    return &FixedWindow{
        limit:      limit,
        window:     window,
        count:      0,
        windowStart: time.Now(),
        config:     cfg,
    }
}

// Wait blocks until request can proceed
func (fw *FixedWindow) Wait(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        if fw.Allow() {
            return nil
        }
        
        waitTime := fw.Reserve()
        if waitTime <= 0 {
            continue
        }
        
        timer := time.NewTimer(waitTime)
        select {
        case <-ctx.Done():
            timer.Stop()
            return ctx.Err()
        case <-timer.C:
            // Continue
        }
    }
}

// Allow returns true if request can proceed
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

// Reserve returns wait time until next available slot
func (fw *FixedWindow) Reserve() time.Duration {
    fw.mu.Lock()
    defer fw.mu.Unlock()
    
    fw.resetWindowIfNeeded()
    
    if fw.count < fw.limit {
        fw.count++
        return 0
    }
    
    // Wait until next window
    elapsed := time.Since(fw.windowStart)
    return fw.window - elapsed
}

// resetWindowIfNeeded resets window if expired (must hold lock)
func (fw *FixedWindow) resetWindowIfNeeded() {
    now := time.Now()
    if now.Sub(fw.windowStart) >= fw.window {
        fw.count = 0
        fw.windowStart = now
    }
}

// RetryAfter returns exponential backoff duration
func (fw *FixedWindow) RetryAfter(attempt int) time.Duration {
    return CalculateBackoff(attempt, fw.config)
}

// Reset resets the window
func (fw *FixedWindow) Reset() {
    fw.mu.Lock()
    defer fw.mu.Unlock()
    
    fw.count = 0
    fw.windowStart = time.Now()
}
```

### Fixed Delay Limiter

```go
package ratelimit

import (
    "context"
    "sync"
    "time"
)

// FixedDelayLimiter enforces a fixed delay between requests
type FixedDelayLimiter struct {
    delay      time.Duration
    lastRequest time.Time
    mu         sync.Mutex
    config     Config
}

// NewFixedDelayLimiter creates a new fixed delay limiter
func NewFixedDelayLimiter(cfg Config) *FixedDelayLimiter {
    if cfg.FixedDelay <= 0 {
        cfg.FixedDelay = 1 * time.Second
    }
    
    return &FixedDelayLimiter{
        delay:  cfg.FixedDelay,
        config: cfg,
    }
}

// Wait blocks for the fixed delay
func (fdl *FixedDelayLimiter) Wait(ctx context.Context) error {
    waitTime := fdl.Reserve()
    if waitTime <= 0 {
        return nil
    }
    
    timer := time.NewTimer(waitTime)
    select {
    case <-ctx.Done():
        timer.Stop()
        return ctx.Err()
    case <-timer.C:
        return nil
    }
}

// Allow always returns true but enforces delay via Reserve
func (fdl *FixedDelayLimiter) Allow() bool {
    return fdl.Reserve() <= 0
}

// Reserve returns time to wait
func (fdl *FixedDelayLimiter) Reserve() time.Duration {
    fdl.mu.Lock()
    defer fdl.mu.Unlock()
    
    if fdl.lastRequest.IsZero() {
        fdl.lastRequest = time.Now()
        return 0
    }
    
    elapsed := time.Since(fdl.lastRequest)
    if elapsed >= fdl.delay {
        fdl.lastRequest = time.Now()
        return 0
    }
    
    waitTime := fdl.delay - elapsed
    fdl.lastRequest = time.Now().Add(waitTime)
    return waitTime
}

// RetryAfter returns exponential backoff duration
func (fdl *FixedDelayLimiter) RetryAfter(attempt int) time.Duration {
    return CalculateBackoff(attempt, fdl.config)
}

// Reset resets the last request time
func (fdl *FixedDelayLimiter) Reset() {
    fdl.mu.Lock()
    defer fdl.mu.Unlock()
    fdl.lastRequest = time.Time{}
}
```

### exponential.go - Exponential Backoff

```go
package ratelimit

import (
    "math"
    "math/rand"
    "time"
)

// CalculateBackoff calculates exponential backoff with jitter
func CalculateBackoff(attempt int, cfg Config) time.Duration {
    if attempt <= 0 {
        return 0
    }
    
    if attempt > cfg.MaxRetries {
        return cfg.MaxBackoff
    }
    
    // Exponential backoff: initial * multiplier^attempt
    backoff := float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt-1))
    
    // Cap at max backoff
    if backoff > float64(cfg.MaxBackoff) {
        backoff = float64(cfg.MaxBackoff)
    }
    
    // Add jitter (±25%)
    jitter := backoff * 0.25 * (2*rand.Float64() - 1)
    backoff += jitter
    
    return time.Duration(backoff)
}

// ShouldRetry returns true if should retry based on attempt count
func ShouldRetry(attempt int, maxRetries int) bool {
    return attempt <= maxRetries
}
```

## Configuration Per Source

### config/ratelimits.yaml

```yaml
rate_limits:
  clinvar:
    strategy: token_bucket
    requests_per_second: 3.0
    burst: 5
    max_retries: 5
    initial_backoff: 1s
    max_backoff: 60s
    backoff_multiplier: 2.0
  
  dbsnp:
    strategy: token_bucket
    requests_per_second: 3.0
    burst: 5
    max_retries: 5
    initial_backoff: 1s
    max_backoff: 60s
    backoff_multiplier: 2.0
  
  opensnp:
    strategy: token_bucket
    requests_per_second: 1.0
    burst: 2
    max_retries: 3
    initial_backoff: 2s
    max_backoff: 120s
    backoff_multiplier: 3.0
  
  pharmgkb:
    strategy: token_bucket
    requests_per_second: 2.0
    burst: 3
    max_retries: 5
    initial_backoff: 1s
    max_backoff: 60s
    backoff_multiplier: 2.0
  
  snpedia:
    strategy: fixed_delay
    fixed_delay: 5s
    max_retries: 3
    initial_backoff: 10s
    max_backoff: 300s
    backoff_multiplier: 3.0
```

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "snp-downloader/internal/ratelimit"
)

func main() {
    // Create limiter for ClinVar
    cfg := ratelimit.Config{
        Strategy:          ratelimit.TokenBucket,
        RequestsPerSec:    3.0,
        Burst:             5,
        MaxRetries:        5,
        InitialBackoff:    1 * time.Second,
        MaxBackoff:        60 * time.Second,
        BackoffMultiplier: 2.0,
    }
    
    limiter := ratelimit.NewLimiter(cfg)
    
    ctx := context.Background()
    
    // Make requests
    for i := 0; i < 10; i++ {
        if err := limiter.Wait(ctx); err != nil {
            fmt.Printf("Error waiting: %v\n", err)
            return
        }
        
        // Make API request here
        fmt.Printf("Request %d at %v\n", i, time.Now())
    }
}
```

## HTTP Client Integration

```go
package client

import (
    "context"
    "net/http"
    "time"
    
    "snp-downloader/internal/ratelimit"
)

// RateLimitedClient wraps http.Client with rate limiting
type RateLimitedClient struct {
    client  *http.Client
    limiter ratelimit.Limiter
}

// NewRateLimitedClient creates a rate-limited HTTP client
func NewRateLimitedClient(limiter ratelimit.Limiter) *RateLimitedClient {
    return &RateLimitedClient{
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
        limiter: limiter,
    }
}

// Do performs an HTTP request with rate limiting
func (c *RateLimitedClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
    // Wait for rate limiter
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, err
    }
    
    // Perform request
    resp, err := c.client.Do(req.WithContext(ctx))
    if err != nil {
        return nil, err
    }
    
    // Handle 429 Too Many Requests
    if resp.StatusCode == http.StatusTooManyRequests {
        resp.Body.Close()
        
        // Use exponential backoff
        backoff := c.limiter.RetryAfter(1)
        time.Sleep(backoff)
        
        // Retry once
        return c.client.Do(req.WithContext(ctx))
    }
    
    return resp, nil
}
```

## Testing

```go
func TestTokenBucket(t *testing.T) {
    cfg := ratelimit.Config{
        RequestsPerSec: 10.0,
        Burst: 10,
    }
    
    limiter := ratelimit.NewTokenBucket(cfg)
    
    // Should allow burst
    for i := 0; i < 10; i++ {
        if !limiter.Allow() {
            t.Errorf("Expected allow on request %d", i)
        }
    }
    
    // Should block after burst
    if limiter.Allow() {
        t.Error("Expected block after burst")
    }
    
    // Should allow after refill
    time.Sleep(200 * time.Millisecond)
    if !limiter.Allow() {
        t.Error("Expected allow after refill")
    }
}
```

## Implementation Tasks

1. Implement base limiter interface
2. Implement token bucket algorithm
3. Implement fixed window algorithm
4. Implement fixed delay limiter
5. Implement exponential backoff
6. Create configuration loading
7. Write comprehensive tests
8. Add benchmarks
9. Document usage patterns

## Success Criteria
- All algorithms work correctly
- Thread-safe operations
- Configurable per source
- Handles 429 responses
- Exponential backoff works
- Tests cover edge cases
- Performance acceptable (<1ms overhead per request)

## Next Feature
After completing this, proceed to **004 - ClinVar Data Source**.
