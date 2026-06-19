import { useState } from 'react'
import { X, LogOut, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'
import { errorMessage } from '../lib/errors'

interface ConvMember { id: number; name: string }

type Choice = 'transfer' | 'delete'

interface Props {
  convId: number
  ownerId: number
  members: ConvMember[]
  onClose: () => void
  onDone: () => void
}

export default function CreatorExitChoiceModal({ convId, ownerId, members, onClose, onDone }: Props) {
  useEscapeKey(onClose)

  const candidates = members.filter(m => m.id !== ownerId)
  const [choice, setChoice] = useState<Choice>(candidates.length > 0 ? 'transfer' : 'delete')
  const [newOwnerId, setNewOwnerId] = useState<number>(candidates[0]?.id ?? 0)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function confirm() {
    setError('')
    setBusy(true)
    try {
      if (choice === 'transfer') {
        if (!newOwnerId) {
          setError('Bitte ein Mitglied auswählen.')
          setBusy(false)
          return
        }
        await api.post(`/chat/conversations/${convId}/transfer-ownership`, { newOwnerId })
        await api.delete(`/chat/conversations/${convId}/members/me`)
      } else {
        if (!confirm_(`Diese Aktion löscht alle Nachrichten endgültig. Fortfahren?`)) {
          setBusy(false)
          return
        }
        await api.delete(`/chat/conversations/${convId}/everyone`)
      }
      onDone()
    } catch (e) {
      setError(errorMessage(e, 'Fehler beim Verarbeiten'))
      setBusy(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">Gruppe verlassen</h2>
          <button onClick={onClose} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Schließen">
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        <p className="text-sm text-brand-text-muted mb-4">
          Du bist der Ersteller dieser Gruppe. Bevor du verlässt, entscheide:
        </p>

        <div className="space-y-3">
          <label className={`flex items-start gap-3 p-3 rounded-md border cursor-pointer transition-colors ${choice === 'transfer' ? 'border-brand-yellow bg-brand-yellow/10' : 'border-brand-border hover:bg-brand-table-select'} ${candidates.length === 0 ? 'opacity-50 cursor-not-allowed' : ''}`}>
            <input
              type="radio"
              name="exit-choice"
              value="transfer"
              checked={choice === 'transfer'}
              onChange={() => setChoice('transfer')}
              disabled={candidates.length === 0}
              className="mt-1"
            />
            <div className="flex-1">
              <div className="flex items-center gap-2 text-sm font-medium text-brand-text">
                <LogOut className="w-4 h-4" />
                Verwaltung übergeben an…
              </div>
              <p className="text-xs text-brand-text-muted mt-1">
                Ein anderes Mitglied wird neuer Ersteller. Du verlässt die Gruppe danach.
              </p>
              {choice === 'transfer' && candidates.length > 0 && (
                <select
                  value={newOwnerId}
                  onChange={e => setNewOwnerId(Number(e.target.value))}
                  className="mt-2 w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                >
                  {candidates.map(m => (
                    <option key={m.id} value={m.id}>{m.name}</option>
                  ))}
                </select>
              )}
              {candidates.length === 0 && (
                <p className="text-xs text-brand-danger mt-1">Keine weiteren Mitglieder vorhanden.</p>
              )}
            </div>
          </label>

          <label className={`flex items-start gap-3 p-3 rounded-md border cursor-pointer transition-colors ${choice === 'delete' ? 'border-brand-yellow bg-brand-yellow/10' : 'border-brand-border hover:bg-brand-table-select'}`}>
            <input
              type="radio"
              name="exit-choice"
              value="delete"
              checked={choice === 'delete'}
              onChange={() => setChoice('delete')}
              className="mt-1"
            />
            <div className="flex-1">
              <div className="flex items-center gap-2 text-sm font-medium text-brand-text">
                <Trash2 className="w-4 h-4 text-brand-danger" />
                Gruppe für alle löschen
              </div>
              <p className="text-xs text-brand-text-muted mt-1">
                Alle Nachrichten werden endgültig entfernt. Diese Aktion kann nicht rückgängig gemacht werden.
              </p>
            </div>
          </label>
        </div>

        {error && <p className="text-sm text-brand-danger mt-3">{error}</p>}

        <div className="pt-5 flex gap-2">
          <button
            onClick={onClose}
            disabled={busy}
            className="flex-1 border border-brand-border rounded-md px-4 py-2.5 sm:py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors disabled:opacity-40"
          >
            Abbrechen
          </button>
          <button
            onClick={confirm}
            disabled={busy || (choice === 'transfer' && candidates.length === 0)}
            className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {busy ? 'Verarbeite…' : 'Bestätigen'}
          </button>
        </div>
      </div>
    </div>
  )
}

// Inline alias for window.confirm to keep the name distinct from the `confirm` function scope.
function confirm_(msg: string): boolean {
  return window.confirm(msg)
}
