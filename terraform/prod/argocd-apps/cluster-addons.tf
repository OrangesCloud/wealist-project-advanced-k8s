# =============================================================================
# Cluster Addons - ArgoCD Applications
# =============================================================================
# 설치 순서 (sync-wave):
#   -2: AWS Load Balancer Controller (webhook 먼저)
#   -1: External Secrets, cert-manager, External DNS, Cluster Autoscaler
#    0: Istio Ingress Gateway (Gateway API로 대체됨)
#    1: External Secrets Config (ClusterSecretStore, ExternalSecret)
#    2+: 서비스들
# =============================================================================

# -----------------------------------------------------------------------------
# AWS Load Balancer Controller (sync-wave: -2)
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "argocd_app_alb_controller" {
  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "aws-load-balancer-controller"
      namespace = "argocd"
      annotations = {
        "argocd.argoproj.io/sync-wave" = "-2"
      }
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
    }
    spec = {
      project = "wealist-prod"
      source = {
        repoURL        = "https://aws.github.io/eks-charts"
        chart          = "aws-load-balancer-controller"
        targetRevision = "1.7.1"
        helm = {
          valuesObject = {
            clusterName = local.cluster_name
            region      = var.aws_region
            vpcId       = local.vpc_id
            serviceAccount = {
              create = true
              name   = "aws-load-balancer-controller"
            }
            enableShield = false
            enableWaf    = false
            enableWafv2  = false
          }
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "kube-system"
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
        syncOptions = ["CreateNamespace=true", "ServerSideApply=true"]
        retry = {
          limit = 5
          backoff = {
            duration    = "5s"
            factor      = 2
            maxDuration = "3m"
          }
        }
      }
    }
  }

  depends_on = [kubernetes_manifest.argocd_project_prod]
}

# -----------------------------------------------------------------------------
# External Secrets Operator (sync-wave: -1)
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "argocd_app_external_secrets" {
  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "external-secrets-operator"
      namespace = "argocd"
      annotations = {
        "argocd.argoproj.io/sync-wave" = "-1"
      }
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
    }
    spec = {
      project = "wealist-prod"
      source = {
        repoURL        = "https://charts.external-secrets.io"
        chart          = "external-secrets"
        targetRevision = "0.10.5"  # v1 API 지원 (v0.10.0+)
        helm = {
          valuesObject = {
            installCRDs = true
            serviceAccount = {
              create = true
              name   = "external-secrets"
            }
          }
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "external-secrets"
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
        syncOptions = ["CreateNamespace=true", "ServerSideApply=true"]
      }
    }
  }

  depends_on = [kubernetes_manifest.argocd_app_alb_controller]
}

# -----------------------------------------------------------------------------
# cert-manager (sync-wave: -1)
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "argocd_app_cert_manager" {
  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "cert-manager"
      namespace = "argocd"
      annotations = {
        "argocd.argoproj.io/sync-wave" = "-1"
      }
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
    }
    spec = {
      project = "wealist-prod"
      source = {
        repoURL        = "https://charts.jetstack.io"
        chart          = "cert-manager"
        targetRevision = "v1.14.3"
        helm = {
          valuesObject = {
            installCRDs = true
            serviceAccount = {
              create = true
              name   = "cert-manager"
            }
          }
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "cert-manager"
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
        syncOptions = ["CreateNamespace=true", "ServerSideApply=true"]
      }
    }
  }

  depends_on = [kubernetes_manifest.argocd_app_alb_controller]
}

# -----------------------------------------------------------------------------
# External DNS (sync-wave: -1)
# -----------------------------------------------------------------------------
# Route53 자동 업데이트: Gateway API 리소스의 annotation 기반으로 DNS 레코드 관리
resource "kubernetes_manifest" "argocd_app_external_dns" {
  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "external-dns"
      namespace = "argocd"
      annotations = {
        "argocd.argoproj.io/sync-wave" = "-1"
      }
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
    }
    spec = {
      project = "wealist-prod"
      source = {
        repoURL        = "https://kubernetes-sigs.github.io/external-dns"
        chart          = "external-dns"
        targetRevision = "1.14.5"
        helm = {
          valuesObject = {
            provider = "aws"
            aws = {
              region = var.aws_region
            }
            domainFilters = ["wealist.co.kr"]
            policy        = "sync"  # create, update, delete records
            registry      = "txt"
            txtOwnerId    = "wealist-prod"
            sources       = ["service", "ingress", "gateway-httproute"]
            serviceAccount = {
              create = true
              name   = "external-dns"
            }
            # Gateway API 지원
            extraArgs = [
              "--gateway-namespace=istio-system",
              "--gateway-label-filter=gateway.networking.k8s.io/gateway-name=istio-ingressgateway"
            ]
          }
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "external-dns"
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
        syncOptions = ["CreateNamespace=true", "ServerSideApply=true"]
      }
    }
  }

  depends_on = [kubernetes_manifest.argocd_app_alb_controller]
}

