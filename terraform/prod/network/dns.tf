# =============================================================================
# Route53 DNS Records
# Managed externally: Hosted Zone (only records are managed here)
# =============================================================================
#
# IMPORTANT:
# - Route53 Hosted Zone is managed outside of Terraform
# - Before setting enable_dns=true, delete manually created DNS records
# - Zone ID: Z0954990337NMPX3FY1D6
#
# =============================================================================

# -----------------------------------------------------------------------------
# Route53 Hosted Zone (Data Source)
# -----------------------------------------------------------------------------

data "aws_route53_zone" "main" {
  count = var.enable_dns ? 1 : 0
  name  = "${var.domain_name}."
}

# =============================================================================
# Frontend Records (wealist.co.kr → CloudFront)
# =============================================================================

# A Record (IPv4)
resource "aws_route53_record" "frontend_a" {
  count   = var.enable_dns ? 1 : 0
  zone_id = data.aws_route53_zone.main[0].zone_id
  name    = var.domain_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.frontend.domain_name
    zone_id                = aws_cloudfront_distribution.frontend.hosted_zone_id
    evaluate_target_health = false
  }
}

# AAAA Record (IPv6)
resource "aws_route53_record" "frontend_aaaa" {
  count   = var.enable_dns ? 1 : 0
  zone_id = data.aws_route53_zone.main[0].zone_id
  name    = var.domain_name
  type    = "AAAA"

  alias {
    name                   = aws_cloudfront_distribution.frontend.domain_name
    zone_id                = aws_cloudfront_distribution.frontend.hosted_zone_id
    evaluate_target_health = false
  }
}

# =============================================================================
# API Records (api.wealist.co.kr → ALB)
# =============================================================================

# A Record (IPv4)
resource "aws_route53_record" "api_a" {
  count   = var.enable_dns && local.alb_enabled ? 1 : 0
  zone_id = data.aws_route53_zone.main[0].zone_id
  name    = "api.${var.domain_name}"
  type    = "A"

  alias {
    name                   = aws_lb.api[0].dns_name
    zone_id                = aws_lb.api[0].zone_id
    evaluate_target_health = true
  }
}

# =============================================================================
# Note: Records NOT managed here
# =============================================================================
#
# The following records are managed outside of Terraform:
#
# - dev.wealist.co.kr    → CloudFront (dev frontend)
# - local.wealist.co.kr  → iptime (local development)
# - ACM validation records (managed by ACM)
#
# =============================================================================
