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
      addon_version = try(v.addon_version, "unknown")
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
    chat_service       = module.pod_identity_chat_service.role_arn
    video_service      = module.pod_identity_video_service.role_arn
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

    Scheduled Scaling: ${var.scheduled_scaling_enabled ? "ENABLED" : "DISABLED"}
      평일 (월-금):
        - Scale Down: ${var.weekday_scale_down_schedule} (UTC) → 새벽 01:00 KST
        - Scale Up:   ${var.weekday_scale_up_schedule} (UTC) → 오전 08:00 KST
      주말 (토-일): ${var.weekend_enabled ? "ENABLED" : "DISABLED"}
        - Scale Down: ${var.weekend_scale_down_schedule} (UTC) → 새벽 03:00 KST
        - Scale Up:   ${var.weekend_scale_up_schedule} (UTC) → 오전 09:00 KST

    Add-ons:
      - vpc-cni (Istio Sidecar 지원)
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
      - chat-service (wealist-prod) - S3 채팅 첨부파일
      - video-service (wealist-prod) - S3 녹화 파일
      - Cluster Autoscaler (kube-system)

    # kubectl 설정:
    aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region}

    # 다음 단계:
    # 1. Gateway API CRDs 설치
    # 2. Istio Sidecar 설치 (Terraform helm-releases.tf에서 자동 설치)
    # 3. ArgoCD 설치 및 앱 배포

  EOT
}

# =============================================================================
# EKS Access - IAM Groups and Roles
# =============================================================================
output "eks_iam_groups" {
  description = "IAM Groups for EKS access - add team members via AWS Console"
  value = {
    admin     = aws_iam_group.eks_admin.name
    developer = aws_iam_group.eks_developer.name
    readonly  = aws_iam_group.eks_readonly.name
  }
}

output "eks_admin_role_arn" {
  description = "EKS Admin role ARN for cluster administrators"
  value       = aws_iam_role.eks_admin.arn
}

output "eks_developer_role_arn" {
  description = "EKS Developer role ARN for backend developers"
  value       = aws_iam_role.eks_developer.arn
}

output "eks_readonly_role_arn" {
  description = "EKS ReadOnly role ARN for view-only access"
  value       = aws_iam_role.eks_readonly.arn
}

output "eks_access_commands" {
  description = "kubectl configuration commands for each access level"
  value = {
    admin     = "aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region} --role-arn ${aws_iam_role.eks_admin.arn}"
    developer = "aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region} --role-arn ${aws_iam_role.eks_developer.arn}"
    readonly  = "aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region} --role-arn ${aws_iam_role.eks_readonly.arn}"
  }
}

output "eks_access_setup_guide" {
  description = "Guide for setting up team member access"
  value       = <<-EOT

    # =============================================================================
    # EKS 팀원 접근 설정 가이드
    # =============================================================================

    ## 1. AWS Console에서 팀원을 IAM Group에 추가

    IAM → User groups → 해당 그룹 선택 → Add users

    Groups:
      - ${aws_iam_group.eks_admin.name}     : 전체 클러스터 관리
      - ${aws_iam_group.eks_developer.name} : wealist-prod, argocd 네임스페이스
      - ${aws_iam_group.eks_readonly.name}  : 조회만 가능

    ## 2. 팀원 PC 설정 (Access Key 필요)

    # AWS CLI 설정
    aws configure

    # kubeconfig 설정 (역할별 명령어)
    # Admin:
    ${aws_iam_role.eks_admin.arn}

    # Developer:
    ${aws_iam_role.eks_developer.arn}

    # ReadOnly:
    ${aws_iam_role.eks_readonly.arn}

  EOT
}

# =============================================================================
# Scheduled Scaling Outputs
# =============================================================================
output "scheduled_scaling" {
  description = "Scheduled scaling configuration for cost optimization"
  value = {
    enabled = var.scheduled_scaling_enabled
    weekday = {
      scale_down = var.weekday_scale_down_schedule  # 01:00 KST
      scale_up   = var.weekday_scale_up_schedule    # 08:00 KST
    }
    weekend = {
      enabled    = var.weekend_enabled
      scale_down = var.weekend_scale_down_schedule  # 03:00 KST
      scale_up   = var.weekend_scale_up_schedule    # 09:00 KST
    }
    info = var.scheduled_scaling_enabled ? "평일: 01:00-08:00 OFF, 주말: 03:00-09:00 OFF" : "24시간 운영 중"
  }
}
