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
	"github.com/OrangesCloud/wealist-advanced-go-pkg/ratelimit"
)

// RouterConfig holds router configuration
type RouterConfig struct {
	Config      *config.Config
	DB          *gorm.DB
	RedisClient *redis.Client
	Logger      *zap.Logger
	ServiceName string // Service name for OTEL tracing
}

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
		serviceName = "chat-service"
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
			WithKeyPrefix("rl:chat:")
		limiter := ratelimit.NewRedisRateLimiter(redisClient, rlConfig, logger)
		r.Use(ratelimit.MiddlewareWithLogger(limiter, ratelimit.UserKey, rlConfig, logger))
		logger.Info("Rate limiting middleware enabled",
			zap.Int("requests_per_minute", cfg.RateLimit.RequestsPerMinute),
			zap.Int("burst_size", cfg.RateLimit.BurstSize))
	} else if cfg.RateLimit.Enabled && redisClient == nil {
		logger.Warn("Rate limiting enabled but Redis is not available, skipping")
	}

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

	// Initialize auth middleware based on ISTIO_JWT_MODE
	var authMiddleware gin.HandlerFunc
	var wsValidator middleware.TokenValidator

	if cfg.Auth.IstioJWTMode {
		// K8s + Istio 환경: Istio가 JWT 검증, Go 서비스는 파싱만
		parser := middleware.NewJWTParser(logger)
		authMiddleware = middleware.IstioAuthMiddleware(parser)
		// WebSocket용 validator는 SmartValidator 사용 (WebSocket은 Istio를 통하지 않을 수 있음)
		wsValidator = middleware.NewSmartValidator(cfg.Auth.ServiceURL, cfg.Auth.JWTIssuer, logger)
		logger.Info("Using Istio JWT mode (parse only)",
			zap.String("auth_service_url", cfg.Auth.ServiceURL))
	} else {
		// Docker Compose / K8s without Istio: SmartValidator로 전체 검증
		validator := middleware.NewSmartValidator(cfg.Auth.ServiceURL, cfg.Auth.JWTIssuer, logger)
		authMiddleware = middleware.AuthMiddleware(validator)
		wsValidator = validator
		logger.Info("Using SmartValidator mode (full validation)",
			zap.String("auth_service_url", cfg.Auth.ServiceURL),
			zap.String("jwt_issuer", cfg.Auth.JWTIssuer))
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(chatService, presenceService, wsValidator, redisClient, logger)

	// Initialize S3 client (optional, for file uploads)
	var fileHandler *handler.FileHandler
	if cfg.S3.Bucket != "" && cfg.S3.Region != "" {
		s3Client, err := client.NewS3Client(&cfg.S3)
		if err != nil {
			logger.Warn("Failed to initialize S3 client, file upload will be disabled", zap.Error(err))
		} else {
			fileHandler = handler.NewFileHandler(s3Client, logger)
			logger.Info("S3 client initialized for file uploads")
		}
	} else {
		logger.Warn("S3 configuration not provided, file upload will be disabled")
	}

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
		authenticated.Use(authMiddleware)
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

			// File upload routes (only if S3 is configured)
			if fileHandler != nil {
				authenticated.POST("/files/presigned-url", fileHandler.GeneratePresignedURL)
			}
		}
	}

	return r
}
