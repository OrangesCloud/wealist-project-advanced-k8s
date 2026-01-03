import { useQuery } from '@tanstack/react-query'
import { useAuth } from '../contexts/AuthContext'
import { getApplications } from '../api/monitoring'
import { getAllUsers } from '../api/users'
import { Activity, Users, Settings, CheckCircle, XCircle, AlertCircle } from 'lucide-react'

export default function Dashboard() {
  const { user, hasRole } = useAuth()

  const { data: applications = [], isLoading: appsLoading } = useQuery({
    queryKey: ['applications'],
    queryFn: getApplications,
  })

  const { data: users = [], isLoading: usersLoading } = useQuery({
    queryKey: ['users'],
    queryFn: getAllUsers,
    enabled: hasRole(['admin']),
  })

  const healthyApps = applications.filter(app => app.health === 'Healthy').length
  const unhealthyApps = applications.filter(app => app.health !== 'Healthy').length
  const syncedApps = applications.filter(app => app.sync === 'Synced').length
  const outOfSyncApps = applications.filter(app => app.sync !== 'Synced').length

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-600 mt-1">Welcome back, {user?.name || user?.email}</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-green-100 rounded-lg">
              <CheckCircle className="w-6 h-6 text-green-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Healthy Apps</p>
              <p className="text-2xl font-bold text-gray-900">
                {appsLoading ? '-' : healthyApps}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-red-100 rounded-lg">
              <XCircle className="w-6 h-6 text-red-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Unhealthy Apps</p>
              <p className="text-2xl font-bold text-gray-900">
                {appsLoading ? '-' : unhealthyApps}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-blue-100 rounded-lg">
              <Activity className="w-6 h-6 text-blue-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Synced Apps</p>
              <p className="text-2xl font-bold text-gray-900">
                {appsLoading ? '-' : syncedApps}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-yellow-100 rounded-lg">
              <AlertCircle className="w-6 h-6 text-yellow-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Out of Sync</p>
              <p className="text-2xl font-bold text-gray-900">
                {appsLoading ? '-' : outOfSyncApps}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Stats */}
      {hasRole(['admin']) && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900">Portal Users</h2>
              <Users className="w-5 h-5 text-gray-400" />
            </div>
            {usersLoading ? (
              <div className="animate-pulse h-8 bg-gray-200 rounded" />
            ) : (
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-gray-600">Total Users</span>
                  <span className="font-medium">{users.length}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Active</span>
                  <span className="font-medium text-green-600">
                    {users.filter(u => u.isActive).length}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Admins</span>
                  <span className="font-medium">
                    {users.filter(u => u.role === 'admin').length}
                  </span>
                </div>
              </div>
            )}
          </div>

          <div className="card">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-gray-900">Quick Actions</h2>
              <Settings className="w-5 h-5 text-gray-400" />
            </div>
            <div className="space-y-2">
              <a
                href="/users"
                className="block px-4 py-2 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
              >
                Manage Users
              </a>
              <a
                href="/config"
                className="block px-4 py-2 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
              >
                App Configuration
              </a>
              <a
                href="/audit"
                className="block px-4 py-2 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
              >
                View Audit Logs
              </a>
            </div>
          </div>
        </div>
      )}

      {/* Applications Table */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">ArgoCD Applications</h2>
        {appsLoading ? (
          <div className="animate-pulse space-y-2">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="h-12 bg-gray-100 rounded" />
            ))}
          </div>
        ) : applications.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No applications found</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-gray-500 border-b">
                  <th className="pb-3 font-medium">Name</th>
                  <th className="pb-3 font-medium">Project</th>
                  <th className="pb-3 font-medium">Health</th>
                  <th className="pb-3 font-medium">Sync</th>
                </tr>
              </thead>
              <tbody>
                {applications.map((app) => (
                  <tr key={app.name} className="border-b last:border-0">
                    <td className="py-3 font-medium">{app.name}</td>
                    <td className="py-3 text-gray-600">{app.project}</td>
                    <td className="py-3">
                      <span className={`badge ${
                        app.health === 'Healthy' ? 'badge-success' :
                        app.health === 'Degraded' ? 'badge-warning' : 'badge-danger'
                      }`}>
                        {app.health}
                      </span>
                    </td>
                    <td className="py-3">
                      <span className={`badge ${
                        app.sync === 'Synced' ? 'badge-success' : 'badge-warning'
                      }`}>
                        {app.sync}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
