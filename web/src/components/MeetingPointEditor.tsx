import { useState } from 'react'
import { api } from '../lib/api'
import { BTN_PRIMARY, BTN_SECONDARY, INPUT, LABEL } from '../lib/buttonStyles'
import { errorMessage } from '../lib/errors'
import type { MeetingFields } from '../lib/meeting'

type MeetingPointEditorProps = {
  gameId: number
  initialTime: string | null | undefined
  initialPlace: string | undefined
  onSaved?: (m: MeetingFields) => void
}

const MAX_PLACE = 100

/**
 * Inline-Editor für Treffzeit und Treffpunkt eines Spiels (spiel-treffpunkt).
 * Eingabe ist eine Uhrzeit am Spieltag; gespeichert wird serverseitig der
 * Abstand zum Anwurf, deshalb wandert die Treffzeit bei einer Verlegung mit.
 * Ruft `PUT /api/games/{id}/meeting`.
 */
export default function MeetingPointEditor({ gameId, initialTime, initialPlace, onSaved }: MeetingPointEditorProps) {
  const [savedTime, setSavedTime] = useState(initialTime ?? '')
  const [savedPlace, setSavedPlace] = useState(initialPlace ?? '')
  const [time, setTime] = useState(savedTime)
  const [place, setPlace] = useState(savedPlace)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const unchanged = time === savedTime && place.trim() === savedPlace
  const placeWithoutTime = time === '' && place.trim() !== ''

  async function submit(meetTime: string, meetPlace: string) {
    setSaving(true)
    setError('')
    try {
      const res = await api.put<MeetingFields>(`/games/${gameId}/meeting`, { meet_time: meetTime, meet_place: meetPlace })
      const m = res.data
      setSavedTime(m.meet_time ?? '')
      setSavedPlace(m.meet_place)
      setTime(m.meet_time ?? '')
      setPlace(m.meet_place)
      onSaved?.(m)
    } catch (e) {
      setError(errorMessage(e, 'Treffzeit konnte nicht gespeichert werden.'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-col sm:flex-row gap-2">
        <div className="sm:w-32 shrink-0">
          <label className={LABEL} htmlFor={`meet-time-${gameId}`}>Treffzeit</label>
          <input
            id={`meet-time-${gameId}`}
            type="time"
            value={time}
            onChange={(e) => setTime(e.target.value)}
            className={INPUT}
          />
        </div>
        <div className="flex-1 min-w-0">
          <label className={LABEL} htmlFor={`meet-place-${gameId}`}>Treffpunkt (optional)</label>
          <input
            id={`meet-place-${gameId}`}
            type="text"
            value={place}
            maxLength={MAX_PLACE}
            onChange={(e) => setPlace(e.target.value)}
            placeholder="z. B. Parkplatz Vereinsheim"
            className={INPUT}
          />
        </div>
      </div>
      <p className="text-xs text-brand-text-muted">Die Treffzeit verschiebt sich mit, wenn der Anwurf verlegt wird.</p>
      <div className="flex flex-wrap justify-end gap-2">
        {savedTime !== '' && (
          <button type="button" onClick={() => submit('', '')} disabled={saving} className={BTN_SECONDARY}>
            Entfernen
          </button>
        )}
        <button
          type="button"
          onClick={() => submit(time, place.trim())}
          disabled={unchanged || placeWithoutTime || saving}
          className={BTN_PRIMARY}
        >
          {saving ? 'Speichern…' : 'Speichern'}
        </button>
      </div>
      {placeWithoutTime && (
        <p className="text-xs text-brand-text-muted">Für einen Treffpunkt bitte auch eine Treffzeit angeben.</p>
      )}
      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
          {error}
        </div>
      )}
    </div>
  )
}
