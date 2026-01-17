package ratelimit

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SourceConfigs represents a map of source name to limiter config.
type SourceConfigs struct {
	RateLimits map[string]Config `yaml:"rate_limits" json:"rate_limits"`
}

// LoadSourceConfigs loads YAML bytes into SourceConfigs.
func LoadSourceConfigs(data []byte) (SourceConfigs, error) {
	var cfgs SourceConfigs
	if err := yaml.Unmarshal(data, &cfgs); err != nil {
		return SourceConfigs{}, err
	}
	// Apply defaults per entry
	for name, cfg := range cfgs.RateLimits {
		cfgs.RateLimits[name] = applyDefaults(cfg)
	}
	return cfgs, nil
}

// Get returns limiter config for a source or default if missing.
func (s SourceConfigs) Get(source string) (Config, error) {
	if s.RateLimits == nil {
		return DefaultConfig(), fmt.Errorf("no rate_limits configured")
	}
	cfg, ok := s.RateLimits[source]
	if !ok {
		return DefaultConfig(), fmt.Errorf("rate_limits for %s not found", source)
	}
	return applyDefaults(cfg), nil
}
