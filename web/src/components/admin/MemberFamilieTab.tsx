import { useState } from 'react'
import { api } from '../../lib/api'
import { errorStatus } from '../../lib/errors'

interface User {
  id: number
  first_name: string
  last_name: string
  email: string
}

interface Props {
  isNew: boolean
  memberId?: number
  memberUserId?: number | null
  users: User[]
  linkedParents: User[]
  onAddParent: (userId: number) => Promise<void>
  onRemoveParent: (userId: number) => Promise<void>
  onReload?: () => void
  saving: boolean
  saved?: boolean
  error: string
}

export default function MemberFamilieTab({
  isNew, memberId, memberUserId, users, linkedParents,
  onAddParent, onRemoveParent, onReload, saving, saved, error
}: Props) {
  const [selectedParent, setSelectedParent] = useState('')
  const [removing, setRemoving] = useState<Record<number, boolean>>({})
  const [proxyLoading, setProxyLoading] = useState(false)
  const [proxyError, setProxyError] = useState('')

  const availableUsers = users.filter(u => !linkedParents.find(p => p.id === u.id))
  const canAddMore = linkedParents.length < 2

  const handleAdd = async () => {
    if (!selectedParent) return
    await onAddParent(Number(selectedParent))
    setSelectedParent('')
  }

  const handleRemove = async (userId: number) => {
    setRemoving(p => ({ ...p, [userId]: true }))
    try {
      await onRemoveParent(userId)
    } finally {
      setRemoving(p => ({ ...p, [userId]: false }))
    }
  }

  const handleCreateProxyAccount = async () => {
    if (!memberId) return
    setProxyLoading(true)
    setProxyError('')
    try {
      await api.post(`/members/${memberId}/proxy-account`)
      onReload?.()
    } catch (e) {
      if (errorStatus(e) === 409) {
        setProxyError('Mitglied hat bereits einen Account.')
      } else {
        setProxyError('Fehler beim Anlegen des Proxy-Accounts.')
      }
    } finally {
      setProxyLoading(false)
    }
  }

  if (isNew) {
    return <div className="text-gray-600">Familie kann nach dem Erstellen hinzugefügt werden.</div>
  }

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Erziehungsberechtigte</h2>

        {linkedParents.length > 0 && (
          <div className="space-y-2 mb-6">
            {linkedParents.map(parent => (
              <div key={parent.id} className="flex items-center justify-between border border-brand-border-subtle rounded-lg p-3 text-sm">
                <div>
                  <span className="font-medium text-brand-text">{parent.first_name} {parent.last_name}</span>
                  <p className="text-xs text-brand-text-muted">{parent.email}</p>
                </div>
                <button
                  onClick={() => handleRemove(parent.id)}
                  disabled={removing[parent.id]}
                  className="text-brand-danger hover:text-brand-danger/80 disabled:opacity-40 text-sm"
                >
                  Entfernen
                </button>
              </div>
            ))}
          </div>
        )}

        {canAddMore && availableUsers.length > 0 && (
          <div className="space-y-2">
            <label className="block text-sm font-medium text-brand-text">Hinzufügen (max. 2)</label>
            <div className="flex gap-2">
              <select
                value={selectedParent}
                onChange={e => setSelectedParent(e.target.value)}
                className="flex-1 border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text"
              >
                <option value="">– Nutzer wählen –</option>
                {availableUsers.map(u => (
                  <option key={u.id} value={u.id}>{u.first_name} {u.last_name} ({u.email})</option>
                ))}
              </select>
              <button
                onClick={handleAdd}
                disabled={!selectedParent || saving}
                className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow disabled:opacity-40"
              >
                Hinzufügen
              </button>
            </div>
          </div>
        )}

        {error && <p className="text-sm text-brand-danger mt-4">{error}</p>}
        {saved && <p className="text-sm text-green-600 mt-4">Gespeichert</p>}
      </div>

      {memberId && memberUserId == null && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-brand-text-muted mb-2">Proxy-Account</h2>
          <p className="text-sm text-brand-text-muted mb-4">
            Dieses Mitglied hat keinen Nutzeraccount. Ein Proxy-Account ermöglicht die Zuordnung im Dienstsystem,
            ohne dass sich das Mitglied einloggen kann.
          </p>
          <button
            onClick={handleCreateProxyAccount}
            disabled={proxyLoading}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {proxyLoading ? 'Anlegen…' : 'Proxy-Account anlegen'}
          </button>
          {proxyError && <p className="text-sm text-brand-danger mt-3">{proxyError}</p>}
        </div>
      )}
    </div>
  )
}
