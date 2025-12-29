# =============================================================================
# Staging Variables
# =============================================================================
# terraform.tfvars 또는 환경변수로 값을 설정하세요.
# 예: TF_VAR_db_host="172.18.0.1"

# -----------------------------------------------------------------------------
# Database
# -----------------------------------------------------------------------------
variable "db_host" {
  description = "Database host (e.g., 172.18.0.1 for Kind, RDS endpoint for AWS)"
  type        = string
  default     = "172.18.0.1"
}

variable "db_port" {
  description = "Database port"
  type        = string
  default     = "5432"
}

variable "db_username" {
  description = "Database username"
  type        = string
  default     = "wealist"
}

variable "db_password" {
  description = "Database password"
  type        = string
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Redis
# -----------------------------------------------------------------------------
variable "redis_host" {
  description = "Redis host"
  type        = string
  default     = "172.18.0.1"
}

variable "redis_port" {
  description = "Redis port"
  type        = string
  default     = "6379"
}

variable "redis_password" {
  description = "Redis password (empty for no auth)"
  type        = string
  default     = ""
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Application
# -----------------------------------------------------------------------------
variable "jwt_secret" {
  description = "JWT signing secret"
  type        = string
  sensitive   = true
}

variable "internal_api_key" {
  description = "Internal API key for service-to-service communication"
  type        = string
  sensitive   = true
}

# -----------------------------------------------------------------------------
# OAuth2 Google
# -----------------------------------------------------------------------------
variable "google_client_id" {
  description = "Google OAuth2 Client ID"
  type        = string
  default     = "placeholder"
}

variable "google_client_secret" {
  description = "Google OAuth2 Client Secret"
  type        = string
  default     = "placeholder"
  sensitive   = true
}

# -----------------------------------------------------------------------------
# LiveKit
# -----------------------------------------------------------------------------
variable "livekit_api_key" {
  description = "LiveKit API Key"
  type        = string
  default     = "devkey"
}

variable "livekit_api_secret" {
  description = "LiveKit API Secret"
  type        = string
  default     = "devsecret"
  sensitive   = true
}

# -----------------------------------------------------------------------------
# MinIO / S3
# -----------------------------------------------------------------------------
variable "minio_access_key" {
  description = "MinIO Access Key"
  type        = string
  default     = "minioadmin"
}

variable "minio_secret_key" {
  description = "MinIO Secret Key"
  type        = string
  default     = "minioadmin"
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Grafana
# -----------------------------------------------------------------------------
variable "grafana_admin_user" {
  description = "Grafana admin username"
  type        = string
  default     = "admin"
}

variable "grafana_admin_password" {
  description = "Grafana admin password"
  type        = string
  default     = "admin"
  sensitive   = true
}
