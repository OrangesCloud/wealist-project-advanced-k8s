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
	"storage-service/internal/client"
	"storage-service/internal/config"
	"storage-service/internal/handler"
	"storage-service/internal/metrics"
	"storage-service/internal/middleware"
	"storage-service/internal/repository"
	"storage-service/internal/service"
)

// Config holds router configuration
type Config struct {
	DB              *gorm.DB
	Logger          *zap.Logger
	JWTSecret       string
	BasePath        string
	S3Client        *client.S3Client
	TokenValidator  middleware.TokenValidator // 공통 모듈의 TokenValidator 인터페이스 사용
	UserClient      client.UserClient
	Metrics         *metrics.Metrics
	RedisClient     *redis.Client
	RateLimitConfig config.RateLimitConfig
}

// Setup sets up the router with all routes
func Setup(cfg Config) *gin.Engine {
	r := gin.New()

	// Initialize metrics if not provided
	m := cfg.Metrics
	if m == nil {
		m = metrics.New()
	}

	// Middleware (using common package)
	r.Use(commonmw.Recovery(cfg.Logger))
	r.Use(commonmw.Logger(cfg.Logger))
	r.Use(commonmw.DefaultCORS())
	r.Use(metrics.HTTPMiddleware(m))

	// Rate limiting middleware
	if cfg.RateLimitConfig.Enabled && cfg.RedisClient != nil {
		rlConfig := ratelimit.DefaultConfig().
			WithRequestsPerMinute(cfg.RateLimitConfig.RequestsPerMinute).
			WithBurstSize(cfg.RateLimitConfig.BurstSize).
			WithKeyPrefix("rl:storage:")
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
	healthChecker := commonhealth.NewHealthChecker(cfg.DB, cfg.RedisClient)
	healthChecker.RegisterRoutes(r, cfg.BasePath)

	// Initialize repositories
	folderRepo := repository.NewFolderRepository(cfg.DB)
	fileRepo := repository.NewFileRepository(cfg.DB)
	shareRepo := repository.NewShareRepository(cfg.DB)
	projectRepo := repository.NewProjectRepository(cfg.DB)

	// Initialize services
	// 각 서비스에 필요한 의존성 주입
	folderService := service.NewFolderService(folderRepo, fileRepo, cfg.Logger)
	fileService := service.NewFileService(fileRepo, folderRepo, cfg.S3Client, cfg.Logger, m) // 메트릭 포함
	shareService := service.NewShareService(shareRepo, fileRepo, folderRepo, cfg.Logger)
	projectService := service.NewProjectService(projectRepo, cfg.UserClient, cfg.Logger)
	accessService := service.NewAccessService(projectRepo, fileRepo, folderRepo, cfg.UserClient, cfg.Logger)

	// Initialize handlers
	folderHandler := handler.NewFolderHandler(folderService, fileService, accessService)
	fileHandler := handler.NewFileHandler(fileService, accessService)
	shareHandler := handler.NewShareHandler(shareService)
	projectHandler := handler.NewProjectHandler(projectService)

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
	// Storage routes (authenticated)
	// ============================================================
	storage := api.Group("/storage")
	storage.Use(authMiddleware)
	{
		// ============================================================
		// Folder routes
		// ============================================================
		folders := storage.Group("/folders")
		{
			folders.POST("", folderHandler.CreateFolder)
			folders.GET("/contents", folderHandler.GetFolderContents)
			folders.GET("/:folderId", folderHandler.GetFolder)
			folders.PUT("/:folderId", folderHandler.UpdateFolder)
			folders.DELETE("/:folderId", folderHandler.DeleteFolder)
			folders.POST("/:folderId/restore", folderHandler.RestoreFolder)
			folders.DELETE("/:folderId/permanent", folderHandler.PermanentDeleteFolder)

			// Folder shares
			folders.GET("/:folderId/shares", shareHandler.GetFolderShares)
		}

		// ============================================================
		// File routes
		// ============================================================
		files := storage.Group("/files")
		{
			files.POST("/upload-url", fileHandler.GenerateUploadURL)
			files.POST("/confirm", fileHandler.ConfirmUpload)
			files.GET("/:fileId", fileHandler.GetFile)
			files.GET("/:fileId/download", fileHandler.GetDownloadURL)
			files.PUT("/:fileId", fileHandler.UpdateFile)
			files.DELETE("/:fileId", fileHandler.DeleteFile)
			files.POST("/:fileId/restore", fileHandler.RestoreFile)
			files.DELETE("/:fileId/permanent", fileHandler.PermanentDeleteFile)

			// File shares
			files.GET("/:fileId/shares", shareHandler.GetFileShares)
		}

		// ============================================================
		// Share routes
		// ============================================================
		shares := storage.Group("/shares")
		{
			shares.POST("", shareHandler.CreateShare)
			shares.GET("/link/:link", shareHandler.GetShareByLink)
			shares.PUT("/:shareId", shareHandler.UpdateShare)
			shares.DELETE("/:shareId", shareHandler.DeleteShare)
		}

		// Shared with me
		storage.GET("/shared-with-me", shareHandler.GetSharedWithMe)

		// ============================================================
		// Project routes
		// ============================================================
		projects := storage.Group("/projects")
		{
			projects.POST("", projectHandler.CreateProject)
			projects.GET("/:projectId", projectHandler.GetProject)
			projects.PUT("/:projectId", projectHandler.UpdateProject)
			projects.DELETE("/:projectId", projectHandler.DeleteProject)

			// Project members
			projects.POST("/:projectId/members", projectHandler.AddMember)
			projects.GET("/:projectId/members", projectHandler.GetMembers)
			projects.PUT("/:projectId/members/:userId", projectHandler.UpdateMember)
			projects.DELETE("/:projectId/members/:userId", projectHandler.RemoveMember)
		}

		// ============================================================
		// Workspace routes
		// ============================================================
		workspaces := storage.Group("/workspaces")
		{
			// Projects in workspace
			workspaces.GET("/:workspaceId/projects", projectHandler.GetWorkspaceProjects)

			workspaces.GET("/:workspaceId/folders", folderHandler.GetWorkspaceFolders)
			workspaces.GET("/:workspaceId/files", fileHandler.GetWorkspaceFiles)
			workspaces.GET("/:workspaceId/files/search", fileHandler.SearchFiles)
			workspaces.GET("/:workspaceId/usage", fileHandler.GetStorageUsage)

			// Trash
			workspaces.GET("/:workspaceId/trash/folders", folderHandler.GetTrashFolders)
			workspaces.GET("/:workspaceId/trash/files", fileHandler.GetTrashFiles)
		}
	}

	// ============================================================
	// Public routes (no auth required for shared links)
	// ============================================================
	public := api.Group("/public/storage")
	{
		public.GET("/shares/link/:link", shareHandler.GetShareByLink)
	}

	return r
}
