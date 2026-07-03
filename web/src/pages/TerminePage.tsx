import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Check, X, HelpCircle, Dumbbell, Home, Plane, Calendar, History } from 'lucide-react'
import EventTypeFilter, { type EventTypeFilterEntry } from '../components/EventTypeFilter'
import { api } from '../lib/api'
import MapsLink from '../components/MapsLink'
import EventNoteIndicator from '../components/EventNoteIndicator'
import { type RsvpDefault } from '../components/RsvpDefaultsEditor'
import { getEventColors } from '../lib/eventColors'
import { buildTeamShortNames } from '../lib/teamName'
import { useAuth } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useCompactHeader } from '../hooks/useCompactHeader'


const WEEKDAYS = ['Sonntag', 'Montag', 'Dienstag', 'Mittwoch', 'Donnerstag', 'Freitag', 'Samstag']

function fmtDate(iso: string) {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAYS[d.getDay()]}, ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}`
}

interface ChildRSVP {
  member_id: number
  name: string
  rsvp: string | null
}

interface VenueRef {
  id: number
  name: string
  street: string
  city: string
  postal_code: string
  note: string
}

interface Session {
  id: number
  date: string
  start_time: string
  end_time: string
  venue?: VenueRef | null
  note: string
  status: 'active' | 'cancelled'
  cancel_reason: string
  team_id: number
  team_name: string
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
  my_rsvp_is_default?: boolean
  children_rsvp?: ChildRSVP[]
  rsvp_default_players: RsvpDefault
  rsvp_default_extended: RsvpDefault
  rsvp_require_reason: number
  rsvp_locks_at?: string
}

interface Game {
  id: number
  date: string
  time: string
  opponent: string
  event_type: string
  is_home: boolean
  season_id: number
  team_names: string
  team_display_short_csv?: string
  team_display_long_csv?: string
  team_ids: number[]
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
  my_rsvp_is_default?: boolean
  children_rsvp?: ChildRSVP[]
  rsvp_default_players: RsvpDefault
  rsvp_default_extended: RsvpDefault
  rsvp_require_reason: number
  rsvp_locks_at?: string
  venue?: VenueRef | null
  note?: string
}

interface Team {
  id: number
  name: string
  age_class: string
  gender: string
  team_number: number
  group_count: number
  is_active: boolean
}

type Termin =
  | { kind: 'training'; data: Session }
  | { kind: 'game'; data: Game }

const ALL_TYPES = new Set(['heim', 'auswärts', 'generisch', 'training'])

function parseFilters(sp: URLSearchParams) {
  const team = parseInt(sp.get('team') ?? '') || null
  const typesRaw = sp.get('types')
  const types = typesRaw
    ? (() => {
        const parsed = new Set(typesRaw.split(',').filter(t => ALL_TYPES.has(t)))
        return parsed.size > 0 ? parsed : new Set(ALL_TYPES)
      })()
    : new Set(ALL_TYPES)
  const past = sp.get('past') === '1'
  const focusRaw = sp.get('focus')
  const focusMatch = focusRaw?.match(/^(training|game)-(\d+)$/)
  const focus = focusMatch ? { kind: focusMatch[1] as 'training' | 'game', id: parseInt(focusMatch[2]) } : null
  return { team, types, past, focus }
}

function sortKey(t: Termin): string {
  if (t.kind === 'training') return t.data.date + 'T' + t.data.start_time
  return t.data.date + 'T' + t.data.time
}

function fmtClockTime(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return new Intl.DateTimeFormat('de-DE', { hour: '2-digit', minute: '2-digit' }).format(d)
}

function RsvpLockNotice({ locksAt, locked }: { locksAt?: string; locked: boolean }) {
  if (!locksAt) return null
  if (locked) {
    return (
      <p className="text-xs text-brand-text-muted">
        Änderungen nur noch beim Trainer möglich.
      </p>
    )
  }
  return (
    <p className="text-xs text-brand-text-subtle">
      Bis {fmtClockTime(locksAt)} Uhr änderbar.
    </p>
  )
}

function RsvpButton({ label, icon, active, activeClass, disabled, onClick }: {
  label: string
  icon: React.ReactNode
  active: boolean
  activeClass: string
  disabled: boolean
  onClick: () => void
}) {
  return (
    <button
      disabled={disabled}
      onClick={onClick}
      className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 sm:py-1 text-xs font-medium border transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
        active
          ? activeClass
          : 'bg-white border-brand-border text-brand-text-muted hover:border-brand-text hover:text-brand-text'
      }`}
    >
      {icon}
      <span className="hidden sm:inline">{label}</span>
    </button>
  )
}

