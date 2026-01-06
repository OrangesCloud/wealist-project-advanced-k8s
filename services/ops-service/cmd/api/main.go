// @title           Ops Service API
// @version         1.0.0
// @description     Admin Portal and Operations Management API
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.wealist.co.kr/support
// @contact.email  support@wealist.co.kr

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8005
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/OrangesCloud/wealist-advanced-go-pkg/otel"
	"ops-service/internal/client"
	"ops-service/internal/config"
	"ops-service/internal/database"
	"ops-service/internal/middleware"
	"ops-service/internal/router"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := initLogger(cfg.Logger.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()

	// Initialize OpenTelemetry
	ctx := context.Background()
	otelCfg := otel.DefaultConfig("ops-service")
	otelShutdown, err := otel.InitProvider(ctx, otelCfg)
	if err != nil {
		logger.Warn("Failed to initialize OpenTelemetry, continuing without tracing",
			zap.Error(err),
		)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := otelShutdown(shutdownCtx); err != nil {
				logger.Error("Failed to shutdown OpenTelemetry", zap.Error(err))
			}
		}()
		logger.Info("OpenTelemetry initialized",
			zap.String("service.name", otelCfg.ServiceName),
			zap.String("otel.endpoint", otelCfg.OTLPEndpoint),
		)
	}

	// Set Gin mode
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	logger.Info("Starting Ops Service",
		zap.String("port", cfg.Server.Port),
		zap.String("mode", cfg.Server.Mode),
		zap.String("base_path", cfg.Server.BasePath),
	)

	// Initialize database with retry
	dbConfig := database.Config{
		DSN:             cfg.Database.GetDSN(),
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	db, err := database.NewWithRetry(dbConfig, 5*time.Second, 30)
	if err != nil {
		logger.Fatal("Failed to connect to database after retries",
			zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// Enable GORM OpenTelemetry tracing
	if err := otel.EnableGORMTracing(db, "ops_db"); err != nil {
		logger.Warn("Failed to enable GORM tracing, continuing without DB tracing",
			zap.Error(err),
		)
	} else {
		logger.Info("GORM OpenTelemetry tracing enabled")
	}

	// Run auto migration
	if cfg.Database.AutoMigrate {
		logger.Info("Running database migrations (DB_AUTO_MIGRATE=true)")
		if err := database.AutoMigrate(db); err != nil {
			logger.Warn("Failed to run database migrations", zap.Error(err))
		} else {
			logger.Info("Database migrations completed")
		}
	} else {
		logger.Info("Database auto-migration disabled (DB_AUTO_MIGRATE=false)")
	}

	// Initialize Redis
	if err := database.InitRedis(logger); err != nil {
		logger.Warn("Failed to initialize Redis, rate limiting will be disabled", zap.Error(err))
	}

	// Enable Redis OpenTelemetry tracing
	if redisClient := database.GetRedis(); redisClient != nil {
		if err := otel.EnableRedisTracing(redisClient); err != nil {
			logger.Warn("Failed to enable Redis tracing",
				zap.Error(err),
			)
		} else {
			logger.Info("Redis OpenTelemetry tracing enabled")
		}
	}

	// Initialize ArgoCD client
	var argoCDClient *client.ArgoCDClient
	if cfg.ArgoCD.ServerURL != "" && cfg.ArgoCD.Token != "" {
		argoCDClient = client.NewArgoCDClient(client.ArgoCDConfig{
			ServerURL: cfg.ArgoCD.ServerURL,
			Token:     cfg.ArgoCD.Token,
			Insecure:  cfg.ArgoCD.Insecure,
		}, logger)
		logger.Info("ArgoCD client initialized",
			zap.String("serverURL", cfg.ArgoCD.ServerURL),
		)
	} else {
		logger.Warn("ArgoCD configuration incomplete, ArgoCD features disabled")
	}

	// Initialize Kubernetes client (for ArgoCD RBAC management)
	var k8sClient *client.K8sClient
	k8sClient, err = client.NewK8sClient(client.K8sConfig{
		InCluster:  cfg.Kubernetes.InCluster,
		KubeConfig: cfg.Kubernetes.KubeConfig,
	}, logger)
	if err != nil {
		logger.Warn("Failed to initialize Kubernetes client, ArgoCD RBAC management disabled",
			zap.Error(err))
		k8sClient = nil
	} else {
		logger.Info("Kubernetes client initialized",
			zap.Bool("inCluster", cfg.Kubernetes.InCluster),
		)
	}

	// Initialize Prometheus client
	var prometheusClient *client.PrometheusClient
	if cfg.Prometheus.BaseURL != "" {
		prometheusClient = client.NewPrometheusClient(client.PrometheusConfig{
			BaseURL: cfg.Prometheus.BaseURL,
			Timeout: cfg.Prometheus.Timeout,
		}, logger)
		logger.Info("Prometheus client initialized",
			zap.String("baseURL", cfg.Prometheus.BaseURL),
			zap.String("namespace", cfg.Prometheus.Namespace),
		)
	} else {
		logger.Warn("Prometheus URL not configured, metrics endpoints will be disabled")
	}

	// Initialize Loki client
	var lokiClient *client.LokiClient
	if cfg.Loki.BaseURL != "" {
		lokiClient = client.NewLokiClient(
			cfg.Loki.BaseURL,
			cfg.Loki.Namespace,
			cfg.Loki.Timeout,
			logger,
		)
		logger.Info("Loki client initialized",
			zap.String("baseURL", cfg.Loki.BaseURL),
			zap.String("namespace", cfg.Loki.Namespace),
		)
	} else {
		logger.Warn("Loki URL not configured, logs endpoints will return empty results")
	}

	// Initialize Auth validator (SmartValidator for RS256 JWKS support)
	var tokenValidator middleware.TokenValidator
	if cfg.AuthAPI.BaseURL != "" {
		jwtIssuer := os.Getenv("JWT_ISSUER")
		if jwtIssuer == "" {
			jwtIssuer = "wealist-auth-service"
		}
		tokenValidator = middleware.NewSmartValidator(cfg.AuthAPI.BaseURL, jwtIssuer, logger)
		logger.Info("SmartValidator initialized",
			zap.String("auth_api_url", cfg.AuthAPI.BaseURL),
			zap.String("jwt_issuer", jwtIssuer))
	}

	// Setup router
	r := router.Setup(router.Config{
		DB:               db,
		Logger:           logger,
		JWTSecret:        cfg.JWT.Secret,
		BasePath:         cfg.Server.BasePath,
		ArgoCDClient:     argoCDClient,
		K8sClient:        k8sClient,
		ArgoCDNamespace:  cfg.ArgoCD.Namespace,
		RedisClient:      database.GetRedis(),
		RateLimitConfig:  cfg.RateLimit,
		ServiceName:      "ops-service",
		TokenValidator:   tokenValidator,
		PrometheusClient: prometheusClient,
		PrometheusNS:     cfg.Prometheus.Namespace,
		LokiClient:       lokiClient,
		LokiNS:           cfg.Loki.Namespace,
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server
	go func() {
		logger.Info("Ops Service started successfully",
			zap.String("address", srv.Addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Development:      zapLevel == zapcore.DebugLevel,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}
