import { useQuery } from '@tanstack/react-query'
import { getApplications } from '../api/monitoring'
import { RefreshCw, ExternalLink } from 'lucide-react'

export default function Monitoring() {
  const { data: applications = [], isLoading, refetch, isFetching } = useQuery({
    queryKey: ['applications'],
    queryFn: getApplications,
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  const healthyCount = applications.filter(app => app.health === 'Healthy').length
  const syncedCount = applications.filter(app => app.sync === 'Synced').length

  const getHealthClass = (health: string) => {
    switch (health) {
      case 'Healthy':
        return 'bg-green-100 text-green-800 border-green-200'
      case 'Degraded':
        return 'bg-yellow-100 text-yellow-800 border-yellow-200'
      case 'Progressing':
        return 'bg-blue-100 text-blue-800 border-blue-200'
      default:
        return 'bg-red-100 text-red-800 border-red-200'
    }
  }

  const getSyncClass = (sync: string) => {
    switch (sync) {
      case 'Synced':
        return 'badge-success'
      case 'OutOfSync':
        return 'badge-warning'
      default:
        return 'badge-info'
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Monitoring</h1>
          <p className="text-gray-600 mt-1">ArgoCD application status overview</p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="btn btn-secondary flex items-center gap-2"
          >
            <RefreshCw size={16} className={isFetching ? 'animate-spin' : ''} />
            Refresh
          </button>
          <a
            href="https://argocd.wealist.co.kr"
            target="_blank"
            rel="noopener noreferrer"
            className="btn btn-primary flex items-center gap-2"
          >
            Open ArgoCD
            <ExternalLink size={16} />
          </a>
        </div>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div className="card">
          <p className="text-sm text-gray-600">Total Applications</p>
          <p className="text-3xl font-bold text-gray-900">{applications.length}</p>
        </div>
        <div className="card">
          <p className="text-sm text-gray-600">Healthy</p>
          <p className="text-3xl font-bold text-green-600">{healthyCount}</p>
        </div>
        <div className="card">
          <p className="text-sm text-gray-600">Synced</p>
          <p className="text-3xl font-bold text-blue-600">{syncedCount}</p>
        </div>
        <div className="card">
          <p className="text-sm text-gray-600">Issues</p>
          <p className="text-3xl font-bold text-red-600">
            {applications.length - healthyCount}
          </p>
        </div>
      </div>

      {/* Applications Grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[...Array(6)].map((_, i) => (
            <div key={i} className="card animate-pulse">
              <div className="h-6 bg-gray-200 rounded w-3/4 mb-4" />
              <div className="h-4 bg-gray-200 rounded w-1/2" />
            </div>
          ))}
        </div>
      ) : applications.length === 0 ? (
        <div className="card text-center py-12">
          <p className="text-gray-500">No applications found</p>
          <p className="text-sm text-gray-400 mt-2">
            Make sure ArgoCD is properly configured
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {applications.map((app) => (
            <div
              key={app.name}
              className={`p-4 rounded-lg border-2 ${getHealthClass(app.health)}`}
            >
              <div className="flex items-start justify-between mb-3">
                <div>
                  <h3 className="font-semibold text-gray-900">{app.name}</h3>
                  <p className="text-sm text-gray-500">{app.project}</p>
                </div>
                <span className={`badge ${getSyncClass(app.sync)}`}>
                  {app.sync}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span
                  className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${getHealthClass(app.health)}`}
                >
                  {app.health}
                </span>
                <span className="text-xs text-gray-500">{app.namespace}</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Grafana Embed Placeholder */}
      <div className="card mt-8">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Metrics Dashboard</h2>
        <div className="bg-gray-100 rounded-lg h-96 flex items-center justify-center">
          <div className="text-center">
            <p className="text-gray-500">Grafana dashboard will be embedded here</p>
            <a
              href="https://grafana.wealist.co.kr"
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary-600 hover:underline mt-2 inline-flex items-center gap-1"
            >
              Open Grafana <ExternalLink size={14} />
            </a>
          </div>
        </div>
      </div>
    </div>
  )
}
