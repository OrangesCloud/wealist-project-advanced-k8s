# =============================================================================
# Bastion Host for SSM Access
# =============================================================================
# RDS, Redis 등 Private 리소스 접속용
# SSM Session Manager로 접속 (SSH 불필요, 퍼블릭 IP 불필요)
#
# 비용: t3.micro (Free Tier 또는 ~$7/월)

# -----------------------------------------------------------------------------
# Amazon Linux 2023 AMI (SSM Agent 기본 포함)
# -----------------------------------------------------------------------------
data "aws_ami" "amazon_linux_2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

# -----------------------------------------------------------------------------
# IAM Role for SSM
# -----------------------------------------------------------------------------
resource "aws_iam_role" "bastion" {
  name = "${local.name_prefix}-bastion-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy_attachment" "bastion_ssm" {
  role       = aws_iam_role.bastion.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "bastion" {
  name = "${local.name_prefix}-bastion-profile"
  role = aws_iam_role.bastion.name

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# Security Group
# -----------------------------------------------------------------------------
resource "aws_security_group" "bastion" {
  name        = "${local.name_prefix}-bastion-sg"
  description = "Security group for Bastion host"
  vpc_id      = module.vpc.vpc_id

  # SSM은 outbound HTTPS만 필요 (inbound 규칙 불필요)
  egress {
    description = "HTTPS for SSM"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # RDS 접속
  egress {
    description     = "PostgreSQL to RDS"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.rds.id]
  }

  # Redis 접속
  egress {
    description     = "Redis to ElastiCache"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.redis.id]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-bastion-sg"
  })
}

# -----------------------------------------------------------------------------
# EC2 Instance
# -----------------------------------------------------------------------------
resource "aws_instance" "bastion" {
  ami                    = data.aws_ami.amazon_linux_2023.id
  instance_type          = "t3.micro"
  subnet_id              = module.vpc.private_subnets[0]
  vpc_security_group_ids = [aws_security_group.bastion.id]
  iam_instance_profile   = aws_iam_instance_profile.bastion.name

  # EBS 최적화 (t3는 기본 지원)
  ebs_optimized = true

  # Root 볼륨
  root_block_device {
    volume_type           = "gp3"
    volume_size           = 8
    encrypted             = true
    kms_key_id            = module.kms.key_arn
    delete_on_termination = true
  }

  # 메타데이터 서비스 v2 (보안 강화)
  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
  }

  # PostgreSQL 클라이언트 설치
  user_data = base64encode(<<-EOF
    #!/bin/bash
    dnf update -y
    dnf install -y postgresql15
    echo "Bastion setup complete"
  EOF
  )

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-bastion"
  })

  lifecycle {
    ignore_changes = [ami]  # AMI 업데이트 시 재생성 방지
  }
}

# -----------------------------------------------------------------------------
# RDS Security Group - Bastion 접근 허용
# -----------------------------------------------------------------------------
resource "aws_security_group_rule" "rds_from_bastion" {
  type                     = "ingress"
  from_port                = 5432
  to_port                  = 5432
  protocol                 = "tcp"
  security_group_id        = aws_security_group.rds.id
  source_security_group_id = aws_security_group.bastion.id
  description              = "PostgreSQL from Bastion"
}

# -----------------------------------------------------------------------------
# Redis Security Group - Bastion 접근 허용
# -----------------------------------------------------------------------------
resource "aws_security_group_rule" "redis_from_bastion" {
  type                     = "ingress"
  from_port                = 6379
  to_port                  = 6379
  protocol                 = "tcp"
  security_group_id        = aws_security_group.redis.id
  source_security_group_id = aws_security_group.bastion.id
  description              = "Redis from Bastion"
}
