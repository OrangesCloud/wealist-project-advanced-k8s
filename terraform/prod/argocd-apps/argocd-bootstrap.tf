# =============================================================================
# ArgoCD Bootstrap (App of Apps Pattern)
# =============================================================================
# ArgoCD가 설치된 후 자동으로 모든 앱을 관리하도록 설정
#
# 배포 순서:
# 1. compute layer에서 ArgoCD Helm 설치
# 2. 이 layer에서 Project + Application 생성
# =============================================================================

# -----------------------------------------------------------------------------
# wealist-prod 네임스페이스 생성
# -----------------------------------------------------------------------------
resource "kubernetes_namespace" "wealist_prod" {
  metadata {
    name = "wealist-prod"
    labels = {
      "istio.io/dataplane-mode" = "ambient"
    }
  }
}

# -----------------------------------------------------------------------------
# ArgoCD Project 생성 (wealist-prod)
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "argocd_project_prod" {
  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "AppProject"
    metadata = {
      name      = "wealist-prod"
      namespace = "argocd"
    }
    spec = {
      description = "Wealist Production Environment"
      sourceRepos = [
        var.git_repo_url,
        # Helm Chart Repositories for cluster addons
        "https://aws.github.io/eks-charts",
        "https://charts.external-secrets.io",
        "https://charts.jetstack.io",
        "https://kubernetes.github.io/autoscaler",
        "https://kubernetes-sigs.github.io/external-dns",
        "https://istio-release.storage.googleapis.com/charts"
      ]
      destinations = [
        {
          namespace = "wealist-prod"
          server    = "https://kubernetes.default.svc"
        },
        {
          namespace = "argocd"
          server    = "https://kubernetes.default.svc"
        },
        {
          namespace = "kube-system"
          server    = "https://kubernetes.default.svc"
        },
        {
          namespace = "external-secrets"
          server    = "https://kubernetes.default.svc"
        },
        {
          namespace = "cert-manager"
          server    = "https://kubernetes.default.svc"
        },
        {
          namespace = "external-dns"
          server    = "https://kubernetes.default.svc"
        },
        {
          namespace = "istio-system"
          server    = "https://kubernetes.default.svc"
        }
      ]
      clusterResourceWhitelist = [
        {
          group = ""
          kind  = "Namespace"
        },
        {
          group = "*"
          kind  = "CustomResourceDefinition"
        },
        {
          group = "*"
          kind  = "ClusterRole"
        },
        {
          group = "*"
          kind  = "ClusterRoleBinding"
        },
        {
          group = "admissionregistration.k8s.io"
          kind  = "*"
        },
        {
          group = "external-secrets.io"
          kind  = "ClusterSecretStore"
        },
        {
          group = "cert-manager.io"
          kind  = "*"
        },
        # AWS Load Balancer Controller requires IngressClass
        {
          group = "networking.k8s.io"
          kind  = "IngressClass"
        },
        # Kubernetes Gateway API (for Istio Ambient mode)
        {
          group = "gateway.networking.k8s.io"
          kind  = "*"
        }
      ]
      namespaceResourceWhitelist = [
        {
          group = "*"
          kind  = "*"
        }
      ]
    }
  }
}

# -----------------------------------------------------------------------------
# ArgoCD Root Application (App of Apps)
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "argocd_root_app" {
  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "wealist-apps-prod"
      namespace = "argocd"
    }
    spec = {
      project = "wealist-prod"
      source = {
        repoURL        = var.git_repo_url
        targetRevision = var.git_target_revision
        path           = var.argocd_apps_path
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "argocd"
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
        syncOptions = [
          "CreateNamespace=true"
        ]
      }
    }
  }

  depends_on = [
    kubernetes_manifest.argocd_project_prod,
    kubernetes_namespace.wealist_prod
  ]
}
