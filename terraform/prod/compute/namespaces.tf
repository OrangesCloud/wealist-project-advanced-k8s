# =============================================================================
# Kubernetes Namespaces
# =============================================================================
# Terraform에서 네임스페이스를 미리 생성하여 라벨 설정
# ArgoCD의 CreateNamespace=true보다 먼저 생성되어 우선 적용됨
#
# Note: kubernetes provider는 main.tf에서 정의됨

# -----------------------------------------------------------------------------
# wealist-prod Namespace (Istio Sidecar Injection 활성화)
# -----------------------------------------------------------------------------
resource "kubernetes_namespace" "wealist_prod" {
  metadata {
    name = "wealist-prod"

    labels = {
      "istio-injection"          = "enabled"
      "app.kubernetes.io/part-of" = "wealist"
    }
  }

  depends_on = [module.eks]
}