# -----------------------------------------------------------------------------
# Cluster Autoscaler (sync-wave: -1)
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "argocd_app_cluster_autoscaler" {
  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "cluster-autoscaler"
      namespace = "argocd"
      annotations = {
        "argocd.argoproj.io/sync-wave" = "-1"
      }
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
    }
    spec = {
      project = "wealist-prod"
      source = {
        repoURL        = "https://kubernetes.github.io/autoscaler"
        chart          = "cluster-autoscaler"
        targetRevision = "9.35.0"
        helm = {
          valuesObject = {
            autoDiscovery = {
              clusterName = local.cluster_name
            }
            awsRegion = var.aws_region
            rbac = {
              serviceAccount = {
                create = true
                name   = "cluster-autoscaler"
              }
            }
          }
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "kube-system"
      }
      syncPolicy = {
        automated = {
          prune    = true
          selfHeal = true
        }
        syncOptions = ["ServerSideApply=true"]
      }
    }
  }

  depends_on = [kubernetes_manifest.argocd_app_alb_controller]
}

# -----------------------------------------------------------------------------
# Istio Ingress Gateway (DISABLED - Using Kubernetes Gateway API instead)
# -----------------------------------------------------------------------------
# NOTE: Istio Ingress Gateway is now managed via Kubernetes Gateway API
# in the istio-config Helm chart. The Gateway API automatically provisions
# Deployment and Service with proper AWS NLB annotations.
#
# Benefits of Gateway API approach:
# - Native Kubernetes API (recommended for Ambient mode)
# - Automatic Deployment/Service provisioning by Istio
# - No sidecar injection issues (Ambient mode compatible)
# - Centralized configuration in Helm charts
#
# To re-enable Helm-based gateway, uncomment the resource below.
# -----------------------------------------------------------------------------
# resource "kubernetes_manifest" "argocd_app_istio_ingress" {
#   manifest = {
#     apiVersion = "argoproj.io/v1alpha1"
#     kind       = "Application"
#     metadata = {
#       name      = "istio-ingressgateway"
#       namespace = "argocd"
#       annotations = {
#         "argocd.argoproj.io/sync-wave" = "0"
#       }
#       finalizers = ["resources-finalizer.argocd.argoproj.io"]
#     }
#     spec = {
#       project = "wealist-prod"
#       source = {
#         repoURL        = "https://istio-release.storage.googleapis.com/charts"
#         chart          = "gateway"
#         targetRevision = "1.28.2"
#         helm = {
#           valuesObject = {
#             service = {
#               type = "LoadBalancer"
#               annotations = {
#                 "service.beta.kubernetes.io/aws-load-balancer-type"            = "external"
#                 "service.beta.kubernetes.io/aws-load-balancer-nlb-target-type" = "ip"
#                 "service.beta.kubernetes.io/aws-load-balancer-scheme"          = "internet-facing"
#               }
#               ports = [
#                 {
#                   name       = "http"
#                   port       = 80
#                   targetPort = 80
#                 },
#                 {
#                   name       = "https"
#                   port       = 443
#                   targetPort = 443
#                 }
#               ]
#             }
#           }
#         }
#       }
#       destination = {
#         server    = "https://kubernetes.default.svc"
#         namespace = "istio-system"
#       }
#       syncPolicy = {
#         automated = {
#           prune    = true
#           selfHeal = true
#         }
#         syncOptions = ["ServerSideApply=true"]
#       }
#     }
#   }
#
#   depends_on = [kubernetes_manifest.argocd_app_alb_controller]
# }
