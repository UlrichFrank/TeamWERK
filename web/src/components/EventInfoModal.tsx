import { useState } from 'react'
import { Home, Plane, Calendar, Dumbbell, X, Check, Pencil, ClipboardList, Trash2, BriefcaseMedical } from 'lucide-react'
import { useEscapeKey } from '../lib/useEscapeKey'
import MapsLink from './MapsLink'
import { api } from '../lib/api'

interface VenueRef {
  id: number
  name: string
  street: string
  city: string
  postal_code: string
  note: string
}

interface Game {
  id: number
  date: string
  time: string
  end_date?: string | null
  opponent: string
  event_type: string
  teams?: Array<{ id: number; name: string }>
  confirmed_count: number
  declined_count: number
  maybe_count: number
  venue?: VenueRef | null
}

interface Training {
  id: number
  title: string
  date: string
  start_time: string
  end_time: string
  team_name?: string
  venue?: VenueRef | null
  confirmed_count: number
  declined_count: number
  maybe_count: number
}

interface AbsenceInfo {
  id: number
  member_id: number
  member_name: string
  can_edit: boolean
  type: 'vacation' | 'injury'
  start_date: string
  end_date: string
  note: string
  created_by: number
}

interface Props {
  type: 'game' | 'training' | 'absence'
  game?: Game
  training?: Training
  absence?: AbsenceInfo
  onClose: () => void
  onEdit?: () => void
  onDienste?: () => void
  canEditAbsence?: boolean
  onAbsenceChanged?: () => void
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr.slice(0, 10) + 'T12:00:00')
  return d.toLocaleDateString('de-DE', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
}

function formatDateRange(startStr: string, endStr: string): string {
  const start = new Date(startStr.slice(0, 10) + 'T12:00:00')
  const end = new Date(endStr.slice(0, 10) + 'T12:00:00')
  const startFmt = start.toLocaleDateString('de-DE', { day: 'numeric', month: 'long' })
  const endFmt = end.toLocaleDateString('de-DE', { day: 'numeric', month: 'long', year: 'numeric' })
  return `${startFmt} – ${endFmt}`
}

function RsvpRow({ confirmed, declined, maybe }: { confirmed: number; declined: number; maybe: number }) {
  return (
    <div className="pt-3 border-t border-brand-border-subtle flex gap-5 text-sm">
      <span className="flex items-center gap-1 text-green-600">
        <Check className="w-4 h-4" />
        <span className="font-medium">{confirmed}</span>
      </span>
      <span className="flex items-center gap-1 text-brand-danger">
        <X className="w-4 h-4" />
        <span className="font-medium">{declined}</span>
      </span>
      <span className="flex items-center gap-1 text-brand-text-muted">
        <span className="font-medium">{maybe}</span>
        <span className="text-xs">vlt.</span>
      </span>
    </div>
  )
}

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

