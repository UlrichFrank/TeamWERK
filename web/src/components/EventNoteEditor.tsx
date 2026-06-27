import { useState } from 'react'
import { api } from '../lib/api'

type EventNoteEditorProps = {
  eventType: 'training' | 'game'
  eventId: number
  initialNote: string
  onSaved?: (newNote: string) => void
}

const MAX = 200

/**
 * Inline-Editor für den Termin-Hinweis: Textarea (max. 200 Zeichen) mit
 * Zeichen-Counter und Speichern-Button. Ruft `PUT /api/{trainings|games}/{id}/note`.
 */
export default function EventNoteEditor({ eventType, eventId, initialNote, onSaved }: EventNoteEditorProps) {
  const [note, setNote] = useState(initialNote)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const path = eventType === 'training' ? `/trainings/${eventId}/note` : `/games/${eventId}/note`
  const tooLong = note.length > MAX
  const unchanged = note === initialNote

  async function save() {
    if (tooLong || unchanged || saving) return
    setSaving(true)
    setError('')
    try {
      await api.put(path, { note })
      onSaved?.(note)
    } catch {
      setError('Hinweis konnte nicht gespeichert werden.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-2">
      <textarea
        value={note}
        onChange={(e) => setNote(e.target.value)}
        rows={3}
        placeholder="Hinweis für die Mannschaft (z. B. Halle gesperrt)"
        className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
      />
      <div className="flex items-center justify-between">
        <span className={`text-xs ${tooLong ? 'text-brand-danger' : 'text-brand-text-muted'}`}>
          {note.length}/{MAX}
        </span>
        <button
          type="button"
          onClick={save}
          disabled={tooLong || unchanged || saving}
          className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {saving ? 'Speichern…' : 'Speichern'}
        </button>
      </div>
      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
          {error}
        </div>
      )}
    </div>
  )
}
