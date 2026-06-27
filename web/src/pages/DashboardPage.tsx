import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Calendar, BarChart2, Users, Car, ArrowRight,
  Home, Plane, Dumbbell, ChevronDown, ChevronRight, Check, Search, Info
} from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useMediaQuery } from '../lib/useMediaQuery'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import EventNoteIndicator from '../components/EventNoteIndicator'

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
  isHome: boolean | null
  isExtended: boolean
  note: string
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

interface CarpoolingPaarung { paarungId: number; partnerName: string; partnerTreffpunkt: string }

interface CarpoolingConfirmed {
  gameId: number
  date: string
  opponent: string
  paarungen: CarpoolingPaarung[]
}

interface CarpoolingOpenRequest {
  sucheId: number
  requesterName: string
  plaetze: number
  treffpunkt: string
}

interface CarpoolingOpenGroup {
  gameId: number
  date: string
  title: string
  requests: CarpoolingOpenRequest[]
}

interface DashboardData {
  currentSeason: Season | null
  meineTermine: NextEvent[]
  meineDienste: MeineDienste | null
  carpoolingConfirmed: CarpoolingConfirmed[]
  carpoolingOpenGroups: CarpoolingOpenGroup[]
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

function ExtendedBadge() {
  return (
    <span className="inline-flex items-center rounded-full bg-brand-blue/10 px-2 py-0.5 text-xs font-semibold text-brand-blue border border-brand-blue/30 whitespace-nowrap flex-shrink-0">
      Erw. Kader
    </span>
  )
}

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

function DashboardRow({
  to, dateISO, icon, title, subtitle, badge,
}: {
  to: string
  dateISO: string
  icon: React.ReactNode
  title: string
  subtitle?: string | React.ReactNode
  badge?: React.ReactNode
}) {
  const d = new Date(dateISO.slice(0, 10) + 'T12:00:00')
  const weekday = d.toLocaleDateString('de-DE', { weekday: 'short' }).replace('.', '')
  const dayMonth = d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
  return (
    <Link
      to={to}
      className="flex items-center gap-3 py-1.5 hover:bg-brand-border-subtle rounded px-2 -mx-2 transition-colors"
    >
      <div className="flex-shrink-0 w-10 text-center">
        <p className="text-xs font-semibold text-brand-text-muted leading-tight">{weekday}</p>
        <p className="text-xs text-brand-text-subtle leading-tight">{dayMonth}</p>
      </div>
      <span className="flex-shrink-0 text-brand-text-muted">{icon}</span>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5 min-w-0">
          <p className="text-sm font-medium text-brand-text truncate">{title}</p>
          {badge}
        </div>
        {subtitle && <p className="text-xs text-brand-text-muted truncate">{subtitle}</p>}
      </div>
      <ArrowRight className="w-4 h-4 flex-shrink-0 text-brand-text-subtle" />
    </Link>
  )
}

function MeineTermineSection({ events }: { events: NextEvent[] }) {
  if (events.length === 0) {
    return (
      <div>
        <p className="text-sm text-brand-text-muted py-1">Keine kommenden Termine.</p>
        <Link to="/termine" className="text-xs text-brand-text-muted hover:text-brand-text flex items-center gap-1 mt-1">
          Alle Termine <ArrowRight className="w-3 h-3" />
        </Link>
      </div>
    )
  }
  return (
    <div className="space-y-1 mt-1">
      <ul className="space-y-1">
        {events.map(e => (
          <li key={`${e.eventType}-${e.id}`}>
            <DashboardRow
              to={e.detailUrl}
              dateISO={e.date}
              icon={e.eventType === 'training'
                ? <Dumbbell className="w-4 h-4" />
                : e.isHome ? <Home className="w-4 h-4" /> : <Plane className="w-4 h-4" />}
              title={e.title}
              subtitle={`${e.teamName} · ${e.time}`}
              badge={(e.note.trim() || e.isExtended) ? (
                <span className="flex items-center gap-1">
                  <EventNoteIndicator variant="icon" note={e.note} />
                  {e.isExtended ? <ExtendedBadge /> : null}
                </span>
              ) : undefined}
            />
          </li>
        ))}
      </ul>

      <Link to="/termine" className="text-xs text-brand-text-muted hover:text-brand-text flex items-center gap-1 pt-2">
        Alle Termine <ArrowRight className="w-3 h-3" />
      </Link>
    </div>
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
        mySlots.length > 0 ? (
          <ul className="space-y-1">
            {mySlots.map((s, i) => (
              <li key={i}>
                <DashboardRow
                  to="/dienste"
                  dateISO={nextGame.date}
                  icon={<Check className="w-4 h-4 text-brand-success" />}
                  title={s.dutyTypeName}
                  subtitle={s.eventTime ? `${nextGame.opponent} · ${s.eventTime}` : nextGame.opponent}
                />
              </li>
            ))}
          </ul>
        ) : (
          <DashboardRow
            to="/dienste"
            dateISO={nextGame.date}
            icon={<Info className="w-4 h-4" />}
            title={`${openSlotsCount} offene Dienst${openSlotsCount !== 1 ? 'e' : ''} verfügbar`}
            subtitle={nextGame.opponent}
          />
        )
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

      <Link to="/dienste" className="text-xs text-brand-text-muted hover:text-brand-text flex items-center gap-1 pt-2">
        Alle Dienste <ArrowRight className="w-3 h-3" />
      </Link>
    </div>
  )
}

function MeinTeamSection() {
  const [teams, setTeams] = useState<{ id: number; name: string; isExtended: boolean }[]>([])

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
            <span className="flex items-center gap-2 text-sm font-medium text-brand-text">
              {t.name}
              {t.isExtended && <ExtendedBadge />}
            </span>
            <ArrowRight className="w-4 h-4 flex-shrink-0 text-brand-text-subtle" />
          </Link>
        </li>
      ))}
    </ul>
  )
}

