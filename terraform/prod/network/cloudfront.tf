# =============================================================================
# CloudFront Distribution for Frontend
# wealist.co.kr → S3 (React SPA)
# =============================================================================

# -----------------------------------------------------------------------------
# Origin Access Control (OAC)
# -----------------------------------------------------------------------------

resource "aws_cloudfront_origin_access_control" "frontend" {
  name                              = "${local.name_prefix}-frontend-oac"
  description                       = "OAC for weAlist frontend"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# -----------------------------------------------------------------------------
# CloudFront Distribution
# -----------------------------------------------------------------------------

resource "aws_cloudfront_distribution" "frontend" {
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  price_class         = var.cloudfront_price_class
  comment             = "weAlist Production Frontend"

  # Custom domain alias (only when DNS is enabled)
  aliases = var.enable_dns ? [var.domain_name] : []

  # ---------------------------------------------------------------------------
  # Origin: S3 Bucket
  # ---------------------------------------------------------------------------
  origin {
    domain_name              = aws_s3_bucket.frontend.bucket_regional_domain_name
    origin_id                = "S3-frontend"
    origin_access_control_id = aws_cloudfront_origin_access_control.frontend.id
  }

  # ---------------------------------------------------------------------------
  # Default Cache Behavior
  # ---------------------------------------------------------------------------
  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-frontend"

    # Managed Cache Policies
    cache_policy_id          = "658327ea-f89d-4fab-a63d-7e88639e58f6" # CachingOptimized
    origin_request_policy_id = "88a5eaf4-2fd4-4709-b370-b4c650ea3fcf" # CORS-S3Origin

    viewer_protocol_policy = "redirect-to-https"
    compress               = true
  }

  # ---------------------------------------------------------------------------
  # SPA Routing (React Router)
  # Return index.html for all 403/404 errors
  # ---------------------------------------------------------------------------
  custom_error_response {
    error_code            = 403
    response_code         = 200
    response_page_path    = "/index.html"
    error_caching_min_ttl = 10
  }

  custom_error_response {
    error_code            = 404
    response_code         = 200
    response_page_path    = "/index.html"
    error_caching_min_ttl = 10
  }

  # ---------------------------------------------------------------------------
  # Geo Restriction
  # ---------------------------------------------------------------------------
  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  # ---------------------------------------------------------------------------
  # SSL Certificate
  # ---------------------------------------------------------------------------
  viewer_certificate {
    # When DNS is enabled, use ACM certificate
    acm_certificate_arn      = var.enable_dns ? data.aws_acm_certificate.cloudfront.arn : null
    ssl_support_method       = var.enable_dns ? "sni-only" : null
    minimum_protocol_version = var.enable_dns ? "TLSv1.2_2021" : null

    # When DNS is disabled, use CloudFront default certificate
    cloudfront_default_certificate = !var.enable_dns
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-frontend-cdn"
  })
}
