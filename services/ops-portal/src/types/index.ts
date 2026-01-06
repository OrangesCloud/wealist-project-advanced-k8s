// Portal User types
export type Role = 'admin' | 'pm' | 'viewer'

export interface PortalUser {
  id: string
  email: string
  name: string
  picture?: string
  role: Role
  isActive: boolean
  lastLoginAt?: string
  createdAt: string
}

// Audit Log types
export type ActionType = 'create' | 'update' | 'delete' | 'login' | 'logout'
export type ResourceType = 'portal_user' | 'argocd_rbac' | 'feature_flag' | 'app_config'

export interface AuditLog {
  id: string
  userId: string
  userEmail: string
  action: ActionType
  resourceType: ResourceType
  resourceId: string
  details?: string
  createdAt: string
}

// App Config types
export interface AppConfig {
  id: string
  key: string
  value: string
  description?: string
  isActive: boolean
  updatedAt: string
}

// ArgoCD Application types
export interface ArgoCDApplication {
  name: string
  namespace: string
  project: string
  sync: string
  health: string
}

// Metrics types
export interface ServiceMetrics {
  serviceName: string
  requestRate: number      // requests per second
  errorRate: number        // percentage
  avgLatency: number       // milliseconds
  p95Latency: number       // milliseconds
  p99Latency: number       // milliseconds
  successRate: number      // percentage
  activeRequests: number   // concurrent requests
}

export interface ClusterMetrics {
  nodeCount: number
  podCount: number
  cpuUsage: number         // percentage
  memoryUsage: number      // percentage
  totalCpuCores: number
  totalMemoryGb: number
  healthyPods: number
  unhealthyPods: number
}

export interface SystemOverview {
  totalRequests: number    // requests in last hour
  avgResponseTime: number  // milliseconds
  errorPercentage: number  // percentage
  activeServices: number
  totalEndpoints: number
}

// API Response types
export interface ApiResponse<T> {
  success: boolean
  message?: string
  data?: T
}

export interface PaginatedResponse<T> {
  success: boolean
  data: T[]
  page: number
  limit: number
  total: number
}
