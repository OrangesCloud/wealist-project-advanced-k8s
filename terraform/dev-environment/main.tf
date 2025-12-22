# =============================================================================
# Dev Environment Configuration
# =============================================================================
# 로컬 PC 개발 환경을 위한 AWS 리소스 설정
# - ECR 리포지토리 (8개 서비스)
# - IAM User (ECR 접근용)
#
# 사용법:
#   1. terraform.tfvars.example을 terraform.tfvars로 복사
#   2. 필요한 변수 설정
#   3. terraform init && terraform apply
#   4. 출력된 Access Key를 로컬 PC에 설정

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
      Environment = "dev"
    }
  }
}

# -----------------------------------------------------------------------------
# Data Sources
# -----------------------------------------------------------------------------
data "aws_caller_identity" "current" {}

# -----------------------------------------------------------------------------
# ECR Repositories
# -----------------------------------------------------------------------------
module "ecr" {
  source = "../modules/ecr"

  repository_names        = var.service_names
  image_tag_mutability    = "MUTABLE"
  scan_on_push            = true
  enable_lifecycle_policy = true
  max_image_count         = var.max_image_count

  tags = {
    Environment = "dev"
  }
}

# -----------------------------------------------------------------------------
# IAM User for Local Development
# -----------------------------------------------------------------------------
resource "aws_iam_user" "dev_ecr_user" {
  name = var.iam_user_name
  path = "/dev/"

  tags = {
    Purpose = "local-dev-ecr-access"
  }
}

# -----------------------------------------------------------------------------
# IAM Policy - ECR Access (Minimal Permissions)
# -----------------------------------------------------------------------------
resource "aws_iam_user_policy" "ecr_access" {
  name = "ecr-access"
  user = aws_iam_user.dev_ecr_user.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ECRGetAuthorizationToken"
        Effect = "Allow"
        Action = [
          "ecr:GetAuthorizationToken"
        ]
        Resource = "*"
      },
      {
        Sid    = "ECRRepositoryAccess"
        Effect = "Allow"
        Action = [
          "ecr:BatchGetImage",
          "ecr:BatchCheckLayerAvailability",
          "ecr:CompleteLayerUpload",
          "ecr:GetDownloadUrlForLayer",
          "ecr:InitiateLayerUpload",
          "ecr:PutImage",
          "ecr:UploadLayerPart",
          "ecr:DescribeRepositories",
          "ecr:ListImages",
          "ecr:DescribeImages"
        ]
        Resource = [
          for name in var.service_names :
          "arn:aws:ecr:${var.aws_region}:${data.aws_caller_identity.current.account_id}:repository/${name}"
        ]
      }
    ]
  })
}

# -----------------------------------------------------------------------------
# IAM Access Key
# -----------------------------------------------------------------------------
resource "aws_iam_access_key" "dev_ecr_user" {
  user = aws_iam_user.dev_ecr_user.name
}
