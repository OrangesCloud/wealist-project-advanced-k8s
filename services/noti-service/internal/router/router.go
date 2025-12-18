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
)

// Setup configures and returns the Gin router with all routes and middleware.
func Setup(cfg *config.Config, db *gorm.DB, redisClient *redis.Client, logger *zap.Logger) *gin.Engine {
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Initialize metrics
	m := metrics.New()

	// Middleware (using common package)
	r.Use(commonmw.Recovery(logger))
	r.Use(commonmw.Logger(logger))
	r.Use(commonmw.DefaultCORS())
	r.Use(metrics.HTTPMiddleware(m))

	// Initialize services
	// 레포지토리와 SSE 서비스 초기화
	notificationRepo := repository.NewNotificationRepository(db)
	sseService := sse.NewSSEService(redisClient, logger)
	// 알림 서비스 초기화 (메트릭 포함)
	notificationService := service.NewNotificationService(notificationRepo, redisClient, cfg, logger, m)

	// Initialize handlers
	validator := middleware.NewAuthServiceValidator(cfg.Auth.ServiceURL, cfg.Auth.SecretKey, logger)
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
		api.GET("/notifications/stream", middleware.SSEAuthMiddleware(validator), notificationHandler.StreamNotifications)

		// Notification routes (require auth via Authorization header)
		notifications := api.Group("/notifications")
		notifications.Use(middleware.AuthMiddleware(validator))
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
		internal.Use(middleware.InternalAuthMiddleware(cfg.Auth.InternalAPIKey))
		{
			internal.POST("/notifications", notificationHandler.CreateNotification)
			internal.POST("/notifications/bulk", notificationHandler.CreateBulkNotifications)
		}
	}

	return r
}
