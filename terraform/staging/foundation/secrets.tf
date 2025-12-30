# =============================================================================
# Staging Secrets - AWS Secrets Manager
# =============================================================================
# ESO가 이 시크릿들을 읽어서 K8s Secret으로 변환합니다.
# terraform apply 후 ESO가 자동으로 동기화합니다.

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = "ap-northeast-2"
}

# -----------------------------------------------------------------------------
# Database Endpoint
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "database_endpoint" {
  name        = "wealist/staging/database/endpoint"
  description = "Staging Database Connection"
}

resource "aws_secretsmanager_secret_version" "database_endpoint" {
  secret_id = aws_secretsmanager_secret.database_endpoint.id
  secret_string = jsonencode({
    host     = var.db_host
    port     = var.db_port
    username = var.db_username
    password = var.db_password
  })
}

# -----------------------------------------------------------------------------
# Redis Endpoint
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "redis_endpoint" {
  name        = "wealist/staging/redis/endpoint"
  description = "Staging Redis Connection"
}

resource "aws_secretsmanager_secret_version" "redis_endpoint" {
  secret_id = aws_secretsmanager_secret.redis_endpoint.id
  secret_string = jsonencode({
    host = var.redis_host
    port = var.redis_port
  })
}

resource "aws_secretsmanager_secret" "redis_auth" {
  name        = "wealist/staging/redis/auth-token"
  description = "Staging Redis Auth Token"
}

resource "aws_secretsmanager_secret_version" "redis_auth" {
  secret_id     = aws_secretsmanager_secret.redis_auth.id
  secret_string = var.redis_password != "" ? var.redis_password : "no-auth"
}

# -----------------------------------------------------------------------------
# Application Secrets
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "jwt_secret" {
  name        = "wealist/staging/app/jwt-secret"
  description = "Staging JWT Secret"
}

resource "aws_secretsmanager_secret_version" "jwt_secret" {
  secret_id     = aws_secretsmanager_secret.jwt_secret.id
  secret_string = var.jwt_secret
}

resource "aws_secretsmanager_secret" "internal_api_key" {
  name        = "wealist/staging/app/internal-api-key"
  description = "Staging Internal API Key"
}

resource "aws_secretsmanager_secret_version" "internal_api_key" {
  secret_id     = aws_secretsmanager_secret.internal_api_key.id
  secret_string = var.internal_api_key
}

# -----------------------------------------------------------------------------
# OAuth2 Google
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "oauth_google" {
  name        = "wealist/staging/oauth/google"
  description = "Staging Google OAuth Credentials"
}

resource "aws_secretsmanager_secret_version" "oauth_google" {
  secret_id = aws_secretsmanager_secret.oauth_google.id
  secret_string = jsonencode({
    client_id     = var.google_client_id
    client_secret = var.google_client_secret
  })
}

# -----------------------------------------------------------------------------
# LiveKit
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "livekit" {
  name        = "wealist/staging/livekit/credentials"
  description = "Staging LiveKit Credentials"
}

resource "aws_secretsmanager_secret_version" "livekit" {
  secret_id = aws_secretsmanager_secret.livekit.id
  secret_string = jsonencode({
    api_key    = var.livekit_api_key
    api_secret = var.livekit_api_secret
  })
}

# -----------------------------------------------------------------------------
# MinIO / S3
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "minio" {
  name        = "wealist/staging/minio/credentials"
  description = "Staging MinIO Credentials"
}

resource "aws_secretsmanager_secret_version" "minio" {
  secret_id = aws_secretsmanager_secret.minio.id
  secret_string = jsonencode({
    access_key = var.minio_access_key
    secret_key = var.minio_secret_key
  })
}

# -----------------------------------------------------------------------------
# Grafana
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "grafana" {
  name        = "wealist/staging/monitoring/grafana"
  description = "Staging Grafana Admin Credentials"
}

resource "aws_secretsmanager_secret_version" "grafana" {
  secret_id = aws_secretsmanager_secret.grafana.id
  secret_string = jsonencode({
    admin_user     = var.grafana_admin_user
    admin_password = var.grafana_admin_password
  })
}
