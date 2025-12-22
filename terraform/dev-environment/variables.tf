# =============================================================================
# Dev Environment Configuration - Variables
# =============================================================================

variable "aws_region" {
  description = "AWS Region"
  type        = string
  default     = "ap-northeast-2"
}

variable "service_names" {
  description = "List of service names for ECR repositories"
  type        = list(string)
  default = [
    "auth-service",
    "user-service",
    "board-service",
    "chat-service",
    "noti-service",
    "storage-service",
    "video-service",
    "frontend"
  ]
}

variable "max_image_count" {
  description = "Maximum number of images to keep per repository"
  type        = number
  default     = 30
}

variable "iam_user_name" {
  description = "Name of the IAM user for local development"
  type        = string
  default     = "wealist-dev-ecr-user"
}
