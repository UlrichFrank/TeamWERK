import { useState, useEffect } from 'react'
import { X } from 'lucide-react'
import { api } from '../lib/api'

interface Training {
  id: number
  title: string
  date: string
  start_time: string
  end_time: string
  location: string
  status: 'active' | 'cancelled'
  note: string
  cancel_reason?: string
  series_id?: number
  team_id: number
  season_id: number
}

interface Series {
  id: number
  name: string
  location: string
  day_of_week: number
  start_time: string
  end_time: string
  valid_from: string
  valid_until: string
  note: string
  rsvp_opt_out: number
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
  const [location, setLocation] = useState(session.location)
  const [status, setStatus] = useState<'active' | 'cancelled'>(session.status)
  const [cancelReason, setCancelReason] = useState(session.cancel_reason ?? '')
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

  const handleSave = async () => {
    setSaving(true)
    setError(null)
    try {
      if (scope === 'this_one') {
        await api.put(`/training-sessions/${session.id}`, {
          title,
          date,
          start_time: startTime,
          end_time: endTime,
          location,
          note: session.note,
          status,
          cancel_reason: cancelReason,
        })
      } else if (series) {
        await api.put(`/training-series/${series.id}`, {
          name: series.name,
          location,
          day_of_week: series.day_of_week,
          start_time: startTime,
          end_time: endTime,
          valid_from: series.valid_from.slice(0, 10),
          valid_until: series.valid_until.slice(0, 10),
          note: series.note,
          rsvp_opt_out: series.rsvp_opt_out,
          rsvp_require_reason: series.rsvp_require_reason,
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
            <input type="text" value={location} onChange={e => setLocation(e.target.value)}
              placeholder="Sporthalle…" className={INPUT} />
          </div>
          {scope === 'this_one' && (
            <>
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Status</label>
                <select value={status} onChange={e => setStatus(e.target.value as 'active' | 'cancelled')} className={INPUT}>
                  <option value="active">Aktiv</option>
                  <option value="cancelled">Abgesagt</option>
                </select>
              </div>
              {status === 'cancelled' && (
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Absagegrund</label>
                  <input type="text" value={cancelReason} onChange={e => setCancelReason(e.target.value)}
                    placeholder="z. B. Hallensperrung" className={INPUT} />
                </div>
              )}
            </>
          )}
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
        </div>

        {confirmDelete && (
          <div className="mt-4 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
            {scope === 'this_one' && 'Diesen Termin wirklich löschen?'}
            {scope === 'this_and_following' && 'Diesen und alle folgenden Termine der Serie löschen?'}
            {scope === 'all' && 'Die gesamte Serie und alle Termine löschen?'}
          </div>
        )}

        <div className="flex gap-2 pt-4">
          <button
            onClick={() => {
              if (!confirmDelete) {
                setConfirmDelete(true)
              } else {
                handleDelete()
              }
            }}
            disabled={deleting || saving || (scope !== 'this_one' && !series)}
            className="bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {deleting ? 'Löschen…' : confirmDelete ? 'Bestätigen' : 'Löschen'}
          </button>
          {confirmDelete && (
            <button onClick={() => setConfirmDelete(false)} className={BTN_SECONDARY}>
              Abbrechen
            </button>
          )}
          {!confirmDelete && (
            <>
              <button onClick={onClose} className={BTN_SECONDARY}>Abbrechen</button>
              <button
                onClick={handleSave}
                disabled={saving || deleting || (scope !== 'this_one' && !series)}
                className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
              >
                {saving ? 'Speichern…' : 'Speichern'}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
