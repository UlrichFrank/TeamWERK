import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Zap, Calendar, BarChart2, Users, Car, UserPlus,
  CircleDot, ArrowRight, Download, ChevronDown, ChevronRight,
  Home, MapPin, MapPinned, Check
} from 'lucide-react'
import { api } from '../lib/api'
import { useAuth, hasFunction } from '../contexts/AuthContext'
import { useMediaQuery } from '../lib/useMediaQuery'
import Accordion from '../components/Accordion'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

interface Season { id: number; name: string; isActive: boolean }
interface Action { id: string; type: string; text: string; link: string; dueDate?: string; eventTime?: string; dutyTypeName?: string; actionNeeded?: boolean }
interface Game { id: number; date: string; opponent: string; isHome: boolean; eventType: string; team: string; slotsCount: number; slotsFilled: number; link: string }
interface TeamStats { team: string; activeMembers: number; totalMembers: number; injuredCount: number; pausedCount: number }
interface RecentAssignment { date: string; dutyType: string; status: string }
interface DutyAccount { season: string; ist: number; soll: number | null; children: number; recentAssignments: RecentAssignment[] }
interface VehicleInfo { seats: number; notes: string; upToDate: boolean }
interface CarpoolingMyEntry { id: number; typ: string }
interface CarpoolingPaarung { paarungId: number; partnerName: string }
interface CarpoolingOpenEntry { typ: string; name: string }
interface CarpoolingHint {
  gameId: number; date: string; opponent: string; bieteCount: number; sucheCount: number
  myEntry: CarpoolingMyEntry | null
  paarungen: CarpoolingPaarung[]
  openEntries: CarpoolingOpenEntry[]
}
interface DashboardData {
  currentSeason: Season | null
  nextGameDate: string | null
  actions: Action[]
  nextGames: Game[]
  teamStats: TeamStats | null
  dutyAccount: DutyAccount | null
  vehicleInfo: VehicleInfo | null
  carpoolingHint?: CarpoolingHint | null
}

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

