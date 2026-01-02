# =============================================================================
# EKS Cluster Configuration
# =============================================================================
# Istio Ambient 모드를 위한 Security Group 포트 설정 포함
# Pod Identity 사용 (IRSA 대신)

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = local.cluster_name
  cluster_version = var.cluster_version

  # -----------------------------------------------------------------------------
  # Network Configuration
  # -----------------------------------------------------------------------------
  vpc_id                   = local.vpc_id
  subnet_ids               = local.private_subnet_ids
  control_plane_subnet_ids = local.private_subnet_ids

  # -----------------------------------------------------------------------------
  # Cluster Endpoint Configuration
  # -----------------------------------------------------------------------------
  cluster_endpoint_public_access       = true
  cluster_endpoint_public_access_cidrs = var.allowed_cidr_blocks
  cluster_endpoint_private_access      = true

  # -----------------------------------------------------------------------------
  # Cluster Admin Permissions
  # -----------------------------------------------------------------------------
  enable_cluster_creator_admin_permissions = true

  # -----------------------------------------------------------------------------
  # KMS Encryption for Secrets
  # -----------------------------------------------------------------------------
  create_kms_key = false
  cluster_encryption_config = {
    provider_key_arn = local.kms_key_arn
    resources        = ["secrets"]
  }

  # -----------------------------------------------------------------------------
  # Control Plane Logging (5가지 전부)
  # -----------------------------------------------------------------------------
  cluster_enabled_log_types = [
    "api",
    "audit",
    "authenticator",
    "controllerManager",
    "scheduler"
  ]

  # -----------------------------------------------------------------------------
  # Node Security Group - Istio Ambient Ports
  # -----------------------------------------------------------------------------
  node_security_group_additional_rules = {
    # Istio HBONE tunnel (ztunnel + Waypoint)
    istio_hbone_ingress = {
      description                   = "Istio HBONE tunnel"
      protocol                      = "tcp"
      from_port                     = 15008
      to_port                       = 15008
      type                          = "ingress"
      source_cluster_security_group = true
    }
    istio_hbone_egress = {
      description                   = "Istio HBONE tunnel"
      protocol                      = "tcp"
      from_port                     = 15008
      to_port                       = 15008
      type                          = "egress"
      source_cluster_security_group = true
    }

    # Istio traffic redirect ports
    istio_redirect_ingress = {
      description                   = "Istio traffic redirect"
      protocol                      = "tcp"
      from_port                     = 15001
      to_port                       = 15006
      type                          = "ingress"
      source_cluster_security_group = true
    }
    istio_redirect_egress = {
      description                   = "Istio traffic redirect"
      protocol                      = "tcp"
      from_port                     = 15001
      to_port                       = 15006
      type                          = "egress"
      source_cluster_security_group = true
    }

    # Istio XDS (istiod communication)
    istio_xds_ingress = {
      description                   = "Istio XDS to istiod"
      protocol                      = "tcp"
      from_port                     = 15012
      to_port                       = 15012
      type                          = "ingress"
      source_cluster_security_group = true
    }
    istio_xds_egress = {
      description                   = "Istio XDS to istiod"
      protocol                      = "tcp"
      from_port                     = 15012
      to_port                       = 15012
      type                          = "egress"
      source_cluster_security_group = true
    }

    # Istio metrics and readiness
    istio_metrics = {
      description                   = "Istio metrics and readiness"
      protocol                      = "tcp"
      from_port                     = 15020
      to_port                       = 15021
      type                          = "ingress"
      source_cluster_security_group = true
    }

    # DNS (CoreDNS)
    dns_tcp = {
      description                   = "DNS TCP"
      protocol                      = "tcp"
      from_port                     = 53
      to_port                       = 53
      type                          = "ingress"
      source_cluster_security_group = true
    }
    dns_udp = {
      description                   = "DNS UDP"
      protocol                      = "udp"
      from_port                     = 53
      to_port                       = 53
      type                          = "ingress"
      source_cluster_security_group = true
    }

    # HTTP services (nginx, frontend, ops-portal 등)
    # 기본 node-to-node ephemeral ports (1025-65535)에 포함되지 않아 별도 규칙 필요
    http_node_to_node = {
      description = "HTTP node to node (nginx, frontend)"
      protocol    = "tcp"
      from_port   = 80
      to_port     = 80
      type        = "ingress"
      self        = true
    }
  }

  # -----------------------------------------------------------------------------
  # Managed Node Groups
  # -----------------------------------------------------------------------------
  eks_managed_node_groups = {
    # 전체 Spot으로 비용 최소화 (~$50/월)
    spot = {
      name            = "${local.name_prefix}-spot"
      use_name_prefix = true

      # Instance types - 다양한 타입으로 Spot 가용성 확보
      instance_types = var.spot_instance_types
      capacity_type  = "SPOT"

      # Scaling configuration
      min_size     = var.spot_min_size
      max_size     = var.spot_max_size
      desired_size = var.spot_desired_size

      # Disk configuration
      disk_size = var.node_disk_size

      # Labels
      labels = {
        "node.kubernetes.io/capacity-type" = "spot"
        "workload-type"                    = "general"
      }

      # Cluster Autoscaler 태그
      tags = {
        "k8s.io/cluster-autoscaler/enabled"               = "true"
        "k8s.io/cluster-autoscaler/${local.cluster_name}" = "owned"
      }
    }
  }

  # -----------------------------------------------------------------------------
  # EKS Add-ons
  # -----------------------------------------------------------------------------
  cluster_addons = {
    # VPC CNI - CRITICAL for Istio Ambient
    vpc-cni = {
      addon_version            = var.addon_versions.vpc_cni
      resolve_conflicts_on_create = "OVERWRITE"
      resolve_conflicts_on_update = "OVERWRITE"

      # Istio Ambient 필수 설정
      configuration_values = jsonencode({
        env = {
          POD_SECURITY_GROUP_ENFORCING_MODE = "standard"
          ENABLE_PREFIX_DELEGATION          = "true"
          WARM_PREFIX_TARGET               = "1"
        }
      })
    }

    # CoreDNS
    coredns = {
      addon_version            = var.addon_versions.coredns
      resolve_conflicts_on_create = "OVERWRITE"
      resolve_conflicts_on_update = "OVERWRITE"
    }

    # kube-proxy
    kube-proxy = {
      addon_version            = var.addon_versions.kube_proxy
      resolve_conflicts_on_create = "OVERWRITE"
    }

    # EBS CSI Driver
    aws-ebs-csi-driver = {
      addon_version            = var.addon_versions.ebs_csi
      resolve_conflicts_on_create = "OVERWRITE"
      service_account_role_arn = module.ebs_csi_irsa.iam_role_arn
    }

    # Pod Identity Agent - Pod Identity 사용에 필수
    eks-pod-identity-agent = {
      addon_version            = var.addon_versions.pod_identity_agent
      resolve_conflicts_on_create = "OVERWRITE"
    }
  }

  tags = local.common_tags
}

# -----------------------------------------------------------------------------
# EBS CSI Driver IRSA (Add-on은 아직 Pod Identity 미지원)
# -----------------------------------------------------------------------------
module "ebs_csi_irsa" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.30"

  role_name_prefix = "${local.name_prefix}-ebs-csi-"

  attach_ebs_csi_policy = true

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:ebs-csi-controller-sa"]
    }
  }

  tags = local.common_tags
}
