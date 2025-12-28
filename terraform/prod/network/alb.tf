# =============================================================================
# Application Load Balancer (ALB)
# For api.wealist.co.kr → Istio Gateway
# Requires: VPC from foundation (or vpc_id/public_subnet_ids variables)
# =============================================================================

# -----------------------------------------------------------------------------
# ALB
# -----------------------------------------------------------------------------

resource "aws_lb" "api" {
  count = local.alb_enabled ? 1 : 0

  name               = "${local.name_prefix}-api-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb[0].id]
  subnets            = local.public_subnet_ids

  enable_deletion_protection = var.alb_deletion_protection
  idle_timeout               = var.alb_idle_timeout

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-api-alb"
  })
}

# -----------------------------------------------------------------------------
# Target Group (Istio Gateway)
# -----------------------------------------------------------------------------

resource "aws_lb_target_group" "istio" {
  count = local.alb_enabled ? 1 : 0

  name        = "${local.name_prefix}-istio-tg"
  port        = 80
  protocol    = "HTTP"
  vpc_id      = local.vpc_id
  target_type = "ip" # Istio Gateway Pod IPs

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 5
    interval            = 30
    path                = var.istio_health_check_path
    port                = var.istio_health_check_port
    protocol            = "HTTP"
    matcher             = "200"
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-istio-tg"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# -----------------------------------------------------------------------------
# HTTPS Listener (443)
# -----------------------------------------------------------------------------

resource "aws_lb_listener" "https" {
  count = local.alb_enabled ? 1 : 0

  load_balancer_arn = aws_lb.api[0].arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"
  certificate_arn   = data.aws_acm_certificate.alb.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.istio[0].arn
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# HTTP Listener (80) - Redirect to HTTPS
# -----------------------------------------------------------------------------

resource "aws_lb_listener" "http" {
  count = local.alb_enabled ? 1 : 0

  load_balancer_arn = aws_lb.api[0].arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }

  tags = local.common_tags
}

# =============================================================================
# Note: Target Registration
# =============================================================================
#
# Targets are NOT registered here. Instead, use AWS Load Balancer Controller
# with TargetGroupBinding CRD to automatically register Istio Gateway Pods.
#
# Example TargetGroupBinding (apply to Kubernetes after terraform apply):
#
# apiVersion: elbv2.k8s.aws/v1beta1
# kind: TargetGroupBinding
# metadata:
#   name: istio-gateway-tgb
#   namespace: istio-system
# spec:
#   serviceRef:
#     name: istio-ingressgateway
#     port: 80
#   targetGroupARN: ${aws_lb_target_group.istio[0].arn}
#   targetType: ip
#
# =============================================================================