function DutyDateGroup({ date, duties }: { date: string; duties: Action[] }) {
  const [open, setOpen] = useState(true)
  const label = formatDate(date)
  return (
    <div>
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between py-1.5 hover:bg-brand-border-subtle rounded px-1 -mx-1 transition-colors min-h-[36px]"
      >
        <span className="text-sm font-semibold text-brand-text">{label}</span>
        {open ? <ChevronDown size={15} className="text-brand-text-muted" /> : <ChevronRight size={15} className="text-brand-text-muted" />}
      </button>
      {open && (
        <ul className="mt-1 mb-2 space-y-1 pl-2 border-l-2 border-brand-border-subtle">
          {duties.map(a => (
            <li key={a.id}>
              <Link
                to={a.link}
                className="flex items-center gap-2 py-1 text-sm text-brand-text hover:text-brand-blue transition-colors"
              >
                <span className="font-mono text-xs text-brand-text-muted w-11 flex-shrink-0">
                  {a.eventTime || '–'}
                </span>
                <span className="flex-1">{a.dutyTypeName || a.text}</span>
                <ArrowRight size={13} className="flex-shrink-0 text-brand-text-subtle" />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function ActionsList({ actions }: { actions: Action[] }) {
  const dutyActions = actions.filter(a => a.type === 'duty' && a.dueDate)
  const otherActions = actions.filter(a => a.type !== 'duty' || !a.dueDate)

  if (actions.length === 0) {
    return <p className="text-sm text-brand-text-muted py-1">Alles erledigt! 🎉</p>
  }

  // Group duty actions by date
  const byDate = new Map<string, Action[]>()
  for (const a of dutyActions) {
    const d = a.dueDate!
    if (!byDate.has(d)) byDate.set(d, [])
    byDate.get(d)!.push(a)
  }
  const sortedDates = [...byDate.keys()].sort()

  return (
    <div className="space-y-1">
      {sortedDates.map(date => (
        <DutyDateGroup key={date} date={date} duties={byDate.get(date)!} />
      ))}
      {otherActions.length > 0 && (
        <ul className="space-y-2 mt-2">
          {otherActions.map(a => (
            <li key={a.id} className="flex items-start gap-2">
              <CircleDot size={16} className="mt-0.5 flex-shrink-0 text-brand-blue" />
              <Link
                to={a.link}
                className="flex-1 text-sm text-brand-text hover:underline flex items-center justify-between gap-1 py-1 sm:py-0.5"
              >
                <span>{a.text}</span>
                <ArrowRight size={14} className="flex-shrink-0 text-brand-text-subtle" />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

function NextGamesList({ games }: { games: Game[] }) {
  if (games.length === 0) {
    return <p className="text-sm text-brand-text-muted py-1">Keine Spiele geplant.</p>
  }
  return (
    <ul className="space-y-2">
      {games.map(g => (
        <li key={g.id}>
          <Link to={`/kalender?date=${g.date.slice(0, 10)}`} className="block hover:bg-brand-border-subtle rounded px-2 py-2 -mx-2 transition-colors">
            <div className="flex items-center justify-between gap-2">
              <div>
                <span className="text-xs text-brand-text-muted mr-2">{formatDate(g.date)}</span>
                <span className="text-sm font-medium inline-flex items-center gap-1">
                  {g.isHome
                    ? <Home className="w-4 h-4 flex-shrink-0" />
                    : <MapPin className="w-4 h-4 flex-shrink-0" />
                  }
                  {g.eventType === 'generisch' ? g.opponent : `Team vs ${g.opponent}`}
                </span>
              </div>
              <ArrowRight size={14} className="text-brand-text-subtle flex-shrink-0" />
            </div>
            <div className="mt-0.5 text-xs text-brand-text-subtle">
              {g.team} · Dienste: {g.slotsFilled}/{g.slotsCount}
            </div>
          </Link>
        </li>
      ))}
    </ul>
  )
}

function DutyAccountTile({ account }: { account: DutyAccount }) {
  const [open, setOpen] = useState(false)
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin' || hasFunction(user, 'vorstand')

  const pct = account.soll ? Math.min(100, Math.round((account.ist / account.soll) * 100)) : null
  const showProgress = account.soll != null && account.soll > 0

  return (
    <div>
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between py-2 hover:bg-brand-border-subtle rounded px-2 -mx-2 transition-colors min-h-[44px]"
      >
        <div>
          <span className="text-sm font-medium text-brand-text">
            Dienstleistungen: {account.ist}{showProgress ? `/${account.soll}` : ''}
          </span>
          {showProgress && (
            <div className="mt-1 h-2 w-32 bg-brand-border-subtle rounded-full overflow-hidden">
              <div
                className="h-full bg-brand-blue rounded-full transition-all"
                style={{ width: `${pct}%` }}
              />
            </div>
          )}
        </div>
        {open ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
      </button>

      {user?.isParent && account.soll != null && (
        <p className="text-xs text-brand-text-muted mt-1">
          Ziel: {account.soll} Dienste (Saison {account.season})
        </p>
      )}

      {open && (
        <div className="mt-2 space-y-1">
          {account.recentAssignments.length === 0 ? (
            <p className="text-xs text-brand-text-muted">Noch keine Dienste diese Saison.</p>
          ) : (
            account.recentAssignments.map((a, i) => {
              const { label, cls } = statusLabel(a.status)
              return (
                <div key={i} className="flex items-center justify-between text-xs py-1 border-b border-brand-border-subtle last:border-0">
                  <span className="text-brand-text-muted">{formatDate(a.date)} — {a.dutyType}</span>
                  <span className={`px-1.5 py-0.5 rounded text-xs ${cls}`}>{label}</span>
                </div>
              )
            })
          )}
        </div>
      )}

      {isAdmin && (
        <a
          href="/api/admin/duty-accounts/export"
          download
          className="mt-3 inline-flex items-center gap-1 text-xs text-brand-text-muted hover:text-brand-text transition-colors"
        >
          <Download size={14} />
          Dienstkonten exportieren
        </a>
      )}
    </div>
  )
}

function TeamStatsCard({ stats }: { stats: TeamStats }) {
  return (
    <div className="mt-3 pt-3 border-t border-brand-border-subtle">
      <p className="text-xs font-semibold uppercase tracking-wider text-brand-text-muted mb-2">{stats.team}</p>
      <div className="grid grid-cols-3 gap-2 text-center">
        <div>
          <div className="text-lg font-bold text-brand-green">{stats.activeMembers}</div>
          <div className="text-xs text-brand-text-muted">Aktiv</div>
        </div>
        <div>
          <div className="text-lg font-bold text-brand-danger">{stats.injuredCount}</div>
          <div className="text-xs text-brand-text-muted">Verletzt</div>
        </div>
        <div>
          <div className="text-lg font-bold text-brand-warning">{stats.pausedCount}</div>
          <div className="text-xs text-brand-text-muted">Pausiert</div>
        </div>
      </div>
    </div>
  )
}

function CarpoolingHintCard({ hint }: { hint: CarpoolingHint | null | undefined }) {
  if (!hint) {
    return (
      <div className="flex items-start gap-2">
        <MapPinned size={16} className="mt-0.5 flex-shrink-0 text-brand-text-muted" />
        <div className="flex-1">
          <p className="text-sm text-brand-text-muted">Keine Auswärtsfahrten geplant.</p>
          <Link to="/mitfahrgelegenheiten" className="text-xs text-brand-text-muted hover:text-brand-text transition-colors flex items-center gap-1 mt-1">
            Zur Übersicht <ArrowRight size={12} />
          </Link>
        </div>
      </div>
    )
  }

  const shownCount = hint.openEntries.length + hint.paarungen.length + (hint.myEntry ? 1 : 0)
  const totalCount = hint.bieteCount + hint.sucheCount
  const weitereCount = totalCount - shownCount
  const hasContent = hint.paarungen.length > 0 || hint.openEntries.length > 0

  return (
    <div className="space-y-2">
      {/* Spielzeile */}
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-medium text-brand-text">
          {formatDate(hint.date)} · vs. {hint.opponent}
        </p>
        <Link
          to="/mitfahrgelegenheiten"
          className="flex-shrink-0 text-xs text-brand-text-muted hover:text-brand-text transition-colors flex items-center gap-1"
        >
          Alle <ArrowRight size={12} />
        </Link>
      </div>

      {/* Mein Eintrag */}
      {hint.myEntry && (
        <span className="inline-block text-xs font-medium px-2 py-0.5 rounded-full bg-brand-yellow/20 text-brand-text">
          {hint.myEntry.typ === 'biete' ? 'Mein Angebot' : 'Mein Gesuch'}
        </span>
      )}

      {/* Bestätigte Paarungen */}
      {hint.paarungen.length > 0 && (
        <div className="space-y-1">
          {hint.paarungen.map(p => (
            <div key={p.paarungId} className="flex items-center gap-1.5 text-xs">
              <Check size={14} className="text-brand-success flex-shrink-0" />
              <span className="font-medium text-brand-text">{p.partnerName}</span>
              <span className="text-brand-text-subtle">— Mitfahrt bestätigt</span>
            </div>
          ))}
        </div>
      )}

      {/* Offene Einträge anderer */}
      {hint.openEntries.length > 0 && (
        <div className="space-y-1 pt-1 border-t border-brand-border-subtle">
          {hint.openEntries.map((e, i) => (
            <div key={i} className="flex items-center gap-1.5 text-xs text-brand-text">
              {e.typ === 'biete'
                ? <Car size={13} className="text-brand-text-muted flex-shrink-0" />
                : <UserPlus size={13} className="text-brand-text-muted flex-shrink-0" />
              }
              <span>{e.name}</span>
              <span className="text-brand-text-subtle">· {e.typ === 'biete' ? 'Angebot' : 'Gesuch'}</span>
            </div>
          ))}
          {weitereCount > 0 && (
            <p className="text-xs text-brand-text-subtle">+ {weitereCount} weitere</p>
          )}
        </div>
      )}

      {/* Fallback wenn gar nichts */}
      {!hasContent && (
        <div className="pt-1">
          <p className="text-xs text-brand-text-muted">Noch keine Einträge.</p>
          <Link to="/mitfahrgelegenheiten" className="text-xs text-brand-text-muted hover:text-brand-text transition-colors flex items-center gap-1 mt-1">
            Als Erstes eintragen <ArrowRight size={12} />
          </Link>
        </div>
      )}
    </div>
  )
}


export default function DashboardPage() {
  const { user } = useAuth()
  const isMobile = useMediaQuery('(max-width: 639px)')

  type State = 'loading' | 'loaded' | 'error'
  const [state, setState] = useState<State>('loading')
  const [data, setData] = useState<DashboardData | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [openSection, setOpenSection] = useState<string>('actions')
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({
    actions: true, games: true, konto: true, team: true, fahrt: true,
  })

  const isOpen = (id: string) => isMobile ? openSection === id : openSections[id]
  const toggle = (id: string) => {
    if (isMobile) {
      setOpenSection(prev => prev === id ? '' : id)
    } else {
      setOpenSections(prev => ({ ...prev, [id]: !prev[id] }))
    }
  }

  const load = useCallback((silent = false) => {
    if (!silent) setState('loading')
    api.get('/dashboard')
      .then(res => {
        setData(res.data)
        setState('loaded')
      })
      .catch(err => {
        setError(err.message)
        setState('error')
      })
  }, [])

  useEffect(() => { load() }, [load])

  useLiveUpdates(event => {
    if (event === 'mitfahrgelegenheiten') load(true)
  })

  if (state === 'loading') {
    return (
      <div className="max-w-2xl mx-auto space-y-3">
        {[1, 2, 3].map(i => (
          <div key={i} className="h-14 bg-brand-border-subtle rounded-lg animate-pulse" />
        ))}
      </div>
    )
  }

  if (state === 'error' || !data) {
    return (
      <div className="text-center py-8">
        <p className="text-sm text-brand-text-muted mb-3">Dashboard konnte nicht geladen werden.</p>
        <p className="text-xs text-brand-text-subtle">{error}</p>
        <button
          onClick={() => { setState('loading'); setError(null); api.get('/dashboard').then(r => { setData(r.data); setState('loaded') }).catch(e => { setError(e.message); setState('error') }) }}
          className="mt-4 px-4 py-2 bg-brand-yellow hover:bg-brand-black hover:text-brand-yellow text-sm font-medium rounded transition-colors"
        >
          Erneut versuchen
        </button>
      </div>
    )
  }

  const hasKontoSection = !!data.dutyAccount
  const hasTeamSection = hasFunction(user, 'trainer') || user?.isParent || hasFunction(user, 'spieler')
  const hasFahrtSection = !!data.vehicleInfo || !!data.carpoolingHint || user?.isParent || hasFunction(user, 'spieler') || hasFunction(user, 'trainer')

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-brand-text">Übersicht</h1>
        {data.currentSeason && (
          <p className="text-sm text-brand-text-muted mt-0.5">Saison {data.currentSeason.name}</p>
        )}
        {data.nextGameDate && (
          <p className="text-sm text-brand-text-muted mt-1">
            Nächster Termin: {formatDate(data.nextGameDate)}
          </p>
        )}
      </div>

      <div className="space-y-2">
        <Accordion id="actions" title="Diese Woche" icon={Zap} isOpen={isOpen('actions')} onToggle={() => toggle('actions')}>
          <ActionsList actions={data.actions} />
        </Accordion>

        <Accordion id="games" title="Nächste Events" icon={Calendar} isOpen={isOpen('games')} onToggle={() => toggle('games')}>
          <NextGamesList games={data.nextGames} />
        </Accordion>

        {hasKontoSection && (
          <Accordion id="konto" title="Dienstkonto" icon={BarChart2} isOpen={isOpen('konto')} onToggle={() => toggle('konto')}>
            {data.dutyAccount && <DutyAccountTile account={data.dutyAccount} />}
            {data.teamStats && <TeamStatsCard stats={data.teamStats} />}
          </Accordion>
        )}

        {hasTeamSection && (
          <Accordion id="team" title="Dein Team" icon={Users} isOpen={isOpen('team')} onToggle={() => toggle('team')}>
            {hasFunction(user, 'trainer') || hasFunction(user, 'vorstand') || user?.role === 'admin' ? (
              <Link to="/mitglieder" className="inline-flex items-center gap-1 text-sm text-brand-text hover:underline py-1">
                Zur Mitgliederliste <ArrowRight size={14} />
              </Link>
            ) : (
              <Link to="/profil" className="inline-flex items-center gap-1 text-sm text-brand-text hover:underline py-1">
                Mein Profil <ArrowRight size={14} />
              </Link>
            )}
          </Accordion>
        )}

        {hasFahrtSection && (
          <Accordion id="fahrt" title="Fahrtgemeinschaften" icon={Car} isOpen={isOpen('fahrt')} onToggle={() => toggle('fahrt')}>
            <CarpoolingHintCard hint={data.carpoolingHint} />
          </Accordion>
        )}
      </div>
    </div>
  )
}
