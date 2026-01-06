import apiClient from './client'
import type { ArgoCDApplication, ServiceMetrics, ClusterMetrics, SystemOverview } from '../types'

export const getApplications = async (): Promise<ArgoCDApplication[]> => {
  const response = await apiClient.get('/monitoring/applications')
  return response.data.data || []
}

export const syncApplication = async (name: string): Promise<void> => {
  await apiClient.post('/monitoring/applications/sync', { name })
}

// Metrics API

export const getSystemOverview = async (): Promise<SystemOverview> => {
  const response = await apiClient.get('/monitoring/metrics/overview')
  return response.data.data || {
    totalRequests: 0,
    avgResponseTime: 0,
    errorPercentage: 0,
    activeServices: 0,
    totalEndpoints: 0,
  }
}

export const getServiceMetrics = async (): Promise<ServiceMetrics[]> => {
  const response = await apiClient.get('/monitoring/metrics/services')
  return response.data.data || []
}

export const getServiceDetail = async (serviceName: string): Promise<ServiceMetrics> => {
  const response = await apiClient.get(`/monitoring/metrics/services/${serviceName}`)
  return response.data.data
}

export const getClusterMetrics = async (): Promise<ClusterMetrics> => {
  const response = await apiClient.get('/monitoring/metrics/cluster')
  return response.data.data || {
    nodeCount: 0,
    podCount: 0,
    cpuUsage: 0,
    memoryUsage: 0,
    totalCpuCores: 0,
    totalMemoryGb: 0,
    healthyPods: 0,
    unhealthyPods: 0,
  }
}
