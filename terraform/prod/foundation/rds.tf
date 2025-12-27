# =============================================================================
# RDS PostgreSQL Configuration
# =============================================================================
# 비용 최적화: db.t4g.small, Single-AZ (~$23/월)
# 추후 트래픽 증가 시 Multi-AZ로 전환 가능
#
# Secrets Manager로 비밀번호 자동 관리 (manage_master_user_password = true)

module "rds" {
  source  = "terraform-aws-modules/rds/aws"
  version = "~> 6.3"

  identifier = "${local.name_prefix}-postgres"

  # -----------------------------------------------------------------------------
  # Engine Configuration
  # -----------------------------------------------------------------------------
  engine               = "postgres"
  engine_version       = "17.7"
  family               = "postgres17"
  major_engine_version = "17"
  instance_class       = var.rds_instance_class

  # -----------------------------------------------------------------------------
  # Storage Configuration
  # -----------------------------------------------------------------------------
  allocated_storage     = var.rds_allocated_storage
  max_allocated_storage = var.rds_max_allocated_storage
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id           = module.kms.key_arn

  # -----------------------------------------------------------------------------
  # Database Configuration
  # -----------------------------------------------------------------------------
  db_name  = "wealist"
  username = "wealist_admin"
  port     = 5432

  # Secrets Manager로 비밀번호 자동 관리
  manage_master_user_password = true

  # -----------------------------------------------------------------------------
  # High Availability
  # -----------------------------------------------------------------------------
  # 비용 절감: Single-AZ (~$23/월 vs $46/월)
  multi_az = var.rds_multi_az

  # -----------------------------------------------------------------------------
  # Network Configuration
  # -----------------------------------------------------------------------------
  db_subnet_group_name   = module.vpc.database_subnet_group_name
  vpc_security_group_ids = [aws_security_group.rds.id]

  # Public 접근 불가 (Private Subnet에만 배치)
  publicly_accessible = false

  # -----------------------------------------------------------------------------
  # Maintenance & Backup
  # -----------------------------------------------------------------------------
  maintenance_window              = "Mon:00:00-Mon:03:00"
  backup_window                   = "03:00-06:00"
  backup_retention_period         = var.rds_backup_retention_days
  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]

  # -----------------------------------------------------------------------------
  # Performance Insights (무료 7일)
  # -----------------------------------------------------------------------------
  performance_insights_enabled          = true
  performance_insights_retention_period = 7

  # -----------------------------------------------------------------------------
  # Parameter Group
  # -----------------------------------------------------------------------------
  parameters = [
    {
      name         = "shared_preload_libraries"
      value        = "pg_stat_statements"
      apply_method = "pending-reboot"  # static parameter
    },
    {
      name  = "log_statement"
      value = "ddl"
    },
    {
      name  = "log_min_duration_statement"
      value = "1000"  # 1초 이상 쿼리 로깅
    }
  ]

  # -----------------------------------------------------------------------------
  # Protection
  # -----------------------------------------------------------------------------
  deletion_protection = var.enable_deletion_protection
  skip_final_snapshot = !var.enable_deletion_protection

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# Security Group for RDS
# -----------------------------------------------------------------------------
resource "aws_security_group" "rds" {
  name        = "${local.name_prefix}-rds-sg"
  description = "Security group for RDS PostgreSQL"
  vpc_id      = module.vpc.vpc_id

  # EKS 노드에서의 접근만 허용 (prod/compute에서 SG ID 참조)
  ingress {
    description = "PostgreSQL from VPC"
    from_port   = 5432
    to_port     = 5432
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
    Name = "${local.name_prefix}-rds-sg"
  })
}
