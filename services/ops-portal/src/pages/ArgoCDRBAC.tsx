import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getArgoCDRBAC, addArgoCDAdmin, removeArgoCDAdmin } from '../api/argocd'
import { Plus, Trash2, Shield, RefreshCw, Copy, CheckCircle } from 'lucide-react'

export default function ArgoCDRBAC() {
  const queryClient = useQueryClient()
  const [newAdminEmail, setNewAdminEmail] = useState('')
  const [copied, setCopied] = useState(false)

  const { data: rbac, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['argocd-rbac'],
    queryFn: getArgoCDRBAC,
  })

  const addMutation = useMutation({
    mutationFn: addArgoCDAdmin,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['argocd-rbac'] })
      setNewAdminEmail('')
    },
  })

  const removeMutation = useMutation({
    mutationFn: removeArgoCDAdmin,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['argocd-rbac'] })
    },
  })

  const handleAddAdmin = (e: React.FormEvent) => {
    e.preventDefault()
    if (newAdminEmail.trim()) {
      addMutation.mutate(newAdminEmail.trim())
    }
  }

  const handleRemoveAdmin = (email: string) => {
    if (window.confirm(`Remove "${email}" from ArgoCD admins?`)) {
      removeMutation.mutate(email)
    }
  }

  const copyPolicyCSV = () => {
    if (rbac?.policyCSV) {
      navigator.clipboard.writeText(rbac.policyCSV)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">ArgoCD RBAC</h1>
          <p className="text-gray-600 mt-1">Manage ArgoCD role-based access control</p>
        </div>
        <button
          onClick={() => refetch()}
          disabled={isFetching}
          className="btn btn-secondary flex items-center gap-2"
        >
          <RefreshCw size={16} className={isFetching ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {isLoading ? (
        <div className="space-y-4">
          <div className="card animate-pulse">
            <div className="h-6 bg-gray-200 rounded w-1/4 mb-4" />
            <div className="h-20 bg-gray-200 rounded" />
          </div>
        </div>
      ) : (
        <>
          {/* Admin Users Section */}
          <div className="card mb-6">
            <div className="flex items-center gap-2 mb-4">
              <Shield className="text-primary-600" size={20} />
              <h2 className="text-lg font-semibold text-gray-900">Admin Users</h2>
            </div>

            {/* Add Admin Form */}
            <form onSubmit={handleAddAdmin} className="flex gap-3 mb-4">
              <input
                type="email"
                value={newAdminEmail}
                onChange={(e) => setNewAdminEmail(e.target.value)}
                placeholder="email@example.com"
                className="input flex-1"
                required
              />
              <button
                type="submit"
                disabled={addMutation.isPending}
                className="btn btn-primary flex items-center gap-2"
              >
                <Plus size={16} />
                {addMutation.isPending ? 'Adding...' : 'Add Admin'}
              </button>
            </form>

            {addMutation.isError && (
              <p className="text-red-600 text-sm mb-4">
                Failed to add admin: {(addMutation.error as Error).message}
              </p>
            )}

            {/* Admin List */}
            {rbac?.adminUsers && rbac.adminUsers.length > 0 ? (
              <ul className="divide-y">
                {rbac.adminUsers.map((email) => (
                  <li
                    key={email}
                    className="py-3 flex items-center justify-between"
                  >
                    <span className="font-mono text-sm">{email}</span>
                    <button
                      onClick={() => handleRemoveAdmin(email)}
                      disabled={removeMutation.isPending}
                      className="p-2 text-gray-500 hover:text-red-600 hover:bg-gray-100 rounded"
                    >
                      <Trash2 size={16} />
                    </button>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-gray-500 text-sm">No admin users configured</p>
            )}
          </div>

          {/* Policy Details */}
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900">Policy Configuration</h2>
              <button
                onClick={copyPolicyCSV}
                className="btn btn-secondary flex items-center gap-2 text-sm"
              >
                {copied ? (
                  <>
                    <CheckCircle size={14} className="text-green-600" />
                    Copied!
                  </>
                ) : (
                  <>
                    <Copy size={14} />
                    Copy Policy
                  </>
                )}
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
              <div>
                <label className="block text-sm text-gray-600 mb-1">Namespace</label>
                <code className="text-sm bg-gray-100 px-3 py-2 rounded block">
                  {rbac?.namespace || 'argocd'}
                </code>
              </div>
              <div>
                <label className="block text-sm text-gray-600 mb-1">Default Role</label>
                <code className="text-sm bg-gray-100 px-3 py-2 rounded block">
                  {rbac?.defaultRole || 'role:readonly'}
                </code>
              </div>
              <div>
                <label className="block text-sm text-gray-600 mb-1">Scopes</label>
                <code className="text-sm bg-gray-100 px-3 py-2 rounded block">
                  {rbac?.scopes || '[groups]'}
                </code>
              </div>
            </div>

            <div>
              <label className="block text-sm text-gray-600 mb-1">Policy CSV</label>
              <pre className="text-sm bg-gray-100 px-4 py-3 rounded overflow-x-auto max-h-64 overflow-y-auto">
                {rbac?.policyCSV || 'No policy defined'}
              </pre>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
