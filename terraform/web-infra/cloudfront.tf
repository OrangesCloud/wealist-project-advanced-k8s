# OAC 생성 (S3 접근 권한 제어)
resource "aws_cloudfront_origin_access_control" "oac" {
  name                              = "wealist-frontend-oac-test"
  description                       = "OAC for weAlist frontend test"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "cdn" {
  origin {
    domain_name              = aws_s3_bucket.frontend_bucket.bucket_regional_domain_name
    origin_id                = "S3-${var.bucket_name}"
    origin_access_control_id = aws_cloudfront_origin_access_control.oac.id
  }

  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"

  # [변경] 커스텀 도메인(aliases) 설정 삭제됨

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD", "OPTIONS"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "S3-${var.bucket_name}"

    forwarded_values {
      query_string = false
      cookies {
        forward = "none"
      }
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400
  }

  # SPA(React) 라우팅 처리를 위한 에러 페이지 설정
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

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  # [변경] ACM 인증서 대신 CloudFront 기본 인증서 사용
  viewer_certificate {
    cloudfront_default_certificate = true
  }
}

# 생성된 CloudFront 주소를 출력
output "cloudfront_domain_name" {
  value = aws_cloudfront_distribution.cdn.domain_name
}