import { useState } from 'react'

interface User {
  id: number
  first_name: string
  last_name: string
  email: string
}

interface Props {
  isNew: boolean
  users: User[]
  linkedParents: User[]
  onAddParent: (userId: number) => Promise<void>
  onRemoveParent: (userId: number) => Promise<void>
  saving: boolean
  saved?: boolean
  error: string
}

export default function MemberFamilieTab({ isNew, users, linkedParents, onAddParent, onRemoveParent, saving, saved, error }: Props) {
  const [selectedParent, setSelectedParent] = useState('')
  const [removing, setRemoving] = useState<Record<number, boolean>>({})

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

  if (isNew) {
    return <div className="text-gray-600">Familie kann nach dem Erstellen hinzugefügt werden.</div>
  }

  return (
    <div className="space-y-6">
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Erziehungsberechtigte</h2>

        {linkedParents.length > 0 && (
          <div className="space-y-2 mb-6">
            {linkedParents.map(parent => (
              <div key={parent.id} className="flex items-center justify-between border border-gray-200 rounded-lg p-3 text-sm">
                <div>
                  <span className="font-medium">{parent.first_name} {parent.last_name}</span>
                  <p className="text-xs text-gray-500">{parent.email}</p>
                </div>
                <button
                  onClick={() => handleRemove(parent.id)}
                  disabled={removing[parent.id]}
                  className="text-red-600 hover:text-red-800 disabled:opacity-40"
                >
                  Entfernen
                </button>
              </div>
            ))}
          </div>
        )}

        {canAddMore && availableUsers.length > 0 && (
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-700">Hinzufügen (max. 2)</label>
            <div className="flex gap-2">
              <select
                value={selectedParent}
                onChange={e => setSelectedParent(e.target.value)}
                className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm"
              >
                <option value="">– Nutzer wählen –</option>
                {availableUsers.map(u => (
                  <option key={u.id} value={u.id}>{u.first_name} {u.last_name} ({u.email})</option>
                ))}
              </select>
              <button
                onClick={handleAdd}
                disabled={!selectedParent || saving}
                className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow disabled:opacity-40"
              >
                Hinzufügen
              </button>
            </div>
          </div>
        )}

        {error && <p className="text-sm text-red-600 mt-4">{error}</p>}
        {saved && <p className="text-sm text-green-600 mt-4">Gespeichert</p>}
      </div>
    </div>
  )
}
