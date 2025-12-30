package config

import (
	"os"
	"strconv"

	commonconfig "github.com/OrangesCloud/wealist-advanced-go-pkg/config"
	"gopkg.in/yaml.v3"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	BurstSize         int  `yaml:"burst_size"`
}

// Config contains all configuration for chat-service.
type Config struct {
	commonconfig.BaseConfig `yaml:",inline"`
	Services                ServicesConfig  `yaml:"services"`
	RateLimit               RateLimitConfig `yaml:"rate_limit"`
}

// ServicesConfig contains service URLs configuration.
type ServicesConfig struct {
	UserServiceURL string `yaml:"user_service_url"`
}

// Load reads configuration from yaml file and environment variables.
func Load(path string) (*Config, error) {
	// Start with defaults
	base := commonconfig.DefaultBaseConfig()
	base.Server.Port = 8001
	base.Server.BasePath = "/api/chats"

	cfg := &Config{
		BaseConfig: base,
	}

	// Load from yaml file if exists
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	// Override with environment variables (common config)
	cfg.LoadFromEnv()

	// Service-specific environment variables
	if userURL := os.Getenv("USER_SERVICE_URL"); userURL != "" {
		cfg.Services.UserServiceURL = userURL
	}

	// Rate Limit environment variables
	if rateLimitEnabled := os.Getenv("RATE_LIMIT_ENABLED"); rateLimitEnabled != "" {
		cfg.RateLimit.Enabled = rateLimitEnabled == "true"
	}
	if rpm := os.Getenv("RATE_LIMIT_PER_MINUTE"); rpm != "" {
		if v, err := strconv.Atoi(rpm); err == nil {
			cfg.RateLimit.RequestsPerMinute = v
		}
	}
	if burst := os.Getenv("RATE_LIMIT_BURST"); burst != "" {
		if v, err := strconv.Atoi(burst); err == nil {
			cfg.RateLimit.BurstSize = v
		}
	}
	if cfg.RateLimit.RequestsPerMinute == 0 {
		cfg.RateLimit.RequestsPerMinute = 60
	}

	return cfg, nil
}
