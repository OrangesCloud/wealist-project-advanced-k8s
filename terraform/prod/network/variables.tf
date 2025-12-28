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
# CloudFront Configuration
# -----------------------------------------------------------------------------

variable "cloudfront_price_class" {
  description = "CloudFront price class (PriceClass_100 = US/EU only, cheapest)"
  type        = string
  default     = "PriceClass_200" # US, EU, Asia
}

variable "cloudfront_default_ttl" {
  description = "Default TTL for CloudFront cache (seconds)"
  type        = number
  default     = 86400 # 1 day
}

variable "cloudfront_min_ttl" {
  description = "Minimum TTL for CloudFront cache (seconds)"
  type        = number
  default     = 0
}

variable "cloudfront_max_ttl" {
  description = "Maximum TTL for CloudFront cache (seconds)"
  type        = number
  default     = 31536000 # 1 year
}

# -----------------------------------------------------------------------------
# S3 Frontend Configuration
# -----------------------------------------------------------------------------

variable "frontend_bucket_name" {
  description = "S3 bucket name for frontend static files"
  type        = string
  default     = "wealist-frontend"
}

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
