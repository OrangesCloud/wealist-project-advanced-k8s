terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "s3" {
    bucket         = "wealist-tf-state-advanced-k8s"
    key            = "web-infra/terraform.tfstate" # 폴더마다 달라야합니다.
    region         = "ap-northeast-2"
    dynamodb_table = "terraform-lock"
    encrypt        = true
  }
}

provider "aws" {
  region = var.aws_region
}


# ---------------------------------------------------------
# 1. S3 버킷 Import 및 설정
# ---------------------------------------------------------

# Terraform 1.5 이상에서 지원하는 import 블록입니다.
# apply 시 자동으로 상태를 가져옵니다.
import {
  to = aws_s3_bucket.frontend_bucket
  id = var.bucket_name
}

resource "aws_s3_bucket" "frontend_bucket" {
  bucket = var.bucket_name
  # 기존 버킷 설정에 맞춰 태그 등을 추가할 수 있습니다.
}

# CloudFront에서 S3에 접근하기 위한 정책 (OAC 방식)
resource "aws_s3_bucket_policy" "frontend_policy" {
  bucket = aws_s3_bucket.frontend_bucket.id
  policy = data.aws_iam_policy_document.s3_policy.json
}

data "aws_iam_policy_document" "s3_policy" {
  statement {
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.frontend_bucket.arn}/*"]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.cdn.arn]
    }
  }
}
