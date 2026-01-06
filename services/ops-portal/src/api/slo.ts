import apiClient from './client'
import type { SLOOverview, BurnRate } from '../types'

export const getSLOOverview = async (): Promise<SLOOverview> => {
  const response = await apiClient.get('/monitoring/slo/overview')
  return response.data.data || {
    services: [],
    overallHealth: 'healthy',
    servicesAtRisk: 0,
    totalServices: 0,
  }
}

export const getBurnRates = async (): Promise<BurnRate[]> => {
  const response = await apiClient.get('/monitoring/slo/burn-rates')
  return response.data.data || []
}
