package domain

// Role represents the role of a portal user
type Role string

const (
	// RoleAdmin has full access to all features
	RoleAdmin Role = "admin"
	// RolePM has access to monitoring and feature flags
	RolePM Role = "pm"
	// RoleViewer has read-only access
	RoleViewer Role = "viewer"
)

// IsValid checks if the role is valid
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RolePM, RoleViewer:
		return true
	default:
		return false
	}
}

// CanManageUsers checks if the role can manage other users
func (r Role) CanManageUsers() bool {
	return r == RoleAdmin
}

// CanManageArgoCD checks if the role can manage ArgoCD RBAC
func (r Role) CanManageArgoCD() bool {
	return r == RoleAdmin
}

// CanViewMonitoring checks if the role can view monitoring
func (r Role) CanViewMonitoring() bool {
	return r == RoleAdmin || r == RolePM || r == RoleViewer
}

// CanManageFeatureFlags checks if the role can manage feature flags
func (r Role) CanManageFeatureFlags() bool {
	return r == RoleAdmin || r == RolePM
}

// CanSearchUsers checks if the role can search users
func (r Role) CanSearchUsers() bool {
	return r == RoleAdmin || r == RolePM
}
