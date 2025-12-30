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

  tags = local.common_tags
}
