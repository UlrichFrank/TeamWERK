import { useState } from 'react'
import { api } from '../lib/api'
import { BTN_PRIMARY, INPUT, LABEL } from '../lib/buttonStyles'
import { errorMessage } from '../lib/errors'
import { formatDuration, parseDuration } from '../lib/time'
import type { MeetingFields } from '../lib/meeting'

type GameInfoEditorProps = {
  gameId: number
  initialNote: string
  initialOffset: number | null | undefined
  initialPlace: string | undefined
  onMeetingSaved?: (m: MeetingFields) => void
}

const MAX_NOTE = 200
const MAX_PLACE = 100
const MAX_OFFSET = 720

function offsetToInput(minutes: number | null | undefined): string {
  return minutes == null ? '' : formatDuration(minutes)
}

/** Minuten aus der Eingabe („1h 30min“, „90“); null = keine Treffzeit, NaN = nicht lesbar. */
function inputToOffset(s: string): number | null {
  if (s.trim() === '') return null
  return /\d/.test(s) ? parseDuration(s) : NaN
}

/**
 * Hinweis und Treffzeit eines Spiels mit einem gemeinsamen Speichern
 * (Kalender-Dialog). Die Treffzeit wird als Abstand in Minuten vor dem Anwurf
 * gepflegt (Format wie die Versätze der Diensttypen: „1h 30min“) — genau so
 * speichert sie der Server, deshalb wandert sie bei einer
 * Verlegung mit. Ruft nur die Routen, deren Wert sich geändert hat:
 * `PUT /api/games/{id}/note` und `PUT /api/games/{id}/meeting`.
 */
export default function GameInfoEditor({ gameId, initialNote, initialOffset, initialPlace, onMeetingSaved }: GameInfoEditorProps) {
  const [savedNote, setSavedNote] = useState(initialNote)
  const [savedOffset, setSavedOffset] = useState<number | null>(initialOffset ?? null)
  const [savedPlace, setSavedPlace] = useState(initialPlace ?? '')
  const [note, setNote] = useState(savedNote)
  const [offset, setOffset] = useState(offsetToInput(savedOffset))
  const [place, setPlace] = useState(savedPlace)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const noteChanged = note !== savedNote
  const offsetMin = inputToOffset(offset)
  const meetingChanged = offsetMin !== savedOffset || place.trim() !== savedPlace
  const noteTooLong = note.length > MAX_NOTE
  const offsetInvalid = offsetMin !== null && (Number.isNaN(offsetMin) || offsetMin > MAX_OFFSET)
  const placeWithoutTime = offsetMin === null && place.trim() !== ''
  const blocked = noteTooLong || offsetInvalid || placeWithoutTime

  async function save() {
    setSaving(true)
    setError('')
    try {
      if (noteChanged) {
        await api.put(`/games/${gameId}/note`, { note })
        setSavedNote(note)
      }
      if (meetingChanged) {
        const res = await api.put<MeetingFields>(`/games/${gameId}/meeting`, {
          meet_offset_minutes: offsetMin,
          meet_place: place.trim(),
        })
        const m = res.data
        setSavedOffset(m.meet_offset_minutes ?? null)
        setOffset(offsetToInput(m.meet_offset_minutes))
        setSavedPlace(m.meet_place)
        setPlace(m.meet_place)
        onMeetingSaved?.(m)
      }
    } catch (e) {
      setError(errorMessage(e, 'Speichern fehlgeschlagen.'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-2">
      <div>
        <label className={LABEL} htmlFor={`game-note-${gameId}`}>Hinweis</label>
        <textarea
          id={`game-note-${gameId}`}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          rows={2}
          placeholder="Hinweis für die Mannschaft (z. B. Halle gesperrt oder Link zum Turnierplan aus Dokumente)"
          className={INPUT}
        />
        <p className={`text-xs text-right ${noteTooLong ? 'text-brand-danger' : 'text-brand-text-muted'}`}>
          {note.length}/{MAX_NOTE}
        </p>
      </div>
      <div className="flex gap-2">
        <div className="shrink-0">
          <label className={LABEL} htmlFor={`meet-offset-${gameId}`}>Treffen vor Anwurf</label>
          <input
            id={`meet-offset-${gameId}`}
            type="text"
            value={offset}
            onChange={(e) => setOffset(e.target.value)}
            onBlur={() => {
              const m = inputToOffset(offset)
              if (m !== null && !Number.isNaN(m)) setOffset(formatDuration(m))
            }}
            placeholder="z. B. 1h 30min"
            className={`${INPUT} w-32 sm:w-40`}
          />
        </div>
        <div className="flex-1 min-w-0">
          <label className={LABEL} htmlFor={`meet-place-${gameId}`}>Treffpunkt</label>
          <input
            id={`meet-place-${gameId}`}
            type="text"
            value={place}
            maxLength={MAX_PLACE}
            onChange={(e) => setPlace(e.target.value)}
            placeholder="optional"
            className={INPUT}
          />
        </div>
      </div>
      {offsetInvalid && (
        <p className="text-xs text-brand-danger">Bitte eine Dauer bis 12h angeben (z. B. 45min oder 1h 30min).</p>
      )}
      {placeWithoutTime && (
        <p className="text-xs text-brand-text-muted">Für einen Treffpunkt bitte auch eine Treffzeit angeben.</p>
      )}
      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
          {error}
        </div>
      )}
      <div className="flex justify-end">
        <button
          type="button"
          onClick={save}
          disabled={(!noteChanged && !meetingChanged) || blocked || saving}
          className={BTN_PRIMARY}
        >
          {saving ? 'Speichern…' : 'Speichern'}
        </button>
      </div>
    </div>
  )
}
