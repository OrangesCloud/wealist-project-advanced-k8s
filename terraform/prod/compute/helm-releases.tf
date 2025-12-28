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
resource "helm_release" "gateway_api_crds" {
  name       = "gateway-api-crds"
  repository = "https://kubernetes-sigs.github.io/gateway-api"
  chart      = "gateway-api"
  version    = "1.2.0"
  namespace  = "kube-system"

  # CRDs만 설치
  set {
    name  = "installCRDs"
    value = "true"
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

  depends_on = [helm_release.gateway_api_crds]
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

# Istio Ingress Gateway (for external traffic)
resource "helm_release" "istio_ingress" {
  name       = "istio-ingressgateway"
  repository = "https://istio-release.storage.googleapis.com/charts"
  chart      = "gateway"
  version    = "1.28.2"
  namespace  = "istio-system"

  # AWS ALB를 통한 외부 노출
  set {
    name  = "service.type"
    value = "LoadBalancer"
  }

  # AWS Load Balancer Controller annotations
  set {
    name  = "service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-type"
    value = "external"
  }

  set {
    name  = "service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-nlb-target-type"
    value = "ip"
  }

  set {
    name  = "service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-scheme"
    value = "internet-facing"
  }

  # SSL 정책 (CloudFront에서 처리하므로 HTTP만)
  set {
    name  = "service.ports[0].name"
    value = "http"
  }

  set {
    name  = "service.ports[0].port"
    value = "80"
  }

  set {
    name  = "service.ports[0].targetPort"
    value = "80"
  }

  set {
    name  = "service.ports[1].name"
    value = "https"
  }

  set {
    name  = "service.ports[1].port"
    value = "443"
  }

  set {
    name  = "service.ports[1].targetPort"
    value = "443"
  }

  depends_on = [
    helm_release.istio_ztunnel,
    helm_release.aws_load_balancer_controller
  ]
}

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

# -----------------------------------------------------------------------------
# AWS Load Balancer Controller
# -----------------------------------------------------------------------------
# Pod Identity로 IAM 권한 부여 (pod-identity.tf 참조)
resource "helm_release" "aws_load_balancer_controller" {
  name       = "aws-load-balancer-controller"
  repository = "https://aws.github.io/eks-charts"
  chart      = "aws-load-balancer-controller"
  version    = "1.7.1"  # 2024-12 stable version
  namespace  = "kube-system"

  set {
    name  = "clusterName"
    value = module.eks.cluster_name
  }

  set {
    name  = "serviceAccount.create"
    value = "true"
  }

  set {
    name  = "serviceAccount.name"
    value = "aws-load-balancer-controller"
  }

  # VPC ID for target group binding
  set {
    name  = "vpcId"
    value = local.vpc_id
  }

  # Region
  set {
    name  = "region"
    value = var.aws_region
  }

  # Enable WAF and Shield integrations
  set {
    name  = "enableShield"
    value = "false"  # Enable if needed
  }

  set {
    name  = "enableWaf"
    value = "false"  # Enable if needed
  }

  set {
    name  = "enableWafv2"
    value = "false"  # Enable if needed
  }

  depends_on = [
    module.eks,
    module.pod_identity_alb_controller
  ]
}

# -----------------------------------------------------------------------------
# External Secrets Operator
# -----------------------------------------------------------------------------
resource "helm_release" "external_secrets" {
  name       = "external-secrets"
  repository = "https://charts.external-secrets.io"
  chart      = "external-secrets"
  version    = "0.9.11"  # 2024-12 stable version
  namespace  = "external-secrets"

  create_namespace = true

  set {
    name  = "serviceAccount.create"
    value = "true"
  }

  set {
    name  = "serviceAccount.name"
    value = "external-secrets"
  }

  depends_on = [
    module.eks,
    module.pod_identity_external_secrets
  ]
}

# -----------------------------------------------------------------------------
# cert-manager
# -----------------------------------------------------------------------------
resource "helm_release" "cert_manager" {
  name       = "cert-manager"
  repository = "https://charts.jetstack.io"
  chart      = "cert-manager"
  version    = "v1.14.3"  # 2024-12 stable version
  namespace  = "cert-manager"

  create_namespace = true

  set {
    name  = "installCRDs"
    value = "true"
  }

  set {
    name  = "serviceAccount.create"
    value = "true"
  }

  set {
    name  = "serviceAccount.name"
    value = "cert-manager"
  }

  depends_on = [
    module.eks,
    module.pod_identity_cert_manager
  ]
}

# -----------------------------------------------------------------------------
# Cluster Autoscaler
# -----------------------------------------------------------------------------
resource "helm_release" "cluster_autoscaler" {
  name       = "cluster-autoscaler"
  repository = "https://kubernetes.github.io/autoscaler"
  chart      = "cluster-autoscaler"
  version    = "9.35.0"  # 2024-12 stable version
  namespace  = "kube-system"

  set {
    name  = "autoDiscovery.clusterName"
    value = module.eks.cluster_name
  }

  set {
    name  = "awsRegion"
    value = var.aws_region
  }

  set {
    name  = "rbac.serviceAccount.create"
    value = "true"
  }

  set {
    name  = "rbac.serviceAccount.name"
    value = "cluster-autoscaler"
  }

  depends_on = [
    module.eks,
    module.pod_identity_cluster_autoscaler
  ]
}

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
