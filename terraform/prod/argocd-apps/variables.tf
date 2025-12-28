# =============================================================================
# Variables
# =============================================================================

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "ap-northeast-2"
}

variable "git_repo_url" {
  description = "Git repository URL for ArgoCD"
  type        = string
  default     = "https://github.com/OrangesCloud/wealist-project-advanced-k8s.git"
}

variable "git_target_revision" {
  description = "Git branch/tag for ArgoCD applications"
  type        = string
  default     = "k8s-deploy-prod"
}

variable "argocd_apps_path" {
  description = "Path to ArgoCD apps in git repo"
  type        = string
  default     = "k8s/argocd/apps/prod"
}
