# =============================================================================
# GitHub OIDC Provider Module
# =============================================================================
# GitHub Actions에서 AWS에 OIDC로 인증하기 위한 설정

# -----------------------------------------------------------------------------
# OIDC Provider - 기존 것 사용 또는 새로 생성
# -----------------------------------------------------------------------------

# 기존 OIDC Provider 조회 (이미 존재하는 경우)
data "aws_iam_openid_connect_provider" "github" {
  count = var.create_oidc_provider ? 0 : 1
  url   = "https://token.actions.githubusercontent.com"
}

# 새 OIDC Provider 생성 (존재하지 않는 경우)
resource "aws_iam_openid_connect_provider" "github" {
  count = var.create_oidc_provider ? 1 : 0

  url = "https://token.actions.githubusercontent.com"

  client_id_list = ["sts.amazonaws.com"]

  # GitHub OIDC thumbprint (2023년 이후 고정값)
  thumbprint_list = [
    "6938fd4d98bab03faadb97b34396831e3780aea1",
    "1c58a3a8518e8759bf075b76b750d4f2df264fcd"
  ]

  tags = var.tags
}

locals {
  oidc_provider_arn = var.create_oidc_provider ? aws_iam_openid_connect_provider.github[0].arn : data.aws_iam_openid_connect_provider.github[0].arn
}

# -----------------------------------------------------------------------------
# IAM Role for GitHub Actions
# -----------------------------------------------------------------------------
locals {
  # Branch-based claims (for push events)
  branch_claims = [
    for branch in var.allowed_branches :
    "repo:${var.github_org}/${var.github_repo}:ref:refs/heads/${branch}"
  ]

  # Environment-based claims (for workflow jobs with environment)
  environment_claims = [
    for env in var.allowed_environments :
    "repo:${var.github_org}/${var.github_repo}:environment:${env}"
  ]

  # Combined claims
  all_claims = concat(local.branch_claims, local.environment_claims)
}

resource "aws_iam_role" "github_actions" {
  name = var.role_name
  path = "/github-actions/"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Federated = local.oidc_provider_arn
        }
        Action = "sts:AssumeRoleWithWebIdentity"
        Condition = {
          StringEquals = {
            "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          }
          StringLike = {
            "token.actions.githubusercontent.com:sub" = local.all_claims
          }
        }
      }
    ]
  })

  tags = var.tags
}

# -----------------------------------------------------------------------------
# IAM Policy - ECR Access
# -----------------------------------------------------------------------------
resource "aws_iam_role_policy" "ecr_access" {
  count = var.enable_ecr_access ? 1 : 0

  name = "ecr-access"
  role = aws_iam_role.github_actions.id

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
          "ecr:CreateRepository",
          "ecr:ListImages",
          "ecr:DescribeImages"
        ]
        Resource = "arn:aws:ecr:${var.aws_region}:${var.aws_account_id}:repository/*"
      }
    ]
  })
}

# -----------------------------------------------------------------------------
# IAM Policy - S3 Access (Frontend Deployment)
# -----------------------------------------------------------------------------
resource "aws_iam_role_policy" "s3_access" {
  count = var.enable_s3_access ? 1 : 0

  name = "s3-access"
  role = aws_iam_role.github_actions.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "S3BucketAccess"
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:DeleteObject",
          "s3:ListBucket",
          "s3:GetBucketLocation"
        ]
        Resource = flatten([
          for bucket in var.s3_bucket_names : [
            "arn:aws:s3:::${bucket}",
            "arn:aws:s3:::${bucket}/*"
          ]
        ])
      }
    ]
  })
}

# -----------------------------------------------------------------------------
# IAM Policy - CloudFront Access (Cache Invalidation)
# -----------------------------------------------------------------------------
resource "aws_iam_role_policy" "cloudfront_access" {
  count = var.enable_cloudfront_access ? 1 : 0

  name = "cloudfront-access"
  role = aws_iam_role.github_actions.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "CloudFrontInvalidation"
        Effect = "Allow"
        Action = [
          "cloudfront:CreateInvalidation",
          "cloudfront:GetInvalidation",
          "cloudfront:ListInvalidations"
        ]
        Resource = "*"
      }
    ]
  })
}
