import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { getDeploymentHistory, getApplicationDeploymentHistory } from '../api/deployments'
import { getApplications } from '../api/monitoring'
import { RefreshCw, GitBranch, Clock, CheckCircle, AlertTriangle, Filter } from 'lucide-react'
import type { DeploymentHistoryEntry } from '../types'

export default function DeploymentHistory() {
  const [selectedApp, setSelectedApp] = useState<string>('')

  const { data: applications = [] } = useQuery({
    queryKey: ['applications'],
    queryFn: getApplications,
  })

  const { data: history = [], isLoading, refetch, isFetching } = useQuery({
    queryKey: ['deploymentHistory', selectedApp],
    queryFn: () => selectedApp
      ? getApplicationDeploymentHistory(selectedApp)
      : getDeploymentHistory(),
    refetchInterval: 60000,
  })

  const handleRefresh = () => {
    refetch()
  }

  const formatDate = (dateStr: string) => {
    if (!dateStr) return '-'
    const date = new Date(dateStr)
    return date.toLocaleString('ko-KR', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const formatRelativeTime = (dateStr: string) => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)

    if (diffMins < 1) return 'just now'
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    return `${diffDays}d ago`
  }

  const shortenRevision = (rev: string) => {
    if (!rev) return '-'
    return rev.substring(0, 7)
  }

  const getHealthBadge = (health: string) => {
    const badges: Record<string, string> = {
      Healthy: 'bg-green-100 text-green-700',
      Degraded: 'bg-yellow-100 text-yellow-700',
      Progressing: 'bg-blue-100 text-blue-700',
      Missing: 'bg-gray-100 text-gray-700',
    }
    return badges[health] || 'bg-red-100 text-red-700'
  }

  const getSyncBadge = (sync: string) => {
    const badges: Record<string, string> = {
      Synced: 'bg-green-100 text-green-700',
      OutOfSync: 'bg-yellow-100 text-yellow-700',
      Unknown: 'bg-gray-100 text-gray-700',
    }
    return badges[sync] || 'bg-gray-100 text-gray-700'
  }

  // Group history by date for timeline view
  const groupedHistory = history.reduce((acc: Record<string, DeploymentHistoryEntry[]>, entry) => {
    const date = entry.deployedAt ? new Date(entry.deployedAt).toLocaleDateString('ko-KR') : 'Unknown'
    if (!acc[date]) acc[date] = []
    acc[date].push(entry)
    return acc
  }, {})

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Deployment History</h1>
          <p className="text-gray-600 mt-1">Track ArgoCD application deployments</p>
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
      <div className="card mb-8">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Filter size={16} className="text-gray-500" />
            <span className="text-sm text-gray-600">Filter by Application:</span>
          </div>
          <select
            value={selectedApp}
            onChange={(e) => setSelectedApp(e.target.value)}
            className="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="">All Applications</option>
            {applications.map((app) => (
              <option key={app.name} value={app.name}>{app.name}</option>
            ))}
          </select>
          {selectedApp && (
            <button
              onClick={() => setSelectedApp('')}
              className="text-sm text-primary-600 hover:text-primary-700"
            >
              Clear
            </button>
          )}
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-100 rounded-lg">
              <GitBranch className="w-5 h-5 text-blue-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Total Deployments</p>
              <p className="text-2xl font-bold text-gray-900">{history.length}</p>
            </div>
          </div>
        </div>
        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-green-100 rounded-lg">
              <CheckCircle className="w-5 h-5 text-green-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Synced</p>
              <p className="text-2xl font-bold text-green-600">
                {history.filter(h => h.syncStatus === 'Synced').length}
              </p>
            </div>
          </div>
        </div>
        <div className="card">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-purple-100 rounded-lg">
              <Clock className="w-5 h-5 text-purple-600" />
            </div>
            <div>
              <p className="text-sm text-gray-600">Latest Deployment</p>
              <p className="text-lg font-bold text-gray-900">
                {history[0] ? formatRelativeTime(history[0].deployedAt) : '-'}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Timeline View */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">Deployment Timeline</h2>
        {isLoading ? (
          <div className="space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-16 bg-gray-100 rounded animate-pulse" />
            ))}
          </div>
        ) : history.length === 0 ? (
          <div className="text-center py-12">
            <GitBranch className="w-12 h-12 text-gray-400 mx-auto mb-3" />
            <p className="text-gray-500">No deployment history available</p>
            {!selectedApp && (
              <p className="text-sm text-gray-400 mt-1">ArgoCD may not be configured</p>
            )}
          </div>
        ) : (
          <div className="space-y-6">
            {Object.entries(groupedHistory).map(([date, entries]) => (
              <div key={date}>
                <div className="flex items-center gap-2 mb-3">
                  <div className="w-2 h-2 bg-primary-500 rounded-full" />
                  <span className="text-sm font-medium text-gray-600">{date}</span>
                </div>
                <div className="ml-4 space-y-2">
                  {entries.map((entry: DeploymentHistoryEntry, idx: number) => (
                    <div
                      key={`${entry.appName}-${entry.revision}-${idx}`}
                      className="flex items-center justify-between p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors"
                    >
                      <div className="flex items-center gap-4">
                        <div className="flex items-center gap-2">
                          {entry.healthStatus === 'Healthy' ? (
                            <CheckCircle className="w-5 h-5 text-green-500" />
                          ) : (
                            <AlertTriangle className="w-5 h-5 text-yellow-500" />
                          )}
                        </div>
                        <div>
                          <p className="font-medium text-gray-900">{entry.appName}</p>
                          <p className="text-sm text-gray-500">
                            <code className="bg-gray-200 px-1 rounded text-xs">
                              {shortenRevision(entry.revision)}
                            </code>
                            <span className="mx-2">·</span>
                            {formatRelativeTime(entry.deployedAt)}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className={`px-2 py-1 rounded text-xs font-medium ${getHealthBadge(entry.healthStatus)}`}>
                          {entry.healthStatus}
                        </span>
                        <span className={`px-2 py-1 rounded text-xs font-medium ${getSyncBadge(entry.syncStatus)}`}>
                          {entry.syncStatus}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Full Table View */}
      {history.length > 0 && (
        <div className="card mt-8">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Detailed History</h2>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-gray-500 border-b">
                  <th className="pb-3 font-medium">Application</th>
                  <th className="pb-3 font-medium">Revision</th>
                  <th className="pb-3 font-medium">Deployed At</th>
                  <th className="pb-3 font-medium text-center">Health</th>
                  <th className="pb-3 font-medium text-center">Sync</th>
                </tr>
              </thead>
              <tbody>
                {history.slice(0, 50).map((entry, idx) => (
                  <tr key={`${entry.appName}-${entry.revision}-${idx}`} className="border-b last:border-0 hover:bg-gray-50">
                    <td className="py-3 font-medium text-gray-900">{entry.appName}</td>
                    <td className="py-3">
                      <code className="bg-gray-100 px-2 py-1 rounded text-sm">
                        {shortenRevision(entry.revision)}
                      </code>
                    </td>
                    <td className="py-3 text-sm text-gray-600">{formatDate(entry.deployedAt)}</td>
                    <td className="py-3 text-center">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${getHealthBadge(entry.healthStatus)}`}>
                        {entry.healthStatus}
                      </span>
                    </td>
                    <td className="py-3 text-center">
                      <span className={`px-2 py-1 rounded text-xs font-medium ${getSyncBadge(entry.syncStatus)}`}>
                        {entry.syncStatus}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  )
}
