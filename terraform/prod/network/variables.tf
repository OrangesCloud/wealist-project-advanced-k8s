# =============================================================================
# Variables
# =============================================================================

variable "aws_region" {
  description = "AWS Region"
  type        = string
  default     = "ap-northeast-2"
}

variable "domain_name" {
  description = "Main domain name"
  type        = string
  default     = "wealist.co.kr"
}

# -----------------------------------------------------------------------------
# VPC Configuration (Override remote state)
# -----------------------------------------------------------------------------

variable "vpc_id" {
  description = "VPC ID (optional - uses remote state if not provided)"
  type        = string
  default     = null
}

variable "vpc_cidr_block" {
  description = "VPC CIDR block (optional - uses remote state if not provided)"
  type        = string
  default     = null
}

variable "public_subnet_ids" {
  description = "List of public subnet IDs (optional - uses remote state if not provided)"
  type        = list(string)
  default     = []
}

# -----------------------------------------------------------------------------
# DNS Toggle
# -----------------------------------------------------------------------------

variable "enable_dns" {
  description = <<-EOT
    Enable Route53 DNS records for the domain.
    Set to false to use CloudFront default domain and ALB DNS name directly.
    IMPORTANT: Before setting to true, delete any manually created Route53 records
    to avoid conflicts.
  EOT
  type        = bool
  default     = false
}

# -----------------------------------------------------------------------------
# ALB Configuration
# -----------------------------------------------------------------------------

variable "alb_idle_timeout" {
  description = "ALB idle timeout in seconds"
  type        = number
  default     = 60
}

variable "alb_deletion_protection" {
  description = "Enable ALB deletion protection"
  type        = bool
  default     = false # Set to true in production after verification
}

# -----------------------------------------------------------------------------
# Frontend (CloudFront + S3) - Managed manually via AWS Console
# -----------------------------------------------------------------------------
# CloudFront uses Flat-Rate Free Plan (November 2025)
# Terraform does not yet support flat-rate pricing plans
# Create CloudFront + S3 manually in AWS Console

# -----------------------------------------------------------------------------
# Target Group Configuration (Istio Gateway)
# -----------------------------------------------------------------------------

variable "istio_health_check_path" {
  description = "Health check path for Istio Gateway"
  type        = string
  default     = "/healthz/ready"
}

variable "istio_health_check_port" {
  description = "Health check port for Istio Gateway"
  type        = string
  default     = "15021"
}
