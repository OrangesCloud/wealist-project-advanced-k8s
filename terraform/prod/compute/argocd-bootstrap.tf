# =============================================================================
# ArgoCD Bootstrap (App of Apps)
# =============================================================================
# ArgoCD 설치 후 자동으로 root-app을 배포하여 GitOps 시작
#
# 배포 순서:
# 1. ArgoCD Helm release (helm-releases.tf)
# 2. AppProject 생성 (wealist-prod)
# 3. Root Application 생성 (k8s/argocd/apps/prod/ 디렉토리 감시)
# 4. ArgoCD가 자동으로 모든 Application sync
# =============================================================================

# -----------------------------------------------------------------------------
# 1. ArgoCD AppProject
# -----------------------------------------------------------------------------
resource "null_resource" "argocd_project" {
  triggers = {
    cluster_name = module.eks.cluster_name
    # 변경 시 재적용을 위한 해시
    project_hash = filemd5("${path.module}/../../../k8s/argocd/projects/wealist-prod.yaml")
  }

  provisioner "local-exec" {
    command = <<-EOT
      aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region}

      # ArgoCD가 준비될 때까지 대기 (최대 5분)
      echo "Waiting for ArgoCD to be ready..."
      kubectl wait --for=condition=available deployment/argocd-server -n argocd --timeout=300s

      # AppProject 적용
      kubectl apply -f ${path.module}/../../../k8s/argocd/projects/wealist-prod.yaml
    EOT
  }

  depends_on = [helm_release.argocd]
}

# -----------------------------------------------------------------------------
# 2. ArgoCD Root Application (App of Apps)
# -----------------------------------------------------------------------------
resource "null_resource" "argocd_root_app" {
  triggers = {
    cluster_name = module.eks.cluster_name
    # 변경 시 재적용을 위한 해시
    root_app_hash = filemd5("${path.module}/../../../k8s/argocd/apps/prod/root-app.yaml")
  }

  provisioner "local-exec" {
    command = <<-EOT
      aws eks update-kubeconfig --name ${module.eks.cluster_name} --region ${var.aws_region}

      # Root Application 적용
      kubectl apply -f ${path.module}/../../../k8s/argocd/apps/prod/root-app.yaml

      echo "ArgoCD Root Application deployed!"
      echo "ArgoCD will now sync all applications from k8s/argocd/apps/prod/"
    EOT
  }

  depends_on = [null_resource.argocd_project, kubernetes_config_map.argocd_cluster_config]
}
