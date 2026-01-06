import { ReactNode } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import {
  LayoutDashboard,
  Users,
  FileText,
  Settings,
  Activity,
  LogOut,
  Shield,
  ShieldCheck,
  AlertTriangle,
  Target,
  GitBranch,
  Terminal,
} from 'lucide-react'

interface NavItem {
  label: string
  path: string
  icon: ReactNode
  roles?: ('admin' | 'pm' | 'viewer')[]
}

const navItems: NavItem[] = [
  { label: 'Dashboard', path: '/', icon: <LayoutDashboard size={20} /> },
  { label: 'Users', path: '/users', icon: <Users size={20} />, roles: ['admin'] },
  { label: 'Audit Logs', path: '/audit', icon: <FileText size={20} />, roles: ['admin'] },
  { label: 'ArgoCD RBAC', path: '/argocd-rbac', icon: <ShieldCheck size={20} />, roles: ['admin'] },
  { label: 'Monitoring', path: '/monitoring', icon: <Activity size={20} /> },
  { label: 'SLO Dashboard', path: '/slo', icon: <Target size={20} /> },
  { label: 'Error Tracker', path: '/errors', icon: <AlertTriangle size={20} /> },
  { label: 'Deployments', path: '/deployments', icon: <GitBranch size={20} /> },
  { label: 'Logs', path: '/logs', icon: <Terminal size={20} /> },
  { label: 'App Config', path: '/config', icon: <Settings size={20} />, roles: ['admin', 'pm'] },
]

export default function MainLayout({ children }: { children: ReactNode }) {
  const { user, logout, hasRole } = useAuth()
  const location = useLocation()

  const filteredNavItems = navItems.filter(
    (item) => !item.roles || hasRole(item.roles)
  )

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Sidebar */}
      <aside className="fixed inset-y-0 left-0 w-64 bg-white border-r border-gray-200">
        <div className="flex items-center h-16 px-6 border-b border-gray-200">
          <Shield className="w-8 h-8 text-primary-600 mr-3" />
          <span className="text-xl font-bold text-gray-900">Admin Portal</span>
        </div>

        <nav className="p-4 space-y-1">
          {filteredNavItems.map((item) => {
            const isActive = location.pathname === item.path
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`flex items-center px-4 py-2.5 rounded-lg transition-colors ${
                  isActive
                    ? 'bg-primary-50 text-primary-700'
                    : 'text-gray-600 hover:bg-gray-100'
                }`}
              >
                {item.icon}
                <span className="ml-3">{item.label}</span>
              </Link>
            )
          })}
        </nav>

        {/* User info at bottom */}
        <div className="absolute bottom-0 left-0 right-0 p-4 border-t border-gray-200">
          <div className="flex items-center mb-3">
            {user?.picture ? (
              <img
                src={user.picture}
                alt={user.name}
                className="w-10 h-10 rounded-full"
              />
            ) : (
              <div className="w-10 h-10 rounded-full bg-primary-100 flex items-center justify-center">
                <span className="text-primary-700 font-medium">
                  {user?.name?.charAt(0) || user?.email?.charAt(0) || '?'}
                </span>
              </div>
            )}
            <div className="ml-3 flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-900 truncate">
                {user?.name || user?.email}
              </p>
              <p className="text-xs text-gray-500 capitalize">{user?.role}</p>
            </div>
          </div>
          <button
            onClick={logout}
            className="flex items-center w-full px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <LogOut size={18} />
            <span className="ml-2">Logout</span>
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="ml-64 min-h-screen">
        <div className="p-8">{children}</div>
      </main>
    </div>
  )
}
