import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  Trophy, CalendarDays, ListOrdered, FileText, Home, MapPin, RefreshCw,
  Grid3x3, TrendingUp, Swords, Scale, PieChart, Users, ChevronDown, ChevronUp,
} from 'lucide-react'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useAuth } from '../contexts/AuthContext'
import { api } from '../lib/api'
import { HEADER_CTRL, HEADER_NEUTRAL, HEADER_FIELD } from '../lib/buttonStyles'
import {
  Staffel, TableRow, ScheduleGame, PlayerStat,
  CrossTable as CrossTableData, ProgressionDay, TeamStats, RefereeStat, Affiliation, TeamMatrix,
  fetchStaffeln, fetchTable, fetchSchedule, fetchRanglisten, syncStaffeln,
  fetchCrossTable, fetchProgression, fetchTeamStats, fetchRefereeStats, fetchAffiliation,
  fetchPlayerGames,
  goalRatio, pointsLabel, sevenMeterRate, matchesSearch, gameSearchFields,
} from '../lib/staffeln'
import { sharedRanks } from '../lib/ranking'
import { isOwnTeam, isOwnPlayer } from '../lib/staffelHighlight'
import SpielberichtPanel from '../components/SpielberichtPanel'
import MapsLink from '../components/MapsLink'
import CrossTable from '../components/staffeln/CrossTable'
import StandingsChart from '../components/staffeln/StandingsChart'
import Spielmatrix from '../components/staffeln/Spielmatrix'
import { GoalTablesView, FairPlayView, DistributionView } from '../components/staffeln/TeamStatsViews'
import RefereeView from '../components/staffeln/RefereeView'

type Tab =
  | 'tabelle' | 'spielplan' | 'kreuztabelle' | 'verlauf'
  | 'tore' | 'fairplay' | 'verteilung' | 'ranglisten' | 'schiedsrichter'

// Neun Reiter sind viel fuer eine Seite, auf Mobile besonders - deshalb
// scrollt die Leiste waagerecht statt umzubrechen, und der gewaehlte Reiter
// steht in der Adresse (design.md, Risks).
//
// Die drei Tor-Tabellen (Torverhaeltnis, Angriff, Verteidigung) liegen
// gemeinsam unter "Tore": es sind drei Sortierungen derselben Zahlen aus
// derselben Antwort, und drei eigene Reiter dafuer haetten die Leiste auf
// zwoelf getrieben, ohne einen Wechsel zu ersparen.
const TABS: { id: Tab; label: string; icon: typeof Trophy }[] = [
  { id: 'tabelle', label: 'Tabelle', icon: Trophy },
  { id: 'spielplan', label: 'Spielplan', icon: CalendarDays },
  { id: 'kreuztabelle', label: 'Kreuztabelle', icon: Grid3x3 },
  { id: 'verlauf', label: 'Verlauf', icon: TrendingUp },
  { id: 'tore', label: 'Tore', icon: Swords },
  { id: 'fairplay', label: 'Fair-Play', icon: Scale },
  { id: 'verteilung', label: 'Verteilung', icon: PieChart },
  { id: 'ranglisten', label: 'Ranglisten', icon: ListOrdered },
  { id: 'schiedsrichter', label: 'Schiedsrichter', icon: Users },
]

// Hinweis fuer einen Reiter, dessen Daten noch gar nicht abgerufen wurden.
const NOCH_NICHTS = 'Noch nichts beim Verband abgerufen.'

const TH = 'bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left'
const TD = 'px-4 py-3 text-sm text-brand-text'
const CARD = 'bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden'

