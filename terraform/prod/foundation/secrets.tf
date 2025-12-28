# =============================================================================
# AWS Secrets Manager - Application Secrets
# =============================================================================
# External Secrets Operator가 참조할 시크릿들
# 기존: RDS (자동), Redis AUTH (elasticache.tf)
# 신규: JWT, Internal API Key, OAuth2, LiveKit, DB/Redis Endpoints

# -----------------------------------------------------------------------------
# Database Credentials (RDS 호스트 + 비밀번호)
# -----------------------------------------------------------------------------
# RDS manage_master_user_password가 생성하는 동적 시크릿(rds!db-xxxx)을
# 고정 경로로 복사하여 ExternalSecret에서 참조 가능하도록 함

# RDS가 생성한 동적 시크릿에서 비밀번호 읽기
data "aws_secretsmanager_secret_version" "rds_password" {
  secret_id = module.rds.db_instance_master_user_secret_arn
}

resource "aws_secretsmanager_secret" "database_endpoint" {
  name       = "wealist/prod/database/endpoint"
  kms_key_id = module.kms.key_arn

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "database_endpoint" {
  secret_id = aws_secretsmanager_secret.database_endpoint.id
  secret_string = jsonencode({
    host     = module.rds.db_instance_address
    username = jsondecode(data.aws_secretsmanager_secret_version.rds_password.secret_string)["username"]
    password = jsondecode(data.aws_secretsmanager_secret_version.rds_password.secret_string)["password"]
  })
}

# -----------------------------------------------------------------------------
# Redis Endpoint (ElastiCache 호스트 주소)
# -----------------------------------------------------------------------------
resource "aws_secretsmanager_secret" "redis_endpoint" {
  name       = "wealist/prod/redis/endpoint"
  kms_key_id = module.kms.key_arn

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "redis_endpoint" {
  secret_id = aws_secretsmanager_secret.redis_endpoint.id
  secret_string = jsonencode({
    host = aws_elasticache_replication_group.redis.primary_endpoint_address
  })
}

# -----------------------------------------------------------------------------
# JWT Secret (자동 생성)
# -----------------------------------------------------------------------------
# auth-service에서 JWT 토큰 서명에 사용
resource "random_password" "jwt_secret" {
  length  = 64
  special = false
}

resource "aws_secretsmanager_secret" "jwt_secret" {
  name       = "wealist/prod/app/jwt-secret"
  kms_key_id = module.kms.key_arn

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "jwt_secret" {
  secret_id     = aws_secretsmanager_secret.jwt_secret.id
  secret_string = base64encode(random_password.jwt_secret.result)
}

# -----------------------------------------------------------------------------
# Internal API Key (자동 생성)
# -----------------------------------------------------------------------------
# 서비스 간 내부 통신 인증에 사용
resource "random_password" "internal_api_key" {
  length  = 32
  special = false
}

resource "aws_secretsmanager_secret" "internal_api_key" {
  name       = "wealist/prod/app/internal-api-key"
  kms_key_id = module.kms.key_arn

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "internal_api_key" {
  secret_id     = aws_secretsmanager_secret.internal_api_key.id
  secret_string = random_password.internal_api_key.result
}

# -----------------------------------------------------------------------------
# OAuth2 Google (수동 입력 - placeholder)
# -----------------------------------------------------------------------------
# Google Cloud Console에서 발급받은 OAuth2 자격 증명
# Terraform apply 후 수동으로 실제 값 입력 필요:
# aws secretsmanager put-secret-value \
#   --secret-id wealist/prod/oauth/google \
#   --secret-string '{"client_id":"...","client_secret":"..."}'
resource "aws_secretsmanager_secret" "oauth_google" {
  name       = "wealist/prod/oauth/google"
  kms_key_id = module.kms.key_arn

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "oauth_google" {
  secret_id = aws_secretsmanager_secret.oauth_google.id
  secret_string = jsonencode({
    client_id     = "PLACEHOLDER-UPDATE-ME"
    client_secret = "PLACEHOLDER-UPDATE-ME"
  })

  lifecycle {
    ignore_changes = [secret_string] # 수동 업데이트 후 덮어쓰지 않음
  }
}

# -----------------------------------------------------------------------------
# LiveKit Credentials (수동 입력 - placeholder)
# -----------------------------------------------------------------------------
# LiveKit Cloud 또는 Self-hosted LiveKit 서버 자격 증명
# Terraform apply 후 수동으로 실제 값 입력 필요:
# aws secretsmanager put-secret-value \
#   --secret-id wealist/prod/livekit/credentials \
#   --secret-string '{"api_key":"...","api_secret":"..."}'
resource "aws_secretsmanager_secret" "livekit" {
  name       = "wealist/prod/livekit/credentials"
  kms_key_id = module.kms.key_arn

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "livekit" {
  secret_id = aws_secretsmanager_secret.livekit.id
  secret_string = jsonencode({
    api_key    = "PLACEHOLDER-UPDATE-ME"
    api_secret = "PLACEHOLDER-UPDATE-ME"
  })

  lifecycle {
    ignore_changes = [secret_string]
  }
}

# -----------------------------------------------------------------------------
# Grafana Admin Password (자동 생성)
# -----------------------------------------------------------------------------
# Grafana 대시보드 관리자 비밀번호
resource "random_password" "grafana_admin" {
  length  = 24
  special = true
}

resource "aws_secretsmanager_secret" "grafana_admin" {
  name       = "wealist/prod/monitoring/grafana"
  kms_key_id = module.kms.key_arn

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "grafana_admin" {
  secret_id = aws_secretsmanager_secret.grafana_admin.id
  secret_string = jsonencode({
    admin_user     = "admin"
    admin_password = random_password.grafana_admin.result
  })
}
