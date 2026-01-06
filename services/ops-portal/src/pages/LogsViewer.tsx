import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getLogs, getLogServices } from '../api/logs'
import { RefreshCw, Search, Terminal, Filter, Clock, AlertCircle, Info, AlertTriangle as WarnIcon } from 'lucide-react'
import type { LogEntry } from '../types'

type TimeRange = '15m' | '1h' | '3h' | '6h' | '24h' | 'custom'

export default function LogsViewer() {
  const [service, setService] = useState<string>('')
  const [level, setLevel] = useState<string>('')
  const [query, setQuery] = useState<string>('')
  const [searchInput, setSearchInput] = useState<string>('')
  const [timeRange, setTimeRange] = useState<TimeRange>('1h')
  const [limit, setLimit] = useState<number>(100)

  const getTimeRangeParams = () => {
    const now = new Date()
    const end = now.toISOString()
    let start: string

    switch (timeRange) {
      case '15m':
        start = new Date(now.getTime() - 15 * 60 * 1000).toISOString()
        break
      case '1h':
        start = new Date(now.getTime() - 60 * 60 * 1000).toISOString()
        break
      case '3h':
        start = new Date(now.getTime() - 3 * 60 * 60 * 1000).toISOString()
        break
      case '6h':
        start = new Date(now.getTime() - 6 * 60 * 60 * 1000).toISOString()
        break
      case '24h':
        start = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString()
        break
      default:
        start = new Date(now.getTime() - 60 * 60 * 1000).toISOString()
    }

    return { start, end }
  }

  const { data: services = [] } = useQuery({
    queryKey: ['logServices'],
    queryFn: getLogServices,
  })

  const { data: logsResult, isLoading, refetch, isFetching } = useQuery({
    queryKey: ['logs', service, level, query, timeRange, limit],
    queryFn: () => {
      const { start, end } = getTimeRangeParams()
      return getLogs({
        service: service || undefined,
        level: level || undefined,
        query: query || undefined,
        start,
        end,
        limit,
      })
    },
    refetchInterval: 30000,
  })

  const logs = logsResult?.entries || []

  const handleSearch = () => {
    setQuery(searchInput)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  const handleRefresh = () => {
    refetch()
  }

  const formatTimestamp = (ts: string) => {
    const date = new Date(ts)
    return date.toLocaleString('ko-KR', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const getLevelBadge = (lvl: string) => {
    const level = lvl?.toLowerCase() || 'info'
    const badges: Record<string, { bg: string; text: string; icon: React.ReactNode }> = {
      error: { bg: 'bg-red-100', text: 'text-red-700', icon: <AlertCircle size={12} /> },
      warn: { bg: 'bg-yellow-100', text: 'text-yellow-700', icon: <WarnIcon size={12} /> },
      warning: { bg: 'bg-yellow-100', text: 'text-yellow-700', icon: <WarnIcon size={12} /> },
      info: { bg: 'bg-blue-100', text: 'text-blue-700', icon: <Info size={12} /> },
      debug: { bg: 'bg-gray-100', text: 'text-gray-700', icon: <Terminal size={12} /> },
    }
    return badges[level] || badges.info
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Logs Viewer</h1>
          <p className="text-gray-600 mt-1">View and search application logs</p>
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

      {/* Filters */}
      <div className="card mb-6">
        <div className="flex flex-wrap items-center gap-4">
          {/* Service filter */}
          <div className="flex items-center gap-2">
            <Filter size={16} className="text-gray-500" />
            <select
              value={service}
              onChange={(e) => setService(e.target.value)}
              className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">All Services</option>
              {services.map((svc) => (
                <option key={svc.name} value={svc.name}>{svc.name}</option>
              ))}
            </select>
          </div>

          {/* Level filter */}
          <select
            value={level}
            onChange={(e) => setLevel(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="">All Levels</option>
            <option value="error">Error</option>
            <option value="warn">Warning</option>
            <option value="info">Info</option>
            <option value="debug">Debug</option>
          </select>

          {/* Time range */}
          <div className="flex items-center gap-2">
            <Clock size={16} className="text-gray-500" />
            <select
              value={timeRange}
              onChange={(e) => setTimeRange(e.target.value as TimeRange)}
              className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="15m">Last 15 minutes</option>
              <option value="1h">Last 1 hour</option>
              <option value="3h">Last 3 hours</option>
              <option value="6h">Last 6 hours</option>
              <option value="24h">Last 24 hours</option>
            </select>
          </div>

          {/* Limit */}
          <select
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="50">50 logs</option>
            <option value="100">100 logs</option>
            <option value="200">200 logs</option>
            <option value="500">500 logs</option>
            <option value="1000">1000 logs</option>
          </select>

          {/* Search */}
          <div className="flex-1 min-w-[200px]">
            <div className="relative">
              <input
                type="text"
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                onKeyPress={handleKeyPress}
                placeholder="Search logs..."
                className="w-full border border-gray-300 rounded-lg pl-10 pr-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
              <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            </div>
          </div>

          <button
            onClick={handleSearch}
            className="btn btn-primary"
          >
            Search
          </button>

          {(service || level || query) && (
            <button
              onClick={() => {
                setService('')
                setLevel('')
                setQuery('')
                setSearchInput('')
              }}
              className="text-sm text-primary-600 hover:text-primary-700"
            >
              Clear filters
            </button>
          )}
        </div>
      </div>

      {/* Summary */}
      <div className="card mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <Terminal className="w-5 h-5 text-gray-500" />
              <span className="text-sm text-gray-600">
                Showing <span className="font-semibold text-gray-900">{logs.length}</span> logs
              </span>
            </div>
            {query && (
              <span className="text-sm text-gray-500">
                matching "<span className="font-medium">{query}</span>"
              </span>
            )}
          </div>
          <div className="flex items-center gap-4 text-sm text-gray-500">
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 bg-red-500 rounded-full" />
              {logs.filter(l => l.level?.toLowerCase() === 'error').length} errors
            </span>
            <span className="flex items-center gap-1">
              <span className="w-2 h-2 bg-yellow-500 rounded-full" />
              {logs.filter(l => ['warn', 'warning'].includes(l.level?.toLowerCase())).length} warnings
            </span>
          </div>
        </div>
      </div>

      {/* Logs List */}
      <div className="card">
        {isLoading ? (
          <div className="space-y-3">
            {[...Array(10)].map((_, i) => (
              <div key={i} className="h-12 bg-gray-100 rounded animate-pulse" />
            ))}
          </div>
        ) : logs.length === 0 ? (
          <div className="text-center py-12">
            <Terminal className="w-12 h-12 text-gray-400 mx-auto mb-3" />
            <p className="text-gray-500">No logs found</p>
            <p className="text-sm text-gray-400 mt-1">
              {service || level || query
                ? 'Try adjusting your filters'
                : 'Loki may not be configured or there are no logs in this time range'}
            </p>
          </div>
        ) : (
          <div className="space-y-1">
            {logs.map((log: LogEntry, idx: number) => {
              const badge = getLevelBadge(log.level)
              return (
                <div
                  key={`${log.timestamp}-${idx}`}
                  className="flex items-start gap-3 p-2 hover:bg-gray-50 rounded font-mono text-sm"
                >
                  {/* Timestamp */}
                  <span className="text-gray-400 whitespace-nowrap text-xs">
                    {formatTimestamp(log.timestamp)}
                  </span>

                  {/* Level badge */}
                  <span className={`px-2 py-0.5 rounded text-xs font-medium flex items-center gap-1 ${badge.bg} ${badge.text}`}>
                    {badge.icon}
                    {log.level?.toUpperCase() || 'INFO'}
                  </span>

                  {/* Service */}
                  <span className="text-purple-600 whitespace-nowrap text-xs">
                    [{log.service}]
                  </span>

                  {/* Message */}
                  <span className="text-gray-800 break-all flex-1">
                    {log.message}
                  </span>

                  {/* Trace ID */}
                  {log.traceId && (
                    <span className="text-gray-400 text-xs whitespace-nowrap" title={`Trace: ${log.traceId}`}>
                      {log.traceId.substring(0, 8)}...
                    </span>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
