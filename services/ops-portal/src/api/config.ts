import apiClient from './client'
import type { AppConfig, ApiResponse } from '../types'

export const getAllConfigs = async (): Promise<AppConfig[]> => {
  const response = await apiClient.get<ApiResponse<AppConfig[]>>('/admin/config')
  return response.data.data || []
}

export const getActiveConfigs = async (): Promise<Record<string, string>> => {
  const response = await apiClient.get<ApiResponse<Record<string, string>>>('/config/active')
  return response.data.data || {}
}

export const getConfigByKey = async (key: string): Promise<AppConfig> => {
  const response = await apiClient.get<ApiResponse<AppConfig>>(`/config/${key}`)
  return response.data.data!
}

export const createConfig = async (key: string, value: string, description?: string): Promise<AppConfig> => {
  const response = await apiClient.post<ApiResponse<AppConfig>>('/admin/config', {
    key,
    value,
    description,
  })
  return response.data.data!
}

export const updateConfig = async (id: string, value: string, description?: string, isActive?: boolean): Promise<AppConfig> => {
  const response = await apiClient.put<ApiResponse<AppConfig>>(`/admin/config/${id}`, {
    value,
    description,
    isActive,
  })
  return response.data.data!
}

export const deleteConfig = async (id: string): Promise<void> => {
  await apiClient.delete(`/admin/config/${id}`)
}
