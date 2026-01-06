package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Logger     LoggerConfig     `yaml:"logger"`
	JWT        JWTConfig        `yaml:"jwt"`
	AuthAPI    AuthAPIConfig    `yaml:"auth_api"`
	CORS       CORSConfig       `yaml:"cors"`
	ArgoCD     ArgoCDConfig     `yaml:"argocd"`
	Kubernetes KubernetesConfig `yaml:"kubernetes"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
}

// PrometheusConfig holds Prometheus API configuration
type PrometheusConfig struct {
	BaseURL   string        `yaml:"base_url"`
	Timeout   time.Duration `yaml:"timeout"`
	Namespace string        `yaml:"namespace"` // Namespace to query metrics for
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            string        `yaml:"port"`
	Mode            string        `yaml:"mode"`
	BasePath        string        `yaml:"base_path"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            string        `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	AutoMigrate     bool          `yaml:"auto_migrate"`
}

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	Level      string `yaml:"level"`
	OutputPath string `yaml:"output_path"`
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret     string        `yaml:"secret"`
	ExpireTime time.Duration `yaml:"expire_time"`
}

// AuthAPIConfig holds Auth Service configuration
type AuthAPIConfig struct {
	BaseURL string        `yaml:"base_url"`
	Timeout time.Duration `yaml:"timeout"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins"`
}

// ArgoCDConfig holds ArgoCD API configuration
type ArgoCDConfig struct {
	ServerURL string `yaml:"server_url"`
	Token     string `yaml:"token"`
	Insecure  bool   `yaml:"insecure"`
	Namespace string `yaml:"namespace"` // ArgoCD namespace (default: "argocd")
}

// KubernetesConfig holds Kubernetes client configuration
type KubernetesConfig struct {
	InCluster  bool   `yaml:"in_cluster"`  // Use in-cluster config
	KubeConfig string `yaml:"kubeconfig"`  // Path to kubeconfig file
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	BurstSize         int  `yaml:"burst_size"`
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	var cfg Config

	data, err := os.ReadFile(configPath)
	if err == nil {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Config file not found, using environment variables and defaults\n")
		cfg = getDefaultConfig()
	}

	cfg.overrideFromEnv()

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func getDefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Port:            "8005",
			Mode:            "debug",
			BasePath:        "/api",
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            "5432",
			User:            "postgres",
			Password:        "",
			DBName:          "ops_db",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			AutoMigrate:     false,
		},
		Logger: LoggerConfig{
			Level:      "info",
			OutputPath: "stdout",
		},
		JWT: JWTConfig{
			Secret:     "",
			ExpireTime: 24 * time.Hour,
		},
		AuthAPI: AuthAPIConfig{
			BaseURL: "http://localhost:8080",
			Timeout: 5 * time.Second,
		},
		CORS: CORSConfig{
			AllowedOrigins: "*",
		},
		ArgoCD: ArgoCDConfig{
			ServerURL: "https://argocd.wealist.co.kr",
			Insecure:  false,
			Namespace: "argocd",
		},
		Kubernetes: KubernetesConfig{
			InCluster:  false,
			KubeConfig: "",
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
		Prometheus: PrometheusConfig{
			BaseURL:   "http://prometheus:9090",
			Timeout:   10 * time.Second,
			Namespace: "wealist-prod",
		},
	}
}

