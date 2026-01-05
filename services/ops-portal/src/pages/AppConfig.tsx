import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getAllConfigs, createConfig, updateConfig, deleteConfig } from '../api/config'
import { useAuth } from '../contexts/AuthContext'
import { Plus, Pencil, Trash2, X } from 'lucide-react'
import type { AppConfig } from '../types'

export default function AppConfigPage() {
  const queryClient = useQueryClient()
  const { hasRole } = useAuth()
  const isAdmin = hasRole(['admin'])

  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingConfig, setEditingConfig] = useState<AppConfig | null>(null)
  const [formData, setFormData] = useState({
    key: '',
    value: '',
    description: '',
    isActive: true,
  })

  const { data: configs = [], isLoading } = useQuery({
    queryKey: ['configs'],
    queryFn: getAllConfigs,
  })

  const createMutation = useMutation({
    mutationFn: () => createConfig(formData.key, formData.value, formData.description),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['configs'] })
      closeModal()
    },
  })

  const updateMutation = useMutation({
    mutationFn: (id: string) =>
      updateConfig(id, formData.value, formData.description, formData.isActive),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['configs'] })
      closeModal()
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['configs'] })
    },
  })

  const openCreateModal = () => {
    setEditingConfig(null)
    setFormData({ key: '', value: '', description: '', isActive: true })
    setIsModalOpen(true)
  }

  const openEditModal = (config: AppConfig) => {
    setEditingConfig(config)
    setFormData({
      key: config.key,
      value: config.value,
      description: config.description || '',
      isActive: config.isActive,
    })
    setIsModalOpen(true)
  }

  const closeModal = () => {
    setIsModalOpen(false)
    setEditingConfig(null)
    setFormData({ key: '', value: '', description: '', isActive: true })
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (editingConfig) {
      updateMutation.mutate(editingConfig.id)
    } else {
      createMutation.mutate()
    }
  }

  const handleDelete = (id: string, key: string) => {
    if (window.confirm(`Are you sure you want to delete "${key}"?`)) {
      deleteMutation.mutate(id)
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">App Configuration</h1>
          <p className="text-gray-600 mt-1">Manage application configuration values</p>
        </div>
        <button
          onClick={openCreateModal}
          className="btn btn-primary flex items-center gap-2"
        >
          <Plus size={20} />
          Add Config
        </button>
      </div>

      <div className="card">
        {isLoading ? (
          <div className="animate-pulse space-y-4">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-16 bg-gray-100 rounded" />
            ))}
          </div>
        ) : configs.length === 0 ? (
          <p className="text-gray-500 text-center py-8">No configurations found</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="text-left text-sm text-gray-500 border-b">
                  <th className="pb-3 font-medium">Key</th>
                  <th className="pb-3 font-medium">Value</th>
                  <th className="pb-3 font-medium">Description</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium">Updated</th>
                  <th className="pb-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {configs.map((config) => (
                  <tr key={config.id} className="border-b last:border-0">
                    <td className="py-4 font-mono text-sm font-medium">
                      {config.key}
                    </td>
                    <td className="py-4">
                      <code className="text-sm bg-gray-100 px-2 py-1 rounded">
                        {config.value.length > 50
                          ? `${config.value.slice(0, 50)}...`
                          : config.value}
                      </code>
                    </td>
                    <td className="py-4 text-sm text-gray-600 max-w-xs truncate">
                      {config.description || '-'}
                    </td>
                    <td className="py-4">
                      <span
                        className={`badge ${
                          config.isActive ? 'badge-success' : 'badge-danger'
                        }`}
                      >
                        {config.isActive ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="py-4 text-sm text-gray-600">
                      {new Date(config.updatedAt).toLocaleDateString()}
                    </td>
                    <td className="py-4">
                      <div className="flex gap-2">
                        <button
                          onClick={() => openEditModal(config)}
                          className="p-2 text-gray-500 hover:text-primary-600 hover:bg-gray-100 rounded"
                        >
                          <Pencil size={16} />
                        </button>
                        {isAdmin && (
                          <button
                            onClick={() => handleDelete(config.id, config.key)}
                            className="p-2 text-gray-500 hover:text-red-600 hover:bg-gray-100 rounded"
                          >
                            <Trash2 size={16} />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-lg">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-bold text-gray-900">
                {editingConfig ? 'Edit Configuration' : 'Add Configuration'}
              </h2>
              <button onClick={closeModal} className="text-gray-500 hover:text-gray-700">
                <X size={20} />
              </button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Key
                </label>
                <input
                  type="text"
                  value={formData.key}
                  onChange={(e) => setFormData({ ...formData, key: e.target.value })}
                  className="input font-mono"
                  placeholder="FEATURE_FLAG_NAME"
                  disabled={!!editingConfig}
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Value
                </label>
                <textarea
                  value={formData.value}
                  onChange={(e) => setFormData({ ...formData, value: e.target.value })}
                  className="input font-mono"
                  rows={3}
                  placeholder="Configuration value"
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Description
                </label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  className="input"
                  placeholder="Optional description"
                />
              </div>
              {editingConfig && (
                <div className="mb-6">
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={formData.isActive}
                      onChange={(e) =>
                        setFormData({ ...formData, isActive: e.target.checked })
                      }
                      className="w-4 h-4 text-primary-600 rounded"
                    />
                    <span className="text-sm text-gray-700">Active</span>
                  </label>
                </div>
              )}
              <div className="flex gap-3">
                <button
                  type="button"
                  onClick={closeModal}
                  className="btn btn-secondary flex-1"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending || updateMutation.isPending}
                  className="btn btn-primary flex-1"
                >
                  {createMutation.isPending || updateMutation.isPending
                    ? 'Saving...'
                    : 'Save'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