export default function TerminePage() {
  const { user, hasCapability } = useAuth()
  const navigate = useNavigate()
  const isTrainer = hasCapability('manage_trainings')
  const isParent = user?.isParent === true
  // Override für RSVP-Cutoff: admin/vorstand/trainer/sportliche_leitung dürfen
  // jederzeit pflegen. `manage_games` deckt genau diese vier ab.
  const canOverrideRsvpCutoff = hasCapability('manage_games')

  const [searchParams, setSearchParams] = useSearchParams()
  const { team: filterTeamId, types: filterTypes, past: showPast, focus } = parseFilters(searchParams)
  const triedPastExpansion = useRef(false)
  const scrollToTodayRef = useRef(false)

  const [termine, setTermine] = useState<Termin[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const teamShortNames = useMemo(() => buildTeamShortNames(teams), [teams])
  const [loading, setLoading] = useState(true)
  const [rsvpLoading, setRsvpLoading] = useState<string | null>(null)
  const [rsvpErrors, setRsvpErrors] = useState<Record<string, string>>({})
  const [pendingRSVP, setPendingRSVP] = useState<{ kind: 'training' | 'game'; id: number; status: 'declined' | 'maybe'; memberId?: number } | null>(null)
  const [modalReason, setModalReason] = useState('')
  const compact = useCompactHeader(950)
  const TERMINE_TYPES: EventTypeFilterEntry[] = [
    ['heim',      'Heim',       <Home className="w-3.5 h-3.5" />],
    ['auswärts',  'Auswärts',   <Plane className="w-3.5 h-3.5" />],
    ['generisch', 'Sonstiges',  <Calendar className="w-3.5 h-3.5" />],
    ['training',  'Training',   <Dumbbell className="w-3.5 h-3.5" />],
  ]

  const updateFilter = (patch: { team?: number | null; types?: Set<string>; past?: boolean }) => {
    const next = new URLSearchParams(searchParams)
    if ('team' in patch) {
      if (patch.team === null) next.delete('team')
      else next.set('team', String(patch.team))
    }
    if ('types' in patch && patch.types) {
      const isDefault = patch.types.size === ALL_TYPES.size && [...ALL_TYPES].every(t => patch.types!.has(t))
      if (isDefault) next.delete('types')
      else next.set('types', [...patch.types].join(','))
    }
    if ('past' in patch) {
      if (patch.past) next.set('past', '1')
      else next.delete('past')
    }
    setSearchParams(next, { replace: true })
  }

  const toggleType = (type: string) => {
    const next = new Set(filterTypes)
    if (next.has(type)) next.delete(type); else next.add(type)
    updateFilter({ types: next })
  }

  const today = new Date().toISOString().slice(0, 10)
  const from = showPast
    ? new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
    : today
  const to = new Date(Date.now() + 180 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)

  const load = () => {
    setLoading(true)
    Promise.all([
      api.get(`/training-sessions?from=${from}&to=${to}`),
      api.get(`/games/my?from=${from}&to=${to}`),
    ])
      .then(([trainingsRes, gamesRes]) => {
        const trainings: Termin[] = (trainingsRes.data ?? []).map((s: Session) => ({ kind: 'training' as const, data: s }))
        const games: Termin[] = (gamesRes.data ?? []).map((g: Game) => ({ kind: 'game' as const, data: g }))
        const merged = [...trainings, ...games].sort((a, b) => sortKey(a).localeCompare(sortKey(b)))
        setTermine(merged)
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
    api.get('/teams').then(r => setTeams(Array.isArray(r.data) ? r.data : (r.data?.teams ?? []))).catch(() => {})
    // load kapselt from/to (aus showPast), soll nur bei showPast-Wechsel neu laufen
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showPast])
  useLiveUpdates((event) => { if (event === 'trainings' || event === 'games' || event === 'event-note') load() })

  const visibleTermine = termine.filter(t => {
    if (focus && t.kind === focus.kind && t.data.id === focus.id) return true
    if (t.kind === 'training') {
      if (!filterTypes.has('training')) return false
      if (filterTeamId !== null && t.data.team_id !== filterTeamId) return false
    } else {
      if (!filterTypes.has(t.data.event_type)) return false
      if (filterTeamId !== null && !t.data.team_ids?.includes(filterTeamId)) return false
    }
    return true
  })

  // Index des ersten nicht-vergangenen Termins (date >= today). Die „heute"-Trennlinie
  // wird nur davor gerendert, wenn mind. ein vergangener Termin darüber steht (> 0) —
  // sonst (alle sichtbaren Termine liegen in Gegenwart/Zukunft) erschiene sie redundant
  // ganz oben.
  const todayIdx = visibleTermine.findIndex(t => t.data.date.slice(0, 10) >= today)
  const showTodayDivider = todayIdx > 0

  const focusNotFound = !loading && !!focus && showPast && !termine.some(t => t.kind === focus.kind && t.data.id === focus.id)

  useEffect(() => {
    if (!focus || loading || showPast || triedPastExpansion.current) return
    const found = termine.some(t => t.kind === focus.kind && t.data.id === focus.id)
    if (!found) {
      triedPastExpansion.current = true
      updateFilter({ past: true })
    }
    // Soll nur bei Fokus-Wechsel / nach dem Laden prüfen; termine/showPast/updateFilter bewusst ausgelassen, um Re-Trigger zu vermeiden
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focus?.kind, focus?.id, loading])

  useEffect(() => { triedPastExpansion.current = false }, [focus?.kind, focus?.id])

  // Nach Toggle „Vergangene" scrollt die Seite ohne dieses Zutun nach oben, weil das
  // Loading-Placeholder die Liste kurz aus dem DOM nimmt. Der „Vergangene"-Button ist
  // nur oben (bei „heute") erreichbar, deshalb springen wir nach dem Reload einfach
  // zur „heute"-Trennlinie — bzw. an den Listenanfang, wenn keine Trennlinie existiert
  // (kein vergangener Termin davor). Focus-Scroll hat Vorrang.
  // Wichtig: Der Scrollcontainer ist das <main> in AppShell (overflow-auto), nicht
  // das window.
  useEffect(() => {
    if (loading || focus) return
    if (!scrollToTodayRef.current) return
    scrollToTodayRef.current = false
    const divider = document.getElementById('today-divider')
    if (divider) {
      divider.scrollIntoView({ behavior: 'auto', block: 'center' })
      return
    }
    document.querySelector('main')?.scrollTo({ top: 0, behavior: 'auto' })
  }, [loading, focus])

  const togglePast = () => {
    scrollToTodayRef.current = true
    updateFilter({ past: !showPast })
  }

  useEffect(() => {
    if (!focus || loading) return
    const el = document.getElementById(`termin-${focus.kind}-${focus.id}`)
    if (!el) return
    el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    el.classList.add('ring-2', 'ring-brand-yellow', 'transition-all')
    const t = setTimeout(() => el.classList.remove('ring-2', 'ring-brand-yellow'), 2000)
    return () => clearTimeout(t)
    // focus über kind/id abgedeckt; nicht das ganze focus-Objekt als Dep, soll nur bei dessen Identität scrollen
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focus?.kind, focus?.id, loading, visibleTermine.length])

  const extractRsvpError = (err: unknown): string => {
    const e = err as { response?: { data?: { error?: string; message?: string } } }
    if (e?.response?.data?.error === 'rsvp_locked' && e.response.data.message) {
      return e.response.data.message
    }
    return 'Fehler beim Speichern. Bitte nochmal versuchen.'
  }

  const respondTraining = async (sessionId: number, status: string, reason = '', memberId?: number) => {
    const key = memberId ? `t-${sessionId}-${memberId}` : `t-${sessionId}`
    setRsvpLoading(key)
    setRsvpErrors(prev => { const n = { ...prev }; delete n[`t-${sessionId}`]; return n })
    try {
      await api.post(`/training-sessions/${sessionId}/respond`, { status, reason, ...(memberId ? { member_id: memberId } : {}) })
      setTermine(prev => prev.map(t => {
        if (t.kind !== 'training' || t.data.id !== sessionId) return t
        if (memberId) {
          return { ...t, data: { ...t.data, children_rsvp: (t.data.children_rsvp ?? []).map(c => c.member_id === memberId ? { ...c, rsvp: status } : c) } }
        }
        return { ...t, data: { ...t.data, my_rsvp: status } }
      }))
    } catch (err) {
      setRsvpErrors(prev => ({ ...prev, [`t-${sessionId}`]: extractRsvpError(err) }))
    } finally {
      setRsvpLoading(null)
    }
  }

  const respondGame = async (gameId: number, status: string, reason = '', memberId?: number) => {
    const key = memberId ? `g-${gameId}-${memberId}` : `g-${gameId}`
    setRsvpLoading(key)
    setRsvpErrors(prev => { const n = { ...prev }; delete n[`g-${gameId}`]; return n })
    try {
      await api.post(`/games/${gameId}/respond`, { status, reason, ...(memberId ? { member_id: memberId } : {}) })
      setTermine(prev => prev.map(t => {
        if (t.kind !== 'game' || t.data.id !== gameId) return t
        if (memberId) {
          return { ...t, data: { ...t.data, children_rsvp: (t.data.children_rsvp ?? []).map(c => c.member_id === memberId ? { ...c, rsvp: status } : c) } }
        }
        return { ...t, data: { ...t.data, my_rsvp: status } }
      }))
    } catch (err) {
      setRsvpErrors(prev => ({ ...prev, [`g-${gameId}`]: extractRsvpError(err) }))
    } finally {
      setRsvpLoading(null)
    }
  }

  const openReasonModal = (kind: 'training' | 'game', id: number, status: 'declined' | 'maybe', memberId?: number) => {
    setModalReason('')
    setPendingRSVP({ kind, id, status, memberId })
  }

  const confirmModal = () => {
    if (!pendingRSVP) return
    const { kind, id, status, memberId } = pendingRSVP
    setPendingRSVP(null)
    if (kind === 'training') {
      respondTraining(id, status, modalReason, memberId)
    } else {
      respondGame(id, status, modalReason, memberId)
    }
    setModalReason('')
  }

  const cancelModal = () => {
    setPendingRSVP(null)
    setModalReason('')
  }

  const pendingChildName = (() => {
    if (!pendingRSVP?.memberId) return null
    const termin = termine.find(t => t.kind === pendingRSVP.kind && t.data.id === pendingRSVP.id)
    if (!termin) return null
    return (termin.data.children_rsvp ?? []).find(c => c.member_id === pendingRSVP.memberId)?.name ?? null
  })()

  return (
    <div>
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-bold text-brand-text shrink-0">Termine</h1>
        <div className="flex items-center gap-1.5 flex-1 flex-nowrap min-w-0">
          <select
            value={filterTeamId ?? ''}
            onChange={e => updateFilter({ team: e.target.value === '' ? null : Number(e.target.value) })}
            className="border border-brand-border rounded-md px-2 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-24 shrink-0"
          >
            <option value="">Teams</option>
            {teams.map(t => (
              <option key={t.id} value={t.id}>{teamShortNames.get(t.id) ?? t.name}</option>
            ))}
          </select>
          <EventTypeFilter
            types={TERMINE_TYPES}
            active={filterTypes}
            onToggle={toggleType}
            compact={compact}
            ariaLabel="Termin-Typ-Filter"
          />
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <button
            onClick={togglePast}
            aria-label="Vergangene anzeigen"
            className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium border transition-colors ${compact ? 'px-2' : 'px-3'} ${
              showPast
                ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
            }`}
          >
            <History className="w-3.5 h-3.5" />
            {!compact && <span>Vergangene</span>}
          </button>
        </div>
      </div>

      {focusNotFound && (
        <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
          Dieser Termin ist nicht verfügbar.
        </div>
      )}

      {loading ? (
        <p className="text-brand-text-muted text-sm">Laden…</p>
      ) : visibleTermine.length === 0 ? (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
          <Dumbbell className="w-10 h-10 mx-auto mb-3 text-brand-text-subtle" />
          <p className="text-brand-text-muted">Keine Termine vorhanden.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {(() => {
          const cards = visibleTermine.map(t => {
            if (t.kind === 'training') {
              const s = t.data
              const key = `t-${s.id}`
              return (
                <div
                  id={`termin-training-${s.id}`}
                  key={key}
                  onClick={() => navigate(`/termine/training/${s.id}`)}
                  className={`rounded-xl shadow border-t-4 p-4 transition-shadow cursor-pointer hover:shadow-md ${
                    s.status === 'cancelled'
                      ? 'bg-brand-surface-card border-brand-border opacity-60'
                      : `${getEventColors('training').card.bg} ${getEventColors('training').card.border}`
                  }`}
                >
                  <div className="flex items-start justify-between gap-4 flex-wrap">
                    <div className="flex items-start gap-3 min-w-0">
                      <Dumbbell className={`w-5 h-5 mt-0.5 shrink-0 ${s.status === 'cancelled' ? 'text-brand-text-muted' : getEventColors('training').card.icon}`} />
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className={`font-semibold text-brand-text ${s.status === 'cancelled' ? 'line-through' : ''}`}>
                            {fmtDate(s.date)}
                          </span>
                          <span className="text-brand-text-muted text-sm">{s.start_time} – {s.end_time}</span>
                          {s.team_name && (
                            <span className="text-brand-text-subtle text-xs">{s.team_name}</span>
                          )}
                          {s.status === 'cancelled' && (
                            <span className="bg-brand-danger-light text-brand-danger text-xs font-medium px-2 py-0.5 rounded-full">
                              Abgesagt
                            </span>
                          )}
                        </div>
                        <span onClick={e => e.stopPropagation()}>
                          <MapsLink venue={s.venue} className="mt-0.5" />
                        </span>
                        {s.status === 'cancelled' && s.cancel_reason && (
                          <p className="text-sm text-brand-danger mt-0.5">{s.cancel_reason}</p>
                        )}
                        <EventNoteIndicator variant="inline" note={s.note ?? ''} className="mt-0.5" />
                      </div>
                    </div>

                    {s.status === 'active' && (
                      <div className="flex items-center gap-1 shrink-0">
                        <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                          <Check className="w-3 h-3 text-green-600" />{s.confirmed_count}
                        </span>
                        <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                          <X className="w-3 h-3 text-brand-danger" />{s.declined_count}
                        </span>
                        <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                          <HelpCircle className="w-3 h-3 text-brand-text-subtle" />{s.maybe_count}
                        </span>
                      </div>
                    )}
                  </div>

                  {s.status === 'active' && (!isTrainer || s.my_rsvp !== null) && (() => {
                    const cutoffLocked = !canOverrideRsvpCutoff && !!s.rsvp_locks_at && Date.now() >= new Date(s.rsvp_locks_at).getTime()
                    return (
                      <div className="mt-3 space-y-2" onClick={e => e.stopPropagation()}>
                        {isParent ? (
                          (s.children_rsvp ?? []).map(child => {
                            const childKey = `t-${s.id}-${child.member_id}`
                            const handleChildDecline = (status: 'declined' | 'maybe') =>
                              s.rsvp_require_reason
                                ? openReasonModal('training', s.id, status, child.member_id)
                                : respondTraining(s.id, status, '', child.member_id)
                            return (
                              <div key={child.member_id} className="space-y-1.5">
                                <span className="text-xs font-medium text-brand-text-muted">{child.name}</span>
                                <div className="flex gap-2">
                                  <RsvpButton label="Zusagen" icon={<Check className="w-4 h-4" />} active={child.rsvp === 'confirmed'} activeClass="bg-green-600 text-white border-green-600" disabled={cutoffLocked || rsvpLoading === childKey} onClick={() => respondTraining(s.id, 'confirmed', '', child.member_id)} />
                                  <RsvpButton label="Vielleicht" icon={<HelpCircle className="w-4 h-4" />} active={child.rsvp === 'maybe'} activeClass="bg-brand-yellow text-brand-black border-brand-yellow" disabled={cutoffLocked || rsvpLoading === childKey} onClick={() => handleChildDecline('maybe')} />
                                  <RsvpButton label="Absagen" icon={<X className="w-4 h-4" />} active={child.rsvp === 'declined'} activeClass="bg-brand-danger text-white border-brand-danger" disabled={cutoffLocked || rsvpLoading === childKey} onClick={() => handleChildDecline('declined')} />
                                </div>
                              </div>
                            )
                          })
                        ) : (
                          <div className="flex gap-2">
                            <RsvpButton label="Zusagen" icon={<Check className="w-4 h-4" />} active={s.my_rsvp === 'confirmed'} activeClass="bg-green-600 text-white border-green-600" disabled={cutoffLocked || rsvpLoading === key} onClick={() => respondTraining(s.id, s.my_rsvp_is_default ? 'confirmed' : (s.my_rsvp === 'confirmed' ? 'maybe' : 'confirmed'))} />
                            <RsvpButton label="Vielleicht" icon={<HelpCircle className="w-4 h-4" />} active={s.my_rsvp === 'maybe'} activeClass="bg-brand-yellow text-brand-black border-brand-yellow" disabled={cutoffLocked || rsvpLoading === key} onClick={() => s.rsvp_require_reason ? openReasonModal('training', s.id, 'maybe') : respondTraining(s.id, 'maybe')} />
                            <RsvpButton label="Absagen" icon={<X className="w-4 h-4" />} active={s.my_rsvp === 'declined'} activeClass="bg-brand-danger text-white border-brand-danger" disabled={cutoffLocked || rsvpLoading === key} onClick={() => s.rsvp_require_reason ? openReasonModal('training', s.id, 'declined') : respondTraining(s.id, 'declined')} />
                          </div>
                        )}
                        {!canOverrideRsvpCutoff && <RsvpLockNotice locksAt={s.rsvp_locks_at} locked={cutoffLocked} />}
                        {rsvpErrors[key] && (
                          <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{rsvpErrors[key]}</p>
                        )}
                      </div>
                    )
                  })()}
                </div>
              )
            }

            // Game card
            const g = t.data
            const key = `g-${g.id}`
            const Icon = g.event_type === 'heim' ? Home : g.event_type === 'auswärts' ? Plane : Calendar
            const label = g.event_type === 'generisch'
              ? g.opponent
              : (g.event_type === 'heim' ? `Heim: ${g.opponent}` : `Auswärts: ${g.opponent}`)
            return (
              <div
                id={`termin-game-${g.id}`}
                key={key}
                onClick={() => navigate(`/termine/${g.event_type === 'generisch' ? 'ereignis' : 'spiel'}/${g.id}`)}
                className={`rounded-xl shadow border-t-4 p-4 transition-shadow cursor-pointer hover:shadow-md ${getEventColors(g.event_type).card.bg} ${getEventColors(g.event_type).card.border}`}
              >
                <div className="flex items-start justify-between gap-4 flex-wrap">
                  <div className="flex items-start gap-3 min-w-0">
                    <Icon className={`w-5 h-5 mt-0.5 shrink-0 ${getEventColors(g.event_type).card.icon}`} />
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-semibold text-brand-text">{fmtDate(g.date)}</span>
                        <span className="text-brand-text-muted text-sm">{g.time} Uhr</span>
                        {(g.team_display_short_csv || g.team_names) && (
                          <span className="text-brand-text-subtle text-xs">
                            {g.team_display_short_csv || g.team_names}
                          </span>
                        )}
                      </div>
                      <p className="text-sm text-brand-text-muted mt-0.5">{label}</p>
                      <span onClick={e => e.stopPropagation()}>
                        <MapsLink venue={g.venue} className="mt-0.5" />
                      </span>
                      <EventNoteIndicator variant="inline" note={g.note ?? ''} className="mt-0.5" />
                    </div>
                  </div>

                  <div className="flex items-center gap-1 shrink-0">
                    <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                      <Check className="w-3 h-3 text-green-600" />{g.confirmed_count}
                    </span>
                    <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                      <X className="w-3 h-3 text-brand-danger" />{g.declined_count}
                    </span>
                    <span className="text-xs text-brand-text-muted bg-white border border-brand-border-subtle rounded px-2 py-1 flex items-center gap-1">
                      <HelpCircle className="w-3 h-3 text-brand-text-subtle" />{g.maybe_count}
                    </span>
                  </div>
                </div>

                {(!isTrainer || g.my_rsvp !== null) && (() => {
                  const cutoffLocked = !canOverrideRsvpCutoff && !!g.rsvp_locks_at && Date.now() >= new Date(g.rsvp_locks_at).getTime()
                  return (
                    <div className="mt-3 space-y-2" onClick={e => e.stopPropagation()}>
                      {isParent ? (
                        (g.children_rsvp ?? []).map(child => {
                          const childKey = `g-${g.id}-${child.member_id}`
                          const handleChildDecline = (status: 'declined' | 'maybe') =>
                            g.rsvp_require_reason
                              ? openReasonModal('game', g.id, status, child.member_id)
                              : respondGame(g.id, status, '', child.member_id)
                          return (
                            <div key={child.member_id} className="space-y-1.5">
                              <span className="text-xs font-medium text-brand-text-muted">{child.name}</span>
                              <div className="flex gap-2">
                                <RsvpButton label="Zusagen" icon={<Check className="w-4 h-4" />} active={child.rsvp === 'confirmed'} activeClass="bg-green-600 text-white border-green-600" disabled={cutoffLocked || rsvpLoading === childKey} onClick={() => respondGame(g.id, 'confirmed', '', child.member_id)} />
                                <RsvpButton label="Vielleicht" icon={<HelpCircle className="w-4 h-4" />} active={child.rsvp === 'maybe'} activeClass="bg-brand-yellow text-brand-black border-brand-yellow" disabled={cutoffLocked || rsvpLoading === childKey} onClick={() => handleChildDecline('maybe')} />
                                <RsvpButton label="Absagen" icon={<X className="w-4 h-4" />} active={child.rsvp === 'declined'} activeClass="bg-brand-danger text-white border-brand-danger" disabled={cutoffLocked || rsvpLoading === childKey} onClick={() => handleChildDecline('declined')} />
                              </div>
                            </div>
                          )
                        })
                      ) : (
                        <div className="flex gap-2">
                          <RsvpButton label="Zusagen" icon={<Check className="w-4 h-4" />} active={g.my_rsvp === 'confirmed'} activeClass="bg-green-600 text-white border-green-600" disabled={cutoffLocked || rsvpLoading === key} onClick={() => respondGame(g.id, g.my_rsvp_is_default ? 'confirmed' : (g.my_rsvp === 'confirmed' ? 'maybe' : 'confirmed'))} />
                          <RsvpButton label="Vielleicht" icon={<HelpCircle className="w-4 h-4" />} active={g.my_rsvp === 'maybe'} activeClass="bg-brand-yellow text-brand-black border-brand-yellow" disabled={cutoffLocked || rsvpLoading === key} onClick={() => g.rsvp_require_reason ? openReasonModal('game', g.id, 'maybe') : respondGame(g.id, 'maybe')} />
                          <RsvpButton label="Absagen" icon={<X className="w-4 h-4" />} active={g.my_rsvp === 'declined'} activeClass="bg-brand-danger text-white border-brand-danger" disabled={cutoffLocked || rsvpLoading === key} onClick={() => g.rsvp_require_reason ? openReasonModal('game', g.id, 'declined') : respondGame(g.id, 'declined')} />
                        </div>
                      )}
                      {!canOverrideRsvpCutoff && <RsvpLockNotice locksAt={g.rsvp_locks_at} locked={cutoffLocked} />}
                      {rsvpErrors[key] && (
                        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{rsvpErrors[key]}</p>
                      )}
                    </div>
                  )
                })()}
              </div>
            )
          })
          if (!showTodayDivider) return cards
          const todayDivider = (
            <div key="today-divider" id="today-divider" className="flex items-center gap-3 py-1" aria-hidden="true">
              <span className="flex-1 border-t border-brand-border-subtle" />
              <span className="text-brand-text-muted text-xs uppercase tracking-wide">heute</span>
              <span className="flex-1 border-t border-brand-border-subtle" />
            </div>
          )
          return [...cards.slice(0, todayIdx), todayDivider, ...cards.slice(todayIdx)]
          })()}
        </div>
      )}

      {pendingRSVP && (
        <div className="fixed inset-0 z-50 bg-black/40 flex items-center justify-center p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
            <h2 className="text-base font-semibold text-brand-text mb-1">
              {pendingRSVP.status === 'declined' ? 'Absagen' : 'Vielleicht'}
              {pendingChildName && <span className="font-normal text-brand-text-muted"> – {pendingChildName}</span>}
            </h2>
            <p className="text-sm text-brand-text-muted mb-4">Bitte gib eine Begründung an.</p>
            <textarea
              autoFocus
              rows={3}
              value={modalReason}
              onChange={e => setModalReason(e.target.value)}
              placeholder="Begründung…"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow resize-none"
            />
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={cancelModal}
                className="rounded-md px-4 py-2 text-sm font-medium border border-brand-border text-brand-text-muted hover:border-brand-text hover:text-brand-text transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={confirmModal}
                disabled={modalReason.trim() === ''}
                className={`rounded-md px-4 py-2 text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${
                  pendingRSVP.status === 'declined'
                    ? 'bg-brand-danger text-white hover:bg-brand-danger/90'
                    : 'bg-brand-yellow text-brand-black hover:bg-brand-black hover:text-brand-yellow'
                }`}
              >
                OK
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
