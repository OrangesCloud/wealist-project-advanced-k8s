# =============================================================================
# ECR Repositories for Production
# =============================================================================
# prod/ prefix로 생성하여 dev ECR과 분리
# Immutable 태그로 프로덕션 이미지 보호

module "ecr_prod" {
  source = "../../modules/ecr"

  repository_names = [
    "prod/auth-service",
    "prod/user-service",
    "prod/board-service",
    "prod/chat-service",
    "prod/noti-service",
    "prod/storage-service",
    "prod/frontend"
  ]

  # Production: Immutable 태그 (덮어쓰기 방지)
  image_tag_mutability    = "IMMUTABLE"
  scan_on_push            = true
  enable_lifecycle_policy = true
  max_image_count         = 50  # Production은 더 많은 이미지 유지

  tags = merge(local.common_tags, {
    ImageType = "production"
  })
}
