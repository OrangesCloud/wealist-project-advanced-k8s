# =============================================================================
# Helm Releases for EKS Add-ons
# =============================================================================
# 설치 순서:
# 1. Gateway API CRDs (Istio 의존성)
# 2. Istio (Base → Istiod → CNI → ztunnel → Ingress Gateway)
# 3. ArgoCD
# 4. 인프라 컴포넌트 (ALB Controller, External Secrets, cert-manager, Cluster Autoscaler)
# 5. ArgoCD Bootstrap App (App of Apps로 나머지 관리)
#
# ArgoCD가 관리하는 항목 (k8s/argocd/apps/prod/):
# - external-secrets-config (ClusterSecretStore + ExternalSecret)
# - wealist-infrastructure (공통 ConfigMap)
# - 마이크로서비스 (auth, user, board, chat, noti, storage, video)
# - istio-config (HTTPRoute, Gateway, AuthorizationPolicy)
# - monitoring (Prometheus, Grafana, Loki)

# -----------------------------------------------------------------------------
# Helm Provider
# -----------------------------------------------------------------------------
provider "helm" {
  kubernetes {
    host                   = module.eks.cluster_endpoint
    cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)

    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
    }
  }
}

# =============================================================================
# 1. Gateway API CRDs (Istio Ambient 필수)
# =============================================================================
# Gateway API는 Helm 차트가 없으므로 kubectl apply로 설치
resource "null_resource" "gateway_api_crds" {
  triggers = {
    cluster_name = module.eks.cluster_name
    version      = "v1.2.0"
  }

  provisioner "local-exec" {
    command = <<-EOT
      aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region}
      kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml
    EOT
  }

  depends_on = [module.eks]
}

# =============================================================================
# 2. Istio Ambient Mode
# =============================================================================
resource "helm_release" "istio_base" {
  name       = "istio-base"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "base"
  version    = "1.28.2"
  namespace  = "istio-system"

  create_namespace = true

  depends_on = [null_resource.gateway_api_crds]
}

resource "helm_release" "istiod" {
  name       = "istiod"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "istiod"
  version    = "1.28.2"
  namespace  = "istio-system"

  set {
    name  = "profile"
    value = "ambient"
  }

  depends_on = [helm_release.istio_base]
}

resource "helm_release" "istio_cni" {
  name       = "istio-cni"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "cni"
  version    = "1.28.2"
  namespace  = "istio-system"

  set {
    name  = "profile"
    value = "ambient"
  }

  depends_on = [helm_release.istiod]
}

resource "helm_release" "istio_ztunnel" {
  name       = "ztunnel"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "ztunnel"
  version    = "1.28.2"
  namespace  = "istio-system"

  depends_on = [helm_release.istio_cni]
}

# =============================================================================
# Istio Ingress Gateway - MOVED TO ArgoCD
# =============================================================================
# Istio Ingress Gateway는 ALB Controller가 필요하므로 ArgoCD에서 관리
# ArgoCD App: cluster-addons (sync-wave: 0)
#
# 설치 순서:
# 1. ArgoCD: AWS Load Balancer Controller (sync-wave: -2)
# 2. ArgoCD: Istio Ingress Gateway (sync-wave: 0)
# =============================================================================

# =============================================================================
# 3. ArgoCD
# =============================================================================
resource "helm_release" "argocd" {
  name       = "argocd"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "5.55.0"  # 2024-12 stable
  namespace  = "argocd"

  create_namespace = true

  # HA 비활성화 (비용 절감)
  set {
    name  = "controller.replicas"
    value = "1"
  }

  set {
    name  = "server.replicas"
    value = "1"
  }

  set {
    name  = "repoServer.replicas"
    value = "1"
  }

  set {
    name  = "applicationSet.replicas"
    value = "1"
  }

  # Insecure mode (TLS termination at ALB)
  set {
    name  = "server.insecure"
    value = "true"
  }

  depends_on = [helm_release.istio_ztunnel]
}

# =============================================================================
# Cluster Info ConfigMap (ArgoCD에서 참조)
# =============================================================================
# ArgoCD에서 cluster-addons 차트 배포 시 이 정보 사용
resource "kubernetes_config_map" "cluster_info" {
  metadata {
    name      = "cluster-info"
    namespace = "argocd"
  }

  data = {
    CLUSTER_NAME = module.eks.cluster_name
    AWS_REGION   = var.aws_region
    VPC_ID       = local.vpc_id
  }

  depends_on = [helm_release.argocd]
}

# =============================================================================
# Namespaces - ArgoCD에서 자동 생성 (CreateNamespace=true)
# =============================================================================
# external-secrets, cert-manager 네임스페이스는 ArgoCD가 생성/관리

# =============================================================================
# 4. ArgoCD Bootstrap - MOVED TO argocd-apps layer
# =============================================================================
# ArgoCD Project, Application은 별도 레이어에서 관리
# 이유: kubernetes_manifest는 plan 시점에 클러스터 연결 필요
#
# 배포 순서:
# 1. compute: EKS + Helm (ArgoCD 설치 포함)
# 2. argocd-apps: ArgoCD Project + Application
#
# 다음 단계:
# cd ../argocd-apps && terraform apply
# =============================================================================
