import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getAllUsers, inviteUser, updateUserRole, deactivateUser, reactivateUser } from '../api/users'
import { UserPlus, MoreVertical } from 'lucide-react'
import type { Role, PortalUser } from '../types'

export default function Users() {
  const queryClient = useQueryClient()
  const [isInviteModalOpen, setIsInviteModalOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<Role>('viewer')
  const [activeDropdown, setActiveDropdown] = useState<string | null>(null)

  const { data: users = [], isLoading } = useQuery({
    queryKey: ['users'],
    queryFn: getAllUsers,
  })

  const inviteMutation = useMutation({
    mutationFn: ({ email, role }: { email: string; role: Role }) =>
      inviteUser(email, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setIsInviteModalOpen(false)
      setInviteEmail('')
      setInviteRole('viewer')
    },
  })

  const updateRoleMutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: Role }) =>
      updateUserRole(id, role),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setActiveDropdown(null)
    },
  })

  const toggleActiveMutation = useMutation({
    mutationFn: ({ id, isActive }: { id: string; isActive: boolean }) =>
      isActive ? deactivateUser(id) : reactivateUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] })
      setActiveDropdown(null)
    },
  })

  const handleInvite = (e: React.FormEvent) => {
    e.preventDefault()
    inviteMutation.mutate({ email: inviteEmail, role: inviteRole })
  }

  const getRoleBadgeClass = (role: Role) => {
    switch (role) {
      case 'admin':
        return 'badge-danger'
      case 'pm':
        return 'badge-info'
      case 'viewer':
        return 'badge-success'
      default:
        return 'badge-info'
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Users</h1>
          <p className="text-gray-600 mt-1">Manage portal user access and roles</p>
        </div>
        <button
          onClick={() => setIsInviteModalOpen(true)}
          className="btn btn-primary flex items-center gap-2"
        >
          <UserPlus size={20} />
          Invite User
        </button>
      </div>

      <div className="card">
        {isLoading ? (
          <div className="animate-pulse space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-16 bg-gray-100 rounded" />
            ))}
          </div>
        ) : users.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No users found</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-gray-500 border-b">
                  <th className="pb-3 font-medium">User</th>
                  <th className="pb-3 font-medium">Role</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium">Last Login</th>
                  <th className="pb-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user: PortalUser) => (
                  <tr key={user.id} className="border-b last:border-0">
                    <td className="py-4">
                      <div className="flex items-center">
                        {user.picture ? (
                          <img
                            src={user.picture}
                            alt={user.name}
                            className="w-10 h-10 rounded-full"
                          />
                        ) : (
                          <div className="w-10 h-10 rounded-full bg-primary-100 flex items-center justify-center">
                            <span className="text-primary-700 font-medium">
                              {user.name?.charAt(0) || user.email.charAt(0)}
                            </span>
                          </div>
                        )}
                        <div className="ml-3">
                          <p className="font-medium text-gray-900">
                            {user.name || '-'}
                          </p>
                          <p className="text-sm text-gray-500">{user.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="py-4">
                      <span className={`badge ${getRoleBadgeClass(user.role)}`}>
                        {user.role}
                      </span>
                    </td>
                    <td className="py-4">
                      <span
                        className={`badge ${
                          user.isActive ? 'badge-success' : 'badge-danger'
                        }`}
                      >
                        {user.isActive ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="py-4 text-gray-600">
                      {user.lastLoginAt
                        ? new Date(user.lastLoginAt).toLocaleDateString()
                        : 'Never'}
                    </td>
                    <td className="py-4">
                      <div className="relative">
                        <button
                          onClick={() =>
                            setActiveDropdown(
                              activeDropdown === user.id ? null : user.id
                            )
                          }
                          className="p-2 hover:bg-gray-100 rounded-lg"
                        >
                          <MoreVertical size={16} />
                        </button>
                        {activeDropdown === user.id && (
                          <div className="absolute right-0 mt-2 w-48 bg-white border border-gray-200 rounded-lg shadow-lg z-10">
                            <div className="py-1">
                              <button
                                onClick={() =>
                                  updateRoleMutation.mutate({
                                    id: user.id,
                                    role: 'admin',
                                  })
                                }
                                className="w-full px-4 py-2 text-left text-sm hover:bg-gray-100"
                              >
                                Set as Admin
                              </button>
                              <button
                                onClick={() =>
                                  updateRoleMutation.mutate({
                                    id: user.id,
                                    role: 'pm',
                                  })
                                }
                                className="w-full px-4 py-2 text-left text-sm hover:bg-gray-100"
                              >
                                Set as PM
                              </button>
                              <button
                                onClick={() =>
                                  updateRoleMutation.mutate({
                                    id: user.id,
                                    role: 'viewer',
                                  })
                                }
                                className="w-full px-4 py-2 text-left text-sm hover:bg-gray-100"
                              >
                                Set as Viewer
                              </button>
                              <hr className="my-1" />
                              <button
                                onClick={() =>
                                  toggleActiveMutation.mutate({
                                    id: user.id,
                                    isActive: user.isActive,
                                  })
                                }
                                className={`w-full px-4 py-2 text-left text-sm hover:bg-gray-100 ${
                                  user.isActive
                                    ? 'text-red-600'
                                    : 'text-green-600'
                                }`}
                              >
                                {user.isActive ? 'Deactivate' : 'Reactivate'}
                              </button>
                            </div>
                          </div>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Invite Modal */}
      {isInviteModalOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-bold text-gray-900 mb-4">Invite User</h2>
            <form onSubmit={handleInvite}>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Email
                </label>
                <input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  className="input"
                  placeholder="user@example.com"
                  required
                />
              </div>
              <div className="mb-6">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Role
                </label>
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value as Role)}
                  className="input"
                >
                  <option value="viewer">Viewer</option>
                  <option value="pm">PM</option>
                  <option value="admin">Admin</option>
                </select>
              </div>
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={() => setIsInviteModalOpen(false)}
                  className="btn btn-secondary flex-1"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={inviteMutation.isPending}
                  className="btn btn-primary flex-1"
                >
                  {inviteMutation.isPending ? 'Inviting...' : 'Invite'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
