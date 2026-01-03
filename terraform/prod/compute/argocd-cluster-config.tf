# =============================================================================
# ArgoCD Cluster Configuration (GitOps Bridge Pattern)
# =============================================================================
# Terraform에서 인프라 값을 ConfigMap으로 주입
# ArgoCD가 이 ConfigMap 값을 참조하여 cluster-addons 배포
#
# 참조: https://github.com/aws-ia/terraform-aws-eks-blueprints-addons
# =============================================================================

# -----------------------------------------------------------------------------
# Cluster Config ConfigMap
# -----------------------------------------------------------------------------
# ALB Controller, External DNS 등이 필요로 하는 인프라 값 저장
resource "kubernetes_config_map" "argocd_cluster_config" {
  metadata {
    name      = "wealist-cluster-config"
    namespace = "argocd"
    labels = {
      "app.kubernetes.io/part-of" = "argocd"
      "wealist.co.kr/managed-by"  = "terraform"
    }
  }

  data = {
    # AWS Infrastructure
    AWS_REGION       = var.aws_region
    AWS_ACCOUNT_ID   = data.aws_caller_identity.current.account_id
    CLUSTER_NAME     = module.eks.cluster_name
    VPC_ID           = local.vpc_id

    # EKS Cluster Info
    CLUSTER_ENDPOINT = module.eks.cluster_endpoint
    CLUSTER_VERSION  = module.eks.cluster_version

    # Domain
    DOMAIN = "wealist.co.kr"
  }

  depends_on = [helm_release.argocd]
}

# Note: data.aws_caller_identity.current is defined in main.tf
