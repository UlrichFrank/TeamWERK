import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Zap, Calendar, BarChart2, Users, Car,
  CircleDot, ArrowRight, Download, ChevronDown, ChevronRight
} from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useMediaQuery } from '../lib/useMediaQuery'
import Accordion from '../components/Accordion'

interface Season { id: number; name: string; isActive: boolean }
interface Action { id: string; type: string; text: string; link: string; dueDate?: string; actionNeeded?: boolean }
interface Game { id: number; date: string; opponent: string; isHome: boolean; team: string; slotsCount: number; slotsFilled: number; link: string }
interface TeamStats { team: string; activeMembers: number; totalMembers: number; injuredCount: number; pausedCount: number }
interface RecentAssignment { date: string; dutyType: string; status: string }
interface DutyAccount { season: string; ist: number; soll: number | null; children: number; recentAssignments: RecentAssignment[] }
interface VehicleInfo { seats: number; notes: string; upToDate: boolean }
interface DashboardData {
  currentSeason: Season | null
  nextGameDate: string | null
  actions: Action[]
  nextGames: Game[]
  teamStats: TeamStats | null
  dutyAccount: DutyAccount | null
  vehicleInfo: VehicleInfo | null
}

function formatDate(iso: string) {
  if (iso.length >= 10) {
    const d = new Date(iso.slice(0, 10) + 'T12:00:00')
    return d.toLocaleDateString('de-DE', { weekday: 'short', day: '2-digit', month: '2-digit' })
  }
  return iso
}

function statusLabel(status: string) {
  if (status === 'fulfilled') return { label: 'Erfüllt', cls: 'bg-green-100 text-green-800' }
  if (status === 'cash_substitute') return { label: 'Ablöse', cls: 'bg-yellow-100 text-yellow-800' }
  return { label: 'Zugesagt', cls: 'bg-blue-100 text-blue-800' }
}

function ActionsList({ actions }: { actions: Action[] }) {
  if (actions.length === 0) {
    return <p className="text-sm text-black/50 py-1">Alles erledigt! 🎉</p>
  }
  return (
    <ul className="space-y-2">
      {actions.map(a => (
        <li key={a.id} className="flex items-start gap-2">
          <CircleDot size={16} className="mt-0.5 flex-shrink-0 text-brand-blue" />
          <Link
            to={a.link}
            className="flex-1 text-sm text-black hover:underline flex items-center justify-between gap-1 py-1 sm:py-0.5"
          >
            <span>{a.text}</span>
            <ArrowRight size={14} className="flex-shrink-0 text-black/40" />
          </Link>
        </li>
      ))}
    </ul>
  )
}

function NextGamesList({ games }: { games: Game[] }) {
  if (games.length === 0) {
    return <p className="text-sm text-black/50 py-1">Keine Spiele geplant.</p>
  }
  return (
    <ul className="space-y-2">
      {games.map(g => (
        <li key={g.id}>
          <Link to={g.link} className="block hover:bg-black/5 rounded px-2 py-2 -mx-2 transition-colors">
            <div className="flex items-center justify-between gap-2">
              <div>
                <span className="text-xs text-black/50 mr-2">{formatDate(g.date)}</span>
                <span className="text-sm font-medium">
                  {g.isHome ? '🏠' : '🚌'} vs. {g.opponent}
                </span>
              </div>
              <ArrowRight size={14} className="text-black/40 flex-shrink-0" />
            </div>
            <div className="mt-0.5 text-xs text-black/40">
              {g.team} · Dienste: {g.slotsFilled}/{g.slotsCount}
            </div>
          </Link>
        </li>
      ))}
    </ul>
  )
}

