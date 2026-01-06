import apiClient from './client'
import type { ErrorOverview, RecentError, ErrorTrendPoint, ServiceErrorSummary } from '../types'

export const getErrorOverview = async (): Promise<ErrorOverview> => {
  const response = await apiClient.get('/monitoring/errors/overview')
  return response.data.data || {
    totalErrors: 0,
    errorRate: 0,
    mostErrorService: '',
    mostErrorCount: 0,
  }
}

export const getRecentErrors = async (limit: number = 50): Promise<RecentError[]> => {
  const response = await apiClient.get(`/monitoring/errors/recent?limit=${limit}`)
  return response.data.data || []
}

export const getErrorTrend = async (): Promise<ErrorTrendPoint[]> => {
  const response = await apiClient.get('/monitoring/errors/trend')
  return response.data.data || []
}

export const getErrorsByService = async (): Promise<ServiceErrorSummary[]> => {
  const response = await apiClient.get('/monitoring/errors/by-service')
  return response.data.data || []
}
