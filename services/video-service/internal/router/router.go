package router

import (
	"video-service/internal/client"
	"video-service/internal/config"
	"video-service/internal/handler"
	"video-service/internal/metrics"
	"video-service/internal/middleware"
	"video-service/internal/repository"
	"video-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	commonhealth "github.com/OrangesCloud/wealist-advanced-go-pkg/health"
	commonmw "github.com/OrangesCloud/wealist-advanced-go-pkg/middleware"
	"github.com/OrangesCloud/wealist-advanced-go-pkg/ratelimit"
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

	// Rate limiting middleware
	if cfg.RateLimit.Enabled && redisClient != nil {
		rlConfig := ratelimit.DefaultConfig().
			WithRequestsPerMinute(cfg.RateLimit.RequestsPerMinute).
			WithBurstSize(cfg.RateLimit.BurstSize).
			WithKeyPrefix("rl:video:")
		limiter := ratelimit.NewRedisRateLimiter(redisClient, rlConfig, logger)
		r.Use(ratelimit.MiddlewareWithLogger(limiter, ratelimit.UserKey, rlConfig, logger))
		logger.Info("Rate limiting middleware enabled",
			zap.Int("requests_per_minute", cfg.RateLimit.RequestsPerMinute),
			zap.Int("burst_size", cfg.RateLimit.BurstSize))
	} else if cfg.RateLimit.Enabled && redisClient == nil {
		logger.Warn("Rate limiting enabled but Redis is not available, skipping")
	}

	// Initialize repositories
	roomRepo := repository.NewRoomRepository(db)

	// Initialize user client for workspace validation
	var userClient client.UserClient
	if cfg.UserAPI.BaseURL != "" {
		userClient = client.NewUserClient(cfg.UserAPI.BaseURL, cfg.UserAPI.Timeout, logger)
		logger.Info("User client initialized",
			zap.String("url", cfg.UserAPI.BaseURL),
			zap.Duration("timeout", cfg.UserAPI.Timeout))
	} else {
		logger.Warn("User service URL not configured, workspace validation will be skipped")
	}

	// Initialize services
	// 룸 서비스 초기화 (메트릭 포함)
	roomService := service.NewRoomService(roomRepo, userClient, cfg.LiveKit, redisClient, logger, m)

	// Initialize auth middleware based on ISTIO_JWT_MODE
	var authMiddleware gin.HandlerFunc

	if cfg.Auth.IstioJWTMode {
		// K8s + Istio 환경: Istio가 JWT 검증, Go 서비스는 파싱만
		parser := middleware.NewJWTParser(logger)
		authMiddleware = middleware.IstioAuthMiddleware(parser)
		logger.Info("Using Istio JWT mode (parse only)",
			zap.String("auth_service_url", cfg.Auth.ServiceURL))
	} else {
		// Docker Compose / K8s without Istio: SmartValidator로 전체 검증
		validator := middleware.NewSmartValidator(cfg.Auth.ServiceURL, cfg.Auth.JWTIssuer, logger)
		authMiddleware = middleware.AuthMiddleware(validator)
		logger.Info("Using SmartValidator mode (full validation)",
			zap.String("auth_service_url", cfg.Auth.ServiceURL),
			zap.String("jwt_issuer", cfg.Auth.JWTIssuer))
	}

	// Initialize handlers
	roomHandler := handler.NewRoomHandler(roomService, logger)

	// Health check routes (using common package)
	healthChecker := commonhealth.NewHealthChecker(db, redisClient)
	healthChecker.RegisterRoutes(r, cfg.Server.BasePath)

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes with base path
	api := r.Group(cfg.Server.BasePath)
	{

		// Authenticated routes
		authenticated := api.Group("")
		authenticated.Use(authMiddleware)
		{
			// Room routes
			authenticated.POST("/rooms", roomHandler.CreateRoom)
			authenticated.GET("/rooms/workspace/:workspaceId", roomHandler.GetWorkspaceRooms)
			authenticated.GET("/rooms/:roomId", roomHandler.GetRoom)
			authenticated.POST("/rooms/:roomId/join", roomHandler.JoinRoom)
			authenticated.POST("/rooms/:roomId/leave", roomHandler.LeaveRoom)
			authenticated.POST("/rooms/:roomId/end", roomHandler.EndRoom)
			authenticated.GET("/rooms/:roomId/participants", roomHandler.GetParticipants)
			authenticated.POST("/rooms/:roomId/transcript", roomHandler.SaveTranscript)

			// Call history routes
			authenticated.GET("/history/workspace/:workspaceId", roomHandler.GetWorkspaceCallHistory)
			authenticated.GET("/history/me", roomHandler.GetMyCallHistory)
			authenticated.GET("/history/:historyId", roomHandler.GetCallHistory)
			authenticated.GET("/history/:historyId/transcript", roomHandler.GetTranscript)
		}
	}

	return r
}