export default function StaffelnPage() {
  const [params, setParams] = useSearchParams()
  const [staffeln, setStaffeln] = useState<Staffel[]>([])
  const [table, setTable] = useState<TableRow[]>([])
  const [games, setGames] = useState<ScheduleGame[]>([])
  const [stats, setStats] = useState<PlayerStat[]>([])
  const [cross, setCross] = useState<CrossTableData | null>(null)
  const [progression, setProgression] = useState<ProgressionDay[]>([])
  const [teamStats, setTeamStats] = useState<TeamStats | null>(null)
  const [referees, setReferees] = useState<RefereeStat[]>([])
  const [matrices, setMatrices] = useState<TeamMatrix[]>([])
  const [affiliation, setAffiliation] = useState<Affiliation>({ teamNames: [], playerIds: [] })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [polling, setPolling] = useState(false)
  const [pollHinweis, setPollHinweis] = useState('')
  const [search, setSearch] = useState('')
  const { hasCapability } = useAuth()

  const selectedId = Number(params.get('staffel')) || 0
  const tab = (params.get('tab') as Tab) || 'tabelle'
  const selected = useMemo(
    () => staffeln.find((s) => s.id === selectedId) ?? staffeln[0],
    [staffeln, selectedId],
  )

  // Die Zuordnungsliste selbst muss nachladbar sein, nicht nur die Daten
  // dahinter: nach dem ersten Abruf wechselt eine Staffel von "noch nichts
  // abgerufen" (id 0) auf eine echte ID. Ohne das bliebe die Seite auf dem
  // alten Zustand stehen und der Abruf sähe wirkungslos aus.
  const reloadStaffeln = () =>
    fetchStaffeln()
      .then(setStaffeln)
      .catch(() => setError('Staffeln konnten nicht geladen werden.'))

  useEffect(() => {
    reloadStaffeln().finally(() => setLoading(false))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Alle Reiter werden gemeinsam geladen: sie zeigen dieselbe Staffel, und ein
  // Reiterwechsel soll nicht auf einen Abruf warten. Die Zugehoerigkeit kommt
  // als eigene Route dazu - die vier Statistik-Routen bleiben dadurch
  // nutzerunabhaengig (design.md Paragraph 10).
  const reload = () => {
    if (!selected || selected.id === 0) {
      setTable([]); setGames([]); setStats([])
      setCross(null); setProgression([]); setTeamStats(null); setReferees([])
      setAffiliation({ teamNames: [], playerIds: [] }); setMatrices([])
      return
    }
    const id = selected.id
    Promise.all([
      fetchTable(id), fetchSchedule(id), fetchRanglisten(id),
      fetchCrossTable(id), fetchProgression(id), fetchTeamStats(id),
      fetchRefereeStats(id), fetchAffiliation(id), fetchPlayerGames(id),
    ])
      .then(([t, g, s, ct, pr, ts, ref, aff, mx]) => {
        setTable(t); setGames(g); setStats(s)
        setCross(ct); setProgression(pr); setTeamStats(ts); setReferees(ref); setAffiliation(aff)
        setMatrices(mx)
      })
      .catch(() => setError('Staffeldaten konnten nicht geladen werden.'))
  }

  useEffect(reload, [selected?.id])
  useLiveUpdates((event) => {
    if (event !== 'bwhv-updated') return
    // Erst die Liste (eine Staffel kann gerade erst eine ID bekommen haben),
    // dann die Daten der gewählten. Wechselt dabei die ID, zieht der Effekt
    // oben ohnehin nach.
    void reloadStaffeln()
    reload()
  })

  // Der Abruf läuft serverseitig im Hintergrund weiter; die Seite lädt über
  // das SSE-Ereignis bwhv-updated nach, sobald etwas ankommt.
  const pollJetzt = async () => {
    if (!selected) return
    setPolling(true)
    setPollHinweis('')
    try {
      // Ohne Snapshot gibt es noch keine Staffel-ID; dann muss erst die
      // Auflösung gegen den Katalog laufen (SyncNow legt die Zeilen an).
      if (selected.id === 0) await syncStaffeln()
      else await api.post(`/staffeln/${selected.id}/poll`)
      setPollHinweis('Abruf gestartet — neue Daten erscheinen hier, sobald sie da sind.')
    } catch {
      setPollHinweis('Der Abruf konnte nicht gestartet werden.')
    } finally {
      setPolling(false)
    }
  }

  const setParam = (key: string, value: string) => {
    const next = new URLSearchParams(params)
    next.set(key, value)
    // Der Sprung zu einem Bericht gilt nur für den Spielplan; ein Reiterwechsel
    // soll ihn nicht bei der nächsten Rückkehr erneut aufklappen.
    if (key === 'tab') next.delete('bericht')
    setParams(next, { replace: true })
  }

  // Sprung aus der Spielmatrix zur Begegnung im Spielplan. Als echter
  // Verlaufseintrag (kein replace), damit "Zurück" in die Matrix führt. Die
  // Suche wird geleert — sie könnte die Zielbegegnung sonst ausblenden.
  const openGame = (bwhvGameId: number) => {
    const next = new URLSearchParams(params)
    next.set('tab', 'spielplan')
    next.set('bericht', String(bwhvGameId))
    setSearch('')
    setParams(next)
  }
  const focusGameId = Number(params.get('bericht')) || null

  if (loading) return <p className="text-sm text-brand-text-muted">Lade…</p>

  if (staffeln.length === 0) {
    return (
      <div>
        <h1 className="text-2xl font-bold text-brand-text mb-4">Staffeln</h1>
          <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            Für diese Saison ist noch keiner Mannschaft eine Staffel zugeordnet. Der Vorstand
            pflegt sie unter Verwaltung → Kader, direkt an der Mannschaft.
          </div>
      </div>
    )
  }

  return (
    <div>
      <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
        <h1 className="text-2xl font-bold text-brand-text">Staffeln</h1>
        <div className="flex items-center gap-2">
          <select
            className={HEADER_FIELD}
            value={selected?.id ?? ''}
            onChange={(e) => setParam('staffel', e.target.value)}
            aria-label="Mannschaft wählen"
          >
            {staffeln.map((s) => (
              <option key={s.id} value={s.id}>
                {s.teamShort || s.teamName || s.code}
              </option>
            ))}
          </select>
          {(tab === 'tabelle' || tab === 'spielplan') && (
            <input
              type="search"
              className={`${HEADER_FIELD} w-40 sm:w-48`}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Suchen…"
              aria-label="Suche in Tabelle und Spielplan"
            />
          )}
          {hasCapability('poll_bwhv') && (
            <button
              onClick={pollJetzt}
              disabled={polling || !selected}
              className={`${HEADER_CTRL} ${HEADER_NEUTRAL}`}
              title="Spielplan, Tabelle und offene Spielberichte dieser Staffel jetzt beim Verband abrufen"
            >
              <RefreshCw className={`w-4 h-4${polling ? ' animate-spin' : ''}`} />
              Jetzt abrufen
            </button>
          )}
        </div>
      </div>

      {selected && (
        <p className="text-sm text-brand-text-muted mb-4">
          {selected.name || selected.code}{' '}
          <span className="text-brand-text-subtle">({selected.code})</span>
        </p>
      )}

      {pollHinweis && (
        <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text mb-4">
          {pollHinweis}
        </div>
      )}

      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4">
          {error}
        </div>
      )}

      <div className="flex gap-2 mb-4 border-b border-brand-border-subtle overflow-x-auto">
        {TABS.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setParam('tab', id)}
            aria-current={tab === id || undefined}
            className={`inline-flex items-center gap-1 px-3 py-2 text-sm font-medium border-b-2 transition-colors shrink-0 whitespace-nowrap ${
              tab === id
                ? 'border-brand-yellow text-brand-text'
                : 'border-transparent text-brand-text-muted hover:text-brand-text'
            }`}
          >
            <Icon className="w-4 h-4" />
            {label}
          </button>
        ))}
      </div>

      {selected && !selected.polled ? (
        <Empty
          text={
            'Diese Staffel ist der Mannschaft zugeordnet, aber es wurde noch nichts beim ' +
            'Verband abgerufen. Der Abruf läuft an Spieltagen automatisch — oder der ' +
            'Vorstand stößt ihn oben mit „Jetzt abrufen" an.'
          }
        />
      ) : (
        <>
          {tab === 'tabelle' && <TableView rows={table.filter((r) => matchesSearch([r.TeamName], search))} searching={search.trim() !== ''} ownTeams={affiliation.teamNames} />}
          {tab === 'spielplan' && (
            <ScheduleView
              games={games.filter((g) => matchesSearch(gameSearchFields(g), search))}
              searching={search.trim() !== ''}
              ownTeams={affiliation.teamNames}
              halfDurationMinutes={staffelHalfDuration(matrices)}
              focusGameId={focusGameId}
            />
          )}
          {tab === 'kreuztabelle' && (
            cross ? <CrossTable data={cross} ownTeams={affiliation.teamNames} /> : <Empty text={NOCH_NICHTS} />
          )}
          {tab === 'verlauf' && (
            <VerlaufView
              days={progression}
              ownTeams={affiliation.teamNames}
              matrices={matrices}
              ownPlayers={affiliation.playerIds}
              onOpenGame={openGame}
            />
          )}
          {tab === 'tore' && (
            teamStats ? <GoalTablesView data={teamStats} ownTeams={affiliation.teamNames} /> : <Empty text={NOCH_NICHTS} />
          )}
          {tab === 'fairplay' && (
            teamStats ? <FairPlayView data={teamStats} ownTeams={affiliation.teamNames} /> : <Empty text={NOCH_NICHTS} />
          )}
          {tab === 'verteilung' && (
            teamStats ? <DistributionView data={teamStats} ownTeams={affiliation.teamNames} /> : <Empty text={NOCH_NICHTS} />
          )}
          {tab === 'ranglisten' && <RanglistenView stats={stats} ownPlayers={affiliation.playerIds} />}
          {tab === 'schiedsrichter' && <RefereeView rows={referees} />}
        </>
      )}
    </div>
  )
}

