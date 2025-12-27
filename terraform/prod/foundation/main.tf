# =============================================================================
# Production Foundation Infrastructure
# =============================================================================
# Production 환경의 기본 인프라 (거의 변경하지 않는 리소스)
# - VPC (네트워크)
# - RDS PostgreSQL (데이터베이스)
# - ElastiCache Redis (캐시)
# - ECR (컨테이너 레지스트리)
# - S3 (파일 스토리지)
# - KMS (암호화)
#
# 배포 순서:
#   1. terraform init
#   2. terraform plan
#   3. terraform apply (사용자가 직접 실행)
#
# 예상 소요 시간: 15-20분 (RDS, NAT Gateway 생성에 시간 소요)

terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.30"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  backend "s3" {
    bucket         = "wealist-tf-state-advanced-k8s"
    key            = "prod/foundation/terraform.tfstate"
    region         = "ap-northeast-2"
    dynamodb_table = "terraform-lock"
    encrypt        = true
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
      Environment = "prod"
      Layer       = "foundation"
    }
  }
}

# -----------------------------------------------------------------------------
# Data Sources
# -----------------------------------------------------------------------------
data "aws_caller_identity" "current" {}
data "aws_availability_zones" "available" {
  state = "available"
}

# -----------------------------------------------------------------------------
# Local Variables
# -----------------------------------------------------------------------------
locals {
  name_prefix = "wealist-prod"

  azs = slice(data.aws_availability_zones.available.names, 0, 3)

  common_tags = {
    Project     = "wealist"
    Environment = "prod"
  }
}
