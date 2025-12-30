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
  field_manager {
    force_conflicts = true
  }

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
        "https://kubernetes-sigs.github.io/metrics-server",
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
        },
        # Metrics Server requires APIService
        {
          group = "apiregistration.k8s.io"
          kind  = "APIService"
        },
        # AWS Load Balancer Controller requires elbv2 CRDs
        {
          group = "elbv2.k8s.aws"
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

# -----------------------------------------------------------------------------
# Wealist Infrastructure Application
# -----------------------------------------------------------------------------
# Git에서 관리하지 않고 Terraform에서 직접 생성
# 이유: RDS/ElastiCache 호스트 정보를 Terraform outputs에서 주입해야 함
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "wealist_infrastructure" {
  field_manager {
    force_conflicts = true
  }

  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "wealist-infrastructure-prod"
      namespace = "argocd"
      annotations = {
        "argocd.argoproj.io/sync-wave" = "0"
      }
    }
    spec = {
      project = "wealist-prod"
      source = {
        repoURL        = var.git_repo_url
        targetRevision = var.git_target_revision
        path           = "k8s/helm/charts/wealist-infrastructure"
        helm = {
          valueFiles = [
            "values.yaml",
            "../../environments/base.yaml",
            "../../environments/prod.yaml"
          ]
          # RDS/ElastiCache 호스트 정보 주입 (Terraform outputs)
          values = yamlencode({
            postgres = {
              external = {
                host = data.terraform_remote_state.foundation.outputs.rds_address
              }
            }
            redis = {
              external = {
                host = data.terraform_remote_state.foundation.outputs.redis_endpoint
              }
            }
            # Monitoring exporters도 동일한 호스트 사용
            postgresExporter = {
              config = {
                host = data.terraform_remote_state.foundation.outputs.rds_address
              }
            }
            redisExporter = {
              config = {
                host = data.terraform_remote_state.foundation.outputs.redis_endpoint
              }
            }
          })
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "wealist-prod"
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
