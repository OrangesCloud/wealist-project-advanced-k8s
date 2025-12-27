package config

import (
	"os"
	"strconv"

	commonconfig "github.com/OrangesCloud/wealist-advanced-go-pkg/config"
	"gopkg.in/yaml.v3"
)

// Config contains all configuration for noti-service.
type Config struct {
	commonconfig.BaseConfig `yaml:",inline"`
	InternalAuth            InternalAuthConfig `yaml:"internal_auth"`
	App                     AppConfig          `yaml:"app"`
	RateLimit               RateLimitConfig    `yaml:"rate_limit"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	BurstSize         int  `yaml:"burst_size"`
}

// InternalAuthConfig contains noti-service specific auth fields.
type InternalAuthConfig struct {
	InternalAPIKey string `yaml:"internal_api_key"`
}

// AppConfig contains notification-specific configuration.
type AppConfig struct {
	CacheUnreadTTL int `yaml:"cache_unread_ttl"` // seconds
	CleanupDays    int `yaml:"cleanup_days"`
}

// Load reads configuration from yaml file and environment variables.
func Load(path string) (*Config, error) {
	// Start with defaults
	base := commonconfig.DefaultBaseConfig()
	base.Server.Port = 8002

	cfg := &Config{
		BaseConfig: base,
		App: AppConfig{
			CacheUnreadTTL: 300, // 5 minutes
			CleanupDays:    30,
		},
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
	if apiKey := os.Getenv("INTERNAL_API_KEY"); apiKey != "" {
		cfg.InternalAuth.InternalAPIKey = apiKey
	}

	// Rate Limit
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
