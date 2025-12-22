# =============================================================================
# OIDC/IAM Configuration - Outputs
# =============================================================================

# -----------------------------------------------------------------------------
# Frontend Role (선택적)
# -----------------------------------------------------------------------------
output "frontend_role_arn" {
  description = "ARN of the IAM role for frontend (set as AWS_ROLE_ARN_FRONTEND in GitHub Secrets)"
  value       = var.enable_frontend_role ? module.github_oidc_frontend[0].role_arn : null
}

output "frontend_role_name" {
  description = "Name of the frontend IAM role"
  value       = var.enable_frontend_role ? module.github_oidc_frontend[0].role_name : null
}

# -----------------------------------------------------------------------------
# Backend Role
# -----------------------------------------------------------------------------
output "backend_role_arn" {
  description = "ARN of the IAM role for backend (set as AWS_ROLE_ARN_BACKEND in GitHub Secrets)"
  value       = module.github_oidc_backend.role_arn
}

output "backend_role_name" {
  description = "Name of the backend IAM role"
  value       = module.github_oidc_backend.role_name
}

# -----------------------------------------------------------------------------
# Shared
# -----------------------------------------------------------------------------
output "oidc_provider_arn" {
  description = "ARN of the GitHub OIDC Provider"
  value       = module.github_oidc_backend.oidc_provider_arn
}
