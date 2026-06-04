import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Check, X, HelpCircle, Dumbbell, Home, MapPin, Calendar, Settings, History } from 'lucide-react'
import { api } from '../lib/api'
import { getEventColors } from '../lib/eventColors'
import { useAuth, hasFunction } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useCompactHeader } from '../hooks/useCompactHeader'


const WEEKDAYS = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']

function fmtDate(iso: string) {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAYS[d.getDay()]} ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.${d.getFullYear()}`
}

interface ChildRSVP {
  member_id: number
  name: string
  rsvp: string | null
}

interface Session {
  id: number
  date: string
  start_time: string
  end_time: string
  location: string
  note: string
  status: 'active' | 'cancelled'
  cancel_reason: string
  team_id: number
  team_name: string
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
  children_rsvp?: ChildRSVP[]
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
  team_ids: number[]
  confirmed_count: number
  declined_count: number
  maybe_count: number
  my_rsvp: string | null
  children_rsvp?: ChildRSVP[]
}

interface Team {
  id: number
  name: string
  is_active: boolean
}

type Termin =
  | { kind: 'training'; data: Session }
  | { kind: 'game'; data: Game }

function sortKey(t: Termin): string {
  if (t.kind === 'training') return t.data.date + 'T' + t.data.start_time
  return t.data.date + 'T' + t.data.time
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
  const { user } = useAuth()
  const navigate = useNavigate()
  const isTrainer = user?.role === 'admin' || hasFunction(user, 'trainer')
  const isParent = user?.isParent === true

  const [termine, setTermine] = useState<Termin[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const [showPast, setShowPast] = useState(false)
  const [loading, setLoading] = useState(true)
  const [rsvpLoading, setRsvpLoading] = useState<string | null>(null)
  const [reasons, setReasons] = useState<Record<string, string>>({})
  const [rsvpErrors, setRsvpErrors] = useState<Record<string, string>>({})
  const [filterTeamId, setFilterTeamId] = useState<number | null>(null)
  const [filterTypes, setFilterTypes] = useState<Set<string>>(new Set(['heim', 'auswärts', 'generisch', 'training']))
  const compact = useCompactHeader(950)

  const toggleType = (type: string) => {
    setFilterTypes(prev => {
      const next = new Set(prev)
      next.has(type) ? next.delete(type) : next.add(type)
      return next
    })
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
  }, [showPast])
  useLiveUpdates((event) => { if (event === 'trainings' || event === 'games') load() })

  const visibleTermine = termine.filter(t => {
    if (t.kind === 'training') {
      if (!filterTypes.has('training')) return false
      if (filterTeamId !== null && t.data.team_id !== filterTeamId) return false
    } else {
      if (!filterTypes.has(t.data.event_type)) return false
      if (filterTeamId !== null && !t.data.team_ids?.includes(filterTeamId)) return false
    }
    return true
  })

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
    } catch {
      setRsvpErrors(prev => ({ ...prev, [`t-${sessionId}`]: 'Fehler beim Speichern. Bitte nochmal versuchen.' }))
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
    } catch {
      setRsvpErrors(prev => ({ ...prev, [`g-${gameId}`]: 'Fehler beim Speichern. Bitte nochmal versuchen.' }))
    } finally {
      setRsvpLoading(null)
    }
  }

  return (
    <div>
      <div className="flex items-center gap-2 mb-6 flex-wrap">
        <h1 className="text-2xl font-bold text-brand-text shrink-0">Termine</h1>
        <div className="flex items-center gap-1.5 flex-1 flex-nowrap min-w-0">
          <select
            value={filterTeamId ?? ''}
            onChange={e => setFilterTeamId(e.target.value === '' ? null : Number(e.target.value))}
            className="border border-brand-border rounded-md px-2 py-1.5 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow min-w-0 shrink"
          >
            <option value="">Alle Teams</option>
            {teams.map(t => (
              <option key={t.id} value={t.id}>{t.name}</option>
            ))}
          </select>
          {([
            ['heim',      'Heim',       <Home className="w-3.5 h-3.5" />],
            ['auswärts',  'Auswärts',   <MapPin className="w-3.5 h-3.5" />],
            ['generisch', 'Sonstiges',  <Calendar className="w-3.5 h-3.5" />],
            ['training',  'Training',   <Dumbbell className="w-3.5 h-3.5" />],
          ] as [string, string, React.ReactNode][]).map(([type, label, icon]) => (
            <button
              key={type}
              onClick={() => toggleType(type)}
              aria-label={label}
              className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium border transition-colors shrink-0 ${compact ? 'px-2' : 'px-3'} ${
                filterTypes.has(type)
                  ? getEventColors(type).filter
                  : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
              }`}
            >
              {icon}
              {!compact && <span>{label}</span>}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-1.5 shrink-0">
          <button
            onClick={() => setShowPast(p => !p)}
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
          {isTrainer && (
            <button
              onClick={() => navigate('/admin/trainings')}
              aria-label="Verwalten"
              className={`flex items-center gap-1 rounded-md py-1.5 text-xs font-medium bg-brand-yellow text-brand-black border border-brand-yellow hover:bg-brand-black hover:text-brand-yellow transition-colors ${compact ? 'px-2' : 'px-3'}`}
            >
              <Settings className="w-3.5 h-3.5" />
              {!compact && <span>Verwalten</span>}
            </button>
          )}
        </div>
      </div>

      {loading ? (
        <p className="text-brand-text-muted text-sm">Laden…</p>
      ) : visibleTermine.length === 0 ? (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
          <Dumbbell className="w-10 h-10 mx-auto mb-3 text-brand-text-subtle" />
          <p className="text-brand-text-muted">Keine Termine vorhanden.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {visibleTermine.map(t => {
            if (t.kind === 'training') {
              const s = t.data
              const key = `t-${s.id}`
              return (
                <div
                  key={key}
                  onClick={isTrainer ? () => navigate(`/termine/training/${s.id}`) : undefined}
                  className={`rounded-xl shadow border-t-4 p-4 transition-shadow ${
                    s.status === 'cancelled'
                      ? 'bg-brand-surface-card border-brand-border opacity-60'
                      : `${getEventColors('training').card.bg} ${getEventColors('training').card.border}`
                  } ${isTrainer ? 'cursor-pointer hover:shadow-md' : ''}`}
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
                        {s.location && (
                          <p className="text-sm text-brand-text-muted mt-0.5">{s.location}</p>
                        )}
                        {s.status === 'cancelled' && s.cancel_reason && (
                          <p className="text-sm text-brand-danger mt-0.5">{s.cancel_reason}</p>
                        )}
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

                  {s.status === 'active' && !isTrainer && (
                    <div className="mt-3 space-y-2" onClick={e => e.stopPropagation()}>
                      {isParent ? (
                        (s.children_rsvp ?? []).map(child => {
                          const childKey = `t-${s.id}-${child.member_id}`
                          const reasonKey = childKey
                          return (
                            <div key={child.member_id} className="space-y-1.5">
                              <span className="text-xs font-medium text-brand-text-muted">{child.name}</span>
                              <div className="flex gap-2">
                                <RsvpButton label="Zusagen" icon={<Check className="w-4 h-4" />} active={child.rsvp === 'confirmed'} activeClass="bg-green-600 text-white border-green-600" disabled={rsvpLoading === childKey} onClick={() => respondTraining(s.id, child.rsvp === 'confirmed' ? 'maybe' : 'confirmed', '', child.member_id)} />
                                <RsvpButton label="Vielleicht" icon={<HelpCircle className="w-4 h-4" />} active={child.rsvp === 'maybe'} activeClass="bg-brand-yellow text-brand-black border-brand-yellow" disabled={rsvpLoading === childKey} onClick={() => respondTraining(s.id, 'maybe', reasons[reasonKey] ?? '', child.member_id)} />
                                <RsvpButton label="Absagen" icon={<X className="w-4 h-4" />} active={child.rsvp === 'declined'} activeClass="bg-brand-danger text-white border-brand-danger" disabled={rsvpLoading === childKey} onClick={() => respondTraining(s.id, 'declined', reasons[reasonKey] ?? '', child.member_id)} />
                              </div>
                              <input type="text" placeholder="Begründung für Absage / Vielleicht (optional)" value={reasons[reasonKey] ?? ''} onChange={e => setReasons(prev => ({ ...prev, [reasonKey]: e.target.value }))} className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
                            </div>
                          )
                        })
                      ) : (
                        <>
                          <div className="flex gap-2">
                            <RsvpButton label="Zusagen" icon={<Check className="w-4 h-4" />} active={s.my_rsvp === 'confirmed'} activeClass="bg-green-600 text-white border-green-600" disabled={rsvpLoading === key} onClick={() => respondTraining(s.id, s.my_rsvp === 'confirmed' ? 'maybe' : 'confirmed')} />
                            <RsvpButton label="Vielleicht" icon={<HelpCircle className="w-4 h-4" />} active={s.my_rsvp === 'maybe'} activeClass="bg-brand-yellow text-brand-black border-brand-yellow" disabled={rsvpLoading === key} onClick={() => respondTraining(s.id, 'maybe', reasons[key] ?? '')} />
                            <RsvpButton label="Absagen" icon={<X className="w-4 h-4" />} active={s.my_rsvp === 'declined'} activeClass="bg-brand-danger text-white border-brand-danger" disabled={rsvpLoading === key} onClick={() => respondTraining(s.id, 'declined', reasons[key] ?? '')} />
                          </div>
                          <input type="text" placeholder="Begründung für Absage / Vielleicht (optional)" value={reasons[key] ?? ''} onChange={e => setReasons(prev => ({ ...prev, [key]: e.target.value }))} className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
                        </>
                      )}
                      {rsvpErrors[key] && (
                        <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{rsvpErrors[key]}</p>
                      )}
                    </div>
                  )}
                </div>
              )
            }

            // Game card
            const g = t.data
            const key = `g-${g.id}`
            const Icon = g.event_type === 'generisch' ? Calendar : (g.is_home ? Home : MapPin)
            const label = g.event_type === 'generisch'
              ? g.opponent
              : (g.is_home ? `Heim: ${g.opponent}` : `Auswärts: ${g.opponent}`)
            return (
              <div
                key={key}
                onClick={isTrainer ? () => navigate(`/termine/spiel/${g.id}`) : undefined}
                className={`rounded-xl shadow border-t-4 p-4 transition-shadow ${getEventColors(g.event_type).card.bg} ${getEventColors(g.event_type).card.border} ${isTrainer ? 'cursor-pointer hover:shadow-md' : ''}`}
              >
                <div className="flex items-start justify-between gap-4 flex-wrap">
                  <div className="flex items-start gap-3 min-w-0">
                    <Icon className={`w-5 h-5 mt-0.5 shrink-0 ${getEventColors(g.event_type).card.icon}`} />
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="font-semibold text-brand-text">{fmtDate(g.date)}</span>
                        <span className="text-brand-text-muted text-sm">{g.time} Uhr</span>
                        {g.team_names && (
                          <span className="text-brand-text-subtle text-xs">
                            {g.team_ids && g.team_ids.length > 1 ? 'Mehrere Teams' : g.team_names}
                          </span>
                        )}
                      </div>
                      <p className="text-sm text-brand-text-muted mt-0.5">{label}</p>
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

                {!isTrainer && (
                  <div className="mt-3 space-y-2" onClick={e => e.stopPropagation()}>
                    {isParent ? (
                      (g.children_rsvp ?? []).map(child => {
                        const childKey = `g-${g.id}-${child.member_id}`
                        const reasonKey = childKey
                        return (
                          <div key={child.member_id} className="space-y-1.5">
                            <span className="text-xs font-medium text-brand-text-muted">{child.name}</span>
                            <div className="flex gap-2">
                              <RsvpButton label="Zusagen" icon={<Check className="w-4 h-4" />} active={child.rsvp === 'confirmed'} activeClass="bg-green-600 text-white border-green-600" disabled={rsvpLoading === childKey} onClick={() => respondGame(g.id, child.rsvp === 'confirmed' ? 'maybe' : 'confirmed', '', child.member_id)} />
                              <RsvpButton label="Vielleicht" icon={<HelpCircle className="w-4 h-4" />} active={child.rsvp === 'maybe'} activeClass="bg-brand-yellow text-brand-black border-brand-yellow" disabled={rsvpLoading === childKey} onClick={() => respondGame(g.id, 'maybe', reasons[reasonKey] ?? '', child.member_id)} />
                              <RsvpButton label="Absagen" icon={<X className="w-4 h-4" />} active={child.rsvp === 'declined'} activeClass="bg-brand-danger text-white border-brand-danger" disabled={rsvpLoading === childKey} onClick={() => respondGame(g.id, 'declined', reasons[reasonKey] ?? '', child.member_id)} />
                            </div>
                            <input type="text" placeholder="Begründung für Absage / Vielleicht (optional)" value={reasons[reasonKey] ?? ''} onChange={e => setReasons(prev => ({ ...prev, [reasonKey]: e.target.value }))} className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
                          </div>
                        )
                      })
                    ) : (
                      <>
                        <div className="flex gap-2">
                          <RsvpButton label="Zusagen" icon={<Check className="w-4 h-4" />} active={g.my_rsvp === 'confirmed'} activeClass="bg-green-600 text-white border-green-600" disabled={rsvpLoading === key} onClick={() => respondGame(g.id, g.my_rsvp === 'confirmed' ? 'maybe' : 'confirmed')} />
                          <RsvpButton label="Vielleicht" icon={<HelpCircle className="w-4 h-4" />} active={g.my_rsvp === 'maybe'} activeClass="bg-brand-yellow text-brand-black border-brand-yellow" disabled={rsvpLoading === key} onClick={() => respondGame(g.id, 'maybe', reasons[key] ?? '')} />
                          <RsvpButton label="Absagen" icon={<X className="w-4 h-4" />} active={g.my_rsvp === 'declined'} activeClass="bg-brand-danger text-white border-brand-danger" disabled={rsvpLoading === key} onClick={() => respondGame(g.id, 'declined', reasons[key] ?? '')} />
                        </div>
                        <input type="text" placeholder="Begründung für Absage / Vielleicht (optional)" value={reasons[key] ?? ''} onChange={e => setReasons(prev => ({ ...prev, [key]: e.target.value }))} className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow" />
                      </>
                    )}
                    {rsvpErrors[key] && (
                      <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{rsvpErrors[key]}</p>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
