import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getAuditLogs } from '../api/audit'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import type { ActionType, ResourceType } from '../types'

export default function AuditLogs() {
  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState<{
    resourceType?: ResourceType
    action?: ActionType
  }>({})

  const { data, isLoading } = useQuery({
    queryKey: ['auditLogs', page, filters],
    queryFn: () => getAuditLogs({ page, limit: 20, ...filters }),
  })

  const logs = data?.data || []
  const totalPages = Math.ceil((data?.total || 0) / 20)

  const getActionBadgeClass = (action: ActionType) => {
    switch (action) {
      case 'create':
        return 'badge-success'
      case 'update':
        return 'badge-info'
      case 'delete':
        return 'badge-danger'
      case 'login':
      case 'logout':
        return 'badge-warning'
      default:
        return 'badge-info'
    }
  }

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Audit Logs</h1>
        <p className="text-gray-600 mt-1">Track all changes made in the portal</p>
      </div>

      {/* Filters */}
      <div className="card mb-6">
        <div className="flex flex-wrap gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Resource Type
            </label>
            <select
              value={filters.resourceType || ''}
              onChange={(e) =>
                setFilters({
                  ...filters,
                  resourceType: (e.target.value || undefined) as ResourceType,
                })
              }
              className="input w-40"
            >
              <option value="">All</option>
              <option value="portal_user">Portal User</option>
              <option value="argocd_rbac">ArgoCD RBAC</option>
              <option value="feature_flag">Feature Flag</option>
              <option value="app_config">App Config</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Action
            </label>
            <select
              value={filters.action || ''}
              onChange={(e) =>
                setFilters({
                  ...filters,
                  action: (e.target.value || undefined) as ActionType,
                })
              }
              className="input w-32"
            >
              <option value="">All</option>
              <option value="create">Create</option>
              <option value="update">Update</option>
              <option value="delete">Delete</option>
              <option value="login">Login</option>
              <option value="logout">Logout</option>
            </select>
          </div>
        </div>
      </div>

      {/* Logs Table */}
      <div className="card">
        {isLoading ? (
          <div className="animate-pulse space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-12 bg-gray-100 rounded" />
            ))}
          </div>
        ) : logs.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No audit logs found</p>
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="text-left text-sm text-gray-500 border-b">
                    <th className="pb-3 font-medium">Timestamp</th>
                    <th className="pb-3 font-medium">User</th>
                    <th className="pb-3 font-medium">Action</th>
                    <th className="pb-3 font-medium">Resource</th>
                    <th className="pb-3 font-medium">Details</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log) => (
                    <tr key={log.id} className="border-b last:border-0">
                      <td className="py-3 text-sm text-gray-600">
                        {new Date(log.createdAt).toLocaleString()}
                      </td>
                      <td className="py-3 text-sm">{log.userEmail}</td>
                      <td className="py-3">
                        <span className={`badge ${getActionBadgeClass(log.action)}`}>
                          {log.action}
                        </span>
                      </td>
                      <td className="py-3 text-sm">
                        <span className="text-gray-600">{log.resourceType}</span>
                        <span className="text-gray-400 ml-1">#{log.resourceId.slice(0, 8)}</span>
                      </td>
                      <td className="py-3 text-sm text-gray-600 max-w-xs truncate">
                        {log.details || '-'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-between mt-4 pt-4 border-t">
                <p className="text-sm text-gray-600">
                  Page {page} of {totalPages} ({data?.total} total)
                </p>
                <div className="flex gap-2">
                  <button
                    onClick={() => setPage((p) => Math.max(1, p - 1))}
                    disabled={page === 1}
                    className="btn btn-secondary flex items-center gap-1"
                  >
                    <ChevronLeft size={16} />
                    Previous
                  </button>
                  <button
                    onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                    disabled={page === totalPages}
                    className="btn btn-secondary flex items-center gap-1"
                  >
                    Next
                    <ChevronRight size={16} />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
