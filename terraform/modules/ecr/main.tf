# =============================================================================
# ECR Repository Module
# =============================================================================
# ECR 리포지토리 생성 및 관리

# -----------------------------------------------------------------------------
# ECR Repositories
# -----------------------------------------------------------------------------
resource "aws_ecr_repository" "services" {
  for_each = toset(var.repository_names)

  name                 = each.value
  image_tag_mutability = var.image_tag_mutability

  image_scanning_configuration {
    scan_on_push = var.scan_on_push
  }

  encryption_configuration {
    encryption_type = "AES256"
  }

  tags = merge(var.tags, {
    Name = each.value
  })
}

# -----------------------------------------------------------------------------
# Lifecycle Policy (Optional)
# -----------------------------------------------------------------------------
resource "aws_ecr_lifecycle_policy" "cleanup" {
  for_each = var.enable_lifecycle_policy ? toset(var.repository_names) : []

  repository = aws_ecr_repository.services[each.key].name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Keep last ${var.max_image_count} images"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = var.max_image_count
        }
        action = {
          type = "expire"
        }
      }
    ]
  })
}

# -----------------------------------------------------------------------------
# Repository Policy (Optional - Cross-account access)
# -----------------------------------------------------------------------------
resource "aws_ecr_repository_policy" "cross_account" {
  for_each = var.cross_account_arns != null ? toset(var.repository_names) : []

  repository = aws_ecr_repository.services[each.key].name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "CrossAccountAccess"
        Effect = "Allow"
        Principal = {
          AWS = var.cross_account_arns
        }
        Action = [
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:BatchCheckLayerAvailability"
        ]
      }
    ]
  })
}
