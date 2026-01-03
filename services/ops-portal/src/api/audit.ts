import apiClient from './client'
import type { AuditLog, PaginatedResponse, ResourceType, ActionType } from '../types'

export interface AuditLogFilters {
  page?: number
  limit?: number
  userId?: string
  resourceType?: ResourceType
  action?: ActionType
  startTime?: string
  endTime?: string
}

export const getAuditLogs = async (filters: AuditLogFilters = {}): Promise<PaginatedResponse<AuditLog>> => {
  const params = new URLSearchParams()

  if (filters.page) params.append('page', String(filters.page))
  if (filters.limit) params.append('limit', String(filters.limit))
  if (filters.userId) params.append('user_id', filters.userId)
  if (filters.resourceType) params.append('resource_type', filters.resourceType)
  if (filters.action) params.append('action', filters.action)
  if (filters.startTime) params.append('start_time', filters.startTime)
  if (filters.endTime) params.append('end_time', filters.endTime)

  const response = await apiClient.get<PaginatedResponse<AuditLog>>(`/admin/audit-logs?${params.toString()}`)
  return response.data
}

export const getAuditLogById = async (id: string): Promise<AuditLog> => {
  const response = await apiClient.get<{ data: AuditLog }>(`/admin/audit-logs/${id}`)
  return response.data.data
}
