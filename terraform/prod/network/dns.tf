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
# -----------------------------------------------------------------------------
# NLB Pattern (Recommended for Istio Ambient Mode)
# -----------------------------------------------------------------------------
# When using Kubernetes Gateway API (enable_alb=false), Route53 should be
# managed externally via:
#
# 1. ExternalDNS (Recommended)
#    - Install ExternalDNS via ArgoCD
#    - Add annotation to Gateway: external-dns.alpha.kubernetes.io/hostname
#    - ExternalDNS automatically creates Route53 records
#
# 2. Manual Update
#    - Get NLB DNS: kubectl get svc -n istio-system istio-ingressgateway-istio
#    - Update Route53 in AWS Console or via AWS CLI:
#      aws route53 change-resource-record-sets --hosted-zone-id Z0954990... \
#        --change-batch '{"Changes":[{"Action":"UPSERT","ResourceRecordSet":{...}}]}'
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
# NOTE: CloudFront is managed manually via AWS Console (Flat-Rate Free Plan)
# DNS records for frontend are also managed manually or via CloudFront console

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
