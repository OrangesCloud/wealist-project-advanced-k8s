# =============================================================================
# Security Groups
# Requires: VPC from foundation (or vpc_id variable)
# =============================================================================

# -----------------------------------------------------------------------------
# ALB Security Group
# -----------------------------------------------------------------------------

resource "aws_security_group" "alb" {
  count = local.alb_enabled ? 1 : 0

  name        = "${local.name_prefix}-alb-sg"
  description = "Security group for API ALB"
  vpc_id      = local.vpc_id

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-alb-sg"
  })
}

# Ingress: HTTP (redirect to HTTPS)
resource "aws_security_group_rule" "alb_ingress_http" {
  count = local.alb_enabled ? 1 : 0

  type              = "ingress"
  from_port         = 80
  to_port           = 80
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.alb[0].id
  description       = "Allow HTTP from anywhere (redirects to HTTPS)"
}

# Ingress: HTTPS
resource "aws_security_group_rule" "alb_ingress_https" {
  count = local.alb_enabled ? 1 : 0

  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.alb[0].id
  description       = "Allow HTTPS from anywhere"
}

# Egress: All traffic to VPC (for target health checks and traffic)
resource "aws_security_group_rule" "alb_egress_vpc" {
  count = local.alb_enabled ? 1 : 0

  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = [local.vpc_cidr_block]
  security_group_id = aws_security_group.alb[0].id
  description       = "Allow all egress to VPC"
}
