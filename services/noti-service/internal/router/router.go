// Package router configures HTTP routing for noti-service.
//
// This package sets up the Gin router with all middleware, handlers,
// and route definitions for the notification API.
package router

import (
	"noti-service/internal/config"
	"noti-service/internal/handler"
	"noti-service/internal/metrics"
	"noti-service/internal/middleware"
	"noti-service/internal/repository"
	"noti-service/internal/service"
	"noti-service/internal/sse"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	commonhealth "github.com/OrangesCloud/wealist-advanced-go-pkg/health"
	commonmw "github.com/OrangesCloud/wealist-advanced-go-pkg/middleware"
	"github.com/OrangesCloud/wealist-advanced-go-pkg/ratelimit"
)

// RouterConfig holds router configuration
type RouterConfig struct {
	Config      *config.Config
	DB          *gorm.DB
	RedisClient *redis.Client
	Logger      *zap.Logger
	ServiceName string
}

// Setup configures and returns the Gin router with all routes and middleware.
func Setup(routerCfg RouterConfig) *gin.Engine {
	cfg := routerCfg.Config
	db := routerCfg.DB
	redisClient := routerCfg.RedisClient
	logger := routerCfg.Logger

	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Initialize metrics
	m := metrics.New()

	// Determine service name for tracing
	serviceName := routerCfg.ServiceName
	if serviceName == "" {
		serviceName = "noti-service"
	}

	// Middleware (using common package with OTEL tracing)
	r.Use(commonmw.Recovery(logger))
	r.Use(commonmw.OTELTracing(serviceName))                 // OpenTelemetry HTTP tracing (otelgin)
	r.Use(commonmw.LoggerWithTracing(logger, serviceName))   // Request logging with trace context
	r.Use(commonmw.DefaultCORS())
	r.Use(metrics.HTTPMiddleware(m))

	// Rate limiting middleware
	if cfg.RateLimit.Enabled && redisClient != nil {
		rlConfig := ratelimit.DefaultConfig().
			WithRequestsPerMinute(cfg.RateLimit.RequestsPerMinute).
			WithBurstSize(cfg.RateLimit.BurstSize).
			WithKeyPrefix("rl:noti:")
		limiter := ratelimit.NewRedisRateLimiter(redisClient, rlConfig, logger)
		r.Use(ratelimit.MiddlewareWithLogger(limiter, ratelimit.UserKey, rlConfig, logger))
		logger.Info("Rate limiting middleware enabled",
			zap.Int("requests_per_minute", cfg.RateLimit.RequestsPerMinute),
			zap.Int("burst_size", cfg.RateLimit.BurstSize))
	} else if cfg.RateLimit.Enabled && redisClient == nil {
		logger.Warn("Rate limiting enabled but Redis is not available, skipping")
	}

	// Initialize services
	// 레포지토리와 SSE 서비스 초기화
	notificationRepo := repository.NewNotificationRepository(db)
	sseService := sse.NewSSEService(redisClient, logger)
	// 알림 서비스 초기화 (메트릭 포함)
	notificationService := service.NewNotificationService(notificationRepo, redisClient, cfg, logger, m)

	// Initialize auth middleware based on ISTIO_JWT_MODE
	var authMiddleware gin.HandlerFunc
	var sseValidator middleware.TokenValidator

	if cfg.Auth.IstioJWTMode {
		// K8s + Istio 환경: Istio가 JWT 검증, Go 서비스는 파싱만
		parser := middleware.NewJWTParser(logger)
		authMiddleware = middleware.IstioAuthMiddleware(parser)
		// SSE용 validator는 SmartValidator 사용 (SSE는 query param으로 토큰 전달)
		sseValidator = middleware.NewSmartValidator(cfg.Auth.ServiceURL, cfg.Auth.JWTIssuer, logger)
		logger.Info("Using Istio JWT mode (parse only)",
			zap.String("auth_service_url", cfg.Auth.ServiceURL))
	} else {
		// Docker Compose / K8s without Istio: SmartValidator로 전체 검증
		validator := middleware.NewSmartValidator(cfg.Auth.ServiceURL, cfg.Auth.JWTIssuer, logger)
		authMiddleware = middleware.AuthMiddleware(validator)
		sseValidator = validator
		logger.Info("Using SmartValidator mode (full validation)",
			zap.String("auth_service_url", cfg.Auth.ServiceURL),
			zap.String("jwt_issuer", cfg.Auth.JWTIssuer))
	}

	notificationHandler := handler.NewNotificationHandler(notificationService, sseService, logger)

	// Health check routes (using common package)
	healthChecker := commonhealth.NewHealthChecker(db, redisClient)
	healthChecker.RegisterRoutes(r, "")

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger documentation (disabled for faster builds)
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	api := r.Group("/api")
	{
		// SSE stream endpoint (uses query param token because EventSource doesn't support headers)
		// SSE는 Istio를 통하지 않을 수 있으므로 SmartValidator 사용
		api.GET("/notifications/stream", middleware.SSEAuthMiddleware(sseValidator), notificationHandler.StreamNotifications)

		// Notification routes (require auth via Authorization header)
		notifications := api.Group("/notifications")
		notifications.Use(authMiddleware)
		notifications.Use(middleware.WorkspaceMiddleware())
		{
			notifications.GET("", middleware.RequireWorkspace(), notificationHandler.GetNotifications)
			notifications.GET("/unread-count", middleware.RequireWorkspace(), notificationHandler.GetUnreadCount)
			notifications.PATCH("/:id/read", notificationHandler.MarkAsRead)
			notifications.POST("/read-all", middleware.RequireWorkspace(), notificationHandler.MarkAllAsRead)
			notifications.DELETE("/:id", notificationHandler.DeleteNotification)
		}

		// Internal API routes (require API key)
		internal := api.Group("/internal")
		internal.Use(middleware.InternalAuthMiddleware(cfg.InternalAuth.InternalAPIKey))
		{
			internal.POST("/notifications", notificationHandler.CreateNotification)
			internal.POST("/notifications/bulk", notificationHandler.CreateBulkNotifications)
		}
	}

	return r
}
