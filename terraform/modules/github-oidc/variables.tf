# =============================================================================
# GitHub OIDC Module - Variables
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
}

variable "github_repo" {
  description = "GitHub Repository name"
  type        = string
}

variable "allowed_branches" {
  description = "List of branch patterns allowed to assume the role"
  type        = list(string)
  default     = ["main"]
}

variable "allowed_environments" {
  description = "List of GitHub Environments allowed to assume the role"
  type        = list(string)
  default     = []
}

variable "role_name" {
  description = "Name of the IAM role for GitHub Actions"
  type        = string
  default     = "github-actions-role"
}

variable "enable_ecr_access" {
  description = "Enable ECR access for the role"
  type        = bool
  default     = true
}

variable "enable_s3_access" {
  description = "Enable S3 access for the role"
  type        = bool
  default     = false
}

variable "s3_bucket_names" {
  description = "List of S3 bucket names to grant access"
  type        = list(string)
  default     = []
}

variable "enable_cloudfront_access" {
  description = "Enable CloudFront access for cache invalidation"
  type        = bool
  default     = false
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}

variable "create_oidc_provider" {
  description = "Whether to create OIDC provider (set to false if it already exists)"
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# EKS Access (for kubectl/Helm deployments in CI/CD)
# -----------------------------------------------------------------------------
variable "enable_eks_access" {
  description = "Enable EKS access for kubectl/Helm deployments"
  type        = bool
  default     = false
}

variable "eks_cluster_arns" {
  description = "List of EKS cluster ARNs to grant access (e.g., arn:aws:eks:region:account:cluster/name)"
  type        = list(string)
  default     = []
}