function DutyAccountTile({ account, role }: { account: DutyAccount; role: string }) {
  const [open, setOpen] = useState(false)
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin' || user?.role === 'vorstand'

  const pct = account.soll ? Math.min(100, Math.round((account.ist / account.soll) * 100)) : null

  return (
    <div>
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center justify-between py-2 hover:bg-black/5 rounded px-2 -mx-2 transition-colors min-h-[44px]"
      >
        <div>
          <span className="text-sm font-medium">
            Dienstleistungen: {account.ist}{account.soll != null ? `/${account.soll}` : ''}
          </span>
          {account.soll != null && (
            <div className="mt-1 h-2 w-32 bg-black/10 rounded-full overflow-hidden">
              <div
                className="h-full bg-brand-blue rounded-full transition-all"
                style={{ width: `${pct}%` }}
              />
            </div>
          )}
        </div>
        {open ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
      </button>

      {role === 'elternteil' && account.children > 0 && (
        <p className="text-xs text-black/50 mt-1">
          Ziel: 5 Dienste × {account.children} Kinder = {account.soll}
        </p>
      )}

      {open && (
        <div className="mt-2 space-y-1">
          {account.recentAssignments.length === 0 ? (
            <p className="text-xs text-black/50">Noch keine Dienste diese Saison.</p>
          ) : (
            account.recentAssignments.map((a, i) => {
              const { label, cls } = statusLabel(a.status)
              return (
                <div key={i} className="flex items-center justify-between text-xs py-1 border-b border-black/5 last:border-0">
                  <span className="text-black/60">{formatDate(a.date)} — {a.dutyType}</span>
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
          className="mt-3 inline-flex items-center gap-1 text-xs text-black/60 hover:text-black transition-colors"
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
    <div className="mt-3 pt-3 border-t border-black/10">
      <p className="text-xs font-semibold uppercase tracking-wider text-black/50 mb-2">{stats.team}</p>
      <div className="grid grid-cols-3 gap-2 text-center">
        <div>
          <div className="text-lg font-bold text-brand-green">{stats.activeMembers}</div>
          <div className="text-xs text-black/50">Aktiv</div>
        </div>
        <div>
          <div className="text-lg font-bold text-red-500">{stats.injuredCount}</div>
          <div className="text-xs text-black/50">Verletzt</div>
        </div>
        <div>
          <div className="text-lg font-bold text-yellow-500">{stats.pausedCount}</div>
          <div className="text-xs text-black/50">Pausiert</div>
        </div>
      </div>
    </div>
  )
}

function VehicleSection({ vehicleInfo }: { vehicleInfo: VehicleInfo | null }) {
  if (!vehicleInfo) {
    return (
      <div className="flex items-start gap-2">
        <CircleDot size={16} className="mt-0.5 flex-shrink-0 text-brand-blue" />
        <Link to="/profil" className="flex-1 text-sm text-black hover:underline flex items-center justify-between gap-1 py-1">
          <span>Fahrzeuginfo fehlt — bitte eintragen</span>
          <ArrowRight size={14} className="flex-shrink-0 text-black/40" />
        </Link>
      </div>
    )
  }
  return (
    <div className="flex items-center justify-between py-1">
      <span className="text-sm">{vehicleInfo.seats} Plätze gemeldet</span>
      <Link to="/profil" className="text-xs text-black/50 hover:text-black transition-colors flex items-center gap-1">
        Ändern <ArrowRight size={12} />
      </Link>
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

  // Mobile: only one section open at a time; Desktop: all open
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

  useEffect(() => {
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

  if (state === 'loading') {
    return (
      <div className="max-w-2xl mx-auto space-y-3">
        {[1, 2, 3].map(i => (
          <div key={i} className="h-14 bg-black/5 rounded-lg animate-pulse" />
        ))}
      </div>
    )
  }

  if (state === 'error' || !data) {
    return (
      <div className="text-center py-8">
        <p className="text-sm text-black/60 mb-3">Dashboard konnte nicht geladen werden.</p>
        <p className="text-xs text-black/40">{error}</p>
        <button
          onClick={() => { setState('loading'); setError(null); api.get('/dashboard').then(r => { setData(r.data); setState('loaded') }).catch(e => { setError(e.message); setState('error') }) }}
          className="mt-4 px-4 py-2 bg-brand-yellow hover:bg-black hover:text-brand-yellow text-sm font-medium rounded transition-colors"
        >
          Erneut versuchen
        </button>
      </div>
    )
  }

  const role = user?.role ?? ''
  const hasKontoSection = !!data.dutyAccount
  const hasTeamSection = role === 'trainer' || role === 'elternteil' || role === 'spieler'
  const hasFahrtSection = !!data.vehicleInfo || role === 'elternteil' || role === 'spieler' || role === 'trainer'

  return (
    <div className="max-w-2xl mx-auto">
      {/* Header */}
      <div className="mb-5">
        <h1 className="text-xl font-bold">Übersicht</h1>
        {data.currentSeason && (
          <p className="text-sm text-black/50 mt-0.5">Saison {data.currentSeason.name}</p>
        )}
        {data.nextGameDate && (
          <p className="text-sm text-black/60 mt-1">
            Nächster Termin: {formatDate(data.nextGameDate)}
          </p>
        )}
      </div>

      <div className="space-y-2">
        {/* DIESE WOCHE */}
        <Accordion id="actions" title="Diese Woche" icon={Zap} isOpen={isOpen('actions')} onToggle={() => toggle('actions')}>
          <ActionsList actions={data.actions} />
        </Accordion>

        {/* NÄCHSTE SPIELE */}
        <Accordion id="games" title="Nächste Spiele" icon={Calendar} isOpen={isOpen('games')} onToggle={() => toggle('games')}>
          <NextGamesList games={data.nextGames} />
        </Accordion>

        {/* KONTO / TEAM-STATS */}
        {hasKontoSection && (
          <Accordion id="konto" title="Dienstkonto" icon={BarChart2} isOpen={isOpen('konto')} onToggle={() => toggle('konto')}>
            {data.dutyAccount && <DutyAccountTile account={data.dutyAccount} role={role} />}
            {data.teamStats && <TeamStatsCard stats={data.teamStats} />}
          </Accordion>
        )}

        {/* DEIN TEAM */}
        {hasTeamSection && (
          <Accordion id="team" title="Dein Team" icon={Users} isOpen={isOpen('team')} onToggle={() => toggle('team')}>
            {user?.role === 'trainer' || user?.role === 'vorstand' || user?.role === 'admin' ? (
              <Link to="/mitglieder" className="inline-flex items-center gap-1 text-sm text-black hover:underline py-1">
                Zur Mitgliederliste <ArrowRight size={14} />
              </Link>
            ) : (
              <Link to="/profil" className="inline-flex items-center gap-1 text-sm text-black hover:underline py-1">
                Mein Profil <ArrowRight size={14} />
              </Link>
            )}
          </Accordion>
        )}

        {/* FAHRTGEMEINSCHAFTEN */}
        {hasFahrtSection && (
          <Accordion id="fahrt" title="Fahrtgemeinschaften" icon={Car} isOpen={isOpen('fahrt')} onToggle={() => toggle('fahrt')}>
            <VehicleSection vehicleInfo={data.vehicleInfo} />
          </Accordion>
        )}
      </div>
    </div>
  )
}
