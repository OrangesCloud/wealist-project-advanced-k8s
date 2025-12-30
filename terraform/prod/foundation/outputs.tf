# =============================================================================
# Production Foundation - Outputs
# =============================================================================
# prod/compute에서 terraform_remote_state로 참조

# =============================================================================
# VPC Outputs
# =============================================================================
output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "vpc_cidr_block" {
  description = "VPC CIDR block"
  value       = module.vpc.vpc_cidr_block
}

output "private_subnet_ids" {
  description = "List of private subnet IDs"
  value       = module.vpc.private_subnets
}

output "public_subnet_ids" {
  description = "List of public subnet IDs"
  value       = module.vpc.public_subnets
}

output "database_subnet_ids" {
  description = "List of database subnet IDs"
  value       = module.vpc.database_subnets
}

output "database_subnet_group_name" {
  description = "Database subnet group name"
  value       = module.vpc.database_subnet_group_name
}

output "nat_gateway_ids" {
  description = "NAT Gateway IDs"
  value       = module.vpc.natgw_ids
}

# =============================================================================
# RDS Outputs
# =============================================================================
output "rds_endpoint" {
  description = "RDS instance endpoint"
  value       = module.rds.db_instance_endpoint
}

output "rds_address" {
  description = "RDS instance address (hostname only)"
  value       = module.rds.db_instance_address
}

output "rds_port" {
  description = "RDS instance port"
  value       = module.rds.db_instance_port
}

output "rds_database_name" {
  description = "RDS database name"
  value       = module.rds.db_instance_name
}

output "rds_master_user_secret_arn" {
  description = "ARN of the Secrets Manager secret containing the master password"
  value       = module.rds.db_instance_master_user_secret_arn
}

output "rds_security_group_id" {
  description = "RDS security group ID"
  value       = aws_security_group.rds.id
}

# =============================================================================
# ElastiCache Redis Outputs
# =============================================================================
output "redis_endpoint" {
  description = "Redis primary endpoint"
  value       = aws_elasticache_replication_group.redis.primary_endpoint_address
}

output "redis_port" {
  description = "Redis port"
  value       = aws_elasticache_replication_group.redis.port
}

output "redis_auth_token_secret_arn" {
  description = "ARN of the Secrets Manager secret containing the Redis auth token"
  value       = aws_secretsmanager_secret.redis_auth.arn
}

output "redis_security_group_id" {
  description = "Redis security group ID"
  value       = aws_security_group.redis.id
}

# =============================================================================
# ECR Outputs
# =============================================================================
output "ecr_registry_url" {
  description = "ECR Registry URL"
  value       = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com"
}

output "ecr_repository_urls" {
  description = "Map of service names to ECR repository URLs"
  value       = module.ecr_prod.repository_urls
}

# =============================================================================
# S3 Outputs
# =============================================================================
output "s3_bucket_name" {
  description = "S3 bucket name for storage service"
  value       = aws_s3_bucket.storage.id
}

output "s3_bucket_arn" {
  description = "S3 bucket ARN"
  value       = aws_s3_bucket.storage.arn
}

output "s3_bucket_regional_domain_name" {
  description = "S3 bucket regional domain name"
  value       = aws_s3_bucket.storage.bucket_regional_domain_name
}

# =============================================================================
# KMS Outputs
# =============================================================================
output "kms_key_arn" {
  description = "KMS key ARN"
  value       = module.kms.key_arn
}

output "kms_key_id" {
  description = "KMS key ID"
  value       = module.kms.key_id
}

# =============================================================================
# Application Secrets Outputs (for External Secrets Operator)
# =============================================================================
output "jwt_secret_arn" {
  description = "ARN of JWT secret in Secrets Manager"
  value       = aws_secretsmanager_secret.jwt_secret.arn
}

output "internal_api_key_secret_arn" {
  description = "ARN of internal API key secret in Secrets Manager"
  value       = aws_secretsmanager_secret.internal_api_key.arn
}

output "oauth_google_secret_arn" {
  description = "ARN of Google OAuth secret (requires manual update after apply)"
  value       = aws_secretsmanager_secret.oauth_google.arn
}

output "livekit_secret_arn" {
  description = "ARN of LiveKit credentials secret (requires manual update after apply)"
  value       = aws_secretsmanager_secret.livekit.arn
}

# =============================================================================
# Summary for prod/compute
# =============================================================================
output "summary" {
  description = "Summary of resources for prod/compute configuration"
  value       = <<-EOT

    # =============================================================================
    # Production Foundation Resources Summary
    # =============================================================================

    VPC:
      ID: ${module.vpc.vpc_id}
      CIDR: ${module.vpc.vpc_cidr_block}
      Private Subnets: ${join(", ", module.vpc.private_subnets)}

    RDS PostgreSQL:
      Endpoint: ${module.rds.db_instance_endpoint}
      Database: ${module.rds.db_instance_name}
      Password Secret: ${module.rds.db_instance_master_user_secret_arn}

    Redis:
      Endpoint: ${aws_elasticache_replication_group.redis.primary_endpoint_address}:${aws_elasticache_replication_group.redis.port}
      Auth Token Secret: ${aws_secretsmanager_secret.redis_auth.arn}

    S3:
      Bucket: ${aws_s3_bucket.storage.id}

    KMS:
      Key ARN: ${module.kms.key_arn}

    # prod/compute에서 remote_state로 참조:
    # data "terraform_remote_state" "foundation" {
    #   backend = "s3"
    #   config = {
    #     bucket = "wealist-tf-state-advanced-k8s"
    #     key    = "prod/foundation/terraform.tfstate"
    #     region = "ap-northeast-2"
    #   }
    # }

  EOT
}
