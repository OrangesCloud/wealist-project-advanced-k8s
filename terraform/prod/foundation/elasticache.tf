# =============================================================================
# ElastiCache Redis Configuration
# =============================================================================
# 비용 최적화: cache.t4g.small, 단일 노드 (~$20/월)
# 세션/토큰 캐싱용

# -----------------------------------------------------------------------------
# Redis Replication Group
# -----------------------------------------------------------------------------
resource "aws_elasticache_replication_group" "redis" {
  replication_group_id = "${local.name_prefix}-redis"
  description          = "Redis cluster for wealist production"

  # -----------------------------------------------------------------------------
  # Engine Configuration
  # -----------------------------------------------------------------------------
  engine               = "redis"
  engine_version       = "7.1"
  node_type            = var.redis_node_type
  port                 = 6379
  parameter_group_name = aws_elasticache_parameter_group.redis.name

  # -----------------------------------------------------------------------------
  # Cluster Configuration
  # -----------------------------------------------------------------------------
  # 비용 절감: 단일 노드 (~$20/월)
  num_cache_clusters         = var.redis_num_cache_clusters
  automatic_failover_enabled = var.redis_num_cache_clusters > 1
  multi_az_enabled          = var.redis_num_cache_clusters > 1

  # -----------------------------------------------------------------------------
  # Security
  # -----------------------------------------------------------------------------
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = random_password.redis_auth.result
  kms_key_id                = module.kms.key_arn

  # -----------------------------------------------------------------------------
  # Network Configuration
  # -----------------------------------------------------------------------------
  subnet_group_name  = aws_elasticache_subnet_group.redis.name
  security_group_ids = [aws_security_group.redis.id]

  # -----------------------------------------------------------------------------
  # Maintenance
  # -----------------------------------------------------------------------------
  maintenance_window       = "tue:05:00-tue:09:00"
  snapshot_window          = "00:00-05:00"
  snapshot_retention_limit = 7

  # -----------------------------------------------------------------------------
  # Updates
  # -----------------------------------------------------------------------------
  apply_immediately          = false
  auto_minor_version_upgrade = true

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# Parameter Group
# -----------------------------------------------------------------------------
resource "aws_elasticache_parameter_group" "redis" {
  family = "redis7"
  name   = "${local.name_prefix}-redis-params"

  parameter {
    name  = "maxmemory-policy"
    value = "volatile-lru"
  }
}

# -----------------------------------------------------------------------------
# Subnet Group
# -----------------------------------------------------------------------------
resource "aws_elasticache_subnet_group" "redis" {
  name       = "${local.name_prefix}-redis-subnet"
  subnet_ids = module.vpc.private_subnets

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# Security Group
# -----------------------------------------------------------------------------
resource "aws_security_group" "redis" {
  name        = "${local.name_prefix}-redis-sg"
  description = "Security group for ElastiCache Redis"
  vpc_id      = module.vpc.vpc_id

  ingress {
    description = "Redis from VPC"
    from_port   = 6379
    to_port     = 6379
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-redis-sg"
  })
}

# -----------------------------------------------------------------------------
# Auth Token
# -----------------------------------------------------------------------------
resource "random_password" "redis_auth" {
  length  = 32
  special = false
}

# Store auth token in Secrets Manager
resource "aws_secretsmanager_secret" "redis_auth" {
  name = "wealist/prod/redis/auth-token"

  tags = local.common_tags
}

resource "aws_secretsmanager_secret_version" "redis_auth" {
  secret_id     = aws_secretsmanager_secret.redis_auth.id
  secret_string = random_password.redis_auth.result
}
