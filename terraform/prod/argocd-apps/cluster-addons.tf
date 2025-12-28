# =============================================================================
# Cluster Addons - ArgoCD Applications
# =============================================================================
# 설치 순서 (sync-wave):
#   -2: AWS Load Balancer Controller (webhook 먼저)
#   -1: External Secrets, cert-manager, Cluster Autoscaler
#    0: Istio Ingress Gateway (ALB Controller 필요)
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
        targetRevision = "0.9.11"
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
# Istio Ingress Gateway (sync-wave: 0)
# -----------------------------------------------------------------------------
resource "kubernetes_manifest" "argocd_app_istio_ingress" {
  manifest = {
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name      = "istio-ingressgateway"
      namespace = "argocd"
      annotations = {
        "argocd.argoproj.io/sync-wave" = "0"
      }
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
    }
    spec = {
      project = "wealist-prod"
      source = {
        repoURL        = "https://istio-release.storage.googleapis.com/charts"
        chart          = "gateway"
        targetRevision = "1.28.2"
        helm = {
          valuesObject = {
            service = {
              type = "LoadBalancer"
              annotations = {
                "service.beta.kubernetes.io/aws-load-balancer-type"            = "external"
                "service.beta.kubernetes.io/aws-load-balancer-nlb-target-type" = "ip"
                "service.beta.kubernetes.io/aws-load-balancer-scheme"          = "internet-facing"
              }
              ports = [
                {
                  name       = "http"
                  port       = 80
                  targetPort = 80
                },
                {
                  name       = "https"
                  port       = 443
                  targetPort = 443
                }
              ]
            }
          }
        }
      }
      destination = {
        server    = "https://kubernetes.default.svc"
        namespace = "istio-system"
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
