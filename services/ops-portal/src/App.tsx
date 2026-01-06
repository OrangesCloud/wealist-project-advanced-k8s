import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './contexts/AuthContext'
import MainLayout from './layouts/MainLayout'
import Dashboard from './pages/Dashboard'
import Login from './pages/Login'
import Users from './pages/Users'
import AuditLogs from './pages/AuditLogs'
import Monitoring from './pages/Monitoring'
import ErrorTracker from './pages/ErrorTracker'
import SLODashboard from './pages/SLODashboard'
import DeploymentHistory from './pages/DeploymentHistory'
import LogsViewer from './pages/LogsViewer'
import AppConfig from './pages/AppConfig'
import ArgoCDRBAC from './pages/ArgoCDRBAC'
import type { Role } from './types'

interface ProtectedRouteProps {
  children: React.ReactNode
  roles?: Role[]
}

function ProtectedRoute({ children, roles }: ProtectedRouteProps) {
  const { isAuthenticated, isLoading, hasRole } = useAuth()

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600" />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (roles && !hasRole(roles)) {
    return <Navigate to="/" replace />
  }

  return <MainLayout>{children}</MainLayout>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <Dashboard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/users"
        element={
          <ProtectedRoute roles={['admin']}>
            <Users />
          </ProtectedRoute>
        }
      />
      <Route
        path="/audit"
        element={
          <ProtectedRoute roles={['admin']}>
            <AuditLogs />
          </ProtectedRoute>
        }
      />
      <Route
        path="/monitoring"
        element={
          <ProtectedRoute>
            <Monitoring />
          </ProtectedRoute>
        }
      />
      <Route
        path="/errors"
        element={
          <ProtectedRoute>
            <ErrorTracker />
          </ProtectedRoute>
        }
      />
      <Route
        path="/slo"
        element={
          <ProtectedRoute>
            <SLODashboard />
          </ProtectedRoute>
        }
      />
      <Route
        path="/deployments"
        element={
          <ProtectedRoute>
            <DeploymentHistory />
          </ProtectedRoute>
        }
      />
      <Route
        path="/logs"
        element={
          <ProtectedRoute>
            <LogsViewer />
          </ProtectedRoute>
        }
      />
      <Route
        path="/config"
        element={
          <ProtectedRoute roles={['admin', 'pm']}>
            <AppConfig />
          </ProtectedRoute>
        }
      />
      <Route
        path="/argocd-rbac"
        element={
          <ProtectedRoute roles={['admin']}>
            <ArgoCDRBAC />
          </ProtectedRoute>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
