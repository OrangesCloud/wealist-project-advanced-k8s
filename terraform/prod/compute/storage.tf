# =============================================================================
# Storage Configuration for EKS
# =============================================================================
# EBS CSI Driver는 EKS 모듈에서 자동 설치됨
# 여기서는 StorageClass 기본값만 설정

# -----------------------------------------------------------------------------
# gp2 StorageClass를 기본값으로 설정
# -----------------------------------------------------------------------------
# EKS는 gp2 StorageClass를 자동 생성하지만 기본값으로 설정하지 않음
# PVC에서 storageClassName을 지정하지 않으면 기본 StorageClass 사용
resource "kubernetes_annotations" "gp2_default" {
  api_version = "storage.k8s.io/v1"
  kind        = "StorageClass"
  metadata {
    name = "gp2"
  }
  annotations = {
    "storageclass.kubernetes.io/is-default-class" = "true"
  }
  force = true

  depends_on = [module.eks]
}

# -----------------------------------------------------------------------------
# gp3 StorageClass (선택적)
# -----------------------------------------------------------------------------
# gp3는 gp2보다 비용 효율적이고 성능이 좋음
# 필요시 주석 해제하여 사용
#
# resource "kubernetes_storage_class" "gp3" {
#   metadata {
#     name = "gp3"
#   }
#   storage_provisioner = "ebs.csi.aws.com"
#   reclaim_policy      = "Delete"
#   volume_binding_mode = "WaitForFirstConsumer"
#   allow_volume_expansion = true
#
#   parameters = {
#     type      = "gp3"
#     encrypted = "true"
#   }
#
#   depends_on = [module.eks]
# }
