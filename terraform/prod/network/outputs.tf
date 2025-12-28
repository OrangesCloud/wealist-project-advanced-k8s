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
# CloudFront Outputs
# -----------------------------------------------------------------------------

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID"
  value       = aws_cloudfront_distribution.frontend.id
}

output "cloudfront_domain_name" {
  description = "CloudFront distribution domain name"
  value       = aws_cloudfront_distribution.frontend.domain_name
}

output "cloudfront_arn" {
  description = "CloudFront distribution ARN"
  value       = aws_cloudfront_distribution.frontend.arn
}

# -----------------------------------------------------------------------------
# S3 Outputs
# -----------------------------------------------------------------------------

output "frontend_bucket_name" {
  description = "Frontend S3 bucket name"
  value       = aws_s3_bucket.frontend.id
}

output "frontend_bucket_arn" {
  description = "Frontend S3 bucket ARN"
  value       = aws_s3_bucket.frontend.arn
}

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
  description = "Frontend URL"
  value       = var.enable_dns ? "https://${var.domain_name}" : "https://${aws_cloudfront_distribution.frontend.domain_name}"
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
    "Frontend:",
    "  CloudFront: ${aws_cloudfront_distribution.frontend.domain_name}",
    "  S3 Bucket: ${aws_s3_bucket.frontend.id}",
    "",
    local.alb_enabled ? "API:\n  ALB DNS: ${aws_lb.api[0].dns_name}" : "API: NOT CREATED (VPC required)",
    "",
    local.alb_enabled ? "Next: terraform output -raw target_group_binding_yaml | kubectl apply -f -" : "Next: Deploy foundation first, then re-run terraform apply",
    ""
  ])
}
