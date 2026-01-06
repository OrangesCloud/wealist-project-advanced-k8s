import { useQuery } from '@tanstack/react-query'
import { useAuth } from '../contexts/AuthContext'
import { getApplications, getSystemOverview, getClusterMetrics } from '../api/monitoring'
import { getAllUsers } from '../api/users'
import {
  Activity,
  Users,
  Settings,
  CheckCircle,
  XCircle,
  AlertCircle,
  Cpu,
  HardDrive,
  Server,
  Clock,
  TrendingUp,
  Zap
} from 'lucide-react'

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

  const { data: systemOverview, isLoading: overviewLoading } = useQuery({
    queryKey: ['systemOverview'],
    queryFn: getSystemOverview,
    refetchInterval: 60000, // Refresh every minute
  })

  const { data: clusterMetrics, isLoading: clusterLoading } = useQuery({
    queryKey: ['clusterMetrics'],
    queryFn: getClusterMetrics,
    refetchInterval: 60000, // Refresh every minute
  })

  const healthyApps = applications.filter(app => app.health === 'Healthy').length
  const unhealthyApps = applications.filter(app => app.health !== 'Healthy').length
  const syncedApps = applications.filter(app => app.sync === 'Synced').length
  const outOfSyncApps = applications.filter(app => app.sync !== 'Synced').length

  const formatNumber = (num: number) => {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
    return num.toFixed(0)
  }

  const formatLatency = (ms: number) => {
    if (ms >= 1000) return (ms / 1000).toFixed(2) + 's'
    return ms.toFixed(0) + 'ms'
  }

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <p className="text-gray-600 mt-1">Welcome back, {user?.name || user?.email}</p>
      </div>

      {/* ArgoCD Application Stats */}
      <h2 className="text-lg font-semibold text-gray-900 mb-4">ArgoCD Applications</h2>
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

      {/* System Overview Metrics */}
      <h2 className="text-lg font-semibold text-gray-900 mb-4">System Overview (Last Hour)</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-indigo-100 rounded-lg">
              <TrendingUp className="w-6 h-6 text-indigo-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Total Requests</p>
              <p className="text-2xl font-bold text-gray-900">
                {overviewLoading ? '-' : formatNumber(systemOverview?.totalRequests || 0)}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-purple-100 rounded-lg">
              <Clock className="w-6 h-6 text-purple-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Avg Response Time</p>
              <p className="text-2xl font-bold text-gray-900">
                {overviewLoading ? '-' : formatLatency(systemOverview?.avgResponseTime || 0)}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-red-100 rounded-lg">
              <AlertCircle className="w-6 h-6 text-red-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Error Rate</p>
              <p className="text-2xl font-bold text-gray-900">
                {overviewLoading ? '-' : `${(systemOverview?.errorPercentage || 0).toFixed(2)}%`}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-green-100 rounded-lg">
              <Zap className="w-6 h-6 text-green-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Active Services</p>
              <p className="text-2xl font-bold text-gray-900">
                {overviewLoading ? '-' : systemOverview?.activeServices || 0}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Cluster Metrics */}
      <h2 className="text-lg font-semibold text-gray-900 mb-4">Cluster Resources</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-cyan-100 rounded-lg">
              <Server className="w-6 h-6 text-cyan-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Nodes</p>
              <p className="text-2xl font-bold text-gray-900">
                {clusterLoading ? '-' : clusterMetrics?.nodeCount || 0}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center">
            <div className="p-3 bg-teal-100 rounded-lg">
              <HardDrive className="w-6 h-6 text-teal-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm text-gray-600">Running Pods</p>
              <p className="text-2xl font-bold text-gray-900">
                {clusterLoading ? '-' : clusterMetrics?.healthyPods || 0}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <div className="p-3 bg-orange-100 rounded-lg">
                <Cpu className="w-6 h-6 text-orange-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm text-gray-600">CPU Usage</p>
                <p className="text-2xl font-bold text-gray-900">
                  {clusterLoading ? '-' : `${(clusterMetrics?.cpuUsage || 0).toFixed(1)}%`}
                </p>
              </div>
            </div>
          </div>
          {!clusterLoading && clusterMetrics && (
            <div className="mt-3">
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className={`h-2 rounded-full ${
                    (clusterMetrics.cpuUsage || 0) > 80 ? 'bg-red-500' :
                    (clusterMetrics.cpuUsage || 0) > 60 ? 'bg-yellow-500' : 'bg-green-500'
                  }`}
                  style={{ width: `${Math.min(clusterMetrics.cpuUsage || 0, 100)}%` }}
                />
              </div>
            </div>
          )}
        </div>

        <div className="card">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <div className="p-3 bg-pink-100 rounded-lg">
                <HardDrive className="w-6 h-6 text-pink-600" />
              </div>
              <div className="ml-4">
                <p className="text-sm text-gray-600">Memory Usage</p>
                <p className="text-2xl font-bold text-gray-900">
                  {clusterLoading ? '-' : `${(clusterMetrics?.memoryUsage || 0).toFixed(1)}%`}
                </p>
              </div>
            </div>
          </div>
          {!clusterLoading && clusterMetrics && (
            <div className="mt-3">
              <div className="w-full bg-gray-200 rounded-full h-2">
                <div
                  className={`h-2 rounded-full ${
                    (clusterMetrics.memoryUsage || 0) > 80 ? 'bg-red-500' :
                    (clusterMetrics.memoryUsage || 0) > 60 ? 'bg-yellow-500' : 'bg-green-500'
                  }`}
                  style={{ width: `${Math.min(clusterMetrics.memoryUsage || 0, 100)}%` }}
                />
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Quick Stats for Admin */}
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
