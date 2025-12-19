package config

import (
	"os"

	commonconfig "github.com/OrangesCloud/wealist-advanced-go-pkg/config"
	"gopkg.in/yaml.v3"
)

// Config contains all configuration for chat-service.
type Config struct {
	commonconfig.BaseConfig `yaml:",inline"`
	Services                ServicesConfig `yaml:"services"`
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
	cfg.BaseConfig.LoadFromEnv()

	// Service-specific environment variables
	if userURL := os.Getenv("USER_SERVICE_URL"); userURL != "" {
		cfg.Services.UserServiceURL = userURL
	}

	return cfg, nil
}
