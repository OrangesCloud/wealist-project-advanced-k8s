import apiClient from './client'
import type { DeploymentHistoryEntry } from '../types'

export const getDeploymentHistory = async (): Promise<DeploymentHistoryEntry[]> => {
  const response = await apiClient.get('/monitoring/deployments/history')
  return response.data.data || []
}

export const getApplicationDeploymentHistory = async (appName: string): Promise<DeploymentHistoryEntry[]> => {
  const response = await apiClient.get(`/monitoring/deployments/${appName}/history`)
  return response.data.data || []
}
