package ratelimit

import "time"

// Config holds rate limiter configuration.
type Config struct {
	Strategy          Strategy      `yaml:"strategy" json:"strategy"`
	RequestsPerSec    float64       `yaml:"requests_per_second" json:"requests_per_second"`
	Burst             int           `yaml:"burst" json:"burst"`
	FixedDelay        time.Duration `yaml:"fixed_delay" json:"fixed_delay"`
	MaxRetries        int           `yaml:"max_retries" json:"max_retries"`
	InitialBackoff    time.Duration `yaml:"initial_backoff" json:"initial_backoff"`
	MaxBackoff        time.Duration `yaml:"max_backoff" json:"max_backoff"`
	BackoffMultiplier float64       `yaml:"backoff_multiplier" json:"backoff_multiplier"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Strategy:          StrategyTokenBucket,
		RequestsPerSec:    3.0,
		Burst:             5,
		FixedDelay:        1 * time.Second,
		MaxRetries:        5,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        60 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

func applyDefaults(cfg Config) Config {
	def := DefaultConfig()
	if cfg.Strategy == "" {
		cfg.Strategy = def.Strategy
	}
	if cfg.RequestsPerSec <= 0 {
		cfg.RequestsPerSec = def.RequestsPerSec
	}
	if cfg.Burst <= 0 {
		cfg.Burst = def.Burst
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = def.MaxRetries
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = def.InitialBackoff
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = def.MaxBackoff
	}
	if cfg.BackoffMultiplier <= 0 {
		cfg.BackoffMultiplier = def.BackoffMultiplier
	}
	if cfg.FixedDelay <= 0 {
		cfg.FixedDelay = def.FixedDelay
	}
	return cfg
}
