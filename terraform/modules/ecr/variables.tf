# =============================================================================
# ECR Module - Variables
# =============================================================================

variable "repository_names" {
  description = "List of ECR repository names to create"
  type        = list(string)
}

variable "image_tag_mutability" {
  description = "Image tag mutability setting (MUTABLE or IMMUTABLE)"
  type        = string
  default     = "MUTABLE"
}

variable "scan_on_push" {
  description = "Enable image scanning on push"
  type        = bool
  default     = true
}

variable "enable_lifecycle_policy" {
  description = "Enable lifecycle policy for image cleanup"
  type        = bool
  default     = true
}

variable "max_image_count" {
  description = "Maximum number of images to keep per repository"
  type        = number
  default     = 30
}

variable "cross_account_arns" {
  description = "List of AWS account ARNs for cross-account access"
  type        = list(string)
  default     = null
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}
