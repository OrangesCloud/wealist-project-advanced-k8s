// Package router provides HTTP routing configuration.
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
	"project-board-api/internal/client"
	"project-board-api/internal/config"
	"project-board-api/internal/converter"
	"project-board-api/internal/database"
	"project-board-api/internal/handler"
	"project-board-api/internal/metrics"
	"project-board-api/internal/middleware"
	"project-board-api/internal/repository"
	"project-board-api/internal/service"
)

type Config struct {
	DB                 *gorm.DB
	Logger             *zap.Logger
	JWTSecret          string // Deprecated: Use AuthServiceURL + JWTIssuer instead
	AuthServiceURL     string // auth-service URL for SmartValidator
	JWTIssuer          string // JWT issuer for JWKS validation
	UserClient         client.UserClient
	BasePath           string
	UserServiceBaseURL string
	Metrics            *metrics.Metrics
	S3Client           *client.S3Client
	RedisClient        *redis.Client
	RateLimitConfig    config.RateLimitConfig
	ServiceName        string // Service name for tracing (default: "board-service")
}

// Setup initializes the router with all dependencies and routes.
func Setup(cfg Config) *gin.Engine {
	// Create Gin router
	router := gin.New()

	// Determine service name for tracing
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "board-service"
	}

	// Apply global middleware chain (using common package)
	router.Use(
		commonmw.Recovery(cfg.Logger),                       // 1. Panic recovery
		commonmw.OTELTracing(serviceName),                   // 2. OpenTelemetry HTTP tracing (otelgin)
		commonmw.LoggerWithTracing(cfg.Logger, serviceName), // 3. Request logging with trace context
		commonmw.DefaultCORS(),                              // 4. CORS configuration (includes X-Workspace-Id)
	)

	// Add metrics middleware if metrics is configured
	if cfg.Metrics != nil {
		router.Use(middleware.Metrics(cfg.Metrics))
		cfg.Logger.Info("Metrics middleware enabled")
	}

	// Add rate limiting middleware if enabled and Redis is available
	if cfg.RateLimitConfig.Enabled && cfg.RedisClient != nil {
		rlConfig := ratelimit.DefaultConfig().
			WithRequestsPerMinute(cfg.RateLimitConfig.RequestsPerMinute).
			WithBurstSize(cfg.RateLimitConfig.BurstSize).
			WithKeyPrefix("rl:board:")
		limiter := ratelimit.NewRedisRateLimiter(cfg.RedisClient, rlConfig, cfg.Logger)
		router.Use(ratelimit.MiddlewareWithLogger(limiter, ratelimit.UserKey, rlConfig, cfg.Logger))
		cfg.Logger.Info("Rate limiting middleware enabled",
			zap.Int("requests_per_minute", cfg.RateLimitConfig.RequestsPerMinute),
			zap.Int("burst_size", cfg.RateLimitConfig.BurstSize))
	} else if cfg.RateLimitConfig.Enabled && cfg.RedisClient == nil {
		cfg.Logger.Warn("Rate limiting enabled but Redis is not available, skipping")
	}

	// Initialize repositories
	projectRepo := repository.NewProjectRepository(cfg.DB)
	boardRepo := repository.NewBoardRepository(cfg.DB)
	participantRepo := repository.NewParticipantRepository(cfg.DB)
	commentRepo := repository.NewCommentRepository(cfg.DB)
	fieldOptionRepo := repository.NewFieldOptionRepository(cfg.DB)
	attachmentRepo := repository.NewAttachmentRepository(cfg.DB)

	// Initialize converters
	fieldOptionConverter := converter.NewFieldOptionConverter(fieldOptionRepo)

	// Initialize services with repository dependencies
	projectService := service.NewProjectService(projectRepo, fieldOptionRepo, attachmentRepo, cfg.S3Client, cfg.UserClient, cfg.Metrics, cfg.Logger)
	boardService := service.NewBoardService(boardRepo, projectRepo, fieldOptionRepo, participantRepo, attachmentRepo, cfg.S3Client, fieldOptionConverter, cfg.Metrics, cfg.Logger)
	participantService := service.NewParticipantService(participantRepo, boardRepo)
	commentService := service.NewCommentService(commentRepo, boardRepo, attachmentRepo, cfg.S3Client, cfg.Logger)
	fieldOptionService := service.NewFieldOptionService(fieldOptionRepo)
	projectMemberService := service.NewProjectMemberService(projectRepo, cfg.UserClient)
	projectJoinRequestService := service.NewProjectJoinRequestService(projectRepo, cfg.UserClient)

	// Initialize handlers with service dependencies
	projectHandler := handler.NewProjectHandler(projectService)
	boardHandler := handler.NewBoardHandler(boardService)
	participantHandler := handler.NewParticipantHandler(participantService)
	commentHandler := handler.NewCommentHandler(commentService)
	fieldOptionHandler := handler.NewFieldOptionHandler(fieldOptionService)
	projectMemberHandler := handler.NewProjectMemberHandler(projectMemberService)
	projectJoinRequestHandler := handler.NewProjectJoinRequestHandler(projectJoinRequestService)
	attachmentHandler := handler.NewAttachmentHandler(cfg.S3Client, attachmentRepo)

	// 💡 WebSocket Handler 초기화
	wsHandler := handler.NewWSHandler(cfg.Logger, cfg.UserClient)

	// Create base path group if configured
	var baseGroup *gin.RouterGroup
	if cfg.BasePath != "" {
		baseGroup = router.Group(cfg.BasePath)
		cfg.Logger.Info("Base path configured for ALB routing", zap.String("base_path", cfg.BasePath))
	} else {
		baseGroup = router.Group("")
		cfg.Logger.Info("No base path configured, using root path")
	}

	// Health check endpoints using common package
	healthChecker := commonhealth.NewHealthChecker(cfg.DB, database.GetRedis())
	healthChecker.RegisterRoutes(router, cfg.BasePath)

	// Swagger documentation endpoint - temporarily disabled for CI compatibility
	// TODO: Re-enable after upgrading gin-swagger to resolve genproto conflict
	// baseGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Metrics endpoint (no authentication required)
	// Add metrics endpoint at root level for compatibility
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Also add metrics endpoint under base path if configured
	if cfg.BasePath != "" {
		baseGroup.GET("/metrics", gin.WrapH(promhttp.Handler()))
		cfg.Logger.Info("Metrics endpoint configured at both root and base path",
			zap.String("root_path", "/metrics"),
			zap.String("base_path", cfg.BasePath+"/metrics"))
	} else {
		cfg.Logger.Info("Metrics endpoint configured at root path", zap.String("path", "/metrics"))
	}

	// Initialize auth middleware based on ISTIO_JWT_MODE
	var authMiddleware gin.HandlerFunc
	istioJWTMode := os.Getenv("ISTIO_JWT_MODE") == "true"

	if istioJWTMode {
		// K8s + Istio 환경: Istio가 JWT 검증, Go 서비스는 파싱만
		parser := middleware.NewJWTParser(cfg.Logger)
		authMiddleware = middleware.IstioAuthMiddleware(parser)
		cfg.Logger.Info("Using Istio JWT mode (parse only)",
			zap.String("auth_service_url", cfg.AuthServiceURL))
	} else if cfg.AuthServiceURL != "" {
		// Docker Compose / K8s without Istio: SmartValidator로 전체 검증
		tokenValidator := middleware.NewSmartValidator(cfg.AuthServiceURL, cfg.JWTIssuer, cfg.Logger)
		authMiddleware = middleware.AuthWithValidator(tokenValidator)
		cfg.Logger.Info("Using SmartValidator mode (full validation)",
			zap.String("auth_service_url", cfg.AuthServiceURL),
			zap.String("jwt_issuer", cfg.JWTIssuer))
	}

	// Setup API routes
	setupRoutes(baseGroup, authMiddleware, projectHandler, boardHandler, participantHandler, commentHandler, fieldOptionHandler, projectMemberHandler, projectJoinRequestHandler, attachmentHandler)

	// 🔥 [중요] WebSocket은 baseGroup에 직접 등록 (chat-service와 동일한 패턴)
	// basePath가 /api/boards일 때: /api/boards/ws/project/:projectId
	baseGroup.GET("/ws/project/:projectId", wsHandler.HandleWebSocket)

	return router
}

