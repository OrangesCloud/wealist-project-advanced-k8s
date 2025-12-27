# =============================================================================
# Pod Identity Module - Variables
# =============================================================================

variable "name" {
  description = "Name prefix for the IAM role (e.g., 'alb-controller', 'external-secrets')"
  type        = string

  validation {
    condition     = length(var.name) > 0 && length(var.name) <= 64
    error_message = "Name must be between 1 and 64 characters."
  }
}

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
}

variable "namespace" {
  description = "Kubernetes namespace where the ServiceAccount exists"
  type        = string
}

variable "service_account" {
  description = "Name of the Kubernetes ServiceAccount to associate with the IAM role"
  type        = string
}

variable "policy_arns" {
  description = "List of IAM policy ARNs to attach to the role"
  type        = list(string)
  default     = []
}

variable "inline_policies" {
  description = "Map of inline IAM policies to attach to the role (name => policy JSON)"
  type        = map(string)
  default     = {}
}

variable "tags" {
  description = "Additional tags for resources"
  type        = map(string)
  default     = {}
}
