# =============================================================================
# VPC Configuration
# =============================================================================
# 3개 AZ에 걸쳐 Public/Private/Database 서브넷 구성
# 비용 최적화: 단일 NAT Gateway ($32/월)
#
# 주의: 단일 NAT Gateway는 SPOF (해당 AZ 장애 시 outbound 불가)
# 고가용성 필요 시 single_nat_gateway = false, one_nat_gateway_per_az = true 로 변경

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.5"

  name = "${local.name_prefix}-vpc"
  cidr = var.vpc_cidr

  azs              = local.azs
  private_subnets  = var.private_subnet_cidrs
  public_subnets   = var.public_subnet_cidrs
  database_subnets = var.database_subnet_cidrs

  # -----------------------------------------------------------------------------
  # NAT Gateway Configuration
  # -----------------------------------------------------------------------------
  # 비용 최적화: 단일 NAT Gateway 사용 ($32/월 vs $96/월)
  enable_nat_gateway     = true
  single_nat_gateway     = true
  one_nat_gateway_per_az = false

  # -----------------------------------------------------------------------------
  # DNS Configuration
  # -----------------------------------------------------------------------------
  enable_dns_hostnames = true
  enable_dns_support   = true

  # -----------------------------------------------------------------------------
  # Database Subnet Group
  # -----------------------------------------------------------------------------
  create_database_subnet_group       = true
  create_database_subnet_route_table = true
  database_subnet_group_name         = "${local.name_prefix}-db-subnet-group"

  # -----------------------------------------------------------------------------
  # VPC Flow Logs (선택적)
  # -----------------------------------------------------------------------------
  enable_flow_log                      = var.enable_vpc_flow_logs
  create_flow_log_cloudwatch_log_group = var.enable_vpc_flow_logs
  create_flow_log_cloudwatch_iam_role  = var.enable_vpc_flow_logs
  flow_log_max_aggregation_interval    = 60

  # -----------------------------------------------------------------------------
  # Subnet Tags for EKS
  # -----------------------------------------------------------------------------
  public_subnet_tags = {
    "kubernetes.io/role/elb"                           = 1
    "kubernetes.io/cluster/${local.name_prefix}-eks"   = "shared"
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb"                  = 1
    "kubernetes.io/cluster/${local.name_prefix}-eks"   = "shared"
  }

  tags = local.common_tags
}
