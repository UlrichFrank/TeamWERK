import { Fragment, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { AlertTriangle, Calendar, Check, Clock, Dumbbell, HelpCircle, Home, Plane, MessageCircle, X } from 'lucide-react'
import { api } from '../lib/api'
import MapsLink from '../components/MapsLink'
import { useAuth } from '../contexts/AuthContext'
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

interface ParticipantItem {
  member_id: number
  member_name: string
  is_extended: boolean
  rsvp_status: string | null
  in_lineup: boolean
  team_id: number
}

interface ParticipantsResponse {
  items: ParticipantItem[]
  hidden_team_ids: number[]
}

interface VenueRef {
  id: number
  name: string
  street: string
  city: string
  postal_code: string
  note: string
}

interface SessionDetail {
  id: number
  date: string
  start_time: string
  end_time: string
  team_id: number
  team_name: string
  venue?: VenueRef | null
  note: string
  status: 'active' | 'cancelled'
  cancel_reason: string
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
  responses: RSVPEntry[]
  rsvp_opt_out?: number
  rsvp_require_reason?: number
}

interface TeamRef {
  id: number
  name: string
  display_short: string
  display_long: string
}

interface GameDetail {
  id: number
  date: string
  time: string
  opponent: string
  event_type: string
  is_home: boolean
  season_id: number
  venue?: VenueRef | null
  rsvp_opt_out?: number
  rsvp_require_reason?: number
  teams?: TeamRef[]
  can?: { edit: boolean; delete: boolean; manage_lineup: boolean }
}

function RsvpConfigBadges({ optOut, requireReason }: { optOut?: number; requireReason?: number }) {
  if (optOut !== 1 && requireReason !== 1) return null
  return (
    <div className="mt-2 flex flex-wrap gap-1.5">
      {optOut === 1 && (
        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-brand-info/10 text-brand-info border border-brand-info/30">
          Opt-Out aktiv
        </span>
      )}
      {requireReason === 1 && (
        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-brand-yellow/20 text-brand-text border border-brand-yellow/40">
          Begründung bei Absage Pflicht
        </span>
      )}
    </div>
  )
}

interface AttendanceItem {
  member_id: number
  member_name: string
  is_extended?: boolean
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
  is_extended?: boolean
  in_lineup?: boolean
  team_id?: number
}

// Eine gruppierte Sektion der Teilnahme-Tabelle (Team- oder Kader-Überschrift + Zeilen).
interface TableSection {
  title: string | null
  rows: TableRow[]
  // Wenn true, hat das Backend gefilterte Member aus diesem Team weggelassen
  // (cross_team_visible=0). Footer „Weitere Mitglieder nicht sichtbar" wird gerendert.
  hasHidden?: boolean
}

export default function TermineDetailPage() {
  const { type, id } = useParams<{ type: string; id: string }>()
  const { hasCapability } = useAuth()

  const [session, setSession] = useState<SessionDetail | null>(null)
  const [game, setGame] = useState<GameDetail | null>(null)
  const [participants, setParticipants] = useState<ParticipantItem[]>([])
  const [hiddenTeamIds, setHiddenTeamIds] = useState<number[]>([])
  const [attendances, setAttendances] = useState<AttendanceItem[]>([])
  const [loading, setLoading] = useState(true)
  const [attendanceMap, setAttendanceMap] = useState<Record<number, boolean>>({})
  const [attendanceError, setAttendanceError] = useState<string | null>(null)
  const [lineupMap, setLineupMap] = useState<Record<number, boolean>>({})
  const [showReasonId, setShowReasonId] = useState<number | null>(null)

  const isTraining = type === 'training'
  // Games carry an authoritative per-item can.manage_lineup; trainings have no per-item
  // flag, so fall back to the manage_trainings capability (admin/trainer/sportliche_leitung).
  const isTrainer = Boolean(game?.can?.manage_lineup ?? hasCapability('manage_trainings'))
  const date = isTraining ? session?.date : game?.date
  const today = new Date().toISOString().slice(0, 10)
  const isPast = date ? date.slice(0, 10) <= today : false

  const applyAttendances = (data: AttendanceItem[]) => {
    setAttendances(data)
    const map: Record<number, boolean> = {}
    for (const a of data) {
      if (a.present !== null) map[a.member_id] = a.present
    }
    setAttendanceMap(map)
  }

  const load = (silent = false) => {
    if (!silent) setLoading(true)
    if (isTraining) {
      Promise.all([
        api.get(`/training-sessions/${id}`),
        api.get(`/training-sessions/${id}/attendances`),
      ])
        .then(([sessionRes, attendancesRes]) => {
          setSession(sessionRes.data)
          applyAttendances(attendancesRes.data ?? [])
        })
        .finally(() => { if (!silent) setLoading(false) })
    } else {
      Promise.all([
        api.get(`/games/${id}`),
        api.get(`/games/${id}/participants`),
      ])
        .then(([gameRes, participantsRes]) => {
          setGame(gameRes.data.game ?? gameRes.data)
          const data: ParticipantsResponse | ParticipantItem[] = participantsRes.data ?? { items: [], hidden_team_ids: [] }
          // Backwards-Kompat: alte Array-Form fallweise erkennen (Tests/Mocks).
          const items: ParticipantItem[] = Array.isArray(data) ? data : (data.items ?? [])
          const hidden: number[] = Array.isArray(data) ? [] : (data.hidden_team_ids ?? [])
          setParticipants(items)
          setHiddenTeamIds(hidden)
          const map: Record<number, boolean> = {}
          for (const p of items) map[p.member_id] = p.in_lineup
          setLineupMap(map)
        })
        .finally(() => { if (!silent) setLoading(false) })
    }
  }

  const loadAttendances = () => {
    if (!isTraining) return
    api.get(`/training-sessions/${id}/attendances`)
      .then(r => applyAttendances(r.data ?? []))
      .catch(() => {})
  }

  useEffect(() => {
    load()
    // load kapselt id/type, soll nur bei deren Änderung neu laufen
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, type])

  useLiveUpdates((event) => {
    if (isTraining && event === 'trainings') { load(true); loadAttendances() }
    else if (!isTraining && event === 'games') load(true)
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

  const saveLineup = async (memberId: number, newValue: boolean) => {
    const updatedMap = { ...lineupMap, [memberId]: newValue }
    setLineupMap(updatedMap)
    const memberIds = Object.entries(updatedMap)
      .filter(([, v]) => v)
      .map(([k]) => parseInt(k))
    try {
      await api.post(`/games/${id}/lineup`, { member_ids: memberIds })
    } catch {
      setLineupMap(prev => ({ ...prev, [memberId]: !newValue }))
    }
  }

  if (loading) return <p className="text-brand-text-muted text-sm p-4">Laden…</p>
  if (isTraining && !session) return <p className="text-brand-danger text-sm p-4">Termin nicht gefunden.</p>
  if (!isTraining && !game) return <p className="text-brand-danger text-sm p-4">Spiel nicht gefunden.</p>

  // --- Training detail ---
  if (isTraining && session) {
    const noRsvpCount = attendances.length - session.confirmed_count - session.declined_count - session.maybe_count
    const showAttendanceCol = isPast

    const tableRows: TableRow[] = attendances.map(a => ({
      member_id: a.member_id,
      member_name: a.member_name,
      is_extended: a.is_extended,
      rsvp_status: a.rsvp_status,
      reason: a.reason,
      present: a.present,
    }))

    return (
      <div className="max-w-2xl space-y-4">
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
                {session.team_name && (
                  <div className="flex items-center gap-2 text-sm text-brand-text-muted">
                    <Dumbbell className="w-4 h-4" />
                    {session.team_name}
                  </div>
                )}
                <div className="flex items-center gap-2 text-sm text-brand-text-muted">
                  <Clock className="w-4 h-4" />
                  {session.start_time} – {session.end_time} Uhr
                </div>
                <MapsLink venue={session.venue} />
              </div>
              {session.note && (
                <p className="mt-3 text-sm text-brand-text bg-white border border-brand-border-subtle rounded-lg p-3">{session.note}</p>
              )}
              <RsvpConfigBadges optOut={session.rsvp_opt_out} requireReason={session.rsvp_require_reason} />
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
  const Icon = g.event_type === 'heim' ? Home : g.event_type === 'auswärts' ? Plane : Calendar
  const gameLabel = g.event_type === 'generisch'
    ? g.opponent
    : (g.event_type === 'heim' ? `Heimspiel vs. ${g.opponent}` : `Auswärtsspiel vs. ${g.opponent}`)

  const toRow = (p: ParticipantItem): TableRow => ({
    member_id: p.member_id,
    member_name: p.member_name,
    rsvp_status: p.rsvp_status,
    reason: null,
    present: null,
    is_extended: p.is_extended,
    in_lineup: p.in_lineup,
    team_id: p.team_id,
  })

  // Bei generischen Ereignissen mit mehreren Teams werden die Teilnehmer nach
  // Team gruppiert; pro Team werden Stamm- und erweiterter Kader gemeinsam nach
  // Vorname (= member_name beginnt mit dem Vornamen) sortiert. In allen anderen
  // Fällen bleibt die bisherige Aufteilung Stammkader / Erweiterter Kader.
  const groupByTeam = g.event_type === 'generisch' && (g.teams?.length ?? 0) > 1
  let sections: TableSection[] | undefined
  if (groupByTeam) {
    const hiddenSet = new Set(hiddenTeamIds)
    sections = (g.teams ?? []).map(team => ({
      title: team.display_long || team.display_short || team.name,
      rows: participants
        .filter(p => p.team_id === team.id)
        .map(toRow)
        .sort((a, b) => a.member_name.localeCompare(b.member_name, 'de')),
      hasHidden: hiddenSet.has(team.id),
    }))
      // Sektionen ohne sichtbare Mitglieder weglassen (auch wenn `hasHidden`
      // technisch true wäre): kein leerer Header.
      .filter(s => s.rows.length > 0)
  }

  // Ohne Team-Gruppierung pro Mitglied nur eine Zeile (ein Mitglied kann in
  // mehreren Team-Kadern stehen).
  const tableRows: TableRow[] = groupByTeam
    ? []
    : Array.from(new Map(participants.map(p => [p.member_id, toRow(p)])).values())

  const confirmedCount = participants.filter(p => p.rsvp_status === 'confirmed').length
  const declinedCount = participants.filter(p => p.rsvp_status === 'declined').length
  const maybeCount = participants.filter(p => p.rsvp_status === 'maybe').length

  return (
    <div className="max-w-2xl space-y-4">
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
            <MapsLink venue={g.venue} className="mt-1.5" />
            <RsvpConfigBadges optOut={g.rsvp_opt_out} requireReason={g.rsvp_require_reason} />
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
        sections={sections}
        showAttendanceCol={false}
        attendanceMap={{}}
        attendanceError={null}
        isTrainer={isTrainer}
        showReasonId={showReasonId}
        setShowReasonId={setShowReasonId}
        onToggleAttendance={() => Promise.resolve()}
        onDismissError={() => {}}
        lineupMap={lineupMap}
        onToggleLineup={isTrainer ? saveLineup : undefined}
      />
    </div>
  )
}

interface RowActions {
  showAttendanceCol: boolean
  attendanceMap: Record<number, boolean>
  isTrainer: boolean
  showReasonId: number | null
  setShowReasonId: (id: number | null) => void
  onToggleAttendance: (memberId: number, value: boolean) => Promise<void>
  lineupMap?: Record<number, boolean>
  onToggleLineup?: (memberId: number, value: boolean) => void
}

function colSpan(a: RowActions) {
  return 2 + (a.lineupMap !== undefined ? 1 : 0) + (a.showAttendanceCol ? 1 : 0)
}

function ParticipantRow({ row, a }: { row: TableRow; a: RowActions }) {
  return (
    <Fragment>
      <tr className="border-b border-brand-border-subtle last:border-0 hover:bg-brand-table-select transition-colors">
        <td className="px-4 py-3 text-sm text-brand-text font-medium">
          <span>{row.member_name}</span>
        </td>
        <td className="px-4 py-3">
          <div className="relative group flex items-center gap-1">
            <RsvpIcon status={row.rsvp_status} />
            {row.reason && (
              <>
                <button
                  onClick={() => a.setShowReasonId(row.member_id === a.showReasonId ? null : row.member_id)}
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
        {a.lineupMap !== undefined && (
          <td className="px-4 py-3 text-center">
            {a.onToggleLineup ? (
              <input
                type="checkbox"
                checked={a.lineupMap[row.member_id] ?? false}
                onChange={e => a.onToggleLineup!(row.member_id, e.target.checked)}
                className="w-4 h-4 rounded border-brand-border"
              />
            ) : (
              a.lineupMap[row.member_id]
                ? <Check className="w-4 h-4 text-green-600 mx-auto" />
                : <span className="text-brand-text-muted text-sm">–</span>
            )}
          </td>
        )}
        {a.showAttendanceCol && (
          <td className="px-4 py-3 text-center">
            <input
              type="checkbox"
              checked={a.attendanceMap[row.member_id] ?? false}
              onChange={a.isTrainer ? e => a.onToggleAttendance(row.member_id, e.target.checked) : undefined}
              readOnly={!a.isTrainer}
              className={`w-4 h-4 rounded border-brand-border ${a.isTrainer ? '' : 'cursor-default opacity-60'}`}
            />
          </td>
        )}
      </tr>
      {a.showReasonId === row.member_id && row.reason && (
        <tr className="bg-brand-surface-card">
          <td colSpan={colSpan(a)} className="px-4 pb-2 text-xs text-brand-text-muted italic">
            {row.reason}
          </td>
        </tr>
      )}
    </Fragment>
  )
}

function ResponseTable({ rows, sections, showAttendanceCol, attendanceMap, attendanceError, isTrainer, showReasonId, setShowReasonId, onToggleAttendance, onDismissError, lineupMap, onToggleLineup }: {
  rows: TableRow[]
  sections?: TableSection[]
  showAttendanceCol: boolean
  attendanceMap: Record<number, boolean>
  attendanceError: string | null
  isTrainer: boolean
  showReasonId: number | null
  setShowReasonId: (id: number | null) => void
  onToggleAttendance: (memberId: number, value: boolean) => Promise<void>
  onDismissError: () => void
  lineupMap?: Record<number, boolean>
  onToggleLineup?: (memberId: number, value: boolean) => void
}) {
  const a: RowActions = { showAttendanceCol, attendanceMap, isTrainer, showReasonId, setShowReasonId, onToggleAttendance, lineupMap, onToggleLineup }

  // Wenn keine expliziten Sektionen übergeben werden, fällt die Tabelle auf die
  // bisherige Aufteilung Stammkader / Erweiterter Kader zurück.
  const effectiveSections: TableSection[] = sections ?? [
    { title: null, rows: rows.filter(r => !r.is_extended) },
    { title: 'Erweiterter Kader', rows: rows.filter(r => r.is_extended) },
  ].filter(s => s.rows.length > 0)

  const isEmpty = effectiveSections.every(s => s.rows.length === 0)

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <div className="px-6 py-4 border-b border-brand-border-subtle">
        <h2 className="font-semibold text-brand-text">Teilnahme</h2>
      </div>
      {isEmpty ? (
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
                {lineupMap !== undefined && (
                  <th className="text-brand-text-muted text-xs uppercase px-4 py-3 text-center">Aufstellung</th>
                )}
                {showAttendanceCol && (
                  <th className="text-brand-text-muted text-xs uppercase px-4 py-3 text-center">Anwesend</th>
                )}
              </tr>
            </thead>
            <tbody>
              {effectiveSections.map((section, si) => (
                <Fragment key={section.title ?? `section-${si}`}>
                  {section.title && (
                    <tr className="border-t-2 border-brand-border bg-brand-surface-card">
                      <td colSpan={colSpan(a)}
                          className="px-4 py-2 text-xs font-semibold text-brand-text-muted uppercase tracking-wide">
                        {section.title}
                      </td>
                    </tr>
                  )}
                  {section.rows.map(row => (
                    <ParticipantRow key={row.member_id} row={row} a={a} />
                  ))}
                  {section.hasHidden && (
                    <tr>
                      <td colSpan={colSpan(a)} className="px-4 py-2 text-xs italic text-brand-text-muted">
                        Weitere Mitglieder nicht sichtbar
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
