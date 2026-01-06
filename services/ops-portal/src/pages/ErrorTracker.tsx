import { useQuery } from '@tanstack/react-query'
import { getErrorOverview, getRecentErrors, getErrorTrend, getErrorsByService } from '../api/errorTracker'
import { RefreshCw, AlertTriangle, TrendingUp, Server, AlertCircle } from 'lucide-react'
import type { ErrorTrendPoint, ServiceErrorSummary } from '../types'

export default function ErrorTracker() {
  const { data: overview, isLoading: overviewLoading, refetch: refetchOverview, isFetching: overviewFetching } = useQuery({
    queryKey: ['errorOverview'],
    queryFn: getErrorOverview,
    refetchInterval: 30000,
  })

  const { data: recentErrors = [], isLoading: errorsLoading, refetch: refetchErrors, isFetching: errorsFetching } = useQuery({
    queryKey: ['recentErrors'],
    queryFn: () => getRecentErrors(50),
    refetchInterval: 30000,
  })

  const { data: errorTrend = [], isLoading: trendLoading, refetch: refetchTrend, isFetching: trendFetching } = useQuery({
    queryKey: ['errorTrend'],
    queryFn: getErrorTrend,
    refetchInterval: 30000,
  })

  const { data: serviceErrors = [], isLoading: serviceLoading, refetch: refetchService, isFetching: serviceFetching } = useQuery({
    queryKey: ['serviceErrors'],
    queryFn: getErrorsByService,
    refetchInterval: 30000,
  })

  const handleRefresh = () => {
    refetchOverview()
    refetchErrors()
    refetchTrend()
    refetchService()
  }

  const isFetching = overviewFetching || errorsFetching || trendFetching || serviceFetching
  // isLoading unused but kept for potential future use
  void (overviewLoading || errorsLoading || trendLoading || serviceLoading)

  const formatTimestamp = (ts: number) => {
    return new Date(ts * 1000).toLocaleTimeString('ko-KR', {
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const formatPercentage = (pct: number) => {
    if (!pct || pct === 0 || isNaN(pct)) return '0%'
    return pct.toFixed(2) + '%'
  }

  const formatCount = (count: number) => {
    if (!count || count === 0) return '0'
    if (count >= 1000) return (count / 1000).toFixed(1) + 'k'
    return count.toFixed(0)
  }

  const getErrorRateColor = (rate: number) => {
    if (rate > 5) return 'text-red-600'
    if (rate > 1) return 'text-yellow-600'
    return 'text-green-600'
  }

  const maxTrendValue = Math.max(...errorTrend.map(p => p.errorRate || 0), 1)

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Error Tracker</h1>
          <p className="text-gray-600 mt-1">Monitor and analyze service errors</p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isFetching}
          className="btn btn-secondary flex items-center gap-2"
        >
          <RefreshCw size={16} className={isFetching ? 'animate-spin' : ''} />
          Refresh
        </button>
      </div>

      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-red-100 rounded-lg">
              <AlertTriangle className="w-5 h-5 text-red-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Total Errors (1h)</p>
              <p className="text-2xl font-bold text-gray-900">
                {overviewLoading ? '-' : formatCount(overview?.totalErrors || 0)}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-orange-100 rounded-lg">
              <TrendingUp className="w-5 h-5 text-orange-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Error Rate (5m)</p>
              <p className={`text-2xl font-bold ${getErrorRateColor(overview?.errorRate || 0)}`}>
                {overviewLoading ? '-' : formatPercentage(overview?.errorRate || 0)}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-purple-100 rounded-lg">
              <Server className="w-5 h-5 text-purple-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Most Errors</p>
              <p className="text-lg font-bold text-gray-900 truncate">
                {overviewLoading ? '-' : overview?.mostErrorService || 'None'}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-100 rounded-lg">
              <AlertCircle className="w-5 h-5 text-blue-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Top Service Errors</p>
              <p className="text-2xl font-bold text-gray-900">
                {overviewLoading ? '-' : formatCount(overview?.mostErrorCount || 0)}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Error Trend Chart */}
      <div className="card mb-8">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Error Rate Trend (Last 1h)</h2>
        {trendLoading ? (
          <div className="h-32 bg-gray-100 rounded animate-pulse" />
        ) : errorTrend.length === 0 ? (
          <div className="text-center py-8 text-gray-500">No trend data available</div>
        ) : (
          <div className="h-32 flex items-end gap-1">
            {errorTrend.map((point: ErrorTrendPoint, idx: number) => {
              const height = maxTrendValue > 0 ? (point.errorRate / maxTrendValue) * 100 : 0
              return (
                <div
                  key={idx}
                  className="flex-1 group relative"
                >
                  <div
                    className={`w-full rounded-t transition-all ${
                      point.errorRate > 5 ? 'bg-red-500' :
                      point.errorRate > 1 ? 'bg-yellow-500' : 'bg-green-500'
                    }`}
                    style={{ height: `${Math.max(height, 2)}%` }}
                  />
                  <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block bg-gray-800 text-white text-xs rounded px-2 py-1 whitespace-nowrap z-10">
                    {formatTimestamp(point.timestamp)}: {formatPercentage(point.errorRate)}
                  </div>
                </div>
              )
            })}
          </div>
        )}
        <div className="flex justify-between text-xs text-gray-500 mt-2">
          <span>1h ago</span>
          <span>Now</span>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Errors by Service */}
        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Errors by Service</h2>
          {serviceLoading ? (
            <div className="space-y-3">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="h-12 bg-gray-100 rounded animate-pulse" />
              ))}
            </div>
          ) : serviceErrors.length === 0 ? (
            <div className="text-center py-8 text-gray-500">No service errors</div>
          ) : (
            <div className="space-y-3">
              {serviceErrors.map((service: ServiceErrorSummary) => (
                <div key={service.serviceName} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div>
                    <p className="font-medium text-gray-900">{service.serviceName}</p>
                    <p className="text-sm text-gray-500">
                      5xx: {formatCount(service.error5xxCount)} | 4xx: {formatCount(service.error4xxCount)}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className={`text-lg font-bold ${getErrorRateColor(service.errorRate)}`}>
                      {formatPercentage(service.errorRate)}
                    </p>
                    <p className="text-sm text-gray-500">{formatCount(service.totalErrors)} total</p>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Recent Errors Table */}
        <div className="card">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Recent 5xx Errors</h2>
          {errorsLoading ? (
            <div className="space-y-3">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="h-12 bg-gray-100 rounded animate-pulse" />
              ))}
            </div>
          ) : recentErrors.length === 0 ? (
            <div className="text-center py-8">
              <AlertTriangle className="w-12 h-12 text-green-500 mx-auto mb-3" />
              <p className="text-gray-500">No recent 5xx errors</p>
              <p className="text-sm text-gray-400 mt-1">All services are healthy</p>
            </div>
          ) : (
            <div className="overflow-y-auto max-h-96">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-white">
                  <tr className="text-left text-gray-500 border-b">
                    <th className="pb-2 font-medium">Service</th>
                    <th className="pb-2 font-medium">Path</th>
                    <th className="pb-2 font-medium text-center">Code</th>
                    <th className="pb-2 font-medium text-right">Count</th>
                  </tr>
                </thead>
                <tbody>
                  {recentErrors.slice(0, 20).map((error, idx) => (
                    <tr key={idx} className="border-b last:border-0 hover:bg-gray-50">
                      <td className="py-2 font-medium text-gray-900 truncate max-w-[120px]">
                        {error.serviceName}
                      </td>
                      <td className="py-2 text-gray-600 truncate max-w-[150px]" title={error.requestPath}>
                        {error.requestPath || '-'}
                      </td>
                      <td className="py-2 text-center">
                        <span className="px-2 py-1 bg-red-100 text-red-700 rounded text-xs font-medium">
                          {error.responseCode}
                        </span>
                      </td>
                      <td className="py-2 text-right font-mono">
                        {formatCount(error.errorCount)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