// setupRoutes configures all API routes
func setupRoutes(
	baseGroup *gin.RouterGroup,
	authMiddleware gin.HandlerFunc,
	projectHandler *handler.ProjectHandler,
	boardHandler *handler.BoardHandler,
	participantHandler *handler.ParticipantHandler,
	commentHandler *handler.CommentHandler,
	fieldOptionHandler *handler.FieldOptionHandler,
	projectMemberHandler *handler.ProjectMemberHandler,
	projectJoinRequestHandler *handler.ProjectJoinRequestHandler,
	attachmentHandler *handler.AttachmentHandler,
) {
	// API group with authentication
	api := baseGroup.Group("/api")
	if authMiddleware != nil {
		api.Use(authMiddleware)
	}
	{
		// Project routes
		projects := api.Group("/projects")
		{
			// Frontend compatibility route (query parameter style)
			projects.GET("", projectHandler.GetProjectsByWorkspaceQuery)

			// Existing routes
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/workspace/:workspaceId", projectHandler.GetProjectsByWorkspace)
			projects.GET("/workspace/:workspaceId/default", projectHandler.GetDefaultProject)

			// New project management extension routes
			projects.GET("/search", projectHandler.SearchProjects)
			projects.GET("/:projectId", projectHandler.GetProject)
			projects.PUT("/:projectId", projectHandler.UpdateProject)
			projects.DELETE("/:projectId", projectHandler.DeleteProject)
			projects.GET("/:projectId/init-settings", projectHandler.GetProjectInitSettings)

			// Project member routes
			projects.GET("/:projectId/members", projectMemberHandler.GetMembers)
			projects.DELETE("/:projectId/members/:memberId", projectMemberHandler.RemoveMember)
			projects.PUT("/:projectId/members/:memberId/role", projectMemberHandler.UpdateMemberRole)

			// Project join request routes
			projects.GET("/:projectId/join-requests", projectJoinRequestHandler.GetJoinRequests)

			// Attachment routes for projects
			projects.GET("/:projectId/attachments", attachmentHandler.GetProjectAttachments)
		}

		// Join request routes (not nested under project)
		joinRequests := api.Group("/join-requests")
		{
			joinRequests.POST("", projectJoinRequestHandler.CreateJoinRequest)
			joinRequests.PUT("/:joinRequestId", projectJoinRequestHandler.UpdateJoinRequest)
		}

		// Board routes
		boards := api.Group("/boards")
		{
			// Frontend compatibility route (query parameter style)
			boards.GET("", boardHandler.GetBoardsByProjectQuery)

			boards.POST("", boardHandler.CreateBoard)
			boards.GET("/:boardId", boardHandler.GetBoard)
			boards.GET("/project/:projectId", boardHandler.GetBoardsByProject)
			boards.PUT("/:boardId", boardHandler.UpdateBoard)
			boards.DELETE("/:boardId", boardHandler.DeleteBoard)
			boards.PUT("/:boardId/move", boardHandler.MoveBoard) // ✅ 이 라인 추가

			// Attachment routes for boards
			boards.GET("/:boardId/attachments", attachmentHandler.GetBoardAttachments)
		}

		// Participant routes
		participants := api.Group("/participants")
		{
			participants.POST("", participantHandler.AddParticipants)
			participants.GET("/board/:boardId", participantHandler.GetParticipants)
			participants.DELETE("/board/:boardId/user/:userId", participantHandler.RemoveParticipant)
		}

		// Comment routes
		comments := api.Group("/comments")
		{
			// Frontend compatibility route (query parameter style)
			comments.GET("", commentHandler.GetCommentsByQuery)

			comments.POST("", commentHandler.CreateComment)
			comments.GET("/board/:boardId", commentHandler.GetComments)
			comments.PUT("/:commentId", commentHandler.UpdateComment)
			comments.DELETE("/:commentId", commentHandler.DeleteComment)

			// Attachment routes for comments
			comments.GET("/:commentId/attachments", attachmentHandler.GetCommentAttachments)
		}

		// Field option routes
		fieldOptions := api.Group("/field-options")
		{
			fieldOptions.GET("", fieldOptionHandler.GetFieldOptions)
			fieldOptions.POST("", fieldOptionHandler.CreateFieldOption)
			fieldOptions.PATCH("/:optionId", fieldOptionHandler.UpdateFieldOption)
			fieldOptions.DELETE("/:optionId", fieldOptionHandler.DeleteFieldOption)
		}

		// Attachment routes (Presigned URL approach)
		attachments := api.Group("/attachments")
		{
			// Generate presigned URL for direct S3 upload
			attachments.POST("/presigned-url", attachmentHandler.GeneratePresignedURL)
			// Save attachment metadata after successful S3 upload
			attachments.POST("", attachmentHandler.SaveAttachmentMetadata)
			// Delete attachment
			attachments.DELETE("/:attachmentId", attachmentHandler.DeleteAttachment)
		}
	}
}
