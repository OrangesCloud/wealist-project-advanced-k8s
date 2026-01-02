# =============================================================================
# KMS Key for Encryption
# =============================================================================
# RDS, ElastiCache, S3, EKS Secrets 암호화에 사용

module "kms" {
  source  = "terraform-aws-modules/kms/aws"
  version = "~> 2.1"

  description = "KMS key for wealist production encryption"
  aliases     = ["wealist-prod"]

  # -----------------------------------------------------------------------------
  # Key Configuration
  # -----------------------------------------------------------------------------
  deletion_window_in_days = 7
  enable_key_rotation     = true

  # -----------------------------------------------------------------------------
  # Key Policy
  # -----------------------------------------------------------------------------
  key_administrators = [
    data.aws_caller_identity.current.arn
  ]

  # RDS, ElastiCache, S3에서 사용 가능하도록 서비스 허용
  key_service_roles_for_autoscaling = [
    "arn:aws:iam::${data.aws_caller_identity.current.account_id}:role/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling"
  ]

  # CloudFront가 S3 KMS 암호화된 객체를 복호화할 수 있도록 허용
  key_statements = [
    {
      sid    = "AllowCloudFrontDecrypt"
      effect = "Allow"
      principals = [
        {
          type        = "Service"
          identifiers = ["cloudfront.amazonaws.com"]
        }
      ]
      actions = [
        "kms:Decrypt",
        "kms:GenerateDataKey*"
      ]
      resources = ["*"]
      conditions = [
        {
          test     = "StringEquals"
          variable = "AWS:SourceArn"
          values   = ["arn:aws:cloudfront::${data.aws_caller_identity.current.account_id}:distribution/${var.cloudfront_distribution_id}"]
        }
      ]
    }
  ]

  tags = local.common_tags
}
