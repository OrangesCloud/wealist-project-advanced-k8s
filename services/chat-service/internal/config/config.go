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

// S3Config holds S3 configuration for chat file uploads
type S3Config struct {
	Bucket         string `yaml:"bucket"`
	Region         string `yaml:"region"`
	AccessKey      string `yaml:"access_key"`      // MinIO용만 필요 (선택적)
	SecretKey      string `yaml:"secret_key"`      // MinIO용만 필요 (선택적)
	Endpoint       string `yaml:"endpoint"`        // 로컬 MinIO용 (선택적)
	PublicEndpoint string `yaml:"public_endpoint"` // 브라우저 접근용 공개 엔드포인트 (presigned URL용)
}

// Config contains all configuration for chat-service.
type Config struct {
	commonconfig.BaseConfig `yaml:",inline"`
	Services                ServicesConfig  `yaml:"services"`
	RateLimit               RateLimitConfig `yaml:"rate_limit"`
	S3                      S3Config        `yaml:"s3"` // S3 configuration
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

	// S3 환경변수 오버라이드
	if s3Bucket := os.Getenv("S3_BUCKET"); s3Bucket != "" {
		cfg.S3.Bucket = s3Bucket
	}
	if s3Region := os.Getenv("S3_REGION"); s3Region != "" {
		cfg.S3.Region = s3Region
	}
	if s3AccessKey := os.Getenv("S3_ACCESS_KEY"); s3AccessKey != "" {
		cfg.S3.AccessKey = s3AccessKey
	}
	if s3SecretKey := os.Getenv("S3_SECRET_KEY"); s3SecretKey != "" {
		cfg.S3.SecretKey = s3SecretKey
	}
	if s3Endpoint := os.Getenv("S3_ENDPOINT"); s3Endpoint != "" {
		cfg.S3.Endpoint = s3Endpoint
	}
	if s3PublicEndpoint := os.Getenv("S3_PUBLIC_ENDPOINT"); s3PublicEndpoint != "" {
		cfg.S3.PublicEndpoint = s3PublicEndpoint
	}

	return cfg, nil
}
