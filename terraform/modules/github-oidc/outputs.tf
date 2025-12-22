# =============================================================================
# GitHub OIDC Module - Outputs
# =============================================================================

output "oidc_provider_arn" {
  description = "ARN of the GitHub OIDC Provider"
  value       = local.oidc_provider_arn
}

output "role_arn" {
  description = "ARN of the IAM role for GitHub Actions"
  value       = aws_iam_role.github_actions.arn
}

output "role_name" {
  description = "Name of the IAM role for GitHub Actions"
  value       = aws_iam_role.github_actions.name
}