/**
 * Der Reiter „Verlauf" trägt zwei Darstellungen derselben Frage „wie ist es
 * gelaufen": oben für die Staffel (Platzierung je Spieltag), darunter für die
 * eigene Mannschaft (Spieler je Begegnung). Ein eigener Reiter dafür hätte die
 * Leiste auf zehn getrieben, und bestehende Verweise auf ?tab=verlauf bleiben
 * so gültig (design.md §12).
 */
function VerlaufView({ days, ownTeams, matrices, ownPlayers, onOpenGame }: {
  days: ProgressionDay[]
  ownTeams: string[]
  matrices: TeamMatrix[]
  ownPlayers: number[]
  onOpenGame: (bwhvGameId: number) => void
}) {
  return (
    <div className="space-y-6">
      <StandingsChart days={days} ownTeams={ownTeams} />
      {matrices.length === 0 ? (
        <Empty
          text={
            'Für die Spielmatrix fehlt noch die Verbindung zur eigenen Mannschaft: sie wird ' +
            'aus der Verknüpfung zwischen Verbands-Begegnung und eigenem Spieltermin ' +
            'abgeleitet, nicht über Namen geraten. Sobald ein eigenes Spiel dieser Staffel ' +
            'importiert ist, erscheint sie hier.'
          }
        />
      ) : (
        matrices.map((m) => <Spielmatrix key={m.team} matrix={m} ownPlayers={ownPlayers} onOpenGame={onOpenGame} />)
      )}
    </div>
  )
}

