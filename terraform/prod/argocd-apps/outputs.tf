# =============================================================================
# Outputs
# =============================================================================

output "namespace" {
  description = "Wealist production namespace"
  value       = kubernetes_namespace.wealist_prod.metadata[0].name
}

output "argocd_project" {
  description = "ArgoCD project name"
  value       = kubernetes_manifest.argocd_project_prod.manifest.metadata.name
}

output "argocd_root_app" {
  description = "ArgoCD root application name"
  value       = kubernetes_manifest.argocd_root_app.manifest.metadata.name
}

output "summary" {
  description = "ArgoCD Apps deployment summary"
  value = join("\n", [
    "",
    "# =============================================================================",
    "# ArgoCD Apps Layer Summary",
    "# =============================================================================",
    "",
    "Namespace: wealist-prod",
    "ArgoCD Project: wealist-prod",
    "ArgoCD Root App: wealist-apps-prod",
    "",
    "Git Repository: ${var.git_repo_url}",
    "Target Revision: ${var.git_target_revision}",
    "Apps Path: ${var.argocd_apps_path}",
    "",
    "# ArgoCD UI 접속:",
    "kubectl port-forward svc/argocd-server -n argocd 8080:443",
    "",
    "# 초기 비밀번호:",
    "kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d",
    ""
  ])
}
