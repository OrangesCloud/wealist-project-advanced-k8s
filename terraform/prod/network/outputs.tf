# =============================================================================
# Outputs
# =============================================================================

# -----------------------------------------------------------------------------
# ALB Outputs (conditional - only when VPC is available)
# -----------------------------------------------------------------------------

output "alb_enabled" {
  description = "Whether ALB was created (requires VPC configuration)"
  value       = local.alb_enabled
}

output "alb_arn" {
  description = "ALB ARN"
  value       = local.alb_enabled ? aws_lb.api[0].arn : null
}

output "alb_dns_name" {
  description = "ALB DNS name"
  value       = local.alb_enabled ? aws_lb.api[0].dns_name : null
}

output "alb_zone_id" {
  description = "ALB hosted zone ID"
  value       = local.alb_enabled ? aws_lb.api[0].zone_id : null
}

output "alb_security_group_id" {
  description = "ALB security group ID"
  value       = local.alb_enabled ? aws_security_group.alb[0].id : null
}

# -----------------------------------------------------------------------------
# Target Group Outputs (for TargetGroupBinding)
# -----------------------------------------------------------------------------

output "istio_target_group_arn" {
  description = "Target Group ARN for Istio Gateway (use with TargetGroupBinding)"
  value       = local.alb_enabled ? aws_lb_target_group.istio[0].arn : null
}

output "target_group_binding_yaml" {
  description = "TargetGroupBinding YAML for Kubernetes (null if ALB not enabled)"
  value = local.alb_enabled ? join("\n", [
    "apiVersion: elbv2.k8s.aws/v1beta1",
    "kind: TargetGroupBinding",
    "metadata:",
    "  name: istio-gateway-tgb",
    "  namespace: istio-system",
    "spec:",
    "  serviceRef:",
    "    name: istio-ingressgateway",
    "    port: 80",
    "  targetGroupARN: ${aws_lb_target_group.istio[0].arn}",
    "  targetType: ip"
  ]) : null
}

# -----------------------------------------------------------------------------
# CloudFront (Managed manually via AWS Console)
# -----------------------------------------------------------------------------
# CloudFront uses Flat-Rate Free Plan (November 2025)
# Terraform does not yet support flat-rate pricing plans
# No outputs here - check AWS Console for CloudFront details

# -----------------------------------------------------------------------------
# S3 (Managed manually via AWS Console with CloudFront)
# -----------------------------------------------------------------------------
# S3 bucket for frontend is created manually with CloudFront Flat-Rate Plan

# -----------------------------------------------------------------------------
# DNS Status
# -----------------------------------------------------------------------------

output "dns_enabled" {
  description = "Whether DNS records are managed by Terraform"
  value       = var.enable_dns
}

# -----------------------------------------------------------------------------
# URLs
# -----------------------------------------------------------------------------

output "frontend_url" {
  description = "Frontend URL (CloudFront managed manually)"
  value       = "https://${var.domain_name} (CloudFront managed via AWS Console)"
}

output "api_url" {
  description = "API URL"
  value = local.alb_enabled ? (
    var.enable_dns ? "https://api.${var.domain_name}" : "https://${aws_lb.api[0].dns_name}"
  ) : "# ALB not created - VPC configuration required"
}

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------

output "summary" {
  description = "Network infrastructure summary"
  value = join("\n", [
    "",
    "# =============================================================================",
    "# Production Network Infrastructure Summary",
    "# =============================================================================",
    "",
    "DNS Enabled: ${var.enable_dns}",
    "ALB Enabled: ${local.alb_enabled}",
    "",
    "Frontend: Managed manually via AWS Console (CloudFront + S3 Flat-Rate Free Plan)",
    "",
    local.alb_enabled ? "API:\n  ALB DNS: ${aws_lb.api[0].dns_name}" : "API: NOT CREATED (VPC required)",
    "",
    local.alb_enabled ? "Next: terraform output -raw target_group_binding_yaml | kubectl apply -f -" : "Next: Deploy foundation first, then re-run terraform apply",
    ""
  ])
}
