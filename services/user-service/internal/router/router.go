package router

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	commonhealth "github.com/OrangesCloud/wealist-advanced-go-pkg/health"
	commonmw "github.com/OrangesCloud/wealist-advanced-go-pkg/middleware"
	"github.com/OrangesCloud/wealist-advanced-go-pkg/ratelimit"
	"user-service/internal/client"
	"user-service/internal/config"
	"user-service/internal/handler"
	"user-service/internal/metrics"
	"user-service/internal/middleware"
	"user-service/internal/repository"
	"user-service/internal/service"
)

// Config holds router configuration
type Config struct {
	DB              *gorm.DB
	Logger          *zap.Logger
	JWTSecret       string
	BasePath        string
	S3Client        *client.S3Client
	TokenValidator  middleware.TokenValidator // 공통 모듈의 TokenValidator 인터페이스 사용
	Metrics         *metrics.Metrics
	RedisClient     *redis.Client
	RateLimitConfig config.RateLimitConfig
	ServiceName     string // Service name for OTEL tracing
}

// Setup sets up the router with all routes
func Setup(cfg Config) *gin.Engine {
	r := gin.New()

	// Initialize metrics if not provided
	m := cfg.Metrics
	if m == nil {
		m = metrics.New()
	}

	// Determine service name for tracing
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "user-service"
	}

	// Middleware (using common package with OTEL tracing)
	r.Use(commonmw.Recovery(cfg.Logger))
	r.Use(commonmw.OTELTracing(serviceName))                   // OpenTelemetry HTTP tracing (otelgin)
	r.Use(commonmw.LoggerWithTracing(cfg.Logger, serviceName)) // Request logging with trace context
	r.Use(commonmw.DefaultCORS())
	r.Use(metrics.HTTPMiddleware(m))

	// Rate limiting middleware
	if cfg.RateLimitConfig.Enabled && cfg.RedisClient != nil {
		rlConfig := ratelimit.DefaultConfig().
			WithRequestsPerMinute(cfg.RateLimitConfig.RequestsPerMinute).
			WithBurstSize(cfg.RateLimitConfig.BurstSize).
			WithKeyPrefix("rl:user:")
		limiter := ratelimit.NewRedisRateLimiter(cfg.RedisClient, rlConfig, cfg.Logger)
		r.Use(ratelimit.MiddlewareWithLogger(limiter, ratelimit.UserKey, rlConfig, cfg.Logger))
		cfg.Logger.Info("Rate limiting middleware enabled",
			zap.Int("requests_per_minute", cfg.RateLimitConfig.RequestsPerMinute),
			zap.Int("burst_size", cfg.RateLimitConfig.BurstSize))
	} else if cfg.RateLimitConfig.Enabled && cfg.RedisClient == nil {
		cfg.Logger.Warn("Rate limiting enabled but Redis is not available, skipping")
	}

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check routes (using common package)
	healthChecker := commonhealth.NewHealthChecker(cfg.DB, nil)
	healthChecker.RegisterRoutes(r, cfg.BasePath)

	// Swagger documentation (disabled for faster builds)
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Initialize repositories
	userRepo := repository.NewUserRepository(cfg.DB)
	workspaceRepo := repository.NewWorkspaceRepository(cfg.DB)
	memberRepo := repository.NewWorkspaceMemberRepository(cfg.DB)
	profileRepo := repository.NewUserProfileRepository(cfg.DB)
	joinReqRepo := repository.NewJoinRequestRepository(cfg.DB)
	attachmentRepo := repository.NewAttachmentRepository(cfg.DB)

	// Initialize services
	// 사용자 서비스 초기화 (메트릭 포함)
	userService := service.NewUserService(userRepo, cfg.Logger, m)
	// 워크스페이스 서비스 초기화 (메트릭 포함)
	workspaceService := service.NewWorkspaceService(
		workspaceRepo,
		memberRepo,
		joinReqRepo,
		profileRepo,
		userRepo,
		cfg.Logger,
		m,
	)
	// 프로필 서비스 초기화 (메트릭 포함)
	profileService := service.NewProfileService(profileRepo, memberRepo, userRepo, cfg.Logger, m)
	attachmentService := service.NewAttachmentService(attachmentRepo, cfg.S3Client, cfg.Logger)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceService)
	profileHandler := handler.NewProfileHandler(profileService, attachmentService)

	// API routes group
	api := r.Group(cfg.BasePath)

	// Auth middleware - check ISTIO_JWT_MODE first
	var authMiddleware gin.HandlerFunc
	istioJWTMode := os.Getenv("ISTIO_JWT_MODE") == "true"

	if istioJWTMode {
		// K8s + Istio 환경: Istio가 JWT 검증, Go 서비스는 파싱만
		parser := middleware.NewJWTParser(cfg.Logger)
		authMiddleware = middleware.IstioAuthMiddleware(parser)
		cfg.Logger.Info("Using Istio JWT mode (parse only)")
	} else if cfg.TokenValidator != nil {
		// Docker Compose / K8s without Istio: SmartValidator로 전체 검증
		authMiddleware = middleware.AuthWithValidator(cfg.TokenValidator)
		cfg.Logger.Info("Using SmartValidator mode (full validation)")
	} else {
		// Fallback: 로컬 JWT 검증
		authMiddleware = middleware.Auth(cfg.JWTSecret)
		cfg.Logger.Info("Using local JWT validation")
	}

	// ============================================================
	// Internal routes (no auth required for service-to-service)
	// ============================================================
	internal := api.Group("/internal")
	{
		internal.GET("/users/:userId/exists", userHandler.UserExists)
		internal.POST("/oauth/login", userHandler.OAuthLogin)
	}

	// ============================================================
	// User routes
	// ============================================================
	users := api.Group("/users")
	{
		users.POST("", userHandler.CreateUser) // Public for OAuth callback
		users.GET("/me", authMiddleware, userHandler.GetMe)
		users.DELETE("/me", authMiddleware, userHandler.DeleteMe)
		users.GET("/:userId", authMiddleware, userHandler.GetUser)
		users.PUT("/:userId", authMiddleware, userHandler.UpdateUser)
		users.PUT("/:userId/restore", authMiddleware, userHandler.RestoreUser)
	}

	// ============================================================
	// Workspace routes
	// ============================================================
	workspaces := api.Group("/workspaces")
	workspaces.Use(authMiddleware)
	{
		workspaces.POST("/create", workspaceHandler.CreateWorkspace)
		workspaces.GET("/all", workspaceHandler.GetAllWorkspaces)
		workspaces.GET("/public/:workspaceName", workspaceHandler.SearchPublicWorkspaces)
		workspaces.GET("/:workspaceId", workspaceHandler.GetWorkspace)
		workspaces.PUT("/ids/:workspaceId", workspaceHandler.UpdateWorkspace)
		workspaces.DELETE("/:workspaceId", workspaceHandler.DeleteWorkspace)
		workspaces.POST("/default", workspaceHandler.SetDefaultWorkspace)

		// Workspace settings
		workspaces.GET("/:workspaceId/settings", workspaceHandler.GetWorkspaceSettings)
		workspaces.PUT("/:workspaceId/settings", workspaceHandler.UpdateWorkspaceSettings)

		// Workspace members
		workspaces.GET("/:workspaceId/members", workspaceHandler.GetMembers)
		workspaces.POST("/:workspaceId/members/invite", workspaceHandler.InviteMember)
		workspaces.PUT("/:workspaceId/members/:memberId/role", workspaceHandler.UpdateMemberRole)
		workspaces.DELETE("/:workspaceId/members/:memberId", workspaceHandler.RemoveMember)
		workspaces.GET("/:workspaceId/validate-member/:userId", workspaceHandler.ValidateMember)

		// Join requests
		workspaces.POST("/join-requests", workspaceHandler.CreateJoinRequest)
		workspaces.GET("/:workspaceId/joinRequests", workspaceHandler.GetJoinRequests)
		workspaces.GET("/:workspaceId/pendingMembers", workspaceHandler.GetJoinRequests) // Alias for frontend compatibility
		workspaces.PUT("/:workspaceId/joinRequests/:requestId", workspaceHandler.ProcessJoinRequest)
	}

	// ============================================================
	// Profile routes
	// ============================================================
	profiles := api.Group("/profiles")
	profiles.Use(authMiddleware)
	{
		profiles.POST("", profileHandler.CreateProfile)
		profiles.GET("/me", profileHandler.GetMyProfile)
		profiles.GET("/all/me", profileHandler.GetAllMyProfiles)
		profiles.PUT("/me", profileHandler.UpdateProfile)
		profiles.GET("/workspace/:workspaceId/user/:userId", profileHandler.GetUserProfile)
		profiles.DELETE("/workspace/:workspaceId", profileHandler.DeleteProfile)

		// Profile image upload
		profiles.POST("/me/image/presigned-url", profileHandler.GeneratePresignedURL)
		profiles.POST("/me/image/attachment", profileHandler.SaveAttachment)
		profiles.PUT("/me/image", profileHandler.ConfirmProfileImage)
	}

	return r
}
