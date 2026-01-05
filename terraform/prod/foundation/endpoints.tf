# =============================================================================
# VPC Endpoints
# =============================================================================
# S3 Gateway Endpoint만 사용 (무료)
#
# Interface Endpoints는 비용 발생 (~$7/월 each):
# - 월 ~778GB 이상 데이터 처리 시에만 NAT Gateway 대비 이득
# - 소규모/중규모 트래픽에서는 오히려 비용 증가
#
# 트래픽 증가 시 아래 주석 해제하여 추가:
# - Secrets Manager, ECR API/DKR, STS, CloudWatch Logs

# -----------------------------------------------------------------------------
# S3 Gateway Endpoint (무료)
# -----------------------------------------------------------------------------
# ECR 이미지 레이어가 S3에 저장되므로 이미지 풀링 시 NAT 우회
# 애플리케이션 파일 스토리지 접근도 NAT 우회
resource "aws_vpc_endpoint" "s3" {
  vpc_id            = module.vpc.vpc_id
  service_name      = "com.amazonaws.${var.aws_region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = module.vpc.private_route_table_ids

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-s3-endpoint"
  })
}

# =============================================================================
# Interface Endpoints (비용 발생 - 필요 시 주석 해제)
# =============================================================================
# 트래픽이 많아지면 아래 Endpoints 추가 고려
# 손익분기점: 월 ~778GB 이상

# # Security Group for Interface Endpoints
# resource "aws_security_group" "vpc_endpoints" {
#   name        = "${local.name_prefix}-vpc-endpoints-sg"
#   description = "Security group for VPC Interface Endpoints"
#   vpc_id      = module.vpc.vpc_id
#
#   ingress {
#     description = "HTTPS from VPC"
#     from_port   = 443
#     to_port     = 443
#     protocol    = "tcp"
#     cidr_blocks = [module.vpc.vpc_cidr_block]
#   }
#
#   egress {
#     from_port   = 0
#     to_port     = 0
#     protocol    = "-1"
#     cidr_blocks = ["0.0.0.0/0"]
#   }
#
#   tags = merge(local.common_tags, {
#     Name = "${local.name_prefix}-vpc-endpoints-sg"
#   })
# }

# # Secrets Manager - ExternalSecrets Operator용
# resource "aws_vpc_endpoint" "secretsmanager" {
#   vpc_id              = module.vpc.vpc_id
#   service_name        = "com.amazonaws.${var.aws_region}.secretsmanager"
#   vpc_endpoint_type   = "Interface"
#   subnet_ids          = module.vpc.private_subnets
#   security_group_ids  = [aws_security_group.vpc_endpoints.id]
#   private_dns_enabled = true
#   tags = merge(local.common_tags, { Name = "${local.name_prefix}-secretsmanager-endpoint" })
# }

# # ECR API - 레지스트리 메타데이터, 인증
# resource "aws_vpc_endpoint" "ecr_api" {
#   vpc_id              = module.vpc.vpc_id
#   service_name        = "com.amazonaws.${var.aws_region}.ecr.api"
#   vpc_endpoint_type   = "Interface"
#   subnet_ids          = module.vpc.private_subnets
#   security_group_ids  = [aws_security_group.vpc_endpoints.id]
#   private_dns_enabled = true
#   tags = merge(local.common_tags, { Name = "${local.name_prefix}-ecr-api-endpoint" })
# }

# # ECR DKR - Docker 이미지 풀링
# resource "aws_vpc_endpoint" "ecr_dkr" {
#   vpc_id              = module.vpc.vpc_id
#   service_name        = "com.amazonaws.${var.aws_region}.ecr.dkr"
#   vpc_endpoint_type   = "Interface"
#   subnet_ids          = module.vpc.private_subnets
#   security_group_ids  = [aws_security_group.vpc_endpoints.id]
#   private_dns_enabled = true
#   tags = merge(local.common_tags, { Name = "${local.name_prefix}-ecr-dkr-endpoint" })
# }

# # STS - Pod Identity 인증
# resource "aws_vpc_endpoint" "sts" {
#   vpc_id              = module.vpc.vpc_id
#   service_name        = "com.amazonaws.${var.aws_region}.sts"
#   vpc_endpoint_type   = "Interface"
#   subnet_ids          = module.vpc.private_subnets
#   security_group_ids  = [aws_security_group.vpc_endpoints.id]
#   private_dns_enabled = true
#   tags = merge(local.common_tags, { Name = "${local.name_prefix}-sts-endpoint" })
# }

# # CloudWatch Logs - 로그 전송
# resource "aws_vpc_endpoint" "logs" {
#   vpc_id              = module.vpc.vpc_id
#   service_name        = "com.amazonaws.${var.aws_region}.logs"
#   vpc_endpoint_type   = "Interface"
#   subnet_ids          = module.vpc.private_subnets
#   security_group_ids  = [aws_security_group.vpc_endpoints.id]
#   private_dns_enabled = true
#   tags = merge(local.common_tags, { Name = "${local.name_prefix}-logs-endpoint" })
# }
