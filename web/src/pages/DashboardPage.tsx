import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Calendar, BarChart2, Users, Car, ArrowRight,
  Home, Plane, Dumbbell, ChevronDown, ChevronRight, Check
} from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useMediaQuery } from '../lib/useMediaQuery'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

// ── Types ─────────────────────────────────────────────────────────────────────

interface Season { id: number; name: string; isActive: boolean }

interface NextEvent {
  id: number
  eventType: 'training' | 'spiel'
  date: string
  time: string
  title: string
  teamName: string
  detailUrl: string
}

interface DiensteSlot {
  dutyTypeName: string
  eventTime: string
}

interface NextDiensteGame {
  id: number
  date: string
  opponent: string
}

interface RecentAssignment { date: string; dutyType: string; status: string }

interface DutyAccount {
  season: string
  ist: number
  soll: number | null
  children: number
  recentAssignments: RecentAssignment[]
}

interface MeineDienste {
  nextGame: NextDiensteGame | null
  mySlots: DiensteSlot[]
  openSlotsCount: number
  dutyAccount: DutyAccount | null
}

interface CarpoolingPaarung { paarungId: number; partnerName: string }

interface CarpoolingConfirmed {
  gameId: number
  date: string
  opponent: string
  paarungen: CarpoolingPaarung[]
}

interface DashboardData {
  currentSeason: Season | null
  meineTermine: NextEvent[]
  meineDienste: MeineDienste | null
  carpoolingConfirmed: CarpoolingConfirmed[]
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatDate(iso: string) {
  if (iso.length >= 10) {
    const d = new Date(iso.slice(0, 10) + 'T12:00:00')
    return d.toLocaleDateString('de-DE', { weekday: 'short', day: '2-digit', month: '2-digit' })
  }
  return iso
}

function statusLabel(status: string) {
  if (status === 'fulfilled') return { label: 'Erfüllt', cls: 'bg-brand-success-light text-brand-success' }
  if (status === 'cash_substitute') return { label: 'Ablöse', cls: 'bg-brand-warning-light text-brand-text' }
  return { label: 'Zugesagt', cls: 'bg-brand-info/10 text-brand-blue' }
}

// ── Sub-components ────────────────────────────────────────────────────────────

function Accordion({
  id, title, icon: Icon, isOpen, onToggle, children,
}: {
  id: string; title: string; icon: React.ElementType; isOpen: boolean
  onToggle: () => void; children: React.ReactNode
}) {
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between px-5 py-4 hover:bg-brand-border-subtle transition-colors min-h-[56px]"
        aria-expanded={isOpen}
        aria-controls={`section-${id}`}
      >
        <span className="flex items-center gap-2 font-semibold text-brand-text">
          <Icon className="w-5 h-5 text-brand-yellow" />
          {title}
        </span>
        {isOpen
          ? <ChevronDown className="w-4 h-4 text-brand-text-muted" />
          : <ChevronRight className="w-4 h-4 text-brand-text-muted" />}
      </button>
      {isOpen && (
        <div id={`section-${id}`} className="px-5 pb-5 pt-1 border-t border-brand-border-subtle">
          {children}
        </div>
      )}
    </div>
  )
}

function MeineTermineSection({ events }: { events: NextEvent[] }) {
  if (events.length === 0) {
    return <p className="text-sm text-brand-text-muted py-1">Keine kommenden Termine.</p>
  }
  return (
    <ul className="space-y-2 mt-1">
      {events.map(e => (
        <li key={`${e.eventType}-${e.id}`}>
          <Link
            to={e.detailUrl}
            className="flex items-center gap-3 py-1.5 hover:bg-brand-border-subtle rounded px-2 -mx-2 transition-colors"
          >
            <span className="flex-shrink-0 text-brand-text-muted">
              {e.eventType === 'training'
                ? <Dumbbell className="w-4 h-4" />
                : e.title.startsWith('vs.') ? <Plane className="w-4 h-4" /> : <Home className="w-4 h-4" />}
            </span>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-brand-text truncate">{e.title}</p>
              <p className="text-xs text-brand-text-muted">{e.teamName} · {e.time}</p>
            </div>
            <ArrowRight className="w-4 h-4 flex-shrink-0 text-brand-text-subtle" />
          </Link>
        </li>
      ))}
    </ul>
  )
}

