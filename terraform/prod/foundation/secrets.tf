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
# JWT RSA Key Pair (수동 입력)
# -----------------------------------------------------------------------------
# auth-service에서 JWT 토큰 RS256 서명/검증에 사용
# 모든 Pod이 동일한 키를 공유해야 함 (Multi-pod 환경 필수)
# openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048
# openssl rsa -pubout -in private_key.pem -out public_key.pem
# aws secretsmanager put-secret-value \
#   --secret-id wealist/prod/app/jwt-rsa-keys \
#   --secret-string '{"public_key":"-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----","private_key":"-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"}'
resource "aws_secretsmanager_secret" "jwt_rsa_keys" {
  name       = "wealist/prod/app/jwt-rsa-keys"
  kms_key_id = module.kms.key_arn

  tags = merge(
    local.common_tags,
    {
      Purpose = "JWT RS256 Token Signing"
    }
  )
}

resource "aws_secretsmanager_secret_version" "jwt_rsa_keys" {
  secret_id = aws_secretsmanager_secret.jwt_rsa_keys.id
  secret_string = jsonencode({
    public_key  = "PLACEHOLDER-UPDATE-ME"
    private_key = "PLACEHOLDER-UPDATE-ME"
  })

  lifecycle {
    ignore_changes = [secret_string] # 수동 업데이트 후 덮어쓰지 않음
  }
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
# OAuth2 ArgoCD (수동 입력 - placeholder)
# -----------------------------------------------------------------------------
# ArgoCD SSO를 위한 Google OAuth2 자격 증명 (앱용 oauth/google과 별도)
# Google Cloud Console에서 ArgoCD 전용 OAuth 클라이언트 생성 후 입력:
# - Redirect URI: https://argocd.wealist.co.kr/api/dex/callback
# aws secretsmanager put-secret-value \
#   --secret-id wealist/prod/oauth/argocd \
#   --secret-string '{"client_id":"...","client_secret":"..."}'
resource "aws_secretsmanager_secret" "oauth_argocd" {
  name       = "wealist/prod/oauth/argocd"
  kms_key_id = module.kms.key_arn

  tags = merge(
    local.common_tags,
    {
      Purpose = "ArgoCD SSO"
    }
  )
}

resource "aws_secretsmanager_secret_version" "oauth_argocd" {
  secret_id = aws_secretsmanager_secret.oauth_argocd.id
  secret_string = jsonencode({
    client_id     = "PLACEHOLDER-UPDATE-ME"
    client_secret = "PLACEHOLDER-UPDATE-ME"
  })

  lifecycle {
    ignore_changes = [secret_string] # 수동 업데이트 후 덮어쓰지 않음
  }
}

# -----------------------------------------------------------------------------
# ArgoCD Server Secret (자동 생성)
# -----------------------------------------------------------------------------
# ArgoCD 서버 인증에 필요한 secretkey
# ExternalSecret이 이 값을 argocd-secret에 주입
resource "aws_secretsmanager_secret" "argocd_server" {
  name       = "wealist/prod/argocd/server"
  kms_key_id = module.kms.key_arn

  tags = merge(
    local.common_tags,
    {
      Purpose = "ArgoCD Server Authentication"
    }
  )
}

resource "random_password" "argocd_secretkey" {
  length  = 32
  special = false
}

resource "aws_secretsmanager_secret_version" "argocd_server" {
  secret_id = aws_secretsmanager_secret.argocd_server.id
  secret_string = jsonencode({
    secretkey = random_password.argocd_secretkey.result
  })
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
# Discord Webhook URL (수동 입력 - placeholder)
# -----------------------------------------------------------------------------
# Discord 서버 설정에서 생성한 Webhook URL
# ArgoCD Notifications에서 배포 알림 전송에 사용
# Terraform apply 후 수동으로 실제 값 입력 필요:
# aws secretsmanager put-secret-value \
#   --secret-id wealist/prod/notifications/discord \
#   --secret-string '{"webhook_url":"https://discord.com/api/webhooks/..."}'
resource "aws_secretsmanager_secret" "discord_webhook" {
  name       = "wealist/prod/notifications/discord"
  kms_key_id = module.kms.key_arn

  tags = merge(
    local.common_tags,
    {
      Purpose = "ArgoCD Notifications"
    }
  )
}

resource "aws_secretsmanager_secret_version" "discord_webhook" {
  secret_id = aws_secretsmanager_secret.discord_webhook.id
  secret_string = jsonencode({
    webhook_url = "https://discord.com/api/webhooks/PLACEHOLDER/PLACEHOLDER-UPDATE-ME"
  })

  lifecycle {
    ignore_changes = [secret_string] # 수동 업데이트 후 덮어쓰지 않음
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
