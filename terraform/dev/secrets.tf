# =============================================================================
# AWS SSM Parameter Store - Dev Environment Secrets
# =============================================================================
# 모든 시크릿은 SecureString으로 AWS 기본 KMS 키로 암호화됨
# Standard tier 사용 (무료)
#
# 사용법:
#   1. terraform.tfvars에 시크릿 값 설정
#   2. terraform apply
#   3. External Secrets Operator로 K8s에서 사용
#
# 파라미터 경로 규칙: /wealist/{environment}/{category}/{key}

# -----------------------------------------------------------------------------
# SSM Parameter Store Module
# -----------------------------------------------------------------------------
module "parameters" {
  source = "../modules/ssm-parameter"

  parameters = {
    # -------------------------------------------------------------------------
    # Google OAuth
    # -------------------------------------------------------------------------
    "/wealist/dev/google-oauth/client-id" = {
      description = "Google OAuth2 Client ID"
      value       = var.google_client_id
    }
    "/wealist/dev/google-oauth/client-secret" = {
      description = "Google OAuth2 Client Secret"
      value       = var.google_client_secret
    }

    # -------------------------------------------------------------------------
    # JWT
    # -------------------------------------------------------------------------
    "/wealist/dev/jwt/secret" = {
      description = "JWT signing secret"
      value       = var.jwt_secret
    }

    # -------------------------------------------------------------------------
    # Database Passwords
    # -------------------------------------------------------------------------
    "/wealist/dev/database/superuser-password" = {
      description = "PostgreSQL superuser password"
      value       = var.db_superuser_password
    }
    "/wealist/dev/database/user-password" = {
      description = "PostgreSQL user service password"
      value       = var.db_user_password
    }
    "/wealist/dev/database/board-password" = {
      description = "PostgreSQL board service password"
      value       = var.db_board_password
    }
    "/wealist/dev/database/chat-password" = {
      description = "PostgreSQL chat service password"
      value       = var.db_chat_password
    }
    "/wealist/dev/database/noti-password" = {
      description = "PostgreSQL notification service password"
      value       = var.db_noti_password
    }
    "/wealist/dev/database/storage-password" = {
      description = "PostgreSQL storage service password"
      value       = var.db_storage_password
    }
    "/wealist/dev/database/video-password" = {
      description = "PostgreSQL video service password"
      value       = var.db_video_password
    }

    # -------------------------------------------------------------------------
    # Redis
    # -------------------------------------------------------------------------
    "/wealist/dev/redis/password" = {
      description = "Redis password"
      value       = var.redis_password
    }

    # -------------------------------------------------------------------------
    # MinIO / S3
    # -------------------------------------------------------------------------
    "/wealist/dev/minio/root-password" = {
      description = "MinIO root password"
      value       = var.minio_root_password
    }
    "/wealist/dev/minio/access-key" = {
      description = "S3/MinIO access key"
      value       = var.s3_access_key
    }
    "/wealist/dev/minio/secret-key" = {
      description = "S3/MinIO secret key"
      value       = var.s3_secret_key
    }

    # -------------------------------------------------------------------------
    # LiveKit
    # -------------------------------------------------------------------------
    "/wealist/dev/livekit/api-key" = {
      description = "LiveKit API key"
      value       = var.livekit_api_key
    }
    "/wealist/dev/livekit/api-secret" = {
      description = "LiveKit API secret"
      value       = var.livekit_api_secret
    }

    # -------------------------------------------------------------------------
    # Internal API Key
    # -------------------------------------------------------------------------
    "/wealist/dev/internal/api-key" = {
      description = "Internal service-to-service API key"
      value       = var.internal_api_key
    }
  }

  tags = {
    Environment = "dev"
    Project     = "wealist"
  }
}

# =============================================================================
# AWS Secrets Manager - Dev Environment Secrets (수동 관리)
# =============================================================================
# Kind 클러스터 setup 스크립트에서 참조하는 시크릿들
# Terraform이 아닌 AWS Console에서 수동 관리 (setup 스크립트 호환)
#
# 필요한 시크릿 목록 (AWS Console에서 생성):
# -----------------------------------------------------------------------------
# | Secret Name                      | 용도                    | 형식 (JSON)
# -----------------------------------------------------------------------------
# | wealist/dev/oauth/argocd         | ArgoCD Google OAuth     | {"client_id":"...", "client_secret":"..."}
# | wealist/dev/discord/webhook      | ArgoCD 배포 알림        | {"webhook_url":"https://discord.com/..."}
# | wealist/dev/argocd/admins        | ArgoCD 관리자 이메일    | {"emails":["email1@...", "email2@..."]}
# | wealist/dev/github/token         | ArgoCD Git 레포 접근    | {"token":"ghp_..."}
# -----------------------------------------------------------------------------
#
# ESO (External Secrets Operator)용 시크릿:
# -----------------------------------------------------------------------------
# | wealist/dev/oauth/google         | Google OAuth (앱용)
# | wealist/dev/app/jwt-secret       | JWT 시크릿
# | wealist/dev/database/endpoint    | DB 연결 정보
# | wealist/dev/redis/endpoint       | Redis 연결 정보
# | wealist/dev/redis/auth-token     | Redis 인증 토큰
# | wealist/dev/minio/credentials    | MinIO 자격증명
# | wealist/dev/livekit/credentials  | LiveKit API 자격증명
# | wealist/dev/app/internal-api-key | 내부 API 키
# | wealist/dev/monitoring/grafana   | Grafana 관리자 비밀번호
# -----------------------------------------------------------------------------