func (c *Config) overrideFromEnv() {
	// Server
	if port := os.Getenv("SERVER_PORT"); port != "" {
		c.Server.Port = port
	}
	if env := os.Getenv("ENV"); env != "" {
		switch env {
		case "dev":
			c.Server.Mode = "debug"
		case "prod":
			c.Server.Mode = "release"
		default:
			c.Server.Mode = env
		}
	}
	if basePath := os.Getenv("SERVER_BASE_PATH"); basePath != "" {
		c.Server.BasePath = basePath
	}

	// Database - DATABASE_URL takes precedence
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		host, port, user, password, dbname, err := parseDatabaseURL(databaseURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse DATABASE_URL: %v\n", err)
		} else {
			c.Database.Host = host
			c.Database.Port = port
			c.Database.User = user
			c.Database.Password = password
			c.Database.DBName = dbname
		}
	}

	if host := os.Getenv("DB_HOST"); host != "" {
		c.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		c.Database.Port = port
	}
	if user := os.Getenv("DB_USER"); user != "" {
		c.Database.User = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		c.Database.Password = password
	}
	if dbname := os.Getenv("DB_NAME"); dbname != "" {
		c.Database.DBName = dbname
	}
	if sslmode := os.Getenv("DB_SSLMODE"); sslmode != "" {
		c.Database.SSLMode = sslmode
	}
	if autoMigrate := os.Getenv("DB_AUTO_MIGRATE"); autoMigrate != "" {
		c.Database.AutoMigrate = autoMigrate == "true"
	}

	// Logger
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		c.Logger.Level = level
	}

	// JWT
	if secret := os.Getenv("SECRET_KEY"); secret != "" {
		c.JWT.Secret = secret
	}
	if secret := os.Getenv("JWT_SECRET"); secret != "" && os.Getenv("SECRET_KEY") == "" {
		c.JWT.Secret = secret
	}

	// Auth API
	if baseURL := os.Getenv("AUTH_SERVICE_URL"); baseURL != "" {
		c.AuthAPI.BaseURL = baseURL
	}

	// CORS
	if origins := os.Getenv("CORS_ORIGINS"); origins != "" {
		c.CORS.AllowedOrigins = origins
	}

	// ArgoCD
	if serverURL := os.Getenv("ARGOCD_SERVER_URL"); serverURL != "" {
		c.ArgoCD.ServerURL = serverURL
	}
	if token := os.Getenv("ARGOCD_TOKEN"); token != "" {
		c.ArgoCD.Token = token
	}
	if insecure := os.Getenv("ARGOCD_INSECURE"); insecure != "" {
		c.ArgoCD.Insecure = insecure == "true"
	}
	if ns := os.Getenv("ARGOCD_NAMESPACE"); ns != "" {
		c.ArgoCD.Namespace = ns
	}
	if c.ArgoCD.Namespace == "" {
		c.ArgoCD.Namespace = "argocd"
	}

	// Kubernetes
	if inCluster := os.Getenv("K8S_IN_CLUSTER"); inCluster != "" {
		c.Kubernetes.InCluster = inCluster == "true"
	}
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		c.Kubernetes.KubeConfig = kubeconfig
	}

	// Rate Limit
	if rateLimitEnabled := os.Getenv("RATE_LIMIT_ENABLED"); rateLimitEnabled != "" {
		c.RateLimit.Enabled = rateLimitEnabled == "true"
	}
	if rpm := os.Getenv("RATE_LIMIT_PER_MINUTE"); rpm != "" {
		if v, err := strconv.Atoi(rpm); err == nil {
			c.RateLimit.RequestsPerMinute = v
		}
	}
	if burst := os.Getenv("RATE_LIMIT_BURST"); burst != "" {
		if v, err := strconv.Atoi(burst); err == nil {
			c.RateLimit.BurstSize = v
		}
	}
	if c.RateLimit.RequestsPerMinute == 0 {
		c.RateLimit.RequestsPerMinute = 60
	}

	// Prometheus
	if prometheusURL := os.Getenv("PROMETHEUS_URL"); prometheusURL != "" {
		c.Prometheus.BaseURL = prometheusURL
	}
	if prometheusNS := os.Getenv("PROMETHEUS_NAMESPACE"); prometheusNS != "" {
		c.Prometheus.Namespace = prometheusNS
	}
}

func (c *Config) validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port == "" {
		return fmt.Errorf("database port is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt secret is required")
	}
	return nil
}

// GetDSN returns the database connection string
func (c *DatabaseConfig) GetDSN() string {
	sslmode := c.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, sslmode,
	)
}

func parseDatabaseURL(databaseURL string) (host, port, user, password, dbname string, err error) {
	if databaseURL == "" {
		return "", "", "", "", "", fmt.Errorf("DATABASE_URL is empty")
	}

	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("invalid DATABASE_URL format: %w", err)
	}

	if u.Scheme != "postgresql" && u.Scheme != "postgres" {
		return "", "", "", "", "", fmt.Errorf("invalid scheme '%s': must be 'postgresql' or 'postgres'", u.Scheme)
	}

	if u.User == nil {
		return "", "", "", "", "", fmt.Errorf("missing user credentials")
	}
	user = u.User.Username()
	password, _ = u.User.Password()

	if u.Host == "" {
		return "", "", "", "", "", fmt.Errorf("missing host")
	}

	hostPort := u.Host
	if strings.Contains(hostPort, ":") {
		parts := strings.Split(hostPort, ":")
		host = parts[0]
		port = parts[1]
	} else {
		host = hostPort
		port = "5432"
	}

	dbname = strings.TrimPrefix(u.Path, "/")
	if dbname == "" {
		return "", "", "", "", "", fmt.Errorf("missing database name")
	}

	return host, port, user, password, dbname, nil
}
