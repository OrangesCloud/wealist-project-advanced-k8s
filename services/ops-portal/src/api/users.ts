import apiClient from './client'
import type { PortalUser, Role, ApiResponse } from '../types'

export const getMe = async (): Promise<PortalUser> => {
  const response = await apiClient.get<ApiResponse<PortalUser>>('/users/me')
  return response.data.data!
}

export const getAllUsers = async (): Promise<PortalUser[]> => {
  const response = await apiClient.get<ApiResponse<PortalUser[]>>('/admin/users')
  return response.data.data || []
}

export const getUserById = async (id: string): Promise<PortalUser> => {
  const response = await apiClient.get<ApiResponse<PortalUser>>(`/admin/users/${id}`)
  return response.data.data!
}

export const inviteUser = async (email: string, role: Role): Promise<PortalUser> => {
  const response = await apiClient.post<ApiResponse<PortalUser>>('/admin/users/invite', {
    email,
    role,
  })
  return response.data.data!
}

export const updateUserRole = async (id: string, role: Role): Promise<void> => {
  await apiClient.put(`/admin/users/${id}/role`, { role })
}

export const deactivateUser = async (id: string): Promise<void> => {
  await apiClient.post(`/admin/users/${id}/deactivate`)
}

export const reactivateUser = async (id: string): Promise<void> => {
  await apiClient.post(`/admin/users/${id}/reactivate`)
}
