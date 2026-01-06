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

// Error Tracker types
export interface RecentError {
  serviceName: string
  requestPath: string
  responseCode: string
  errorCount: number
  timestamp: number
}

export interface ErrorTrendPoint {
  timestamp: number
  errorRate: number
  errorCount: number
}

export interface ServiceErrorSummary {
  serviceName: string
  totalErrors: number
  errorRate: number
  error5xxCount: number
  error4xxCount: number
}

export interface ErrorOverview {
  totalErrors: number
  errorRate: number
  mostErrorService: string
  mostErrorCount: number
}

// SLO Dashboard types
export interface ServiceSLO {
  serviceName: string
  availability: number
  availabilityTarget: number
  availabilityMet: boolean
  latencyP50: number
  latencyP99: number
  latencyTarget: number
  latencyMet: boolean
  errorBudgetRemaining: number
  errorBudgetConsumed: number
}

export interface SLOOverview {
  services: ServiceSLO[]
  overallHealth: 'healthy' | 'degraded' | 'critical'
  servicesAtRisk: number
  totalServices: number
}

export interface BurnRate {
  serviceName: string
  rate1h: number
  rate6h: number
  rate24h: number
  alerting: boolean
}

// Deployment History types
export interface DeploymentHistoryEntry {
  appName: string
  revision: string
  deployedAt: string
  syncStatus: string
  healthStatus: string
  commitMessage?: string
}

// Logs Viewer types
export interface LogEntry {
  timestamp: string
  service: string
  level: string
  message: string
  traceId?: string
  spanId?: string
  labels?: Record<string, string>
}

export interface LogQueryResult {
  entries: LogEntry[]
  totalCount: number
}

export interface ServiceInfo {
  name: string
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
