import { useState } from 'react'
import { X, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'
import VenuePicker from './VenuePicker'

interface Game {
  id: number
  date: string
  time: string
  end_time?: string | null
  opponent: string
  event_type: string
  venue?: { id: number; name: string; street: string; city: string; postal_code: string; note: string } | null
}

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_SECONDARY = 'border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors'

interface Props {
  game: Game
  onClose: () => void
  onSaved: () => void
  onDeleted?: () => void
}

export default function GameEditModal({ game, onClose, onSaved, onDeleted }: Props) {
  const isGeneric = game.event_type === 'generisch'
  const [opponent, setOpponent] = useState(game.opponent)
  const [date, setDate] = useState(game.date.slice(0, 10))
  const [time, setTime] = useState(game.time)
  const [endTime, setEndTime] = useState(game.end_time ?? '')
  const [eventType, setEventType] = useState(game.event_type)
  const [venueId, setVenueId] = useState<number | null>(game.venue?.id ?? null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEscapeKey(onClose)

  const handleDelete = async () => {
    setDeleting(true)
    setError(null)
    try {
      await api.delete(`/admin/kalender/${game.id}?delete_slots=true`)
      onDeleted?.()
    } catch {
      setError('Löschen fehlgeschlagen.')
      setConfirmDelete(false)
    } finally {
      setDeleting(false)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      await api.put(`/admin/kalender/${game.id}`, {
        date,
        time,
        end_time: isGeneric ? (endTime || null) : null,
        opponent,
        event_type: eventType,
        venue_id: venueId,
      })
      onSaved()
    } catch {
      setError('Speichern fehlgeschlagen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">
            {isGeneric ? 'Event bearbeiten' : 'Spieltag bearbeiten'}
          </h2>
          <button onClick={onClose} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Schließen">
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">
              {isGeneric ? 'Event-Name' : 'Gegner'}
            </label>
            <input type="text" value={opponent} onChange={e => setOpponent(e.target.value)}
              placeholder={isGeneric ? 'Event-Name…' : 'Gegner…'} className={INPUT} />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Datum</label>
            <input type="date" value={date} onChange={e => setDate(e.target.value)} className={INPUT} />
          </div>
          <div className={isGeneric ? 'grid grid-cols-2 gap-3' : ''}>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">
                {isGeneric ? 'Beginn' : 'Uhrzeit'}
              </label>
              <input type="time" value={time} onChange={e => setTime(e.target.value)} className={INPUT} />
            </div>
            {isGeneric && (
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Ende</label>
                <input type="time" value={endTime} onChange={e => setEndTime(e.target.value)} className={INPUT} />
              </div>
            )}
          </div>
          {!isGeneric && (
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Typ</label>
              <select value={eventType} onChange={e => setEventType(e.target.value)} className={INPUT}>
                <option value="heim">Heimspiel</option>
                <option value="auswärts">Auswärtsspiel</option>
              </select>
            </div>
          )}
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Ort</label>
            <VenuePicker value={venueId} onChange={setVenueId} />
          </div>
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
        </div>

        {confirmDelete ? (
          <div className="mt-4 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg">
            <p className="text-sm text-brand-danger mb-3">Event und alle zugehörigen Dienst-Slots löschen?</p>
            <div className="flex gap-2">
              <button onClick={() => setConfirmDelete(false)} className={BTN_SECONDARY}>Abbrechen</button>
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="flex-1 bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {deleting ? 'Löschen…' : 'Ja, löschen'}
              </button>
            </div>
          </div>
        ) : (
          <div className="flex gap-2 pt-4">
            {onDeleted && (
              <button
                onClick={() => setConfirmDelete(true)}
                className="p-2 text-brand-text-muted hover:text-brand-danger hover:bg-brand-danger-light rounded-md transition-colors"
                aria-label="Event löschen"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}
            <button onClick={onClose} className={BTN_SECONDARY}>Abbrechen</button>
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {saving ? 'Speichern…' : 'Speichern'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
