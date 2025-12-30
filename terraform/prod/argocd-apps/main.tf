# =============================================================================
# ArgoCD Applications Layer
# Requires: compute layer (EKS + ArgoCD installed)
# =============================================================================

terraform {
  required_version = ">= 1.9"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25"
    }
  }

  backend "s3" {
    bucket         = "wealist-tf-state-advanced-k8s"
    key            = "prod/argocd-apps/terraform.tfstate"
    region         = "ap-northeast-2"
    dynamodb_table = "terraform-lock"
    encrypt        = true
  }
}

# -----------------------------------------------------------------------------
# Providers
# -----------------------------------------------------------------------------

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = local.common_tags
  }
}

# -----------------------------------------------------------------------------
# Data Sources - Remote State
# -----------------------------------------------------------------------------

data "terraform_remote_state" "foundation" {
  backend = "s3"

  config = {
    bucket = "wealist-tf-state-advanced-k8s"
    key    = "prod/foundation/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

data "terraform_remote_state" "compute" {
  backend = "s3"

  config = {
    bucket = "wealist-tf-state-advanced-k8s"
    key    = "prod/compute/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

# EKS Cluster Data
data "aws_eks_cluster" "cluster" {
  name = data.terraform_remote_state.compute.outputs.cluster_name
}

data "aws_eks_cluster_auth" "cluster" {
  name = data.terraform_remote_state.compute.outputs.cluster_name
}

# -----------------------------------------------------------------------------
# Kubernetes Provider (uses existing EKS cluster)
# -----------------------------------------------------------------------------

provider "kubernetes" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.cluster.token
}

# -----------------------------------------------------------------------------
# Locals
# -----------------------------------------------------------------------------

locals {
  name_prefix = "wealist-prod"
  environment = "prod"

  # Cluster info from remote state
  cluster_name = data.terraform_remote_state.compute.outputs.cluster_name
  vpc_id       = data.terraform_remote_state.foundation.outputs.vpc_id

  common_tags = {
    Project     = "wealist"
    Environment = local.environment
    ManagedBy   = "terraform"
    Layer       = "argocd-apps"
  }
}
