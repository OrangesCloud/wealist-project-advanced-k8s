package router

import (
	"chat-service/internal/client"
	"chat-service/internal/config"
	"chat-service/internal/handler"
	"chat-service/internal/metrics"
	"chat-service/internal/middleware"
	"chat-service/internal/repository"
	"chat-service/internal/service"
	"chat-service/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	// swaggerFiles "github.com/swaggo/files"
	// ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"gorm.io/gorm"

	commonhealth "github.com/OrangesCloud/wealist-advanced-go-pkg/health"
	commonmw "github.com/OrangesCloud/wealist-advanced-go-pkg/middleware"
)

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

	// Initialize repositories
	chatRepo := repository.NewChatRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	presenceRepo := repository.NewPresenceRepository(db)

	// Initialize user client for workspace validation
	var userClient client.UserClient
	if cfg.UserAPI.BaseURL != "" {
		userClient = client.NewUserClient(cfg.UserAPI.BaseURL, cfg.UserAPI.Timeout, logger)
		logger.Info("User client initialized", zap.String("url", cfg.UserAPI.BaseURL))
	} else {
		logger.Warn("User service URL not configured, workspace validation will be skipped")
	}

	// Initialize services (메트릭 연동)
	chatService := service.NewChatService(chatRepo, messageRepo, userClient, redisClient, logger, m)
	presenceService := service.NewPresenceService(presenceRepo, redisClient, logger, m)

	// Initialize validator
	validator := middleware.NewAuthServiceValidator(cfg.Auth.ServiceURL, cfg.Auth.SecretKey, logger)

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(chatService, presenceService, validator, redisClient, logger)

	// Initialize handlers
	chatHandler := handler.NewChatHandler(chatService, presenceService, logger)
	messageHandler := handler.NewMessageHandler(chatService, logger)
	presenceHandler := handler.NewPresenceHandler(presenceService, logger)

	// Health check routes (using common package)
	healthChecker := commonhealth.NewHealthChecker(db, redisClient)
	healthChecker.RegisterRoutes(r, cfg.Server.BasePath)

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger documentation (disabled for faster builds)
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes with base path
	api := r.Group(cfg.Server.BasePath)
	{

		// WebSocket endpoints (static route must come before dynamic route)
		api.GET("/ws/presence", wsHub.HandlePresenceWebSocket)
		api.GET("/ws/:chatId", wsHub.HandleChatWebSocket)

		// Authenticated routes
		authenticated := api.Group("")
		authenticated.Use(middleware.AuthMiddleware(validator))
		{
			// Chat routes
			authenticated.POST("", chatHandler.CreateChat)
			authenticated.GET("/my", chatHandler.GetMyChats)
			authenticated.GET("/workspace/:workspaceId", chatHandler.GetWorkspaceChats)
			authenticated.GET("/:chatId", chatHandler.GetChat)
			authenticated.DELETE("/:chatId", chatHandler.DeleteChat)
			authenticated.POST("/:chatId/participants", chatHandler.AddParticipants)
			authenticated.DELETE("/:chatId/participants/:userId", chatHandler.RemoveParticipant)

			// Message routes
			authenticated.GET("/messages/:chatId", messageHandler.GetMessages)
			authenticated.POST("/messages/:chatId", messageHandler.SendMessage)
			authenticated.DELETE("/messages/:messageId", messageHandler.DeleteMessage)
			authenticated.POST("/messages/read", messageHandler.MarkMessagesAsRead)
			authenticated.GET("/messages/:chatId/unread", messageHandler.GetUnreadCount)
			authenticated.PUT("/messages/:chatId/last-read", messageHandler.UpdateLastRead)

			// Presence routes
			authenticated.GET("/presence/online", presenceHandler.GetOnlineUsers)
			authenticated.GET("/presence/status/:userId", presenceHandler.GetUserStatus)
		}
	}

	return r
}
