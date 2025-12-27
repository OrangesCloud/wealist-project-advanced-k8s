// @title           User Service API
// @version         1.0
// @description     User and Workspace management API
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.wealist.co.kr/support
// @contact.email  support@wealist.co.kr

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
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

	_ "user-service/docs" // Swagger docs import

	"user-service/internal/config"
	"user-service/internal/database"
	"user-service/internal/middleware"
	"user-service/internal/router"
	"user-service/internal/client"
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

	// Set Gin mode
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	logger.Info("Starting User Service",
		zap.String("port", cfg.Server.Port),
		zap.String("mode", cfg.Server.Mode),
		zap.String("base_path", cfg.Server.BasePath),
	)

	// Initialize database with retry (wait for DB to be ready)
	dbConfig := database.Config{
		DSN:             cfg.Database.GetDSN(),
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
	}

	// Retry up to 30 times (5s interval = ~2.5 minutes total wait)
	db, err := database.NewWithRetry(dbConfig, 5*time.Second, 30)
	if err != nil {
		logger.Fatal("Failed to connect to database after retries",
			zap.Error(err))
	}
	logger.Info("Database connected successfully")

	// Run auto migration (conditional based on DB_AUTO_MIGRATE env)
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

	// Initialize S3 client
	var s3Client *client.S3Client
	if cfg.S3.Bucket != "" && cfg.S3.Region != "" {
		s3Client, err = client.NewS3Client(&cfg.S3)
		if err != nil {
			logger.Warn("Failed to initialize S3 client", zap.Error(err))
		} else {
			logger.Info("S3 client initialized",
				zap.String("bucket", cfg.S3.Bucket),
				zap.String("region", cfg.S3.Region),
			)
		}
	} else {
		logger.Warn("S3 configuration incomplete, profile image uploads disabled")
	}

	// Initialize Redis for rate limiting
	if err := database.InitRedis(logger); err != nil {
		logger.Warn("Failed to initialize Redis, rate limiting will be disabled", zap.Error(err))
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
		DB:              db,
		Logger:          logger,
		JWTSecret:       cfg.JWT.Secret,
		BasePath:        cfg.Server.BasePath,
		S3Client:        s3Client,
		TokenValidator:  tokenValidator,
		RedisClient:     database.GetRedis(),
		RateLimitConfig: cfg.RateLimit,
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
		logger.Info("User Service started successfully",
			zap.String("address", srv.Addr),
			zap.String("swagger", fmt.Sprintf("http://localhost:%s/swagger/index.html", cfg.Server.Port)),
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
