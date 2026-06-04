import { Fragment, useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { AlertTriangle, Check, ChevronLeft, Clock, Dumbbell, HelpCircle, Home, MapPin, Plane, MessageCircle, X } from 'lucide-react'
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

interface RSVPEntry {
  member_id: number
  member_name: string
  status: 'confirmed' | 'declined' | 'maybe'
  reason: string | null
  responded_by: number
  responded_at: string
}

interface SessionDetail {
  id: number
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
  responses: RSVPEntry[]
}

interface GameDetail {
  id: number
  date: string
  time: string
  opponent: string
  event_type: string
  is_home: boolean
  season_id: number
}

interface AttendanceItem {
  member_id: number
  member_name: string
  rsvp_status: string | null
  reason: string | null
  present: boolean | null
}

interface TableRow {
  member_id: number
  member_name: string
  rsvp_status: string | null
  reason: string | null
  present: boolean | null
}

export default function TermineDetailPage() {
  const { type, id } = useParams<{ type: string; id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isTrainer = user?.role === 'admin' || hasFunction(user, 'trainer')

  const [session, setSession] = useState<SessionDetail | null>(null)
  const [game, setGame] = useState<GameDetail | null>(null)
  const [gameResponses, setGameResponses] = useState<RSVPEntry[]>([])
  const [attendances, setAttendances] = useState<AttendanceItem[]>([])
  const [loading, setLoading] = useState(true)
  const [attendanceMap, setAttendanceMap] = useState<Record<number, boolean>>({})
  const [attendanceError, setAttendanceError] = useState<string | null>(null)
  const [showReasonId, setShowReasonId] = useState<number | null>(null)

  const isTraining = type === 'training'
  const date = isTraining ? session?.date : game?.date
  const today = new Date().toISOString().slice(0, 10)
  const isPast = date ? date.slice(0, 10) <= today : false

  const load = () => {
    setLoading(true)
    if (isTraining) {
      api.get(`/training-sessions/${id}`)
        .then(r => setSession(r.data))
        .finally(() => setLoading(false))
    } else {
      Promise.all([
        api.get(`/kalender/${id}`),
        api.get(`/games/${id}/responses`),
      ])
        .then(([gameRes, responsesRes]) => {
          setGame(gameRes.data.game ?? gameRes.data)
          setGameResponses(responsesRes.data ?? [])
        })
        .finally(() => setLoading(false))
    }
  }

  const loadAttendances = () => {
    if (!isTraining) return
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
    if (isTraining) loadAttendances()
  }, [id, type])

  useLiveUpdates((event) => {
    if ((isTraining && event === 'trainings') || (!isTraining && event === 'games')) load()
  })

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
  if (isTraining && !session) return <p className="text-brand-danger text-sm p-4">Termin nicht gefunden.</p>
  if (!isTraining && !game) return <p className="text-brand-danger text-sm p-4">Spiel nicht gefunden.</p>

  // --- Training detail ---
  if (isTraining && session) {
    const noRsvpCount = attendances.length - session.confirmed_count - session.declined_count - session.maybe_count
    const showAttendanceCol = isTrainer && isPast

    const tableRows: TableRow[] = attendances.map(a => ({
      member_id: a.member_id,
      member_name: a.member_name,
      rsvp_status: a.rsvp_status,
      reason: a.reason,
      present: a.present,
    }))

    return (
      <div className="max-w-2xl space-y-4">
        <button
          onClick={() => navigate('/termine')}
          className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text transition-colors mb-2"
        >
          <ChevronLeft className="w-4 h-4" /> Zurück zu Termine
        </button>

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

        <ResponseTable
          rows={tableRows}
          showAttendanceCol={showAttendanceCol}
          attendanceMap={attendanceMap}
          attendanceError={attendanceError}
          isTrainer={isTrainer}
          showReasonId={showReasonId}
          setShowReasonId={setShowReasonId}
          onToggleAttendance={toggleAttendance}
          onDismissError={() => setAttendanceError(null)}
        />
      </div>
    )
  }

  // --- Game detail ---
  const g = game!
  const Icon = g.is_home ? Home : Plane
  const gameLabel = g.event_type === 'generisch'
    ? g.opponent
    : (g.is_home ? `Heimspiel vs. ${g.opponent}` : `Auswärtsspiel vs. ${g.opponent}`)

  const tableRows: TableRow[] = gameResponses.map(r => ({
    member_id: r.member_id,
    member_name: r.member_name,
    rsvp_status: r.status,
    reason: r.reason,
    present: null,
  }))

  const confirmedCount = gameResponses.filter(r => r.status === 'confirmed').length
  const declinedCount = gameResponses.filter(r => r.status === 'declined').length
  const maybeCount = gameResponses.filter(r => r.status === 'maybe').length

  return (
    <div className="max-w-2xl space-y-4">
      <button
        onClick={() => navigate('/termine')}
        className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text transition-colors mb-2"
      >
        <ChevronLeft className="w-4 h-4" /> Zurück zu Termine
      </button>

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <div className="flex items-start gap-3">
          <Icon className="w-6 h-6 mt-0.5 text-brand-text-muted shrink-0" />
          <div className="flex-1 min-w-0">
            <h1 className="text-xl font-bold text-brand-text">{fmtDate(g.date)}</h1>
            <p className="text-brand-text-muted mt-1">{gameLabel}</p>
            <div className="mt-3 flex items-center gap-2 text-sm text-brand-text-muted">
              <Clock className="w-4 h-4" />
              {g.time} Uhr
            </div>
            <div className="mt-4 flex flex-wrap gap-2">
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700">
                <Check className="w-3 h-3" /> {confirmedCount}
              </span>
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-brand-danger-light text-brand-danger">
                <X className="w-3 h-3" /> {declinedCount}
              </span>
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-brand-border-subtle text-brand-text-muted">
                <HelpCircle className="w-3 h-3" /> {maybeCount}
              </span>
            </div>
          </div>
        </div>
      </div>

      <ResponseTable
        rows={tableRows}
        showAttendanceCol={false}
        attendanceMap={{}}
        attendanceError={null}
        isTrainer={isTrainer}
        showReasonId={showReasonId}
        setShowReasonId={setShowReasonId}
        onToggleAttendance={() => Promise.resolve()}
        onDismissError={() => {}}
      />
    </div>
  )
}

function ResponseTable({ rows, showAttendanceCol, attendanceMap, attendanceError, isTrainer, showReasonId, setShowReasonId, onToggleAttendance, onDismissError }: {
  rows: TableRow[]
  showAttendanceCol: boolean
  attendanceMap: Record<number, boolean>
  attendanceError: string | null
  isTrainer: boolean
  showReasonId: number | null
  setShowReasonId: (id: number | null) => void
  onToggleAttendance: (memberId: number, value: boolean) => Promise<void>
  onDismissError: () => void
}) {
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <div className="px-6 py-4 border-b border-brand-border-subtle">
        <h2 className="font-semibold text-brand-text">Teilnahme</h2>
      </div>
      {rows.length === 0 ? (
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
              {rows.map(row => (
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
                          onChange={e => onToggleAttendance(row.member_id, e.target.checked)}
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
              <button onClick={onDismissError} aria-label="Schließen">
                <X className="w-3 h-3" />
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
