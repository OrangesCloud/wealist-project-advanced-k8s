import api from './client'

export interface RBACInfo {
  namespace: string
  adminUsers: string[]
  policyCSV: string
  defaultRole: string
  scopes: string
}

export interface Application {
  name: string
  namespace: string
  project: string
  sync: string
  health: string
}

// Get ArgoCD RBAC configuration
export const getArgoCDRBAC = async (): Promise<RBACInfo> => {
  const response = await api.get('/admin/argocd/rbac')
  return response.data.data
}

// Add ArgoCD admin user
export const addArgoCDAdmin = async (email: string): Promise<void> => {
  await api.post('/admin/argocd/rbac/admins', { email })
}

// Remove ArgoCD admin user
export const removeArgoCDAdmin = async (email: string): Promise<void> => {
  await api.delete(`/admin/argocd/rbac/admins/${encodeURIComponent(email)}`)
}

// Get ArgoCD applications for monitoring
export const getApplications = async (): Promise<Application[]> => {
  const response = await api.get('/monitoring/applications')
  return response.data.data || []
}

// Sync an ArgoCD application
export const syncApplication = async (name: string): Promise<void> => {
  await api.post('/monitoring/applications/sync', { name })
}
