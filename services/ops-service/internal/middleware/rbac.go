package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"ops-service/internal/domain"
	"ops-service/internal/response"
)

// PortalUserGetter interface for getting portal user by email
type PortalUserGetter interface {
	GetByEmail(email string) (*domain.PortalUser, error)
}

// RBACMiddleware creates middleware that checks if user has required role
func RBACMiddleware(userGetter PortalUserGetter, logger *zap.Logger, requiredRoles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		email := GetEmail(c)
		if email == "" {
			response.Unauthorized(c, "Email not found in token")
			c.Abort()
			return
		}

		user, err := userGetter.GetByEmail(email)
		if err != nil {
			logger.Warn("Portal user not found", zap.String("email", email), zap.Error(err))
			response.Forbidden(c, "Access denied: Not a portal user")
			c.Abort()
			return
		}

		if !user.IsActive {
			response.Forbidden(c, "Access denied: User is inactive")
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, role := range requiredRoles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			response.Forbidden(c, "Access denied: Insufficient permissions")
			c.Abort()
			return
		}

		// Set portal user in context
		c.Set("portalUser", user)
		c.Set("portalUserRole", user.Role)

		c.Next()
	}
}

// RequireAdmin requires admin role
func RequireAdmin(userGetter PortalUserGetter, logger *zap.Logger) gin.HandlerFunc {
	return RBACMiddleware(userGetter, logger, domain.RoleAdmin)
}

// RequireAdminOrPM requires admin or PM role
func RequireAdminOrPM(userGetter PortalUserGetter, logger *zap.Logger) gin.HandlerFunc {
	return RBACMiddleware(userGetter, logger, domain.RoleAdmin, domain.RolePM)
}

// RequireAnyRole requires any valid role (authenticated portal user)
func RequireAnyRole(userGetter PortalUserGetter, logger *zap.Logger) gin.HandlerFunc {
	return RBACMiddleware(userGetter, logger, domain.RoleAdmin, domain.RolePM, domain.RoleViewer)
}

// GetPortalUser gets portal user from context
func GetPortalUser(c *gin.Context) *domain.PortalUser {
	if user, exists := c.Get("portalUser"); exists {
		return user.(*domain.PortalUser)
	}
	return nil
}

// GetPortalUserRole gets portal user role from context
func GetPortalUserRole(c *gin.Context) domain.Role {
	if role, exists := c.Get("portalUserRole"); exists {
		return role.(domain.Role)
	}
	return ""
}
