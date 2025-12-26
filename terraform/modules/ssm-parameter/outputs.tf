# =============================================================================
# SSM Parameter Store Module - Outputs
# =============================================================================

output "parameter_arns" {
  description = "Map of parameter names to their ARNs"
  value = {
    for name, param in aws_ssm_parameter.this : name => param.arn
  }
}

output "parameter_names" {
  description = "List of created parameter names"
  value       = keys(aws_ssm_parameter.this)
}

output "parameter_versions" {
  description = "Map of parameter names to their versions"
  value = {
    for name, param in aws_ssm_parameter.this : name => param.version
  }
}
