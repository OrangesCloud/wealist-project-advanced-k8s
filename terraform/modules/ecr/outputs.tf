# =============================================================================
# ECR Module - Outputs
# =============================================================================

output "repository_urls" {
  description = "Map of repository names to their URLs"
  value = {
    for name, repo in aws_ecr_repository.services : name => repo.repository_url
  }
}

output "repository_arns" {
  description = "Map of repository names to their ARNs"
  value = {
    for name, repo in aws_ecr_repository.services : name => repo.arn
  }
}

output "registry_id" {
  description = "The registry ID (AWS Account ID)"
  value       = length(aws_ecr_repository.services) > 0 ? values(aws_ecr_repository.services)[0].registry_id : null
}
