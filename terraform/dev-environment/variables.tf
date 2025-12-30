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

# =============================================================================
# Secrets Variables
# =============================================================================
# 이 값들은 terraform.tfvars에서 설정하세요 (git에 커밋하지 마세요!)

# -----------------------------------------------------------------------------
# Google OAuth
# -----------------------------------------------------------------------------
variable "google_client_id" {
  description = "Google OAuth Client ID"
  type        = string
  sensitive   = true
}

variable "google_client_secret" {
  description = "Google OAuth Client Secret"
  type        = string
  sensitive   = true
}

# -----------------------------------------------------------------------------
# JWT
# -----------------------------------------------------------------------------
variable "jwt_secret" {
  description = "JWT signing secret (minimum 32 characters)"
  type        = string
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Database Passwords
# -----------------------------------------------------------------------------
variable "db_superuser_password" {
  description = "PostgreSQL superuser password"
  type        = string
  sensitive   = true
}

variable "db_user_password" {
  description = "User service database password"
  type        = string
  sensitive   = true
}

variable "db_board_password" {
  description = "Board service database password"
  type        = string
  sensitive   = true
}

variable "db_chat_password" {
  description = "Chat service database password"
  type        = string
  sensitive   = true
}

variable "db_noti_password" {
  description = "Notification service database password"
  type        = string
  sensitive   = true
}

variable "db_storage_password" {
  description = "Storage service database password"
  type        = string
  sensitive   = true
}

variable "db_video_password" {
  description = "Video service database password"
  type        = string
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Redis
# -----------------------------------------------------------------------------
variable "redis_password" {
  description = "Redis password"
  type        = string
  sensitive   = true
}

# -----------------------------------------------------------------------------
# MinIO / S3
# -----------------------------------------------------------------------------
variable "minio_root_password" {
  description = "MinIO root password"
  type        = string
  sensitive   = true
}

variable "s3_access_key" {
  description = "S3/MinIO access key"
  type        = string
  sensitive   = true
}

variable "s3_secret_key" {
  description = "S3/MinIO secret key"
  type        = string
  sensitive   = true
}

# -----------------------------------------------------------------------------
# LiveKit
# -----------------------------------------------------------------------------
variable "livekit_api_key" {
  description = "LiveKit API key"
  type        = string
  sensitive   = true
}

variable "livekit_api_secret" {
  description = "LiveKit API secret"
  type        = string
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Internal
# -----------------------------------------------------------------------------
variable "internal_api_key" {
  description = "Internal API key for service-to-service communication"
  type        = string
  sensitive   = true
}
