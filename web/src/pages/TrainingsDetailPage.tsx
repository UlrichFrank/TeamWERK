import { Fragment, useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { AlertTriangle, Check, ChevronLeft, Clock, Dumbbell, HelpCircle, MapPin, MessageCircle, X } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth, hasFunction } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

const WEEKDAYS = ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag']

function fmtDate(iso: string) {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAYS[d.getDay()]}, ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}`
}

function RsvpIcon({ status }: { status: string | null }) {
  if (status === 'confirmed') return <Check className="w-4 h-4 text-green-600" />
  if (status === 'declined') return <X className="w-4 h-4 text-brand-danger" />
  if (status === 'maybe') return <HelpCircle className="w-4 h-4 text-brand-text-subtle" />
  return <span className="text-brand-text-muted text-sm">–</span>
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

interface TableRow {
  member_id: number
  member_name: string
  rsvp_status: string | null
  reason: string | null
  present: boolean | null
}

export default function TrainingsDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isTrainer = user?.role === 'admin' || hasFunction(user, 'trainer')

  const [session, setSession] = useState<SessionDetail | null>(null)
  const [attendances, setAttendances] = useState<AttendanceItem[]>([])
  const [loading, setLoading] = useState(true)
  const [attendanceMap, setAttendanceMap] = useState<Record<number, boolean>>({})
  const [attendanceError, setAttendanceError] = useState<string | null>(null)
  const [showReasonId, setShowReasonId] = useState<number | null>(null)

  const today = new Date().toISOString().slice(0, 10)
  const isPast = session ? session.date.slice(0, 10) <= today : false

  const load = () => {
    api.get(`/training-sessions/${id}`)
      .then(r => setSession(r.data))
      .finally(() => setLoading(false))
  }

  const loadAttendances = () => {
    api.get(`/training-sessions/${id}/attendances`).then(r => {
      const data: AttendanceItem[] = r.data ?? []
      setAttendances(data)
      const map: Record<number, boolean> = {}
      for (const a of data) {
        if (a.present !== null) map[a.member_id] = a.present
      }
      setAttendanceMap(map)
    })
  }

  useEffect(() => {
    load()
    if (isTrainer) loadAttendances()
  }, [id])
  useLiveUpdates((event) => { if (event === 'trainings') load() })

  const toggleAttendance = async (memberId: number, newValue: boolean) => {
    setAttendanceMap(prev => ({ ...prev, [memberId]: newValue }))
    const entries = attendances.map(a => ({
      member_id: a.member_id,
      present: a.member_id === memberId ? newValue : (attendanceMap[a.member_id] ?? false),
    }))
    try {
      await api.post(`/training-sessions/${id}/attendances`, entries)
      setAttendanceError(null)
    } catch {
      setAttendanceMap(prev => ({ ...prev, [memberId]: !newValue }))
      setAttendanceError('Fehler beim Speichern. Bitte nochmal versuchen.')
    }
  }

  if (loading) return <p className="text-brand-text-muted text-sm p-4">Laden…</p>
  if (!session) return <p className="text-brand-danger text-sm p-4">Termin nicht gefunden.</p>

  const responseMap = Object.fromEntries(session.responses.map(r => [r.member_id, r]))
  const noRsvpCount = attendances.length - session.confirmed_count - session.declined_count - session.maybe_count
  const showAttendanceCol = isTrainer && isPast

  const tableRows: TableRow[] = isTrainer
    ? attendances.map(a => ({
        member_id: a.member_id,
        member_name: a.member_name,
        rsvp_status: a.rsvp_status,
        reason: responseMap[a.member_id]?.reason ?? null,
        present: a.present,
      }))
    : session.responses.map(r => ({
        member_id: r.member_id,
        member_name: r.member_name,
        rsvp_status: r.status,
        reason: r.reason,
        present: null,
      }))

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
            {/* Stat badges */}
            <div className="mt-4 flex flex-wrap gap-2">
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700">
                <Check className="w-3 h-3" /> {session.confirmed_count}
              </span>
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-brand-danger-light text-brand-danger">
                <X className="w-3 h-3" /> {session.declined_count}
              </span>
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-brand-border-subtle text-brand-text-muted">
                <HelpCircle className="w-3 h-3" /> {session.maybe_count}
              </span>
              {isTrainer && attendances.length > 0 && noRsvpCount > 0 && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-brand-border-subtle text-brand-text-muted">
                  – {noRsvpCount}
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Unified participation table */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <div className="px-6 py-4 border-b border-brand-border-subtle">
          <h2 className="font-semibold text-brand-text">Teilnahme</h2>
        </div>
        {tableRows.length === 0 ? (
          <p className="px-6 py-4 text-sm text-brand-text-muted">
            {isTrainer ? 'Keine Mitglieder gefunden.' : 'Noch keine Rückmeldungen.'}
          </p>
        ) : (
          <>
            <table className="w-full">
              <thead>
                <tr className="border-b border-brand-border-subtle">
                  <th className="text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Mitglied</th>
                  <th className="text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Rückmeldung</th>
                  {showAttendanceCol && (
                    <th className="text-brand-text-muted text-xs uppercase px-4 py-3 text-center">Anwesend</th>
                  )}
                </tr>
              </thead>
              <tbody>
                {tableRows.map(row => (
                  <Fragment key={row.member_id}>
                    <tr className="border-b border-brand-border-subtle last:border-0 hover:bg-brand-table-select transition-colors">
                      <td className="px-4 py-3 text-sm text-brand-text font-medium">{row.member_name}</td>
                      <td className="px-4 py-3">
                        <div className="relative group flex items-center gap-1">
                          <RsvpIcon status={row.rsvp_status} />
                          {row.reason && (
                            <>
                              <button
                                onClick={() => setShowReasonId(row.member_id === showReasonId ? null : row.member_id)}
                                aria-label="Kommentar anzeigen"
                              >
                                <MessageCircle className="w-3 h-3 text-brand-text-muted" />
                              </button>
                              <div className="hidden group-hover:block absolute left-0 top-full z-10 mt-1 w-48 rounded-md bg-brand-text px-2 py-1 text-xs text-white shadow-lg pointer-events-none">
                                {row.reason}
                              </div>
                            </>
                          )}
                        </div>
                      </td>
                      {showAttendanceCol && (
                        <td className="px-4 py-3 text-center">
                          <input
                            type="checkbox"
                            checked={attendanceMap[row.member_id] ?? false}
                            onChange={e => toggleAttendance(row.member_id, e.target.checked)}
                            className="w-4 h-4 rounded border-brand-border"
                          />
                        </td>
                      )}
                    </tr>
                    {showReasonId === row.member_id && row.reason && (
                      <tr className="bg-brand-surface-card">
                        <td colSpan={showAttendanceCol ? 3 : 2} className="px-4 pb-2 text-xs text-brand-text-muted italic">
                          {row.reason}
                        </td>
                      </tr>
                    )}
                  </Fragment>
                ))}
              </tbody>
            </table>
            {attendanceError && (
              <div className="px-4 py-2 text-xs text-brand-danger bg-brand-danger-light border-t border-brand-danger/20 flex items-center justify-between">
                {attendanceError}
                <button onClick={() => setAttendanceError(null)} aria-label="Schließen">
                  <X className="w-3 h-3" />
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