function MeineDiensteSection({ dienste }: { dienste: MeineDienste | null }) {
  const [accountOpen, setAccountOpen] = useState(false)

  if (!dienste) return null

  const { nextGame, mySlots, openSlotsCount, dutyAccount } = dienste
  const showProgress = dutyAccount?.soll != null && dutyAccount.soll > 0
  const pct = showProgress ? Math.min(100, Math.round(((dutyAccount?.ist ?? 0) / (dutyAccount?.soll ?? 1)) * 100)) : 0

  return (
    <div className="space-y-3 mt-1">
      {nextGame ? (
        <div>
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">
            {formatDate(nextGame.date)} · {nextGame.opponent}
          </p>
          {mySlots.length > 0 ? (
            <ul className="space-y-1">
              {mySlots.map((s, i) => (
                <li key={i} className="flex items-center gap-2 text-sm text-brand-text py-0.5">
                  <Check className="w-4 h-4 text-brand-success flex-shrink-0" />
                  <span className="flex-1">{s.dutyTypeName}</span>
                  {s.eventTime && <span className="text-xs text-brand-text-muted">{s.eventTime}</span>}
                </li>
              ))}
            </ul>
          ) : (
            <Link
              to="/dienste"
              className="flex items-center justify-between text-sm text-brand-text hover:bg-brand-border-subtle rounded px-2 py-1.5 -mx-2 transition-colors"
            >
              <span>{openSlotsCount} offene Dienst{openSlotsCount !== 1 ? 'e' : ''} verfügbar</span>
              <ArrowRight className="w-4 h-4 text-brand-text-subtle" />
            </Link>
          )}
        </div>
      ) : (
        <p className="text-sm text-brand-text-muted">Kein kommendes Spiel mit Diensten.</p>
      )}

      {dutyAccount && (
        <div className="pt-3 border-t border-brand-border-subtle">
          <button
            onClick={() => setAccountOpen(o => !o)}
            className="w-full flex items-center justify-between hover:bg-brand-border-subtle rounded px-2 py-1.5 -mx-2 transition-colors min-h-[36px]"
          >
            <div>
              <span className="text-sm font-medium text-brand-text">
                Dienstkonto: {dutyAccount.ist}{showProgress ? `/${dutyAccount.soll}` : ''}
              </span>
              {showProgress && (
                <div className="mt-1 h-1.5 w-32 bg-brand-border-subtle rounded-full overflow-hidden">
                  <div className="h-full bg-brand-blue rounded-full" style={{ width: `${pct}%` }} />
                </div>
              )}
            </div>
            {accountOpen ? <ChevronDown className="w-4 h-4 text-brand-text-muted" /> : <ChevronRight className="w-4 h-4 text-brand-text-muted" />}
          </button>
          {accountOpen && (
            <div className="mt-2 space-y-1">
              {dutyAccount.recentAssignments.length === 0 ? (
                <p className="text-xs text-brand-text-muted">Noch keine Dienste diese Saison.</p>
              ) : dutyAccount.recentAssignments.map((a, i) => {
                const { label, cls } = statusLabel(a.status)
                return (
                  <div key={i} className="flex items-center justify-between text-xs py-1 border-b border-brand-border-subtle last:border-0">
                    <span className="text-brand-text-muted">{formatDate(a.date)} — {a.dutyType}</span>
                    <span className={`px-1.5 py-0.5 rounded ${cls}`}>{label}</span>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function MeinTeamSection() {
  const [teams, setTeams] = useState<{ id: number; name: string }[]>([])

  useEffect(() => {
    api.get('/teams/my').then(r => setTeams(r.data ?? [])).catch(() => {})
  }, [])

  if (teams.length === 0) {
    return <p className="text-sm text-brand-text-muted py-1">Kein Team zugeordnet.</p>
  }

  return (
    <ul className="space-y-2 mt-1">
      {teams.map(t => (
        <li key={t.id}>
          <Link
            to={`/mein-team?team=${t.id}`}
            className="flex items-center justify-between py-1.5 hover:bg-brand-border-subtle rounded px-2 -mx-2 transition-colors"
          >
            <span className="text-sm font-medium text-brand-text">{t.name}</span>
            <ArrowRight className="w-4 h-4 text-brand-text-subtle" />
          </Link>
        </li>
      ))}
    </ul>
  )
}

function FahrgemeinschaftenSection({ confirmed }: { confirmed: CarpoolingConfirmed[] }) {
  const withPairings = confirmed.filter(c => c.paarungen.length > 0)

  if (withPairings.length === 0) {
    return (
      <div>
        <p className="text-sm text-brand-text-muted py-1">Keine bestätigten Fahrgemeinschaften.</p>
        <Link to="/mitfahrgelegenheiten" className="text-xs text-brand-text-muted hover:text-brand-text flex items-center gap-1 mt-1">
          Zur Übersicht <ArrowRight className="w-3 h-3" />
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-3 mt-1">
      {withPairings.map(g => (
        <div key={g.gameId}>
          <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-1">
            {formatDate(g.date)} · {g.opponent}
          </p>
          <ul className="space-y-1">
            {g.paarungen.map(p => (
              <li key={p.paarungId} className="flex items-center gap-1.5 text-sm text-brand-text">
                <Check className="w-4 h-4 text-brand-success flex-shrink-0" />
                <span>{p.partnerName}</span>
              </li>
            ))}
          </ul>
        </div>
      ))}
      <Link to="/mitfahrgelegenheiten" className="text-xs text-brand-text-muted hover:text-brand-text flex items-center gap-1">
        Alle Mitfahrten <ArrowRight className="w-3 h-3" />
      </Link>
    </div>
  )
}

// ── Main ──────────────────────────────────────────────────────────────────────

export default function DashboardPage() {
  const { user } = useAuth()
  const isMobile = useMediaQuery('(max-width: 639px)')

  type LoadState = 'loading' | 'loaded' | 'error'
  const [loadState, setLoadState] = useState<LoadState>('loading')
  const [data, setData] = useState<DashboardData | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [openSection, setOpenSection] = useState<string>('termine')
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({
    termine: true, dienste: true, team: true, fahrt: true,
  })

  const isOpen = (id: string) => isMobile ? openSection === id : openSections[id]
  const toggle = (id: string) => {
    if (isMobile) setOpenSection(prev => prev === id ? '' : id)
    else setOpenSections(prev => ({ ...prev, [id]: !prev[id] }))
  }

  const load = useCallback((silent = false) => {
    if (!silent) setLoadState('loading')
    api.get('/dashboard')
      .then(res => { setData(res.data); setLoadState('loaded') })
      .catch(err => { setError(err.message); setLoadState('error') })
  }, [])

  useEffect(() => { load() }, [load])

  useLiveUpdates(event => {
    if (event === 'mitfahrgelegenheiten') load(true)
  })

  if (loadState === 'loading') {
    return (
      <div className="max-w-2xl mx-auto space-y-3">
        {[1, 2, 3].map(i => <div key={i} className="h-14 bg-brand-border-subtle rounded-lg animate-pulse" />)}
      </div>
    )
  }

  if (loadState === 'error' || !data) {
    return (
      <div className="text-center py-8">
        <p className="text-sm text-brand-text-muted mb-3">Dashboard konnte nicht geladen werden.</p>
        <p className="text-xs text-brand-text-subtle">{error}</p>
        <button onClick={() => load()} className="mt-4 px-4 py-2 bg-brand-yellow hover:bg-brand-black hover:text-brand-yellow text-sm font-medium rounded transition-colors">
          Erneut versuchen
        </button>
      </div>
    )
  }

  const showDienste = !!data.meineDienste
  const showFahrt = data.carpoolingConfirmed.length > 0 || !!user

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-text">Übersicht</h1>
        {data.currentSeason && (
          <p className="text-sm text-brand-text-muted mt-0.5">Saison {data.currentSeason.name}</p>
        )}
      </div>

      <div className="space-y-2">
        <Accordion id="termine" title="Meine Termine" icon={Calendar} isOpen={isOpen('termine')} onToggle={() => toggle('termine')}>
          <MeineTermineSection events={data.meineTermine} />
        </Accordion>

        {showDienste && (
          <Accordion id="dienste" title="Meine Dienste" icon={BarChart2} isOpen={isOpen('dienste')} onToggle={() => toggle('dienste')}>
            <MeineDiensteSection dienste={data.meineDienste} />
          </Accordion>
        )}

        <Accordion id="team" title="Mein Team" icon={Users} isOpen={isOpen('team')} onToggle={() => toggle('team')}>
          <MeinTeamSection />
        </Accordion>

        {showFahrt && (
          <Accordion id="fahrt" title="Fahrgemeinschaften" icon={Car} isOpen={isOpen('fahrt')} onToggle={() => toggle('fahrt')}>
            <FahrgemeinschaftenSection confirmed={data.carpoolingConfirmed} />
          </Accordion>
        )}
      </div>
    </div>
  )
}
