# =============================================================================
# Production Compute - Outputs
# =============================================================================
# ArgoCD, Helm 배포, 모니터링 설정에 필요한 정보

# =============================================================================
# EKS Cluster Outputs
# =============================================================================
output "cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "cluster_endpoint" {
  description = "EKS cluster API endpoint"
  value       = module.eks.cluster_endpoint
}

output "cluster_certificate_authority_data" {
  description = "Base64 encoded certificate data for cluster authentication"
  value       = module.eks.cluster_certificate_authority_data
  sensitive   = true
}

output "cluster_oidc_issuer_url" {
  description = "OIDC issuer URL for the cluster"
  value       = module.eks.cluster_oidc_issuer_url
}

output "oidc_provider_arn" {
  description = "ARN of the OIDC provider (for IRSA if needed)"
  value       = module.eks.oidc_provider_arn
}

output "cluster_security_group_id" {
  description = "Security group ID for the cluster control plane"
  value       = module.eks.cluster_security_group_id
}

output "node_security_group_id" {
  description = "Security group ID for the worker nodes"
  value       = module.eks.node_security_group_id
}

# =============================================================================
# Node Group Outputs
# =============================================================================
output "node_groups" {
  description = "Map of EKS managed node groups"
  value = {
    for k, v in module.eks.eks_managed_node_groups : k => {
      node_group_id        = v.node_group_id
      node_group_arn       = v.node_group_arn
      node_group_status    = v.node_group_status
      node_group_resources = v.node_group_resources
    }
  }
}

# =============================================================================
# Add-on Outputs
# =============================================================================
output "cluster_addons" {
  description = "Map of EKS cluster add-ons"
  value = {
    for k, v in module.eks.cluster_addons : k => {
      addon_version = v.addon_version
      status        = v.status
    }
  }
}

# =============================================================================
# Pod Identity Outputs
# =============================================================================
output "pod_identity_roles" {
  description = "Map of Pod Identity IAM role ARNs"
  value = {
    alb_controller     = module.pod_identity_alb_controller.role_arn
    external_secrets   = module.pod_identity_external_secrets.role_arn
    external_dns       = module.pod_identity_external_dns.role_arn
    cert_manager       = module.pod_identity_cert_manager.role_arn
    storage_service    = module.pod_identity_storage_service.role_arn
    board_service      = module.pod_identity_board_service.role_arn
    user_service       = module.pod_identity_user_service.role_arn
    cluster_autoscaler = module.pod_identity_cluster_autoscaler.role_arn
  }
}

# =============================================================================
# kubeconfig 생성 명령어
# =============================================================================
output "configure_kubectl" {
  description = "Command to configure kubectl"
  value       = "aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region}"
}

# =============================================================================
# Summary
# =============================================================================
output "summary" {
  description = "Summary of EKS cluster for deployment"
  value       = <<-EOT

    # =============================================================================
    # Production EKS Cluster Summary
    # =============================================================================

    Cluster:
      Name: ${module.eks.cluster_name}
      Version: ${var.cluster_version}
      Endpoint: ${module.eks.cluster_endpoint}

    Node Groups:
      - spot: ${var.spot_desired_size} nodes (min: ${var.spot_min_size}, max: ${var.spot_max_size})
        Instance Types: ${join(", ", var.spot_instance_types)}

    Add-ons:
      - vpc-cni (Istio Ambient 지원)
      - coredns
      - kube-proxy
      - aws-ebs-csi-driver
      - eks-pod-identity-agent

    Pod Identity Associations:
      - AWS Load Balancer Controller (kube-system)
      - External Secrets Operator (external-secrets)
      - External DNS (external-dns)
      - cert-manager (cert-manager)
      - storage-service (wealist-prod) - S3 파일 스토리지
      - board-service (wealist-prod) - S3 첨부파일
      - user-service (wealist-prod) - S3 프로필 이미지
      - Cluster Autoscaler (kube-system)

    # kubectl 설정:
    aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region}

    # 다음 단계:
    # 1. Gateway API CRDs 설치
    # 2. Istio Ambient 설치: istioctl install --set profile=ambient
    # 3. ArgoCD 설치 및 앱 배포

  EOT
}
