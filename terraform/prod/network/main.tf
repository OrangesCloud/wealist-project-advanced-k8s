# =============================================================================
# Production Network Infrastructure
# ALB, CloudFront, Route53 DNS Records
# =============================================================================

terraform {
  required_version = ">= 1.9"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "wealist-tf-state-advanced-k8s"
    key            = "prod/network/terraform.tfstate"
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

# CloudFront requires ACM certificates in us-east-1
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"

  default_tags {
    tags = local.common_tags
  }
}

# -----------------------------------------------------------------------------
# Data Sources - Remote State (Optional)
# -----------------------------------------------------------------------------

data "terraform_remote_state" "foundation" {
  count   = var.vpc_id == null ? 1 : 0
  backend = "s3"

  config = {
    bucket = "wealist-tf-state-advanced-k8s"
    key    = "prod/foundation/terraform.tfstate"
    region = "ap-northeast-2"
  }
}

# -----------------------------------------------------------------------------
# Data Sources - VPC (when using variable override)
# -----------------------------------------------------------------------------

data "aws_vpc" "selected" {
  count = var.vpc_id != null ? 1 : 0
  id    = var.vpc_id
}

data "aws_subnets" "public" {
  count = var.vpc_id != null && length(var.public_subnet_ids) == 0 ? 1 : 0

  filter {
    name   = "vpc-id"
    values = [var.vpc_id]
  }

  filter {
    name   = "tag:Name"
    values = ["*public*"]
  }
}

# -----------------------------------------------------------------------------
# Locals
# -----------------------------------------------------------------------------

locals {
  name_prefix = "wealist-prod"
  environment = "prod"

  common_tags = {
    Project     = "wealist"
    Environment = local.environment
    ManagedBy   = "terraform"
    Layer       = "network"
  }

  # VPC and Subnets - prefer variables, fallback to remote state
  # Use try() to handle missing remote state outputs gracefully
  _foundation_vpc_id     = try(data.terraform_remote_state.foundation[0].outputs.vpc_id, null)
  _foundation_subnet_ids = try(data.terraform_remote_state.foundation[0].outputs.public_subnet_ids, [])
  _foundation_vpc_cidr   = try(data.terraform_remote_state.foundation[0].outputs.vpc_cidr_block, null)
  _discovered_subnet_ids = try(data.aws_subnets.public[0].ids, [])
  _discovered_vpc_cidr   = try(data.aws_vpc.selected[0].cidr_block, null)

  # Final values - prefer variable, then remote state, then discovered
  vpc_id = var.vpc_id != null ? var.vpc_id : local._foundation_vpc_id

  public_subnet_ids = (
    length(var.public_subnet_ids) > 0 ? var.public_subnet_ids :
    length(local._foundation_subnet_ids) > 0 ? local._foundation_subnet_ids :
    local._discovered_subnet_ids
  )

  vpc_cidr_block = (
    var.vpc_cidr_block != null ? var.vpc_cidr_block :
    local._foundation_vpc_cidr != null ? local._foundation_vpc_cidr :
    local._discovered_vpc_cidr
  )

  # Flag to indicate if ALB can be created
  alb_enabled = local.vpc_id != null && length(local.public_subnet_ids) > 0
}
