import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Check, X, HelpCircle, Dumbbell, ChevronLeft, MapPin, Clock, AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth, hasFunction } from '../contexts/AuthContext'

const WEEKDAYS = ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag']

function fmtDate(iso: string) {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAYS[d.getDay()]}, ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}`
}

interface TrainingResponse {
  member_id: number
  member_name: string
  status: 'confirmed' | 'declined' | 'maybe'
  reason: string | null
  responded_by: number
  responded_at: string
}

interface SessionDetail {
  id: number
  series_id: number | null
  team_id: number
  date: string
  start_time: string
  end_time: string
  location: string
  note: string
  status: 'active' | 'cancelled'
  cancel_reason: string
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
  responses: TrainingResponse[]
}

interface AttendanceItem {
  member_id: number
  member_name: string
  rsvp_status: string | null
  present: boolean | null
}

const statusIcon = { confirmed: <Check className="w-4 h-4 text-green-600" />, declined: <X className="w-4 h-4 text-brand-danger" />, maybe: <HelpCircle className="w-4 h-4 text-brand-text-subtle" /> }
const statusLabel = { confirmed: 'Zugesagt', declined: 'Abgesagt', maybe: 'Vielleicht' }

export default function TrainingsDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isTrainer = user?.role === 'admin' || hasFunction(user, 'trainer')
  const canRsvp = !isTrainer

  const [session, setSession] = useState<SessionDetail | null>(null)
  const [attendances, setAttendances] = useState<AttendanceItem[]>([])
  const [loading, setLoading] = useState(true)
  const [rsvpLoading, setRsvpLoading] = useState(false)
  const [declineReason, setDeclineReason] = useState('')
  const [showDeclineForm, setShowDeclineForm] = useState(false)
  const [attendanceSaving, setAttendanceSaving] = useState(false)
  const [attendanceMap, setAttendanceMap] = useState<Record<number, boolean>>({})
  const [attendanceLoaded, setAttendanceLoaded] = useState(false)

  const today = new Date().toISOString().slice(0, 10)
  const isPast = session ? session.date <= today : false

  const load = () => {
    api.get(`/training-sessions/${id}`)
      .then(r => setSession(r.data))
      .finally(() => setLoading(false))
  }

  const loadAttendances = () => {
    api.get(`/training-sessions/${id}/attendances`).then(r => {
      setAttendances(r.data ?? [])
      const map: Record<number, boolean> = {}
      for (const a of r.data ?? []) {
        if (a.present !== null) map[a.member_id] = a.present
      }
      setAttendanceMap(map)
      setAttendanceLoaded(true)
    })
  }

  useEffect(() => {
    load()
    if (isTrainer) loadAttendances()
  }, [id])

  const respond = async (status: string, reason = '') => {
    setRsvpLoading(true)
    try {
      await api.post(`/training-sessions/${id}/respond`, { status, reason })
      setSession(prev => prev ? { ...prev, my_rsvp: status } : prev)
      setShowDeclineForm(false)
    } finally {
      setRsvpLoading(false)
    }
  }

  const saveAttendances = async () => {
    setAttendanceSaving(true)
    try {
      const entries = attendances.map(a => ({
        member_id: a.member_id,
        present: attendanceMap[a.member_id] ?? false,
      }))
      await api.post(`/training-sessions/${id}/attendances`, entries)
      loadAttendances()
    } finally {
      setAttendanceSaving(false)
    }
  }

  if (loading) return <p className="text-brand-text-muted text-sm p-4">Laden…</p>
  if (!session) return <p className="text-brand-danger text-sm p-4">Termin nicht gefunden.</p>

  const confirmed = session.responses.filter(r => r.status === 'confirmed')
  const declined = session.responses.filter(r => r.status === 'declined')
  const maybe = session.responses.filter(r => r.status === 'maybe')

  return (
    <div className="max-w-2xl space-y-4">
      <button
        onClick={() => navigate('/trainings')}
        className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text transition-colors mb-2"
      >
        <ChevronLeft className="w-4 h-4" /> Zurück zu Trainings
      </button>

      {/* Session Info */}
      <div className={`bg-brand-surface-card rounded-xl shadow border-t-4 p-6 ${session.status === 'cancelled' ? 'border-brand-border' : 'border-brand-yellow'}`}>
        <div className="flex items-start gap-3">
          <Dumbbell className="w-6 h-6 mt-0.5 text-brand-text-muted shrink-0" />
          <div className="flex-1 min-w-0">
            <h1 className={`text-xl font-bold text-brand-text ${session.status === 'cancelled' ? 'line-through opacity-60' : ''}`}>
              {fmtDate(session.date)}
            </h1>
            {session.status === 'cancelled' && (
              <div className="mt-2 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-start gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                <span>Training abgesagt{session.cancel_reason ? `: ${session.cancel_reason}` : '.'}</span>
              </div>
            )}
            <div className="mt-3 space-y-1.5">
              <div className="flex items-center gap-2 text-sm text-brand-text-muted">
                <Clock className="w-4 h-4" />
                {session.start_time} – {session.end_time} Uhr
              </div>
              {session.location && (
                <div className="flex items-center gap-2 text-sm text-brand-text-muted">
                  <MapPin className="w-4 h-4" />
                  {session.location}
                </div>
              )}
            </div>
            {session.note && (
              <p className="mt-3 text-sm text-brand-text bg-white border border-brand-border-subtle rounded-lg p-3">{session.note}</p>
            )}
          </div>
        </div>
      </div>

      {/* RSVP for spieler/elternteil */}
      {canRsvp && session.status === 'active' && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-brand-text mb-3">Meine Rückmeldung</h2>
          {session.my_rsvp && (
            <p className="text-sm text-brand-text-muted mb-3">
              Aktuelle Antwort: <strong>{statusLabel[session.my_rsvp as keyof typeof statusLabel]}</strong>
            </p>
          )}
          <div className="flex gap-2 flex-wrap">
            <button
              disabled={rsvpLoading}
              onClick={() => respond('confirmed')}
              className={`flex items-center gap-2 rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium transition-colors disabled:opacity-40 ${
                session.my_rsvp === 'confirmed'
                  ? 'bg-green-600 text-white'
                  : 'bg-brand-surface-card border border-brand-border text-brand-text hover:bg-green-50 hover:border-green-300'
              }`}
            >
              <Check className="w-4 h-4" /> Zusagen
            </button>
            <button
              disabled={rsvpLoading}
              onClick={() => respond('maybe')}
              className={`flex items-center gap-2 rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium transition-colors disabled:opacity-40 ${
                session.my_rsvp === 'maybe'
                  ? 'bg-brand-yellow text-brand-black'
                  : 'bg-brand-surface-card border border-brand-border text-brand-text hover:bg-yellow-50'
              }`}
            >
              <HelpCircle className="w-4 h-4" /> Vielleicht
            </button>
            <button
              disabled={rsvpLoading}
              onClick={() => setShowDeclineForm(!showDeclineForm)}
              className={`flex items-center gap-2 rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium transition-colors disabled:opacity-40 ${
                session.my_rsvp === 'declined'
                  ? 'bg-brand-danger text-white'
                  : 'bg-brand-surface-card border border-brand-border text-brand-text hover:bg-red-50 hover:border-brand-danger/30'
              }`}
            >
              <X className="w-4 h-4" /> Absagen
            </button>
          </div>
          {showDeclineForm && (
            <div className="mt-3 space-y-2">
              <input
                type="text"
                placeholder="Begründung (optional)"
                value={declineReason}
                onChange={e => setDeclineReason(e.target.value)}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
              />
              <button
                onClick={() => respond('declined', declineReason)}
                disabled={rsvpLoading}
                className="bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40"
              >
                Absagen bestätigen
              </button>
            </div>
          )}
        </div>
      )}

      {/* Responses list */}
      {session.status === 'active' && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
          <div className="px-6 py-4 border-b border-brand-border-subtle">
            <h2 className="font-semibold text-brand-text">
              Rückmeldungen
              <span className="ml-2 text-sm font-normal text-brand-text-muted">
                {session.confirmed_count} ✓ · {session.declined_count} ✗ · {session.maybe_count} ?
              </span>
            </h2>
          </div>
          {[...confirmed, ...maybe, ...declined].length === 0 ? (
            <p className="px-6 py-4 text-sm text-brand-text-muted">Noch keine Rückmeldungen.</p>
          ) : (
            <ul className="divide-y divide-brand-border-subtle">
              {[...confirmed, ...maybe, ...declined].map(resp => (
                <li key={resp.member_id} className="px-6 py-3 flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-brand-text truncate">{resp.member_name}</p>
                    {resp.reason && (
                      <p className="text-xs text-brand-text-muted mt-0.5">{resp.reason}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    {statusIcon[resp.status]}
                    <span className="text-xs text-brand-text-muted hidden sm:inline">{statusLabel[resp.status]}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {/* Attendance section (trainer only, past sessions) */}
      {isTrainer && isPast && attendanceLoaded && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
          <div className="px-6 py-4 border-b border-brand-border-subtle flex items-center justify-between">
            <h2 className="font-semibold text-brand-text">Anwesenheit</h2>
            <button
              onClick={saveAttendances}
              disabled={attendanceSaving}
              className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
            >
              {attendanceSaving ? 'Speichern…' : 'Speichern'}
            </button>
          </div>
          {attendances.length === 0 ? (
            <p className="px-6 py-4 text-sm text-brand-text-muted">Keine Mitglieder gefunden.</p>
          ) : (
            <ul className="divide-y divide-brand-border-subtle">
              {attendances.map(a => (
                <li key={a.member_id} className="px-6 py-3 flex items-center justify-between">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-brand-text">{a.member_name}</p>
                    {a.rsvp_status && (
                      <p className="text-xs text-brand-text-muted mt-0.5">
                        RSVP: {statusLabel[a.rsvp_status as keyof typeof statusLabel] ?? a.rsvp_status}
                      </p>
                    )}
                  </div>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <span className="text-xs text-brand-text-muted">Anwesend</span>
                    <input
                      type="checkbox"
                      checked={attendanceMap[a.member_id] ?? false}
                      onChange={e => setAttendanceMap(prev => ({ ...prev, [a.member_id]: e.target.checked }))}
                      className="w-4 h-4 rounded border-brand-border"
                    />
                  </label>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
