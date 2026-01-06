import { useQuery } from '@tanstack/react-query'
import { getSLOOverview, getBurnRates } from '../api/slo'
import { RefreshCw, Target, TrendingDown, CheckCircle, XCircle, AlertTriangle } from 'lucide-react'
import type { ServiceSLO, BurnRate } from '../types'

export default function SLODashboard() {
  const { data: overview, isLoading: overviewLoading, refetch: refetchOverview, isFetching: overviewFetching } = useQuery({
    queryKey: ['sloOverview'],
    queryFn: getSLOOverview,
    refetchInterval: 60000, // Refresh every minute
  })

  const { data: burnRates = [], isLoading: burnLoading, refetch: refetchBurn, isFetching: burnFetching } = useQuery({
    queryKey: ['burnRates'],
    queryFn: getBurnRates,
    refetchInterval: 60000,
  })

  const handleRefresh = () => {
    refetchOverview()
    refetchBurn()
  }

  const isFetching = overviewFetching || burnFetching
  void (overviewLoading || burnLoading) // Suppress unused warning

  const formatPercentage = (pct: number, decimals: number = 2) => {
    if (!pct || isNaN(pct)) return '0%'
    return pct.toFixed(decimals) + '%'
  }

  const formatLatency = (ms: number) => {
    if (!ms || ms === 0 || isNaN(ms)) return '-'
    if (ms >= 1000) return (ms / 1000).toFixed(2) + 's'
    return ms.toFixed(0) + 'ms'
  }

  const formatBurnRate = (rate: number) => {
    if (!rate || isNaN(rate)) return '0x'
    return rate.toFixed(1) + 'x'
  }

  const getHealthColor = (health: string) => {
    switch (health) {
      case 'healthy': return 'text-green-600 bg-green-100'
      case 'degraded': return 'text-yellow-600 bg-yellow-100'
      case 'critical': return 'text-red-600 bg-red-100'
      default: return 'text-gray-600 bg-gray-100'
    }
  }

  const getErrorBudgetColor = (remaining: number) => {
    if (remaining > 50) return 'bg-green-500'
    if (remaining > 20) return 'bg-yellow-500'
    return 'bg-red-500'
  }

  const getBurnRateColor = (rate: number) => {
    if (rate > 10) return 'text-red-600'
    if (rate > 3) return 'text-yellow-600'
    return 'text-green-600'
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">SLO Dashboard</h1>
          <p className="text-gray-600 mt-1">Service Level Objectives and Error Budget tracking</p>
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
            <div className={`p-2 rounded-lg ${getHealthColor(overview?.overallHealth || 'healthy')}`}>
              <Target className="w-5 h-5" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Overall Health</p>
              <p className={`text-xl font-bold capitalize ${getHealthColor(overview?.overallHealth || 'healthy').split(' ')[0]}`}>
                {overview?.overallHealth || 'Loading...'}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-100 rounded-lg">
              <CheckCircle className="w-5 h-5 text-blue-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Total Services</p>
              <p className="text-2xl font-bold text-gray-900">
                {overview?.totalServices || 0}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-red-100 rounded-lg">
              <AlertTriangle className="w-5 h-5 text-red-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">At Risk</p>
              <p className="text-2xl font-bold text-red-600">
                {overview?.servicesAtRisk || 0}
              </p>
            </div>
          </div>
        </div>

        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-purple-100 rounded-lg">
              <TrendingDown className="w-5 h-5 text-purple-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">SLO Target</p>
              <p className="text-2xl font-bold text-gray-900">99.9%</p>
            </div>
          </div>
        </div>
      </div>

      {/* Service SLO Table */}
      <div className="card mb-8">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Service SLO Status</h2>
        {!overview?.services || overview.services.length === 0 ? (
          <div className="text-center py-8 text-gray-500">No SLO data available</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-gray-500 border-b">
                  <th className="pb-3 font-medium">Service</th>
                  <th className="pb-3 font-medium text-center">Availability</th>
                  <th className="pb-3 font-medium text-center">Status</th>
                  <th className="pb-3 font-medium text-center">P50 Latency</th>
                  <th className="pb-3 font-medium text-center">P99 Latency</th>
                  <th className="pb-3 font-medium">Error Budget</th>
                </tr>
              </thead>
              <tbody>
                {overview.services.map((slo: ServiceSLO) => (
                  <tr key={slo.serviceName} className="border-b last:border-0 hover:bg-gray-50">
                    <td className="py-3 font-medium text-gray-900">
                      {slo.serviceName}
                    </td>
                    <td className="py-3 text-center">
                      <span className={`font-mono ${
                        slo.availabilityMet ? 'text-green-600' : 'text-red-600'
                      }`}>
                        {formatPercentage(slo.availability, 3)}
                      </span>
                      <span className="text-gray-400 text-xs ml-1">
                        / {formatPercentage(slo.availabilityTarget, 1)}
                      </span>
                    </td>
                    <td className="py-3 text-center">
                      {slo.availabilityMet && slo.latencyMet ? (
                        <span className="inline-flex items-center gap-1 px-2 py-1 bg-green-100 text-green-700 rounded text-xs">
                          <CheckCircle size={12} /> Met
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 px-2 py-1 bg-red-100 text-red-700 rounded text-xs">
                          <XCircle size={12} /> Breached
                        </span>
                      )}
                    </td>
                    <td className="py-3 text-center font-mono text-sm">
                      {formatLatency(slo.latencyP50)}
                    </td>
                    <td className="py-3 text-center">
                      <span className={`font-mono text-sm ${
                        slo.latencyMet ? 'text-green-600' : 'text-red-600'
                      }`}>
                        {formatLatency(slo.latencyP99)}
                      </span>
                      <span className="text-gray-400 text-xs ml-1">
                        / {formatLatency(slo.latencyTarget)}
                      </span>
                    </td>
                    <td className="py-3">
                      <div className="flex items-center gap-2">
                        <div className="flex-1 h-2 bg-gray-200 rounded-full overflow-hidden">
                          <div
                            className={`h-full transition-all ${getErrorBudgetColor(slo.errorBudgetRemaining)}`}
                            style={{ width: `${Math.min(slo.errorBudgetRemaining, 100)}%` }}
                          />
                        </div>
                        <span className="text-xs font-mono w-12 text-right">
                          {formatPercentage(slo.errorBudgetRemaining, 0)}
                        </span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Burn Rates */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Error Budget Burn Rate</h2>
        <p className="text-sm text-gray-500 mb-4">
          Burn rate indicates how fast error budget is being consumed. Rate &gt; 1x means faster than sustainable.
        </p>
        {burnRates.length === 0 ? (
          <div className="text-center py-8 text-gray-500">No burn rate data available</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {burnRates.map((br: BurnRate) => (
              <div
                key={br.serviceName}
                className={`p-4 rounded-lg border-2 ${
                  br.alerting ? 'border-red-500 bg-red-50' : 'border-gray-200 bg-white'
                }`}
              >
                <div className="flex items-center justify-between mb-3">
                  <h3 className="font-semibold text-gray-900">{br.serviceName}</h3>
                  {br.alerting && (
                    <AlertTriangle className="w-5 h-5 text-red-500" />
                  )}
                </div>
                <div className="grid grid-cols-3 gap-2 text-center">
                  <div>
                    <p className="text-xs text-gray-500">1h</p>
                    <p className={`font-mono font-bold ${getBurnRateColor(br.rate1h)}`}>
                      {formatBurnRate(br.rate1h)}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-gray-500">6h</p>
                    <p className={`font-mono font-bold ${getBurnRateColor(br.rate6h)}`}>
                      {formatBurnRate(br.rate6h)}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-gray-500">24h</p>
                    <p className={`font-mono font-bold ${getBurnRateColor(br.rate24h)}`}>
                      {formatBurnRate(br.rate24h)}
                    </p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
