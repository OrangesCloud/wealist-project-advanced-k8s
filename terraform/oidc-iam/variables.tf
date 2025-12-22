# =============================================================================
# OIDC/IAM Configuration - Variables
# =============================================================================

variable "aws_account_id" {
  description = "AWS Account ID"
  type        = string
}

variable "aws_region" {
  description = "AWS Region"
  type        = string
  default     = "ap-northeast-2"
}

variable "github_org" {
  description = "GitHub Organization name"
  type        = string
  default     = "OrangesCloud"
}

variable "github_repo" {
  description = "GitHub Repository name"
  type        = string
  default     = "wealist-project-advanced-k8s"
}

variable "role_name_prefix" {
  description = "Prefix for IAM role names"
  type        = string
  default     = "wealist-github-actions"
}

# -----------------------------------------------------------------------------
# Frontend 설정
# -----------------------------------------------------------------------------
variable "enable_frontend_role" {
  description = "Enable frontend IAM role creation"
  type        = bool
  default     = true
}

variable "frontend_branches" {
  description = "List of branches allowed to assume the frontend role"
  type        = list(string)
  default     = ["dev-frontend"]
}

variable "s3_bucket_names" {
  description = "List of S3 bucket names for frontend deployment"
  type        = list(string)
  default     = []
}

# -----------------------------------------------------------------------------
# Backend 설정
# -----------------------------------------------------------------------------
variable "backend_branches" {
  description = "List of branches allowed to assume the backend role"
  type        = list(string)
  default = [
    "service-deploy-dev",
    "service-deploy-prod",
    "k8s-deploy-dev",
    "k8s-deploy-prod"
  ]
}
