import apiClient from './client'
import type { LogQueryResult, ServiceInfo } from '../types'

export interface LogQueryParams {
  service?: string
  level?: string
  query?: string
  start?: string
  end?: string
  limit?: number
}

export const getLogs = async (params: LogQueryParams = {}): Promise<LogQueryResult> => {
  const searchParams = new URLSearchParams()
  if (params.service) searchParams.set('service', params.service)
  if (params.level) searchParams.set('level', params.level)
  if (params.query) searchParams.set('query', params.query)
  if (params.start) searchParams.set('start', params.start)
  if (params.end) searchParams.set('end', params.end)
  if (params.limit) searchParams.set('limit', params.limit.toString())

  const queryString = searchParams.toString()
  const url = queryString ? `/monitoring/logs?${queryString}` : '/monitoring/logs'

  const response = await apiClient.get(url)
  return response.data.data || { entries: [], totalCount: 0 }
}

export const getLogServices = async (): Promise<ServiceInfo[]> => {
  const response = await apiClient.get('/monitoring/logs/services')
  return response.data.data || []
}
