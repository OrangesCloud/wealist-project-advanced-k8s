package config

import (
	"os"

	commonconfig "github.com/OrangesCloud/wealist-advanced-go-pkg/config"
	"gopkg.in/yaml.v3"
)

// Config contains all configuration for noti-service.
type Config struct {
	commonconfig.BaseConfig `yaml:",inline"`
	InternalAuth            InternalAuthConfig `yaml:"internal_auth"`
	App                     AppConfig          `yaml:"app"`
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
	cfg.BaseConfig.LoadFromEnv()

	// Service-specific environment variables
	if apiKey := os.Getenv("INTERNAL_API_KEY"); apiKey != "" {
		cfg.InternalAuth.InternalAPIKey = apiKey
	}

	return cfg, nil
}
