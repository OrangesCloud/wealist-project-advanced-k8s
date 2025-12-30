# =============================================================================
# Production Foundation - Variables
# =============================================================================

variable "aws_region" {
  description = "AWS Region"
  type        = string
  default     = "ap-northeast-2"
}

# =============================================================================
# VPC Configuration
# =============================================================================
variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "private_subnet_cidrs" {
  description = "CIDR blocks for private subnets"
  type        = list(string)
  default     = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
}

variable "public_subnet_cidrs" {
  description = "CIDR blocks for public subnets"
  type        = list(string)
  default     = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
}

variable "database_subnet_cidrs" {
  description = "CIDR blocks for database subnets"
  type        = list(string)
  default     = ["10.0.201.0/24", "10.0.202.0/24", "10.0.203.0/24"]
}

variable "enable_vpc_flow_logs" {
  description = "Enable VPC Flow Logs"
  type        = bool
  default     = false  # 비용 절감을 위해 기본 비활성화
}

# =============================================================================
# RDS Configuration
# =============================================================================
variable "rds_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t4g.small"  # 2 vCPU, 2GB RAM (~$23/월)
}

variable "rds_allocated_storage" {
  description = "Initial allocated storage in GB"
  type        = number
  default     = 20
}

variable "rds_max_allocated_storage" {
  description = "Maximum allocated storage for autoscaling in GB"
  type        = number
  default     = 100
}

variable "rds_multi_az" {
  description = "Enable Multi-AZ deployment"
  type        = bool
  default     = false  # 비용 절감: Single-AZ
}

variable "rds_backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 7
}

# =============================================================================
# ElastiCache Redis Configuration
# =============================================================================
variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t4g.small"  # 1.4GB RAM (~$20/월)
}

variable "redis_num_cache_clusters" {
  description = "Number of cache clusters (nodes)"
  type        = number
  default     = 1  # 비용 절감: 단일 노드
}

# =============================================================================
# S3 Configuration
# =============================================================================
variable "cors_allowed_origins" {
  description = "Allowed origins for CORS"
  type        = list(string)
  default     = ["https://wealist.co.kr", "https://www.wealist.co.kr"]
}

# =============================================================================
# Protection
# =============================================================================
variable "enable_deletion_protection" {
  description = "Enable deletion protection for RDS"
  type        = bool
  default     = true
}
