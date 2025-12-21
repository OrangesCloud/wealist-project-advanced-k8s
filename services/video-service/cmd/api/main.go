// @title           Video Service API
// @version         1.0
// @description     Video call room management API with LiveKit integration

// @contact.name   API Support
// @contact.url    http://www.wealist.co.kr/support
// @contact.email  support@wealist.co.kr

// @host      localhost:8004
// @BasePath  /api/video

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
	"video-service/internal/config"
	"video-service/internal/database"
	"video-service/internal/router"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := initLogger(cfg.Server.Env, cfg.Server.LogLevel)
	defer func() { _ = logger.Sync() }()

	logger.Info("Starting video service",
		zap.String("env", cfg.Server.Env),
		zap.Int("port", cfg.Server.Port),
		zap.String("basePath", cfg.Server.BasePath),
		zap.String("liveKitWSUrl", cfg.LiveKit.WSUrl), // Log the WSUrl
	)

	// Initialize database
	db, err := database.NewDB(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	sqlDB, _ := db.DB()
	defer func() { _ = sqlDB.Close() }()

	// Initialize Redis for rate limiting
	if err := database.InitRedis(logger); err != nil {
		logger.Warn("Failed to initialize Redis, rate limiting will be disabled", zap.Error(err))
	}
	redisClient := database.GetRedis()

	// Setup router
	r := router.Setup(cfg, db, redisClient, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server listening", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initLogger(env, level string) *zap.Logger {
	var config zap.Config

	if env == "production" || env == "prod" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return logger
}