function TableView({ rows, ownTeams, searching }: { rows: TableRow[]; ownTeams: string[]; searching: boolean }) {
  if (rows.length === 0) return <Empty text={searching ? 'Keine Mannschaft passt zur Suche.' : 'Noch kein Tabellenstand abgerufen.'} />
  return (
    <div className={CARD}>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr>
              <th className={TH}>#</th>
              <th className={TH}>Mannschaft</th>
              <th className={TH}>Sp</th>
              <th className={`${TH} hidden sm:table-cell`}>S</th>
              <th className={`${TH} hidden sm:table-cell`}>U</th>
              <th className={`${TH} hidden sm:table-cell`}>N</th>
              <th className={TH}>Tore</th>
              <th className={TH}>Punkte</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => {
              const own = isOwnTeam(ownTeams, r.TeamName)
              return (
              <tr
                key={`${r.Position}-${r.TeamName}`}
                aria-current={own || undefined}
                className={own
                  ? 'bg-brand-table-select font-semibold'
                  : 'hover:bg-brand-table-select transition-colors'}
              >
                <td className={TD}>{r.Position}</td>
                <td className={`${TD} font-medium`}>{r.TeamName}</td>
                <td className={TD}>{r.Games}</td>
                <td className={`${TD} hidden sm:table-cell`}>{r.Won}</td>
                <td className={`${TD} hidden sm:table-cell`}>{r.Drawn}</td>
                <td className={`${TD} hidden sm:table-cell`}>{r.Lost}</td>
                <td className={TD}>{goalRatio(r)}</td>
                <td className={`${TD} font-medium`}>{pointsLabel(r)}</td>
              </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function ScheduleView({ games, ownTeams, searching, halfDurationMinutes, focusGameId }: {
  games: ScheduleGame[]
  ownTeams: string[]
  searching: boolean
  halfDurationMinutes: number | null
  /** Begegnung, zu der gesprungen wurde (?bericht=) — aufklappen und zeigen. */
  focusGameId: number | null
}) {
  // Genau ein Bericht ist aufgeklappt: der Bericht ist lang (zwei Mannschafts-
  // listen plus Spielverlauf), mehrere gleichzeitig machten die Liste unbenutzbar.
  const [openReportId, setOpenReportId] = useState<number | null>(focusGameId)
  const focusVorhanden = games.some((g) => g.ID === focusGameId)

  // Erst scrollen, wenn die Begegnung gerendert ist: der Spielplan kommt
  // asynchron, beim Sprung kann die Liste noch leer sein.
  useEffect(() => {
    if (focusGameId === null || !focusVorhanden) return
    setOpenReportId(focusGameId)
    document.getElementById(`spiel-${focusGameId}`)?.scrollIntoView?.({ block: 'start' })
  }, [focusGameId, focusVorhanden])
  if (games.length === 0) return <Empty text={searching ? 'Keine Begegnung passt zur Suche.' : 'Noch kein Spielplan abgerufen.'} />
  return (
    <div className="space-y-2">
      {games.map((g) => {
        const own = isOwnTeam(ownTeams, g.HomeTeam) || isOwnTeam(ownTeams, g.GuestTeam)
        return (
        <div
          key={g.ID}
          id={`spiel-${g.ID}`}
          aria-current={own || undefined}
          className={`rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-4 ${
            own ? 'bg-brand-table-select font-semibold' : 'bg-brand-surface-card'
          }`}
        >
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="text-xs text-brand-text-muted mb-1">
                {formatDate(g.Date)} {g.Time && `· ${g.Time}`}
                {g.GameID !== null && (
                  <span className="ml-2 inline-flex items-center gap-1 text-brand-text">
                    <Home className="w-3 h-3" /> eigenes Spiel
                  </span>
                )}
              </div>
              <div className="text-sm text-brand-text font-medium truncate">
                {g.HomeTeam} <span className="text-brand-text-subtle">–</span> {g.GuestTeam}
              </div>
              {g.Venue ? (
                <div className="mt-1">
                  <MapsLink venue={g.Venue} className="text-xs" />
                </div>
              ) : g.HallNumber && (
                <div className="text-xs text-brand-text-subtle mt-1 inline-flex items-center gap-1">
                  <MapPin className="w-3 h-3" /> Halle {g.HallNumber}
                </div>
              )}
            </div>
            <div className="text-right shrink-0">
              {g.HomeGoals !== null && g.GuestGoals !== null ? (
                <>
                  <div className="text-lg font-bold text-brand-text">
                    {g.HomeGoals}:{g.GuestGoals}
                  </div>
                  {g.HomeGoalsHT !== null && (
                    <div className="text-xs text-brand-text-muted">
                      ({g.HomeGoalsHT}:{g.GuestGoalsHT})
                    </div>
                  )}
                </>
              ) : (
                <span className="text-sm text-brand-text-subtle">—</span>
              )}
              {g.HasReport && (
                <button
                  type="button"
                  onClick={() => setOpenReportId(openReportId === g.ID ? null : g.ID)}
                  aria-expanded={openReportId === g.ID}
                  className="text-xs text-brand-text-muted hover:text-brand-text transition-colors mt-1 inline-flex items-center gap-1"
                >
                  <FileText className="w-3 h-3" /> Bericht
                  {openReportId === g.ID
                    ? <ChevronUp className="w-3 h-3" />
                    : <ChevronDown className="w-3 h-3" />}
                </button>
              )}
            </div>
          </div>
          {g.HasReport && openReportId === g.ID && (
            <div className="mt-3 pt-3 border-t border-brand-border-subtle font-normal">
              <SpielberichtPanel bwhvGameId={g.ID} halfDurationMinutes={halfDurationMinutes} />
            </div>
          )}
        </div>
        )
      })}
    </div>
  )
}

function RanglistenView({ stats, ownPlayers }: { stats: PlayerStat[]; ownPlayers: number[] }) {
  if (stats.length === 0) return <Empty text="Noch keine ausgewerteten Spielberichte." />

  const scorer = [...stats].filter((s) => s.goals > 0).slice(0, 20)
  // Nur Spieler mit mindestens einem Versuch: "nie geworfen" soll sich nicht
  // wie "immer verworfen" lesen. Sortiert wird nach getroffenen Siebenmetern,
  // nicht nach der Quote - ein einzelner verwandelter Wurf ergaebe sonst 100 %
  // und die Spitze der Liste.
  const sevenM = [...stats]
    .filter((s) => s.sevenMAttempts > 0)
    .sort((a, b) => b.sevenMGoals - a.sevenMGoals || a.sevenMMissed - b.sevenMMissed)
    .slice(0, 10)
  const fair = [...stats]
    .filter((s) => s.fairPlayScore > 0)
    .sort((a, b) => b.fairPlayScore - a.fairPlayScore)
    .slice(0, 10)

  return (
    <div className="space-y-6">
      <Ranking
        title="Torschützen" rows={scorer} ownPlayers={ownPlayers}
        tieKey={(s) => s.goals}
        render={(s) => `${s.goals} Tore in ${s.games} ${s.games === 1 ? 'Spiel' : 'Spielen'}`}
      />
      <Ranking
        title="Siebenmeter"
        hint="Nur Spieler mit mindestens einem Versuch."
        rows={sevenM}
        ownPlayers={ownPlayers}
        tieKey={(s) => `${s.sevenMGoals}|${s.sevenMMissed}`}
        render={(s) =>
          `${s.sevenMGoals}/${s.sevenMAttempts} · ${s.sevenMMissed} daneben · ` +
          `${sevenMeterRate(s)} % · ${s.games} ${s.games === 1 ? 'Spiel' : 'Spiele'}`
        }
      />
      <Ranking
        title="Meiste Strafen"
        rows={fair}
        ownPlayers={ownPlayers}
        tieKey={(s) => s.fairPlayScore}
        render={(s) =>
          [s.twoMin && `${s.twoMin}× 2 min`, s.warnings && `${s.warnings}× Verwarnung`, s.disq && `${s.disq}× Disq.`]
            .filter(Boolean)
            .join(' · ')
        }
      />
    </div>
  )
}

function Ranking({
  title, hint, rows, ownPlayers, render, tieKey,
}: {
  title: string
  hint?: string
  rows: PlayerStat[]
  ownPlayers: number[]
  /** Wert, über den Gleichstand entschieden wird (lib/ranking.ts). */
  tieKey: (s: PlayerStat) => string | number
  render: (s: PlayerStat) => string
}) {
  if (rows.length === 0) return null
  const ranks = sharedRanks(rows, tieKey)
  return (
    <div className={CARD}>
      <div className="px-4 py-3 border-b border-brand-border-subtle">
        <h2 className="text-sm font-medium text-brand-text">{title}</h2>
        {hint && <p className="text-xs text-brand-text-muted mt-1">{hint}</p>}
      </div>
      <ul>
        {rows.map((s, i) => {
          const own = isOwnPlayer(ownPlayers, s.playerId)
          return (
          <li
            key={s.playerId}
            aria-current={own || undefined}
            className={`flex items-center justify-between gap-3 px-4 py-2 ${
              own
                ? 'bg-brand-table-select font-semibold'
                : 'hover:bg-brand-table-select transition-colors'
            }`}
          >
            <span className="flex items-center gap-3 min-w-0">
              <span className="text-xs text-brand-text-subtle w-5 shrink-0">{ranks[i]}</span>
              <span className="min-w-0">
                <span className="text-sm text-brand-text block truncate">{s.name}</span>
                <span className="text-xs text-brand-text-muted block truncate">{s.teamName}</span>
              </span>
            </span>
            <span className="text-sm text-brand-text shrink-0">{render(s)}</span>
          </li>
          )
        })}
      </ul>
    </div>
  )
}

// Alle Mannschaften einer Staffel spielen in derselben Altersklasse — die
// gepflegte Halbzeitdauer der eigenen Mannschaft gilt deshalb für jede
// Begegnung der Staffel. Ohne eigene Mannschaft bleibt sie offen, der Ablauf
// leitet die Achse dann aus dem Verlauf ab.
function staffelHalfDuration(matrices: TeamMatrix[]): number | null {
  return matrices.find((m) => m.halfDurationMinutes !== null)?.halfDurationMinutes ?? null
}

function Empty({ text }: { text: string }) {
  return (
    <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
      {text}
    </div>
  )
}

// SQLite liefert DATE-Felder als ISO-Timestamp; für die Anzeige zählt nur der
// Datumsanteil (docs/agent/06-gotchas.md).
function formatDate(date: string): string {
  const d = new Date(`${date.slice(0, 10)}T12:00:00`)
  return d.toLocaleDateString('de-DE', { weekday: 'short', day: '2-digit', month: '2-digit', year: '2-digit' })
}