type FahrtRow =
  | { kind: 'zusage'; date: string; key: string; paarungId: number; partnerName: string; partnerTreffpunkt: string; opponent: string }
  | { kind: 'gesuch'; date: string; key: string; sucheId: number; requesterName: string; plaetze: number; treffpunkt: string; gameTitle: string }

function FahrgemeinschaftenSection({ confirmed, openGroups }: { confirmed: CarpoolingConfirmed[] | undefined; openGroups: CarpoolingOpenGroup[] | undefined }) {
  const rows: FahrtRow[] = []
  for (const g of confirmed ?? []) {
    for (const p of g.paarungen ?? []) {
      rows.push({ kind: 'zusage', date: g.date, key: `z-${p.paarungId}`, paarungId: p.paarungId, partnerName: p.partnerName, partnerTreffpunkt: p.partnerTreffpunkt ?? '', opponent: g.opponent })
    }
  }
  for (const g of openGroups ?? []) {
    for (const req of g.requests ?? []) {
      rows.push({ kind: 'gesuch', date: g.date, key: `g-${req.sucheId}`, sucheId: req.sucheId, requesterName: req.requesterName, plaetze: req.plaetze, treffpunkt: req.treffpunkt ?? '', gameTitle: g.title })
    }
  }
  rows.sort((a, b) => {
    const dateCmp = a.date.slice(0, 10).localeCompare(b.date.slice(0, 10))
    if (dateCmp !== 0) return dateCmp
    if (a.kind === b.kind) return 0
    return a.kind === 'zusage' ? -1 : 1
  })

  if (rows.length === 0) {
    return (
      <div>
        <p className="text-sm text-brand-text-muted py-1">Keine Fahrgemeinschaften oder offenen Gesuche.</p>
        <Link to="/mitfahrgelegenheiten" className="text-xs text-brand-text-muted hover:text-brand-text flex items-center gap-1 mt-1">
          Alle Fahrgemeinschaften <ArrowRight className="w-3 h-3" />
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-1 mt-1">
      <ul className="space-y-1">
        {rows.map(row => {
          if (row.kind === 'zusage') {
            const subtitle = row.partnerTreffpunkt
              ? `${row.opponent} · ${row.partnerTreffpunkt}`
              : row.opponent
            return (
              <li key={row.key}>
                <DashboardRow
                  to={`/mitfahrgelegenheiten#paarung-${row.paarungId}`}
                  dateISO={row.date}
                  icon={<Check className="w-4 h-4 text-brand-success" />}
                  title={row.partnerName}
                  subtitle={subtitle}
                />
              </li>
            )
          }
          const plaetzeText = `${row.plaetze} ${row.plaetze === 1 ? 'Platz' : 'Plätze'}`
          const subtitle = row.treffpunkt
            ? `${plaetzeText} · ${row.treffpunkt}`
            : `${plaetzeText} · ${row.gameTitle}`
          return (
            <li key={row.key}>
              <DashboardRow
                to={`/mitfahrgelegenheiten#suche-${row.sucheId}`}
                dateISO={row.date}
                icon={<Search className="w-4 h-4" />}
                title={row.requesterName}
                subtitle={subtitle}
              />
            </li>
          )
        })}
      </ul>

      <Link to="/mitfahrgelegenheiten" className="text-xs text-brand-text-muted hover:text-brand-text flex items-center gap-1 pt-2">
        Alle Fahrgemeinschaften <ArrowRight className="w-3 h-3" />
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
    if (event === 'mitfahrgelegenheiten' || event === 'games' || event === 'trainings' || event === 'duties' || event === 'absences' || event === 'event-note') load(true)
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
  const showFahrt = (data.carpoolingConfirmed?.length ?? 0) > 0 || !!user

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

        {showFahrt && (
          <Accordion id="fahrt" title="Fahrgemeinschaften" icon={Car} isOpen={isOpen('fahrt')} onToggle={() => toggle('fahrt')}>
            <FahrgemeinschaftenSection confirmed={data.carpoolingConfirmed} openGroups={data.carpoolingOpenGroups} />
          </Accordion>
        )}

        <Accordion id="team" title="Mein Team" icon={Users} isOpen={isOpen('team')} onToggle={() => toggle('team')}>
          <MeinTeamSection />
        </Accordion>
      </div>
    </div>
  )
}
