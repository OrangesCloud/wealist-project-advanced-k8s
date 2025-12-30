# =============================================================================
# S3 Bucket for Storage Service
# =============================================================================
# 파일 업로드/다운로드용 S3 버킷
# KMS 암호화, 버전 관리, Lifecycle 정책 적용

resource "aws_s3_bucket" "storage" {
  bucket = "${local.name_prefix}-files-${data.aws_caller_identity.current.account_id}"

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-files"
  })
}

# -----------------------------------------------------------------------------
# Versioning
# -----------------------------------------------------------------------------
resource "aws_s3_bucket_versioning" "storage" {
  bucket = aws_s3_bucket.storage.id
  versioning_configuration {
    status = "Enabled"
  }
}

# -----------------------------------------------------------------------------
# Encryption (KMS)
# -----------------------------------------------------------------------------
resource "aws_s3_bucket_server_side_encryption_configuration" "storage" {
  bucket = aws_s3_bucket.storage.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = module.kms.key_arn
      sse_algorithm     = "aws:kms"
    }
    bucket_key_enabled = true
  }
}

# -----------------------------------------------------------------------------
# Public Access Block
# -----------------------------------------------------------------------------
resource "aws_s3_bucket_public_access_block" "storage" {
  bucket = aws_s3_bucket.storage.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# -----------------------------------------------------------------------------
# Lifecycle Rules
# -----------------------------------------------------------------------------
resource "aws_s3_bucket_lifecycle_configuration" "storage" {
  bucket = aws_s3_bucket.storage.id

  rule {
    id     = "transition-to-ia"
    status = "Enabled"

    # 모든 객체에 적용
    filter {}

    # 90일 후 Infrequent Access로 이동
    transition {
      days          = 90
      storage_class = "STANDARD_IA"
    }

    # 365일 후 Glacier로 이동
    transition {
      days          = 365
      storage_class = "GLACIER"
    }
  }

  rule {
    id     = "delete-old-versions"
    status = "Enabled"

    # 모든 객체에 적용
    filter {}

    # 이전 버전은 30일 후 삭제
    noncurrent_version_expiration {
      noncurrent_days = 30
    }
  }
}

# -----------------------------------------------------------------------------
# CORS Configuration (Frontend 업로드용)
# -----------------------------------------------------------------------------
resource "aws_s3_bucket_cors_configuration" "storage" {
  bucket = aws_s3_bucket.storage.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "PUT", "POST", "DELETE", "HEAD"]
    allowed_origins = var.cors_allowed_origins
    expose_headers  = ["ETag", "Content-Length", "Content-Type"]
    max_age_seconds = 3600
  }
}
