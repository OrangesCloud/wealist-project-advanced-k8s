# =============================================================================
# Dev Environment Configuration - Outputs
# =============================================================================

output "ecr_registry_url" {
  description = "ECR Registry URL"
  value       = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com"
}

output "ecr_repository_urls" {
  description = "Map of service names to ECR repository URLs"
  value       = module.ecr.repository_urls
}

output "dev_user_name" {
  description = "Name of the IAM user for local development"
  value       = aws_iam_user.dev_ecr_user.name
}

output "dev_user_arn" {
  description = "ARN of the IAM user"
  value       = aws_iam_user.dev_ecr_user.arn
}

output "dev_user_access_key_id" {
  description = "Access Key ID for the IAM user"
  value       = aws_iam_access_key.dev_ecr_user.id
}

output "dev_user_secret_access_key" {
  description = "Secret Access Key for the IAM user (use 'terraform output -raw dev_user_secret_access_key' to see)"
  value       = aws_iam_access_key.dev_ecr_user.secret
  sensitive   = true
}

output "aws_configure_commands" {
  description = "Commands to configure AWS CLI for local development"
  value       = <<-EOT

    # AWS CLI 프로필 설정
    aws configure --profile wealist-dev
    # Access Key ID: ${aws_iam_access_key.dev_ecr_user.id}
    # Secret Access Key: (terraform output -raw dev_user_secret_access_key 로 확인)
    # Default region: ${var.aws_region}
    # Default output format: json

    # ECR 로그인
    aws ecr get-login-password --region ${var.aws_region} --profile wealist-dev | \
      docker login --username AWS --password-stdin ${data.aws_caller_identity.current.account_id}.dkr.ecr.${var.aws_region}.amazonaws.com

  EOT
}

# =============================================================================
# SSM Parameter Store Outputs
# =============================================================================
output "parameter_arns" {
  description = "Map of parameter names to their ARNs"
  value       = module.parameters.parameter_arns
}

output "parameter_names" {
  description = "List of created parameter names"
  value       = module.parameters.parameter_names
}

output "ssm_usage_info" {
  description = "Information on how to use the SSM parameters"
  value       = <<-EOT

    # =============================================================================
    # AWS SSM Parameter Store - 생성된 파라미터
    # =============================================================================

    파라미터 목록:
    ${join("\n    ", module.parameters.parameter_names)}

    # AWS CLI로 파라미터 값 확인:
    aws ssm get-parameter --name "/wealist/dev/google-oauth/client-id" --with-decryption --query Parameter.Value --output text

    # 특정 경로의 모든 파라미터 조회:
    aws ssm get-parameters-by-path --path "/wealist/dev" --recursive --with-decryption

    # External Secrets Operator에서 사용:
    # k8s/helm/charts/external-secrets/ 참조

  EOT
}
