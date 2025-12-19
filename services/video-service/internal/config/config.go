package config

import (
	"os"

	commonconfig "github.com/OrangesCloud/wealist-advanced-go-pkg/config"
	"gopkg.in/yaml.v3"
)

// Config contains all configuration for video-service.
type Config struct {
	commonconfig.BaseConfig `yaml:",inline"`
	LiveKit                 LiveKitConfig  `yaml:"livekit"`
	Services                ServicesConfig `yaml:"services"`
	CORS                    CORSConfig     `yaml:"cors"`
}

// LiveKitConfig contains LiveKit server configuration.
type LiveKitConfig struct {
	Host      string `yaml:"host"`
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	WSUrl     string `yaml:"ws_url"`
}

// ServicesConfig contains service URLs configuration.
type ServicesConfig struct {
	UserServiceURL string `yaml:"user_service_url"`
}

// CORSConfig contains CORS configuration.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins"`
}

// Load reads configuration from yaml file and environment variables.
func Load(path string) (*Config, error) {
	// Start with defaults
	base := commonconfig.DefaultBaseConfig()
	base.Server.Port = 8004
	base.Server.BasePath = "/api/video"

	cfg := &Config{
		BaseConfig: base,
		LiveKit: LiveKitConfig{
			Host:  "http://localhost:7880",
			WSUrl: "ws://localhost:7880",
		},
		CORS: CORSConfig{
			AllowedOrigins: "*",
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
	if userURL := os.Getenv("USER_SERVICE_URL"); userURL != "" {
		cfg.Services.UserServiceURL = userURL
	}

	// LiveKit configuration
	if lkHost := os.Getenv("LIVEKIT_HOST"); lkHost != "" {
		cfg.LiveKit.Host = lkHost
	}
	if lkAPIKey := os.Getenv("LIVEKIT_API_KEY"); lkAPIKey != "" {
		cfg.LiveKit.APIKey = lkAPIKey
	}
	if lkAPISecret := os.Getenv("LIVEKIT_API_SECRET"); lkAPISecret != "" {
		cfg.LiveKit.APISecret = lkAPISecret
	}
	if lkWSUrl := os.Getenv("LIVEKIT_WS_URL"); lkWSUrl != "" {
		cfg.LiveKit.WSUrl = lkWSUrl
	}

	// CORS configuration
	if corsOrigins := os.Getenv("CORS_ORIGINS"); corsOrigins != "" {
		cfg.CORS.AllowedOrigins = corsOrigins
	}

	return cfg, nil
}