export default function EventInfoModal({ type, game, training, absence, onClose, onEdit, onDienste, canEditAbsence, onAbsenceChanged }: Props) {
  useEscapeKey(onClose)

  const [editing, setEditing] = useState(false)
  const [editType, setEditType] = useState<'vacation' | 'injury'>(absence?.type ?? 'vacation')
  const [editStart, setEditStart] = useState(absence?.start_date ?? '')
  const [editEnd, setEditEnd] = useState(absence?.end_date ?? '')
  const [editNote, setEditNote] = useState(absence?.note ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const eventTypeLabel = game?.event_type === 'heim' ? 'Heimspiel'
    : game?.event_type === 'auswärts' ? 'Auswärtsspiel'
    : 'Event'

  const EventIcon = game?.event_type === 'heim' ? Home
    : game?.event_type === 'auswärts' ? Plane
    : Calendar

  const absenceTypeLabel = absence?.type === 'vacation' ? 'Urlaub' : 'Verletzung'
  const AbsenceIcon = absence?.type === 'injury' ? BriefcaseMedical : Plane

  async function handleSaveAbsence() {
    if (!absence) return
    if (editStart > editEnd) { setError('Startdatum muss vor dem Enddatum liegen.'); return }
    setSaving(true)
    setError('')
    try {
      await api.put(`/absences/${absence.id}`, { type: editType, start_date: editStart, end_date: editEnd, note: editNote })
      onAbsenceChanged?.()
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      setError(status === 409
        ? 'Eine Abwesenheit dieses Typs überschneidet sich bereits mit diesem Zeitraum.'
        : 'Fehler beim Speichern.')
      setSaving(false)
    }
  }

  async function handleDeleteAbsence() {
    if (!absence) return
    await api.delete(`/absences/${absence.id}`)
    onAbsenceChanged?.()
  }

  return (
    <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            {type === 'game'
              ? <EventIcon className="w-5 h-5 text-brand-text-muted" />
              : type === 'training'
              ? <Dumbbell className="w-5 h-5 text-brand-green" />
              : <AbsenceIcon className="w-5 h-5 text-brand-text-muted" />}
            <h2 className="text-lg font-bold text-brand-text">
              {type === 'game' ? eventTypeLabel
                : type === 'training' ? (training?.title || 'Training')
                : editing ? (editType === 'vacation' ? 'Urlaub' : 'Verletzung') + ' bearbeiten'
                : absenceTypeLabel}
            </h2>
          </div>
          <div className="flex items-center gap-1">
            {onEdit && (
              <button onClick={onEdit} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Bearbeiten">
                <Pencil className="w-4 h-4 text-brand-text-muted" />
              </button>
            )}
            {onDienste && (
              <button onClick={onDienste} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Dienste">
                <ClipboardList className="w-4 h-4 text-brand-text-muted" />
              </button>
            )}
            {type === 'absence' && canEditAbsence && !editing && (
              <button onClick={() => setEditing(true)} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Bearbeiten">
                <Pencil className="w-4 h-4 text-brand-text-muted" />
              </button>
            )}
            <button onClick={onClose} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Schließen">
              <X className="w-5 h-5 text-brand-text-muted" />
            </button>
          </div>
        </div>

        {type === 'game' && game ? (
          <div className="space-y-2 text-sm">
            {game.opponent && (
              <div className="flex justify-between">
                <span className="text-brand-text-muted">{game.event_type === 'generisch' ? 'Event-Name' : 'Gegner'}</span>
                <span className="font-medium text-brand-text">{game.opponent}</span>
              </div>
            )}
            <div className="flex justify-between">
              <span className="text-brand-text-muted">Datum</span>
              <span className="font-medium text-brand-text">
                {game.end_date && game.end_date.slice(0, 10) !== game.date.slice(0, 10)
                  ? formatDateRange(game.date, game.end_date)
                  : formatDate(game.date)}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-brand-text-muted">Uhrzeit</span>
              <span className="font-medium text-brand-text">{game.time}</span>
            </div>
            {game.teams && game.teams.length > 0 && (
              <div className="flex justify-between">
                <span className="text-brand-text-muted">{game.teams.length === 1 ? 'Team' : 'Teams'}</span>
                <span className="font-medium text-brand-text">{game.teams.map(t => t.name).join(', ')}</span>
              </div>
            )}
            {game.venue && (
              <div className="flex justify-between items-center">
                <span className="text-brand-text-muted">Ort</span>
                <MapsLink venue={game.venue} />
              </div>
            )}
            <RsvpRow confirmed={game.confirmed_count} declined={game.declined_count} maybe={game.maybe_count} />
          </div>
        ) : type === 'training' && training ? (
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-brand-text-muted">Datum</span>
              <span className="font-medium text-brand-text">{formatDate(training.date)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-brand-text-muted">Uhrzeit</span>
              <span className="font-medium text-brand-text">{training.start_time}–{training.end_time}</span>
            </div>
            {training.team_name && (
              <div className="flex justify-between">
                <span className="text-brand-text-muted">Team</span>
                <span className="font-medium text-brand-text">{training.team_name}</span>
              </div>
            )}
            {training.venue && (
              <div className="flex justify-between items-center">
                <span className="text-brand-text-muted">Ort</span>
                <MapsLink venue={training.venue} />
              </div>
            )}
            <RsvpRow confirmed={training.confirmed_count} declined={training.declined_count} maybe={training.maybe_count} />
          </div>
        ) : type === 'absence' && absence ? (
          editing ? (
            <div className="space-y-3">
              <div>
                <label className="block text-xs text-brand-text-muted mb-1">Typ</label>
                <select value={editType} onChange={e => setEditType(e.target.value as 'vacation' | 'injury')} className={INPUT}>
                  <option value="vacation">Urlaub</option>
                  <option value="injury">Verletzung</option>
                </select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs text-brand-text-muted mb-1">Von</label>
                  <input type="date" value={editStart} onChange={e => setEditStart(e.target.value)} className={INPUT} />
                </div>
                <div>
                  <label className="block text-xs text-brand-text-muted mb-1">Bis</label>
                  <input type="date" value={editEnd} onChange={e => setEditEnd(e.target.value)} className={INPUT} />
                </div>
              </div>
              <div>
                <label className="block text-xs text-brand-text-muted mb-1">Notiz</label>
                <input type="text" value={editNote} onChange={e => setEditNote(e.target.value)} placeholder="Optional" className={INPUT} />
              </div>
              {error && <p className="text-sm text-brand-danger">{error}</p>}
            </div>
          ) : (
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-brand-text-muted">Mitglied</span>
                <span className="font-medium text-brand-text">{absence.member_name}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-brand-text-muted">Von</span>
                <span className="font-medium text-brand-text">{formatDate(absence.start_date)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-brand-text-muted">Bis</span>
                <span className="font-medium text-brand-text">{formatDate(absence.end_date)}</span>
              </div>
              {absence.note && (
                <div className="flex justify-between">
                  <span className="text-brand-text-muted">Notiz</span>
                  <span className="font-medium text-brand-text">{absence.note}</span>
                </div>
              )}
            </div>
          )
        ) : null}

        <div className="pt-4 flex gap-2">
          {type === 'absence' && editing ? (
            <>
              <button
                onClick={handleDeleteAbsence}
                className="p-2 text-brand-text-muted hover:text-brand-danger hover:bg-brand-danger-light rounded-md transition-colors"
                aria-label="Löschen"
              >
                <Trash2 className="w-4 h-4" />
              </button>
              <button
                onClick={() => { setEditing(false); setError('') }}
                className="border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleSaveAbsence}
                disabled={saving}
                className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {saving ? 'Speichern…' : 'Speichern'}
              </button>
            </>
          ) : (
            <button
              onClick={onClose}
              className="w-full border border-brand-border rounded-md px-4 py-2.5 sm:py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors"
            >
              Schließen
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
