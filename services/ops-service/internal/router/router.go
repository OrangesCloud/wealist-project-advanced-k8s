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

	"ops-service/internal/client"
	"ops-service/internal/config"
	"ops-service/internal/domain"
	"ops-service/internal/handler"
	"ops-service/internal/metrics"
	"ops-service/internal/middleware"
	"ops-service/internal/repository"
	"ops-service/internal/service"
)

// Config holds router configuration
type Config struct {
	DB               *gorm.DB
	Logger           *zap.Logger
	JWTSecret        string
	BasePath         string
	TokenValidator   middleware.TokenValidator
	ArgoCDClient     *client.ArgoCDClient
	K8sClient        *client.K8sClient
	ArgoCDNamespace  string
	Metrics          *metrics.Metrics
	RedisClient      *redis.Client
	RateLimitConfig  config.RateLimitConfig
	ServiceName      string
	PrometheusClient *client.PrometheusClient
	PrometheusNS     string
}

// Setup sets up the router with all routes
func Setup(cfg Config) *gin.Engine {
	r := gin.New()

	// Initialize metrics if not provided
	m := cfg.Metrics
	if m == nil {
		m = metrics.New()
	}

	// Determine service name
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "ops-service"
	}

	// Middleware
	r.Use(commonmw.Recovery(cfg.Logger))
	r.Use(commonmw.OTELTracing(serviceName))
	r.Use(commonmw.LoggerWithTracing(cfg.Logger, serviceName))
	r.Use(commonmw.DefaultCORS())
	r.Use(metrics.HTTPMiddleware(m))

	// Rate limiting middleware
	if cfg.RateLimitConfig.Enabled && cfg.RedisClient != nil {
		rlConfig := ratelimit.DefaultConfig().
			WithRequestsPerMinute(cfg.RateLimitConfig.RequestsPerMinute).
			WithBurstSize(cfg.RateLimitConfig.BurstSize).
			WithKeyPrefix("rl:ops:")
		limiter := ratelimit.NewRedisRateLimiter(cfg.RedisClient, rlConfig, cfg.Logger)
		r.Use(ratelimit.MiddlewareWithLogger(limiter, ratelimit.UserKey, rlConfig, cfg.Logger))
		cfg.Logger.Info("Rate limiting middleware enabled",
			zap.Int("requests_per_minute", cfg.RateLimitConfig.RequestsPerMinute),
			zap.Int("burst_size", cfg.RateLimitConfig.BurstSize))
	}

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check routes
	healthChecker := commonhealth.NewHealthChecker(cfg.DB, cfg.RedisClient)
	healthChecker.RegisterRoutes(r, cfg.BasePath)

	// Initialize repositories
	userRepo := repository.NewPortalUserRepository(cfg.DB)
	auditRepo := repository.NewAuditLogRepository(cfg.DB)
	configRepo := repository.NewAppConfigRepository(cfg.DB)

	// Initialize services
	auditService := service.NewAuditLogService(auditRepo, cfg.Logger)
	userService := service.NewPortalUserService(userRepo, auditService, cfg.Logger)
	configService := service.NewAppConfigService(configRepo, auditService, cfg.Logger)

	// Initialize ArgoCD RBAC service
	var argoCDService *service.ArgoCDRBACService
	if cfg.K8sClient != nil {
		argoCDService = service.NewArgoCDRBACService(service.ArgoCDRBACServiceConfig{
			K8sClient:    cfg.K8sClient,
			ArgoCDClient: cfg.ArgoCDClient,
			AuditLogRepo: auditRepo,
			ArgoCDNS:     cfg.ArgoCDNamespace,
			Logger:       cfg.Logger,
		})
		cfg.Logger.Info("ArgoCD RBAC service initialized")
	}

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userService, auditService)
	userHandler := handler.NewUserHandler(userService)
	auditHandler := handler.NewAuditHandler(auditService)
	configHandler := handler.NewConfigHandler(configService)
	argoCDHandler := handler.NewArgoCDHandler(argoCDService, cfg.Logger)
	metricsHandler := handler.NewMetricsHandler(cfg.PrometheusClient, cfg.PrometheusNS, cfg.Logger)

	// API routes group
	api := r.Group(cfg.BasePath)

	// Auth middleware setup
	var authMiddleware gin.HandlerFunc
	istioJWTMode := os.Getenv("ISTIO_JWT_MODE") == "true"

	if istioJWTMode {
		parser := middleware.NewJWTParser(cfg.Logger)
		authMiddleware = middleware.IstioAuthMiddleware(parser)
		cfg.Logger.Info("Using Istio JWT mode (parse only)")
	} else if cfg.TokenValidator != nil {
		authMiddleware = middleware.AuthWithValidator(cfg.TokenValidator)
		cfg.Logger.Info("Using SmartValidator mode (full validation)")
	} else {
		cfg.Logger.Warn("No token validator configured")
		authMiddleware = func(c *gin.Context) {
			c.Next()
		}
	}

	// ============================================================
	// Public routes (no auth required)
	// ============================================================
	public := api.Group("")
	{
		// App config for clients
		public.GET("/config/active", configHandler.GetActive)
		public.GET("/config/:key", configHandler.GetByKey)
	}

	// ============================================================
	// Auth routes (requires JWT auth)
	// ============================================================
	auth := api.Group("/auth")
	auth.Use(authMiddleware)
	{
		auth.GET("/callback", authHandler.OAuthCallback)
		auth.POST("/logout", authHandler.Logout)
	}

	// ============================================================
	// User routes (requires portal user)
	// ============================================================
	users := api.Group("/users")
	users.Use(authMiddleware)
	users.Use(middleware.RequireAnyRole(userService, cfg.Logger))
	{
		users.GET("/me", userHandler.GetMe)
	}

	// ============================================================
	// Admin routes (requires admin role)
	// ============================================================
	admin := api.Group("/admin")
	admin.Use(authMiddleware)
	admin.Use(middleware.RequireAdmin(userService, cfg.Logger))
	{
		// User management
		admin.GET("/users", userHandler.GetAll)
		admin.GET("/users/:id", userHandler.GetByID)
		admin.POST("/users/invite", userHandler.InviteUser)
		admin.PUT("/users/:id/role", userHandler.UpdateRole)
		admin.POST("/users/:id/deactivate", userHandler.DeactivateUser)
		admin.POST("/users/:id/reactivate", userHandler.ReactivateUser)

		// Audit logs
		admin.GET("/audit-logs", auditHandler.List)
		admin.GET("/audit-logs/:id", auditHandler.GetByID)

		// App config management
		admin.GET("/config", configHandler.GetAll)
		admin.POST("/config", configHandler.Create)
		admin.PUT("/config/:id", configHandler.Update)
		admin.DELETE("/config/:id", configHandler.Delete)

		// ArgoCD RBAC management (admin only)
		admin.GET("/argocd/rbac", argoCDHandler.GetRBAC)
		admin.POST("/argocd/rbac/admins", argoCDHandler.AddAdmin)
		admin.DELETE("/argocd/rbac/admins/:email", argoCDHandler.RemoveAdmin)
	}

	// ============================================================
	// PM routes (requires admin or PM role)
	// ============================================================
	pm := api.Group("/pm")
	pm.Use(authMiddleware)
	pm.Use(middleware.RequireAdminOrPM(userService, cfg.Logger))
	{
		// PM can view and manage configs
		pm.GET("/config", configHandler.GetAll)
		pm.POST("/config", configHandler.Create)
		pm.PUT("/config/:id", configHandler.Update)
	}

	// ============================================================
	// Monitoring routes (all portal users)
	// ============================================================
	monitoring := api.Group("/monitoring")
	monitoring.Use(authMiddleware)
	monitoring.Use(middleware.RequireAnyRole(userService, cfg.Logger))
	{
		// ArgoCD applications
		monitoring.GET("/applications", argoCDHandler.GetApplications)

		// Prometheus metrics
		monitoring.GET("/metrics/overview", metricsHandler.GetSystemOverview)
		monitoring.GET("/metrics/services", metricsHandler.GetServiceMetrics)
		monitoring.GET("/metrics/services/:serviceName", metricsHandler.GetServiceDetail)
		monitoring.GET("/metrics/cluster", metricsHandler.GetClusterMetrics)
	}

	// ============================================================
	// PM monitoring routes (admin or PM can trigger syncs)
	// ============================================================
	pmMonitoring := api.Group("/monitoring")
	pmMonitoring.Use(authMiddleware)
	pmMonitoring.Use(middleware.RequireAdminOrPM(userService, cfg.Logger))
	{
		// Sync operations require admin or PM role
		pmMonitoring.POST("/applications/sync", argoCDHandler.SyncApplication)
	}

	// RBAC middleware for role-based routes
	_ = domain.RoleAdmin // Just to use the domain package

	return r
}
