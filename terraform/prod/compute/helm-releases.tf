# =============================================================================
# Helm Releases for EKS Add-ons
# =============================================================================
# 설치 순서 (Bootstrap 순환 의존성 해결):
# 1. Gateway API CRDs (Istio 의존성)
# 2. Istio (Base → Istiod) - Sidecar Mode
# 3. External Secrets Operator + ClusterSecretStore ← ArgoCD보다 먼저!
# 4. ArgoCD (ESO가 argocd-secret 생성 가능)
# 5. ArgoCD Bootstrap App (App of Apps로 나머지 관리)
#
# 왜 ESO를 Terraform에서 설치하는가?
#   - ArgoCD는 argocd-secret이 있어야 시작 가능
#   - argocd-secret은 ExternalSecret이 생성
#   - ExternalSecret은 ESO가 필요
#   - ESO를 ArgoCD로 설치하면 순환 의존성 발생!
#   - 따라서 ESO는 Terraform에서 ArgoCD보다 먼저 설치
#
# ArgoCD가 관리하는 항목 (k8s/argocd/apps/prod/):
# - external-secrets-config (ExternalSecret 리소스들)
# - wealist-infrastructure (공통 ConfigMap)
# - 마이크로서비스 (auth, user, board, chat, noti, storage)
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
# 1. Gateway API CRDs (Istio Gateway API 지원)
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
# 2. Istio Sidecar Mode
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

  # Sidecar 모드: default profile 사용 (profile 설정 불필요)
  # Sidecar injection은 네임스페이스 라벨(istio-injection=enabled)로 제어

  depends_on = [helm_release.istio_base]
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
# 3. External Secrets Operator (ArgoCD보다 먼저 설치 - Bootstrap 필수)
# =============================================================================
# 왜 ArgoCD보다 먼저?
#   - ArgoCD는 argocd-secret이 있어야 시작 가능 (createSecret=false 설정)
#   - argocd-secret은 ExternalSecret이 생성
#   - ExternalSecret은 ESO + ClusterSecretStore가 필요
#   - ESO를 ArgoCD로 설치하면 순환 의존성 발생!
#
# 설치 구성요소:
#   - external-secrets Helm chart (CRDs 포함)
#   - ClusterSecretStore (AWS Secrets Manager 연동)
# =============================================================================
resource "helm_release" "external_secrets" {
  name       = "external-secrets"
  repository = "https://charts.external-secrets.io"
  chart      = "external-secrets"
  version    = "1.2.0"  # K8s 1.34 호환
  namespace  = "external-secrets"

  create_namespace = true

  # CRDs 설치
  set {
    name  = "installCRDs"
    value = "true"
  }

  # v1beta1 → v1 conversion webhook 비활성화
  set {
    name  = "crds.conversion.enabled"
    value = "false"
  }

  # Service Account 설정 (Pod Identity 연결용)
  set {
    name  = "serviceAccount.create"
    value = "true"
  }

  set {
    name  = "serviceAccount.name"
    value = "external-secrets"
  }

  # Webhook 리소스
  set {
    name  = "webhook.resources.requests.memory"
    value = "64Mi"
  }

  set {
    name  = "webhook.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "webhook.resources.limits.memory"
    value = "128Mi"
  }

  set {
    name  = "webhook.resources.limits.cpu"
    value = "200m"
  }

  # Cert Controller 리소스
  set {
    name  = "certController.resources.requests.memory"
    value = "64Mi"
  }

  set {
    name  = "certController.resources.requests.cpu"
    value = "50m"
  }

  set {
    name  = "certController.resources.limits.memory"
    value = "128Mi"
  }

  set {
    name  = "certController.resources.limits.cpu"
    value = "200m"
  }

  depends_on = [
    helm_release.istiod,
    module.pod_identity_external_secrets
  ]
}

# ESO CRDs가 완전히 등록될 때까지 대기
resource "time_sleep" "wait_for_eso_crds" {
  depends_on = [helm_release.external_secrets]

  create_duration = "30s"
}

# -----------------------------------------------------------------------------
# ClusterSecretStore - AWS Secrets Manager
# -----------------------------------------------------------------------------
# Pod Identity를 통해 AWS Secrets Manager에 접근
# kubectl_manifest 사용: kubernetes_manifest는 plan 시 CRD 검증을 하지만,
# ESO가 아직 설치 안됐으면 CRD가 없어서 실패함. kubectl은 apply 시에만 검증.
resource "kubectl_manifest" "cluster_secret_store" {
  yaml_body = <<-YAML
    apiVersion: external-secrets.io/v1
    kind: ClusterSecretStore
    metadata:
      name: aws-secrets-manager
    spec:
      provider:
        aws:
          service: SecretsManager
          region: ${var.aws_region}
  YAML

  depends_on = [time_sleep.wait_for_eso_crds]
}

