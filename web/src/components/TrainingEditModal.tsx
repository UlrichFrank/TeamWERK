import { useState, useEffect } from 'react'
import { X, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import VenuePicker from './VenuePicker'
import RsvpDefaultsEditor, { type RsvpDefault } from './RsvpDefaultsEditor'

interface VenueRef { id: number; name: string; street: string; city: string; postal_code: string; note: string }

interface Training {
  id: number
  title: string
  date: string
  start_time: string
  end_time: string
  venue?: VenueRef | null
  status: 'active' | 'cancelled'
  note: string
  cancel_reason?: string
  series_id?: number
  team_id: number
  season_id: number
  rsvp_default_players?: RsvpDefault
  rsvp_default_extended?: RsvpDefault
  rsvp_require_reason?: number
}

interface Series {
  id: number
  name: string
  venue_id?: number | null
  day_of_week: number
  start_time: string
  end_time: string
  valid_from: string
  valid_until: string
  note: string
  rsvp_default_players: RsvpDefault
  rsvp_default_extended: RsvpDefault
  rsvp_require_reason: number
}

type Scope = 'this_one' | 'this_and_following' | 'all'

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_SECONDARY = 'border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors'

interface Props {
  session: Training
  teamName?: string
  onClose: () => void
  onSaved: () => void
}

export default function TrainingEditModal({ session, teamName, onClose, onSaved }: Props) {
  const [title, setTitle] = useState(session.title)
  const [date, setDate] = useState(session.date.slice(0, 10))
  const [startTime, setStartTime] = useState(session.start_time)
  const [endTime, setEndTime] = useState(session.end_time)
  const [venueId, setVenueId] = useState<number | null>(session.venue?.id ?? null)
  const [rsvpDefaultPlayers, setRsvpDefaultPlayers] = useState<RsvpDefault>(session.rsvp_default_players ?? 'none')
  const [rsvpDefaultExtended, setRsvpDefaultExtended] = useState<RsvpDefault>(session.rsvp_default_extended ?? 'none')
  const [rsvpRequireReason, setRsvpRequireReason] = useState<boolean>(session.rsvp_require_reason === 1)
  const [scope, setScope] = useState<Scope>('this_one')
  const [series, setSeries] = useState<Series | null>(null)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (session.series_id) {
      api.get(`/training-series?team_id=${session.team_id}`)
        .then(r => {
          const list: Series[] = Array.isArray(r.data) ? r.data : []
          const found = list.find(s => s.id === session.series_id)
          if (found) setSeries(found)
        })
        .catch(() => {})
    }
  }, [session.series_id, session.team_id])

  // When scope swaps between session and series, pre-populate the RSVP checkboxes
  // with the matching source so the user sees the current value for the chosen scope.
  useEffect(() => {
    if (scope === 'this_one') {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
      setRsvpDefaultPlayers(session.rsvp_default_players ?? 'none')
      setRsvpDefaultExtended(session.rsvp_default_extended ?? 'none')
      setRsvpRequireReason(session.rsvp_require_reason === 1)
    } else if (series) {
      setRsvpDefaultPlayers(series.rsvp_default_players ?? 'none')
      setRsvpDefaultExtended(series.rsvp_default_extended ?? 'none')
      setRsvpRequireReason(series.rsvp_require_reason === 1)
    }
  }, [scope, series, session.rsvp_default_players, session.rsvp_default_extended, session.rsvp_require_reason])

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      if (scope === 'this_one') {
        // note bewusst NICHT mitsenden: der Termin-Hinweis wird ausschließlich
        // über EventNoteEditor (PUT /trainings/{id}/note + Debounce) gepflegt.
        // UpdateSession behandelt note als Tri-State (fehlt = unverändert).
        await api.put(`/training-sessions/${session.id}`, {
          title,
          date,
          start_time: startTime,
          end_time: endTime,
          venue_id: venueId,
          status: session.status,
          cancel_reason: session.cancel_reason ?? '',
          rsvp_default_players: rsvpDefaultPlayers,
          rsvp_default_extended: rsvpDefaultExtended,
          rsvp_require_reason: rsvpRequireReason ? 1 : 0,
        })
      } else if (series) {
        await api.put(`/training-series/${series.id}`, {
          name: series.name,
          venue_id: venueId,
          day_of_week: series.day_of_week,
          start_time: startTime,
          end_time: endTime,
          valid_from: series.valid_from.slice(0, 10),
          valid_until: series.valid_until.slice(0, 10),
          note: series.note,
          rsvp_default_players: rsvpDefaultPlayers,
          rsvp_default_extended: rsvpDefaultExtended,
          rsvp_require_reason: rsvpRequireReason ? 1 : 0,
          scope: scope === 'this_and_following' ? 'this_and_following' : 'all',
          from_date: scope === 'this_and_following' ? session.date.slice(0, 10) : undefined,
        })
      }
      onSaved()
    } catch {
      setError('Speichern fehlgeschlagen.')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    setDeleting(true)
    setError(null)
    try {
      if (scope === 'this_one') {
        await api.delete(`/training-sessions/${session.id}`)
      } else if (series) {
        const fromParam = scope === 'this_and_following'
          ? `&from=${session.date.slice(0, 10)}`
          : ''
        await api.delete(`/training-series/${series.id}?scope=${scope}${fromParam}`)
      }
      onSaved()
    } catch {
      setError('Löschen fehlgeschlagen.')
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl border-t-4 border-brand-yellow p-6 w-full max-w-md shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-brand-text">Training bearbeiten</h2>
          <button onClick={onClose} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Schließen">
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        {teamName && (
          <p className="text-sm text-brand-text-muted mb-4">Mannschaft: <span className="font-medium text-brand-text">{teamName}</span></p>
        )}

        {session.series_id && (
          <div className="mb-4">
            <label className="block text-sm font-medium text-brand-text-muted mb-2">Bearbeitungsumfang</label>
            <div className="space-y-2">
              {([
                ['this_one', 'Nur dieser Termin'],
                ['this_and_following', 'Dieser und folgende'],
                ['all', 'Alle der Serie'],
              ] as [Scope, string][]).map(([val, label]) => (
                <label key={val} className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="scope"
                    value={val}
                    checked={scope === val}
                    onChange={() => { setScope(val); setConfirmDelete(false) }}
                    className="accent-brand-yellow"
                  />
                  <span className="text-sm text-brand-text">{label}</span>
                </label>
              ))}
            </div>
          </div>
        )}

        <div className="space-y-3">
          {scope === 'this_one' && (
            <>
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Titel</label>
                <input type="text" value={title} onChange={e => setTitle(e.target.value)}
                  placeholder="z. B. Konditionstraining" className={INPUT} />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Datum</label>
                <input type="date" value={date} onChange={e => setDate(e.target.value)} className={INPUT} />
              </div>
            </>
          )}
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Startzeit</label>
            <input type="time" value={startTime} onChange={e => setStartTime(e.target.value)} className={INPUT} />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Endzeit</label>
            <input type="time" value={endTime} onChange={e => setEndTime(e.target.value)} className={INPUT} />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Ort</label>
            <VenuePicker value={venueId} onChange={setVenueId} />
          </div>
          <RsvpDefaultsEditor
            defaultPlayers={rsvpDefaultPlayers}
            defaultExtended={rsvpDefaultExtended}
            requireReason={rsvpRequireReason}
            onChangePlayers={setRsvpDefaultPlayers}
            onChangeExtended={setRsvpDefaultExtended}
            onChangeRequireReason={setRsvpRequireReason}
          />
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
        </div>

        {confirmDelete ? (
          <div className="mt-4 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg">
            <p className="text-sm text-brand-danger mb-3">
              {scope === 'this_one' && 'Diesen Termin wirklich löschen?'}
              {scope === 'this_and_following' && 'Diesen und alle folgenden Termine der Serie löschen?'}
              {scope === 'all' && 'Die gesamte Serie und alle Termine löschen?'}
            </p>
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
            <button
              onClick={() => setConfirmDelete(true)}
              disabled={deleting || saving || (scope !== 'this_one' && !series)}
              className="p-2 text-brand-text-muted hover:text-brand-danger hover:bg-brand-danger-light rounded-md transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              aria-label="Training löschen"
            >
              <Trash2 className="w-4 h-4" />
            </button>
            <button onClick={onClose} className={BTN_SECONDARY}>Abbrechen</button>
            <button
              onClick={handleSave}
              disabled={saving || deleting || (scope !== 'this_one' && !series)}
              className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
            >
              {saving ? 'Speichern…' : 'Speichern'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
