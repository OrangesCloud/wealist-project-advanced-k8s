# =============================================================================
# OIDC/IAM Configuration
# =============================================================================
# GitHub Actions → AWS 인증을 위한 OIDC Provider 및 IAM Role 설정
# Frontend와 Backend를 분리하여 최소 권한 원칙 적용
#
# 사용법:
#   1. terraform.tfvars.example을 terraform.tfvars로 복사
#   2. aws_account_id 설정
#   3. terraform init && terraform apply
#   4. 출력된 role_arn을 GitHub Secrets에 등록:
#      - AWS_ROLE_ARN_FRONTEND
#      - AWS_ROLE_ARN_BACKEND

terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# -----------------------------------------------------------------------------
# Provider Configuration
# -----------------------------------------------------------------------------
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "wealist"
      ManagedBy   = "terraform"
      Environment = "shared"
    }
  }
}

# -----------------------------------------------------------------------------
# Frontend Role - S3, CloudFront 권한만 (선택적)
# -----------------------------------------------------------------------------
module "github_oidc_frontend" {
  count  = var.enable_frontend_role ? 1 : 0
  source = "../modules/github-oidc"

  aws_account_id   = var.aws_account_id
  aws_region       = var.aws_region
  github_org       = var.github_org
  github_repo      = var.github_repo
  allowed_branches = var.frontend_branches
  role_name        = "${var.role_name_prefix}-frontend"

  create_oidc_provider = false

  # Frontend 권한
  enable_ecr_access        = false
  enable_s3_access         = true
  s3_bucket_names          = var.s3_bucket_names
  enable_cloudfront_access = true

  tags = {
    Purpose = "github-actions-frontend"
  }
}

# -----------------------------------------------------------------------------
# Backend Role - ECR 권한만
# -----------------------------------------------------------------------------
module "github_oidc_backend" {
  source = "../modules/github-oidc"

  aws_account_id   = var.aws_account_id
  aws_region       = var.aws_region
  github_org       = var.github_org
  github_repo      = var.github_repo
  allowed_branches = var.backend_branches
  role_name        = "${var.role_name_prefix}-backend"

  create_oidc_provider = false

  # Backend 권한
  enable_ecr_access        = true
  enable_s3_access         = false
  s3_bucket_names          = []
  enable_cloudfront_access = false

  tags = {
    Purpose = "github-actions-backend"
  }
}
