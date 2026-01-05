# =============================================================================
# AWS SSM Parameter Store Module
# =============================================================================
# 시크릿을 AWS Systems Manager Parameter Store에 저장하는 모듈
# SecureString 타입으로 암호화 저장 (AWS 기본 KMS 키 사용)
#
# 사용법:
#   module "parameters" {
#     source = "../modules/ssm-parameter"
#     parameters = {
#       "/wealist/dev/google-oauth/client-id" = {
#         description = "Google OAuth Client ID"
#         value       = var.google_client_id
#         type        = "SecureString"
#       }
#     }
#   }

# -----------------------------------------------------------------------------
# SSM Parameters
# -----------------------------------------------------------------------------
resource "aws_ssm_parameter" "this" {
  for_each = var.parameters

  name        = each.key
  description = lookup(each.value, "description", "Managed by Terraform")
  type        = lookup(each.value, "type", "SecureString")
  value       = each.value.value
  tier        = lookup(each.value, "tier", "Standard")  # Standard = 무료

  # AWS 기본 KMS 키 사용 (SecureString의 경우)
  # key_id를 지정하지 않으면 기본 키 사용

  tags = merge(var.tags, lookup(each.value, "tags", {}))
}