# -----------------------------------------------------------------------------
# ArgoCD Namespace (ExternalSecret보다 먼저 생성)
# -----------------------------------------------------------------------------
resource "kubernetes_namespace" "argocd" {
  metadata {
    name = "argocd"
  }

  depends_on = [kubectl_manifest.cluster_secret_store]
}

# -----------------------------------------------------------------------------
# ExternalSecret - argocd-secret (ArgoCD 시작 전에 필요)
# -----------------------------------------------------------------------------
# ArgoCD는 argocd-secret이 있어야 시작 가능
# ExternalSecret이 AWS Secrets Manager에서 credentials를 가져와 생성
resource "kubectl_manifest" "argocd_external_secret" {
  yaml_body = <<-YAML
    apiVersion: external-secrets.io/v1
    kind: ExternalSecret
    metadata:
      name: argocd-oauth-secret
      namespace: argocd
    spec:
      refreshInterval: 1h
      secretStoreRef:
        name: aws-secrets-manager
        kind: ClusterSecretStore
      target:
        name: argocd-secret
        creationPolicy: Owner
        template:
          metadata:
            labels:
              app.kubernetes.io/name: argocd-secret
              app.kubernetes.io/part-of: argocd
          data:
            server.secretkey: "{{ .server_secretkey }}"
            dex.google.clientID: "{{ .dex_google_clientID }}"
            dex.google.clientSecret: "{{ .dex_google_clientSecret }}"
      data:
        - secretKey: server_secretkey
          remoteRef:
            key: "wealist/prod/argocd/server"
            property: secretkey
        - secretKey: dex_google_clientID
          remoteRef:
            key: "wealist/prod/oauth/argocd"
            property: client_id
        - secretKey: dex_google_clientSecret
          remoteRef:
            key: "wealist/prod/oauth/argocd"
            property: client_secret
  YAML

  depends_on = [kubernetes_namespace.argocd]
}

# argocd-secret이 생성될 때까지 대기
resource "time_sleep" "wait_for_argocd_secret" {
  depends_on = [kubectl_manifest.argocd_external_secret]

  create_duration = "30s"
}

# =============================================================================
# 4. ArgoCD
# =============================================================================
resource "helm_release" "argocd" {
  name       = "argocd"
  repository = "https://argoproj.github.io/argo-helm"
  chart      = "argo-cd"
  version    = "5.55.0"  # 2024-12 stable
  namespace  = "argocd"

  # Namespace는 Terraform이 먼저 생성 (ExternalSecret용)
  create_namespace = false

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

  # Insecure mode (TLS termination at NLB)
  # server.insecure: deployment command line flag
  # configs.params: argocd-cmd-params-cm ConfigMap (이게 우선)
  set {
    name  = "server.insecure"
    value = "true"
  }

  set {
    name  = "configs.params.server\\.insecure"
    value = "true"
  }

  # =========================================
  # SSO Configuration (Dex + Google OAuth)
  # =========================================
  # ConfigMap(argocd-cm)에서 Dex config 설정
  # ExternalSecret(argocd-oauth-secret)에서 credentials 주입
  set {
    name  = "dex.enabled"
    value = "true"
  }

  # =========================================
  # Secret Management
  # =========================================
  # argocd-secret은 ExternalSecret이 관리 (Helm 생성 비활성화)
  # ArgoCD가 자기 자신을 sync할 때 Helm이 만든 secret을 삭제하는 것 방지
  set {
    name  = "configs.secret.createSecret"
    value = "false"
  }

  # argocd-secret이 먼저 생성되어야 ArgoCD가 시작 가능
  depends_on = [time_sleep.wait_for_argocd_secret]
}

# =============================================================================
# Cluster Info ConfigMap - argocd-cluster-config.tf에서 관리
# =============================================================================
# wealist-cluster-config ConfigMap으로 통합
# 참조: argocd-cluster-config.tf

# =============================================================================
# Namespaces - ArgoCD에서 자동 생성 (CreateNamespace=true)
# =============================================================================
# external-secrets, cert-manager 네임스페이스는 ArgoCD가 생성/관리

# =============================================================================
# 4. ArgoCD Bootstrap - argocd-bootstrap.tf에서 관리
# =============================================================================
# ArgoCD Project, Application은 null_resource로 kubectl apply
# 참조: argocd-bootstrap.tf
#
# 배포 순서:
# 1. EKS 클러스터 생성
# 2. Istio + ArgoCD Helm 설치
# 3. ArgoCD cluster-config ConfigMap 생성 (GitOps Bridge)
# 4. ArgoCD AppProject + Root App 적용 (kubectl)
# 5. ArgoCD가 자동으로 k8s/argocd/apps/prod/ sync
# =============================================================================
