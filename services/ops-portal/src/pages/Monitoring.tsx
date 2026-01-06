import { useQuery } from '@tanstack/react-query'
import { getApplications, getServiceMetrics } from '../api/monitoring'
import { RefreshCw, ExternalLink, Activity, Clock, AlertTriangle, CheckCircle } from 'lucide-react'
import type { ServiceMetrics } from '../types'

export default function Monitoring() {
  const { data: applications = [], isLoading: appsLoading, refetch: refetchApps, isFetching: appsFetching } = useQuery({
    queryKey: ['applications'],
    queryFn: getApplications,
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  const { data: serviceMetrics = [], isLoading: metricsLoading, refetch: refetchMetrics, isFetching: metricsFetching } = useQuery({
    queryKey: ['serviceMetrics'],
    queryFn: getServiceMetrics,
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  const handleRefresh = () => {
    refetchApps()
    refetchMetrics()
  }

  const isFetching = appsFetching || metricsFetching

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

  const formatLatency = (ms: number) => {
    if (!ms || ms === 0) return '-'
    if (ms >= 1000) return (ms / 1000).toFixed(2) + 's'
    return ms.toFixed(0) + 'ms'
  }

  const formatRate = (rate: number) => {
    if (!rate || rate === 0) return '-'
    if (rate < 0.01) return '< 0.01/s'
    return rate.toFixed(2) + '/s'
  }

  const formatPercentage = (pct: number) => {
    if (!pct || pct === 0) return '0%'
    if (isNaN(pct)) return '-'
    return pct.toFixed(2) + '%'
  }

  const getServiceHealthColor = (metrics: ServiceMetrics) => {
    if (metrics.errorRate > 5) return 'border-red-500 bg-red-50'
    if (metrics.errorRate > 1) return 'border-yellow-500 bg-yellow-50'
    if (metrics.requestRate > 0) return 'border-green-500 bg-green-50'
    return 'border-gray-300 bg-gray-50'
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Monitoring</h1>
          <p className="text-gray-600 mt-1">Service metrics and ArgoCD application status</p>
        </div>
        <div className="flex gap-3">
          <button
            onClick={handleRefresh}
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

      {/* Service Metrics Table */}
      <div className="card mb-8">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Service Metrics (5m average)</h2>
        {metricsLoading ? (
          <div className="animate-pulse space-y-3">
            {[...Array(7)].map((_, i) => (
              <div key={i} className="h-12 bg-gray-100 rounded" />
            ))}
          </div>
        ) : serviceMetrics.length === 0 ? (
          <div className="text-center py-8">
            <AlertTriangle className="w-12 h-12 text-yellow-500 mx-auto mb-3" />
            <p className="text-gray-500">No service metrics available</p>
            <p className="text-sm text-gray-400 mt-1">Prometheus may not be configured or reachable</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-gray-500 border-b">
                  <th className="pb-3 font-medium">Service</th>
                  <th className="pb-3 font-medium text-right">Request Rate</th>
                  <th className="pb-3 font-medium text-right">Success Rate</th>
                  <th className="pb-3 font-medium text-right">Error Rate</th>
                  <th className="pb-3 font-medium text-right">Avg Latency</th>
                  <th className="pb-3 font-medium text-right">P95 Latency</th>
                  <th className="pb-3 font-medium text-right">P99 Latency</th>
                </tr>
              </thead>
              <tbody>
                {serviceMetrics.map((metric) => (
                  <tr key={metric.serviceName} className="border-b last:border-0 hover:bg-gray-50">
                    <td className="py-3">
                      <div className="flex items-center gap-2">
                        {metric.requestRate > 0 ? (
                          <Activity className="w-4 h-4 text-green-500" />
                        ) : (
                          <Activity className="w-4 h-4 text-gray-300" />
                        )}
                        <span className="font-medium">{metric.serviceName}</span>
                      </div>
                    </td>
                    <td className="py-3 text-right font-mono text-sm">
                      {formatRate(metric.requestRate)}
                    </td>
                    <td className="py-3 text-right">
                      <span className={`font-mono text-sm ${
                        metric.successRate >= 99 ? 'text-green-600' :
                        metric.successRate >= 95 ? 'text-yellow-600' :
                        metric.successRate > 0 ? 'text-red-600' : 'text-gray-400'
                      }`}>
                        {metric.requestRate > 0 ? formatPercentage(metric.successRate) : '-'}
                      </span>
                    </td>
                    <td className="py-3 text-right">
                      <span className={`font-mono text-sm ${
                        metric.errorRate > 5 ? 'text-red-600' :
                        metric.errorRate > 1 ? 'text-yellow-600' : 'text-green-600'
                      }`}>
                        {metric.requestRate > 0 ? formatPercentage(metric.errorRate) : '-'}
                      </span>
                    </td>
                    <td className="py-3 text-right font-mono text-sm">
                      {formatLatency(metric.avgLatency)}
                    </td>
                    <td className="py-3 text-right font-mono text-sm">
                      {formatLatency(metric.p95Latency)}
                    </td>
                    <td className="py-3 text-right font-mono text-sm">
                      {formatLatency(metric.p99Latency)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Service Cards Grid */}
      <h2 className="text-lg font-semibold text-gray-900 mb-4">Service Health Overview</h2>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 mb-8">
        {metricsLoading ? (
          [...Array(7)].map((_, i) => (
            <div key={i} className="card animate-pulse">
              <div className="h-6 bg-gray-200 rounded w-3/4 mb-4" />
              <div className="h-4 bg-gray-200 rounded w-1/2" />
            </div>
          ))
        ) : serviceMetrics.length === 0 ? null : (
          serviceMetrics.map((metric) => (
            <div
              key={metric.serviceName}
              className={`p-4 rounded-lg border-2 ${getServiceHealthColor(metric)}`}
            >
              <div className="flex items-center justify-between mb-3">
                <h3 className="font-semibold text-gray-900 truncate">{metric.serviceName}</h3>
                {metric.errorRate > 5 ? (
                  <AlertTriangle className="w-5 h-5 text-red-500 flex-shrink-0" />
                ) : metric.requestRate > 0 ? (
                  <CheckCircle className="w-5 h-5 text-green-500 flex-shrink-0" />
                ) : (
                  <Activity className="w-5 h-5 text-gray-400 flex-shrink-0" />
                )}
              </div>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-gray-600">Requests</span>
                  <span className="font-medium">{formatRate(metric.requestRate)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Latency</span>
                  <span className="font-medium">{formatLatency(metric.avgLatency)}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-600">Errors</span>
                  <span className={`font-medium ${
                    metric.errorRate > 5 ? 'text-red-600' :
                    metric.errorRate > 1 ? 'text-yellow-600' : 'text-green-600'
                  }`}>
                    {metric.requestRate > 0 ? formatPercentage(metric.errorRate) : '-'}
                  </span>
                </div>
              </div>
            </div>
          ))
        )}
      </div>

      {/* Applications Grid */}
      <h2 className="text-lg font-semibold text-gray-900 mb-4">ArgoCD Applications</h2>
      {appsLoading ? (
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
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
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

      {/* External Links */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">External Dashboards</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <a
            href="https://grafana.wealist.co.kr"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-3 p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
          >
            <div className="p-2 bg-orange-100 rounded-lg">
              <Activity className="w-5 h-5 text-orange-600" />
            </div>
            <div>
              <p className="font-medium text-gray-900">Grafana</p>
              <p className="text-sm text-gray-500">Metrics & Dashboards</p>
            </div>
            <ExternalLink size={16} className="ml-auto text-gray-400" />
          </a>

          <a
            href="https://argocd.wealist.co.kr"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-3 p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
          >
            <div className="p-2 bg-blue-100 rounded-lg">
              <Activity className="w-5 h-5 text-blue-600" />
            </div>
            <div>
              <p className="font-medium text-gray-900">ArgoCD</p>
              <p className="text-sm text-gray-500">Continuous Delivery</p>
            </div>
            <ExternalLink size={16} className="ml-auto text-gray-400" />
          </a>

          <a
            href="https://kiali.wealist.co.kr"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-3 p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
          >
            <div className="p-2 bg-purple-100 rounded-lg">
              <Clock className="w-5 h-5 text-purple-600" />
            </div>
            <div>
              <p className="font-medium text-gray-900">Kiali</p>
              <p className="text-sm text-gray-500">Service Mesh</p>
            </div>
            <ExternalLink size={16} className="ml-auto text-gray-400" />
          </a>
        </div>
      </div>
    </div>
  )
}
