# =============================================================================
# SSM Parameter Store Module - Variables
# =============================================================================

variable "parameters" {
  description = "Map of parameter names to their configurations"
  type = map(object({
    value       = string
    description = optional(string, "Managed by Terraform")
    type        = optional(string, "SecureString")
    tier        = optional(string, "Standard")
    tags        = optional(map(string), {})
  }))
}

variable "tags" {
  description = "Tags to apply to all parameters"
  type        = map(string)
  default     = {}
}
