# =============================================================================
# Production Compute - Variables
# =============================================================================

variable "aws_region" {
  description = "AWS Region"
  type        = string
  default     = "ap-northeast-2"
}

# =============================================================================
# EKS Cluster Configuration
# =============================================================================
variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster (Istio Ambient 1.28 호환)"
  type        = string
  default     = "1.34"
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access the EKS API endpoint"
  type        = list(string)
  default     = ["0.0.0.0/0"]  # 프로덕션에서는 제한 필요
}

# =============================================================================
# Node Group Configuration
# =============================================================================
variable "spot_instance_types" {
  description = "Instance types for Spot node group (다양한 타입으로 Spot 가용성 확보)"
  type        = list(string)
  default = [
    "t3.large",    # 2 vCPU, 8GB RAM (기본)
    "t3a.large",   # AMD 버전 (더 저렴)
    "t3.xlarge",   # Fallback: 4 vCPU, 16GB RAM
    "t3a.xlarge"   # Fallback AMD 버전
  ]
}

variable "spot_min_size" {
  description = "Minimum number of Spot nodes"
  type        = number
  default     = 2  # 3→2 for cost optimization (2026-01-03)
}

variable "spot_max_size" {
  description = "Maximum number of Spot nodes"
  type        = number
  default     = 6
}

variable "spot_desired_size" {
  description = "Desired number of Spot nodes (t3.large 8GB × 2 = 16GB, 70 pods capacity)"
  type        = number
  default     = 2  # 3→2 for cost optimization (2026-01-03)
}

variable "node_disk_size" {
  description = "Disk size in GB for worker nodes"
  type        = number
  default     = 50
}

# =============================================================================
# EKS Add-on Versions
# =============================================================================
variable "addon_versions" {
  description = "Versions for EKS managed add-ons (compatible with EKS 1.34 + Istio Ambient 1.28)"
  type = object({
    vpc_cni            = string
    coredns            = string
    kube_proxy         = string
    ebs_csi            = string
    pod_identity_agent = string
  })
  default = {
    vpc_cni            = "v1.21.1-eksbuild.1"    # Istio Ambient 호환
    coredns            = "v1.12.4-eksbuild.1"    # EKS 1.34용
    kube_proxy         = "v1.34.1-eksbuild.2"    # EKS 1.34용
    ebs_csi            = "v1.54.0-eksbuild.1"
    pod_identity_agent = "v1.3.10-eksbuild.2"
  }
}

# =============================================================================
# Scheduled Scaling Configuration
# =============================================================================
variable "scheduled_scaling_enabled" {
  description = "Enable scheduled scaling for cost optimization (노드 자동 on/off)"
  type        = bool
  default     = true
}

# -----------------------------------------------------------------------------
# 평일 스케줄 (월-금)
# -----------------------------------------------------------------------------
variable "weekday_scale_down_schedule" {
  description = "Cron for weekday scale down (UTC). 16:00 UTC Sun-Thu = 01:00 KST Mon-Fri"
  type        = string
  default     = "0 16 * * 0-4"  # 평일 새벽 1시 KST
}

variable "weekday_scale_up_schedule" {
  description = "Cron for weekday scale up (UTC). 23:00 UTC Sun-Thu = 08:00 KST Mon-Fri"
  type        = string
  default     = "0 23 * * 0-4"  # 평일 오전 8시 KST
}

# -----------------------------------------------------------------------------
# 주말 스케줄 (토-일)
# -----------------------------------------------------------------------------
variable "weekend_enabled" {
  description = "Enable weekend scheduled scaling (주말 스케줄 적용 여부)"
  type        = bool
  default     = true
}

variable "weekend_scale_down_schedule" {
  description = "Cron for weekend scale down (UTC). 18:00 UTC Fri-Sat = 03:00 KST Sat-Sun"
  type        = string
  default     = "0 18 * * 5,6"  # 주말 새벽 3시 KST
}

variable "weekend_scale_up_schedule" {
  description = "Cron for weekend scale up (UTC). 00:00 UTC Sat-Sun = 09:00 KST Sat-Sun"
  type        = string
  default     = "0 0 * * 6,0"  # 주말 오전 9시 KST
}