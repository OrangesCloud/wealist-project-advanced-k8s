import apiClient from './client'
import type { ArgoCDApplication } from '../types'

export const getApplications = async (): Promise<ArgoCDApplication[]> => {
  const response = await apiClient.get('/monitoring/applications')
  return response.data.data || []
}

export const syncApplication = async (name: string): Promise<void> => {
  await apiClient.post('/monitoring/applications/sync', { name })
}
