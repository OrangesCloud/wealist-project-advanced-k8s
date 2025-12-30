# =============================================================================
# Pod Identity Module
# =============================================================================
# EKS Pod Identity를 사용하여 Kubernetes ServiceAccount에 IAM Role을 연결합니다.
# IRSA보다 간단하며, OIDC Provider 설정이 필요 없습니다.
#
# 사용법:
#   module "pod_identity_alb" {
#     source          = "../../modules/pod-identity"
#     name            = "alb-controller"
#     cluster_name    = module.eks.cluster_name
#     namespace       = "kube-system"
#     service_account = "aws-load-balancer-controller"
#     policy_arns     = [aws_iam_policy.alb_controller.arn]
#   }

# -----------------------------------------------------------------------------
# IAM Role for Pod Identity
# -----------------------------------------------------------------------------
resource "aws_iam_role" "this" {
  name = "${var.name}-pod-identity"
  path = "/pod-identity/"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "pods.eks.amazonaws.com"
        }
        Action = [
          "sts:AssumeRole",
          "sts:TagSession"
        ]
      }
    ]
  })

  tags = merge(var.tags, {
    Name        = "${var.name}-pod-identity"
    ClusterName = var.cluster_name
    Namespace   = var.namespace
    ServiceAccount = var.service_account
  })
}

# -----------------------------------------------------------------------------
# Attach Managed Policies
# -----------------------------------------------------------------------------
resource "aws_iam_role_policy_attachment" "managed" {
  count = length(var.policy_arns)

  role       = aws_iam_role.this.name
  policy_arn = var.policy_arns[count.index]
}

# -----------------------------------------------------------------------------
# Inline Policies (optional, multiple)
# -----------------------------------------------------------------------------
resource "aws_iam_role_policy" "inline" {
  for_each = var.inline_policies

  name   = each.key
  role   = aws_iam_role.this.id
  policy = each.value
}

# -----------------------------------------------------------------------------
# Pod Identity Association
# -----------------------------------------------------------------------------
resource "aws_eks_pod_identity_association" "this" {
  cluster_name    = var.cluster_name
  namespace       = var.namespace
  service_account = var.service_account
  role_arn        = aws_iam_role.this.arn

  tags = var.tags
}
