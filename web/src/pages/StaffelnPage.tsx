import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Trophy, CalendarDays, ListOrdered, FileText, Home, MapPin, RefreshCw } from 'lucide-react'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { useAuth } from '../contexts/AuthContext'
import { api } from '../lib/api'
import { HEADER_CTRL, HEADER_NEUTRAL, HEADER_FIELD } from '../lib/buttonStyles'
import {
  Staffel, TableRow, ScheduleGame, PlayerStat,
  fetchStaffeln, fetchTable, fetchSchedule, fetchRanglisten, syncStaffeln,
  goalRatio, pointsLabel, sevenMeterRate,
} from '../lib/staffeln'

type Tab = 'tabelle' | 'spielplan' | 'ranglisten'

const TABS: { id: Tab; label: string; icon: typeof Trophy }[] = [
  { id: 'tabelle', label: 'Tabelle', icon: Trophy },
  { id: 'spielplan', label: 'Spielplan', icon: CalendarDays },
  { id: 'ranglisten', label: 'Ranglisten', icon: ListOrdered },
]

const TH = 'bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left'
const TD = 'px-4 py-3 text-sm text-brand-text'
const CARD = 'bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden'

export default function StaffelnPage() {
  const [params, setParams] = useSearchParams()
  const [staffeln, setStaffeln] = useState<Staffel[]>([])
  const [table, setTable] = useState<TableRow[]>([])
  const [games, setGames] = useState<ScheduleGame[]>([])
  const [stats, setStats] = useState<PlayerStat[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [polling, setPolling] = useState(false)
  const [pollHinweis, setPollHinweis] = useState('')
  const { hasCapability } = useAuth()

  const selectedId = Number(params.get('staffel')) || 0
  const tab = (params.get('tab') as Tab) || 'tabelle'
  const selected = useMemo(
    () => staffeln.find((s) => s.id === selectedId) ?? staffeln[0],
    [staffeln, selectedId],
  )

  useEffect(() => {
    fetchStaffeln()
      .then(setStaffeln)
      .catch(() => setError('Staffeln konnten nicht geladen werden.'))
      .finally(() => setLoading(false))
  }, [])

  const reload = () => {
    if (!selected || selected.id === 0) { setTable([]); setGames([]); setStats([]); return }
    Promise.all([
      fetchTable(selected.id),
      fetchSchedule(selected.id),
      fetchRanglisten(selected.id),
    ])
      .then(([t, g, s]) => { setTable(t); setGames(g); setStats(s) })
      .catch(() => setError('Staffeldaten konnten nicht geladen werden.'))
  }

  useEffect(reload, [selected?.id])
  useLiveUpdates((event) => { if (event === 'bwhv-updated') reload() })

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
    setParams(next, { replace: true })
  }

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
                {s.teamName || s.code}
              </option>
            ))}
          </select>
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

      <div className="flex gap-2 mb-4 border-b border-brand-border-subtle">
        {TABS.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setParam('tab', id)}
            className={`inline-flex items-center gap-1 px-3 py-2 text-sm font-medium border-b-2 transition-colors ${
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
          {tab === 'tabelle' && <TableView rows={table} />}
          {tab === 'spielplan' && <ScheduleView games={games} />}
          {tab === 'ranglisten' && <RanglistenView stats={stats} />}
        </>
      )}
    </div>
  )
}

function TableView({ rows }: { rows: TableRow[] }) {
  if (rows.length === 0) return <Empty text="Noch kein Tabellenstand abgerufen." />
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
            {rows.map((r) => (
              <tr key={`${r.Position}-${r.TeamName}`} className="hover:bg-brand-table-select transition-colors">
                <td className={TD}>{r.Position}</td>
                <td className={`${TD} font-medium`}>{r.TeamName}</td>
                <td className={TD}>{r.Games}</td>
                <td className={`${TD} hidden sm:table-cell`}>{r.Won}</td>
                <td className={`${TD} hidden sm:table-cell`}>{r.Drawn}</td>
                <td className={`${TD} hidden sm:table-cell`}>{r.Lost}</td>
                <td className={TD}>{goalRatio(r)}</td>
                <td className={`${TD} font-medium`}>{pointsLabel(r)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function ScheduleView({ games }: { games: ScheduleGame[] }) {
  if (games.length === 0) return <Empty text="Noch kein Spielplan abgerufen." />
  return (
    <div className="space-y-2">
      {games.map((g) => (
        <div key={g.ID} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-4">
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
              {g.HallNumber && (
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
                <div className="text-xs text-brand-text-muted mt-1 inline-flex items-center gap-1">
                  <FileText className="w-3 h-3" /> Bericht
                </div>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

function RanglistenView({ stats }: { stats: PlayerStat[] }) {
  if (stats.length === 0) return <Empty text="Noch keine ausgewerteten Spielberichte." />

  const scorer = [...stats].filter((s) => s.goals > 0).slice(0, 20)
  const sevenM = [...stats]
    .filter((s) => s.sevenMAttempts > 0)
    .sort((a, b) => (sevenMeterRate(b) ?? 0) - (sevenMeterRate(a) ?? 0))
    .slice(0, 10)
  const fair = [...stats]
    .filter((s) => s.fairPlayScore > 0)
    .sort((a, b) => b.fairPlayScore - a.fairPlayScore)
    .slice(0, 10)

  return (
    <div className="space-y-6">
      <Ranking title="Torschützen" rows={scorer} render={(s) => `${s.goals} Tore`} />
      <Ranking
        title="Siebenmeter-Quote"
        rows={sevenM}
        render={(s) => `${sevenMeterRate(s)} % (${s.sevenMGoals}/${s.sevenMAttempts})`}
      />
      <Ranking
        title="Meiste Strafen"
        rows={fair}
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
  title, rows, render,
}: { title: string; rows: PlayerStat[]; render: (s: PlayerStat) => string }) {
  if (rows.length === 0) return null
  return (
    <div className={CARD}>
      <h2 className="text-sm font-medium text-brand-text px-4 py-3 border-b border-brand-border-subtle">{title}</h2>
      <ul>
        {rows.map((s, i) => (
          <li
            key={s.playerId}
            className="flex items-center justify-between gap-3 px-4 py-2 hover:bg-brand-table-select transition-colors"
          >
            <span className="flex items-center gap-3 min-w-0">
              <span className="text-xs text-brand-text-subtle w-5 shrink-0">{i + 1}</span>
              <span className="min-w-0">
                <span className="text-sm text-brand-text block truncate">{s.name}</span>
                <span className="text-xs text-brand-text-muted block truncate">{s.teamName}</span>
              </span>
            </span>
            <span className="text-sm text-brand-text shrink-0">{render(s)}</span>
          </li>
        ))}
      </ul>
    </div>
  )
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
