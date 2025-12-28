# =============================================================================
# ACM Certificates (Existing - Data Sources)
# =============================================================================

# -----------------------------------------------------------------------------
# CloudFront Certificate (us-east-1)
# CloudFront requires certificates in us-east-1 region
# -----------------------------------------------------------------------------

data "aws_acm_certificate" "cloudfront" {
  provider = aws.us_east_1
  domain   = var.domain_name
  statuses = ["ISSUED"]

  # This should match:
  # arn:aws:acm:us-east-1:290008131187:certificate/2a16643d-18e9-4d34-98bf-7be8e1613d33
}

# -----------------------------------------------------------------------------
# ALB Certificate (ap-northeast-2)
# ALB uses certificate in the same region
# -----------------------------------------------------------------------------

data "aws_acm_certificate" "alb" {
  domain   = var.domain_name
  statuses = ["ISSUED"]

  # This should match:
  # arn:aws:acm:ap-northeast-2:290008131187:certificate/193d87ce-e04a-469d-9196-7624ed113bcc
}
