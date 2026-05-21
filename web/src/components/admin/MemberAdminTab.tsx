import { useState } from 'react'

interface User {
  id: number
  name: string
  email: string
}

interface Props {
  isNew: boolean
  users: User[]
  currentUserId: number | null
  onLinkUser: (userId: number | null) => Promise<void>
  saving: boolean
  saved: boolean
  error: string
}

export default function MemberAdminTab({ isNew, users, currentUserId, onLinkUser, saving, saved, error }: Props) {
  const [selectedUser, setSelectedUser] = useState<string>(currentUserId ? String(currentUserId) : '')

  const currentUser = currentUserId ? users.find(u => u.id === currentUserId) : null

  const handleSave = async () => {
    const userId = selectedUser ? Number(selectedUser) : null
    await onLinkUser(userId)
  }

  if (isNew) {
    return <div className="text-gray-600">Nutzer-Verknüpfung kann nach dem Erstellen vorgenommen werden.</div>
  }

  return (
    <div className="space-y-6">
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Nutzer verknüpfen</h2>

        {currentUser && (
          <div className="mb-6 p-3 bg-blue-50 border border-blue-200 rounded-lg text-sm">
            <p className="font-medium text-blue-900">Aktuell verknüpft:</p>
            <p className="text-blue-700">{currentUser.name} ({currentUser.email})</p>
          </div>
        )}

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Nutzer ändern</label>
          <select
            value={selectedUser}
            onChange={e => setSelectedUser(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm mb-3"
          >
            <option value="">– Keine Verknüpfung –</option>
            {users.map(u => (
              <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
            ))}
          </select>

          <button
            onClick={handleSave}
            disabled={saving}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow disabled:opacity-40"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
        </div>

        {saved && <p className="text-sm text-green-600 mt-3">Gespeichert</p>}
        {error && <p className="text-sm text-red-600 mt-3">{error}</p>}
      </div>
    </div>
  )
}
