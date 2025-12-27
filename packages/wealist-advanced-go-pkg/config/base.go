// Package config provides common configuration structures for wealist services.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// BaseConfig contains configuration common to all services.
type BaseConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Auth     AuthConfig     `yaml:"auth"`
	UserAPI  UserAPIConfig  `yaml:"user_api"`
}

// ServerConfig contains HTTP server configuration.
type ServerConfig struct {
	Port     int    `yaml:"port"`
	BasePath string `yaml:"base_path"`
	Env      string `yaml:"env"`
	LogLevel string `yaml:"log_level"`
}

// DatabaseConfig contains database connection configuration.
type DatabaseConfig struct {
	URL         string `yaml:"url"`
	Host        string `yaml:"host"`
	Port        string `yaml:"port"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	DBName      string `yaml:"dbname"`
	SSLMode     string `yaml:"sslmode"`
	AutoMigrate bool   `yaml:"auto_migrate"`
}

// RedisConfig contains Redis connection configuration.
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// AuthConfig contains authentication configuration.
type AuthConfig struct {
	ServiceURL   string `yaml:"service_url"`
	SecretKey    string `yaml:"secret_key"`     // Deprecated: use JWTIssuer with SmartValidator
	JWTIssuer    string `yaml:"jwt_issuer"`     // JWT issuer for JWKS validation
	IstioJWTMode bool   `yaml:"istio_jwt_mode"` // If true, Istio handles JWT validation, Go services only parse
}

// UserAPIConfig contains User API client configuration.
type UserAPIConfig struct {
	BaseURL string        `yaml:"base_url"`
	Timeout time.Duration `yaml:"timeout"`
}

// DefaultBaseConfig returns a BaseConfig with default values.
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Server: ServerConfig{
			Port:     8080,
			BasePath: "/api",
			Env:      "dev",
			LogLevel: "debug",
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   0,
		},
		UserAPI: UserAPIConfig{
			Timeout: 5 * time.Second,
		},
	}
}

// LoadFromEnv overrides BaseConfig fields from environment variables.
func (c *BaseConfig) LoadFromEnv() {
	// Server config
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			c.Server.Port = p
		}
	}
	if basePath := os.Getenv("SERVER_BASE_PATH"); basePath != "" {
		c.Server.BasePath = basePath
	}
	if env := os.Getenv("ENV"); env != "" {
		c.Server.Env = env
	}
	if env := os.Getenv("NODE_ENV"); env != "" {
		c.Server.Env = env
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.Server.LogLevel = logLevel
	}

	// Database config - individual components
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		c.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		c.Database.Port = dbPort
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		c.Database.User = dbUser
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		c.Database.Password = dbPassword
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		c.Database.DBName = dbName
	}
	if dbSSLMode := os.Getenv("DB_SSLMODE"); dbSSLMode != "" {
		c.Database.SSLMode = dbSSLMode
	}
	// DATABASE_URL takes precedence over individual components
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		c.Database.URL = dbURL
	} else if c.Database.Host != "" && c.Database.DBName != "" {
		// Build DATABASE_URL from individual components
		c.Database.URL = c.Database.BuildURL()
	}
	if autoMigrate := os.Getenv("DB_AUTO_MIGRATE"); autoMigrate != "" {
		c.Database.AutoMigrate = autoMigrate == "true"
	}

	// Redis config
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		c.Redis.Host = redisHost
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		if p, err := strconv.Atoi(redisPort); err == nil {
			c.Redis.Port = p
		}
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		c.Redis.Password = redisPassword
	}

	// Auth config
	if authURL := os.Getenv("AUTH_SERVICE_URL"); authURL != "" {
		c.Auth.ServiceURL = authURL
	}
	if secretKey := os.Getenv("SECRET_KEY"); secretKey != "" {
		c.Auth.SecretKey = secretKey
	}
	if jwtIssuer := os.Getenv("JWT_ISSUER"); jwtIssuer != "" {
		c.Auth.JWTIssuer = jwtIssuer
	}
	if c.Auth.JWTIssuer == "" {
		c.Auth.JWTIssuer = "wealist-auth-service" // default issuer
	}
	// Istio JWT Mode: true면 Istio가 검증, Go 서비스는 파싱만
	if istioJWTMode := os.Getenv("ISTIO_JWT_MODE"); istioJWTMode != "" {
		c.Auth.IstioJWTMode = istioJWTMode == "true"
	}

	// User API config
	if userURL := os.Getenv("USER_SERVICE_URL"); userURL != "" {
		c.UserAPI.BaseURL = userURL
	}
	if timeout := os.Getenv("USER_API_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			c.UserAPI.Timeout = d
		}
	}
	if c.UserAPI.Timeout == 0 {
		c.UserAPI.Timeout = 5 * time.Second
	}
}

// GetEnvString returns environment variable value or default.
func GetEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt returns environment variable as int or default.
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// GetEnvBool returns environment variable as bool or default.
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

// GetEnvDuration returns environment variable as duration or default.
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

// BuildURL constructs a PostgreSQL connection URL from individual components.
// User and password are URL-encoded to handle special characters.
func (d *DatabaseConfig) BuildURL() string {
	port := d.Port
	if port == "" {
		port = "5432"
	}
	sslmode := d.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	// URL-encode user and password to handle special characters
	encodedUser := url.QueryEscape(d.User)
	encodedPassword := url.QueryEscape(d.Password)
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		encodedUser, encodedPassword, d.Host, port, d.DBName, sslmode)
}
