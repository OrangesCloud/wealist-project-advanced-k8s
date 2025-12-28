# =============================================================================
# Production Compute Infrastructure
# =============================================================================
# EKS 클러스터 및 관련 리소스 (변경이 더 빈번한 리소스)
# - EKS Cluster
# - Node Groups (Spot)
# - EKS Add-ons
# - Pod Identity Associations
#
# 의존성: prod/foundation이 먼저 배포되어야 함
#
# 배포 순서:
#   1. terraform init
#   2. terraform plan
#   3. terraform apply (사용자가 직접 실행)
#
# 예상 소요 시간: 15-20분

terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.30"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    null = {
      source  = "hashicorp/null"
      version = "~> 3.2"
    }
  }

  backend "s3" {
    bucket         = "wealist-tf-state-advanced-k8s"
    key            = "prod/compute/terraform.tfstate"
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
      Layer       = "compute"
    }
  }
}

# Kubernetes provider (EKS 생성 후 사용)
provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
  }
}

# -----------------------------------------------------------------------------
# Data Sources
# -----------------------------------------------------------------------------
data "aws_caller_identity" "current" {}

# Foundation 레이어의 outputs 참조
data "terraform_remote_state" "foundation" {
  backend = "s3"

  config = {
    bucket = "wealist-tf-state-advanced-k8s"
    key    = "prod/foundation/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

# -----------------------------------------------------------------------------
# Local Variables
# -----------------------------------------------------------------------------
locals {
  name_prefix  = "wealist-prod"
  cluster_name = "${local.name_prefix}-eks"

  common_tags = {
    Project     = "wealist"
    Environment = "prod"
  }

  # Foundation outputs 참조
  vpc_id             = data.terraform_remote_state.foundation.outputs.vpc_id
  private_subnet_ids = data.terraform_remote_state.foundation.outputs.private_subnet_ids
  kms_key_arn        = data.terraform_remote_state.foundation.outputs.kms_key_arn
}
