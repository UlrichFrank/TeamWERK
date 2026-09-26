import { useEffect, useState } from 'react'
import { AlertTriangle, Download, Users, Activity, TrendingUp, CircleDot } from 'lucide-react'
import {
  ReportDetail, PlayerLine, EventLine,
  fetchReport, fetchReportForGame, gameClock, sevenMeterRate,
} from '../lib/staffeln'
import { HEADER_CTRL, HEADER_PRIMARY, HEADER_NEUTRAL } from '../lib/buttonStyles'
import TorMomentum from './staffeln/TorMomentum'

const CARD = 'bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-4'
const TH = 'bg-brand-surface-card text-brand-text-muted text-xs uppercase px-3 py-2 text-left'
const TD = 'px-3 py-2 text-sm text-brand-text'

interface Props {
  /** BWHV-Begegnung (aus dem Staffel-Spielplan). */
  bwhvGameId?: number
  /** Eigener Spieltermin (aus der Detailansicht). Genau eines von beiden. */
  gameId?: number
  /** Gepflegte Halbzeitdauer der Altersklasse, falls bekannt — sonst leitet
   * der Ablauf die Achse aus dem Verlauf ab (docs/agent/06-gotchas.md). */
  halfDurationMinutes?: number | null
}

/** Panel mit dem offiziellen BWHV-Spielbericht zu einer Begegnung. */
export default function SpielberichtPanel({ bwhvGameId, gameId, halfDurationMinutes = null }: Props) {
  const [report, setReport] = useState<ReportDetail | null>(null)
  const [state, setState] = useState<'laden' | 'da' | 'fehlt' | 'fehler'>('laden')

  useEffect(() => {
    let active = true
    setState('laden')
    const load = gameId !== undefined ? fetchReportForGame(gameId) : fetchReport(bwhvGameId!)
    load
      .then((r) => { if (active) { setReport(r); setState('da') } })
      .catch((err) => {
        if (!active) return
        setState(err?.response?.status === 404 ? 'fehlt' : 'fehler')
      })
    return () => { active = false }
  }, [bwhvGameId, gameId])

  if (state === 'laden') return <p className="text-sm text-brand-text-muted">Lade Spielbericht…</p>
  if (state === 'fehlt') {
    return (
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
        Für dieses Spiel liegt noch kein Spielbericht vor. Er wird abgerufen, sobald der
        Verband ihn freigibt.
      </div>
    )
  }
  if (state === 'fehler' || !report) {
    return (
      <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
        Der Spielbericht konnte nicht geladen werden.
      </div>
    )
  }

  const home = report.players.filter((p) => p.side === 'home')
  const guest = report.players.filter((p) => p.side === 'guest')
  // Die Mannschaftsnamen kommen aus dem Bericht selbst — kein verdrahteter
  // Vereinsname in der Oberfläche.
  const { homeTeam, guestTeam } = report

  return (
    <div className="space-y-4">
      {report.warnings.length > 0 && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
          <p className="font-medium inline-flex items-center gap-1 mb-1">
            <AlertTriangle className="w-4 h-4" />
            Der Bericht ist in sich nicht schlüssig
          </p>
          <ul className="list-disc ml-5">
            {report.warnings.map((w) => <li key={w}>{w}</li>)}
          </ul>
        </div>
      )}

      <div className={CARD}>
        <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-brand-text">
          {report.spectators && (
            <span><span className="text-brand-text-muted">Zuschauer:</span> {report.spectators}</span>
          )}
          {report.referees && (
            <span><span className="text-brand-text-muted">Schiedsrichter:</span> {report.referees}</span>
          )}
          {report.hasPdf && (
            <a
              href={`/api/bwhv-reports/${report.reportId}/pdf`}
              className="inline-flex items-center gap-1 text-brand-text hover:text-brand-blue transition-colors"
            >
              <Download className="w-4 h-4" /> Bericht als PDF
            </a>
          )}
        </div>
      </div>

      <Spielverlauf report={report} halfDurationMinutes={halfDurationMinutes} />

      <div className="grid gap-4 sm:grid-cols-2">
        <Mannschaftsliste title={homeTeam} players={home} />
        <Mannschaftsliste title={guestTeam} players={guest} />
      </div>

      <Verlauf events={report.events} />
    </div>
  )
}

function Mannschaftsliste({ title, players }: { title: string; players: PlayerLine[] }) {
  if (players.length === 0) return null
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden">
      <h3 className="text-sm font-medium text-brand-text px-3 py-2 border-b border-brand-border-subtle inline-flex items-center gap-1">
        <Users className="w-4 h-4" /> {title}
      </h3>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr>
              <th className={TH}>Nr.</th>
              <th className={TH}>Name</th>
              <th className={TH}>Tore</th>
              <th className={TH}>7m</th>
              <th className={TH}>Strafen</th>
            </tr>
          </thead>
          <tbody>
            {players.map((p) => (
              <tr key={p.playerId} className="hover:bg-brand-table-select transition-colors">
                <td className={TD}>{p.number ?? '—'}</td>
                <td className={TD}>
                  {p.name}
                  {p.memberId === null && (
                    <span
                      className="ml-1 text-xs text-brand-text-subtle"
                      title="Diesem Eintrag ist kein Vereinsmitglied zugeordnet"
                    >
                      ·
                    </span>
                  )}
                </td>
                <td className={TD}>{p.goals || ''}</td>
                <td className={TD}>
                  {p.sevenMAttempts > 0 ? `${p.sevenMGoals}/${p.sevenMAttempts}` : ''}
                </td>
                <td className={TD}>
                  {[p.twoMin && `${p.twoMin}×2′`, p.warnings && 'V', p.disq && 'D']
                    .filter(Boolean)
                    .join(' ')}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

type Ansicht = 'kurve' | 'ablauf'

/**
 * Spielverlauf mit zwei Darstellungen derselben Ereignisliste: die Torkurve
 * (Differenz über die Spielzeit) und der Ablauf (Tor-Momentum je Halbzeit).
 * Umschaltbar statt nebeneinander — beide zeigen dieselbe Frage, und der Ablauf
 * ist allein schon zwei Achsen hoch.
 */
function Spielverlauf({ report, halfDurationMinutes }: { report: ReportDetail; halfDurationMinutes: number | null }) {
  const [ansicht, setAnsicht] = useState<Ansicht>('kurve')
  const { events, homeTeam, guestTeam } = report
  const hatTore = events.some((e) => e.scoreHome !== null && e.scoreGuest !== null)
  if (!hatTore) return null

  const schalter = (a: Ansicht, label: string, Icon: typeof Activity) => (
    <button
      type="button"
      onClick={() => setAnsicht(a)}
      aria-pressed={ansicht === a}
      className={`${HEADER_CTRL} ${ansicht === a ? HEADER_PRIMARY : HEADER_NEUTRAL}`}
    >
      <Icon className="w-4 h-4" /> {label}
    </button>
  )

  return (
    <div className={CARD}>
      <div className="flex flex-wrap items-center justify-between gap-2 mb-2">
        <h3 className="text-sm font-medium text-brand-text inline-flex items-center gap-1">
          <Activity className="w-4 h-4" /> Spielverlauf
        </h3>
        <div className="flex gap-2" role="group" aria-label="Darstellung des Spielverlaufs">
          {schalter('kurve', 'Kurve', TrendingUp)}
          {schalter('ablauf', 'Ablauf', CircleDot)}
        </div>
      </div>
      {ansicht === 'kurve' ? (
        <Torkurve events={events} homeTeam={homeTeam} guestTeam={guestTeam} />
      ) : (
        <TorMomentum
          events={events}
          homeTeam={homeTeam}
          guestTeam={guestTeam}
          homeGoalsHt={report.homeGoalsHt}
          guestGoalsHt={report.guestGoalsHt}
          halfDurationMinutes={halfDurationMinutes}
        />
      )}
    </div>
  )
}

/**
 * Torkurve: Spielstand über die Spielzeit als Flächen-Differenz.
 *
 * Bewusst inline-SVG statt Diagramm-Bibliothek — die Kurve ist ein Polygonzug
 * über höchstens ~60 Punkte, und eine Bibliothek wäre auf dem 1-GB-VPS
 * zusätzliches Bundle ohne Gegenwert.
 */
function Torkurve({ events, homeTeam, guestTeam }: { events: EventLine[]; homeTeam: string; guestTeam: string }) {
  const goals = events.filter((e) => e.scoreHome !== null && e.scoreGuest !== null)
  if (goals.length < 2) {
    return <p className="text-sm text-brand-text-muted">Zu wenige Tore für eine Kurve.</p>
  }

  const maxSec = Math.max(...goals.map((e) => e.gameSecond), 1)
  const maxDiff = Math.max(...goals.map((e) => Math.abs((e.scoreHome ?? 0) - (e.scoreGuest ?? 0))), 1)

  const W = 600
  const H = 120
  const x = (sec: number) => (sec / maxSec) * W
  const y = (diff: number) => H / 2 - (diff / maxDiff) * (H / 2 - 8)

  const points = goals.map((e) => `${x(e.gameSecond).toFixed(1)},${y((e.scoreHome ?? 0) - (e.scoreGuest ?? 0)).toFixed(1)}`)
  const path = `M0,${H / 2} L${points.join(' L')}`

  return (
    <div>
      <div className="flex justify-between text-xs text-brand-text-muted mb-1">
        <span className="truncate">{homeTeam} führt</span>
        <span className="truncate">{guestTeam} führt</span>
      </div>
      <svg viewBox={`0 0 ${W} ${H}`} className="w-full h-24" role="img"
           aria-label={`Torkurve: Führungswechsel zwischen ${homeTeam} und ${guestTeam}`}>
        <line x1="0" y1={H / 2} x2={W} y2={H / 2} stroke="#D1D5DB" strokeWidth="1" />
        <path d={path} fill="none" stroke="#3E4A98" strokeWidth="2" strokeLinejoin="round" />
      </svg>
      <p className="text-xs text-brand-text-subtle mt-1">
        Über der Linie führt {homeTeam}, darunter {guestTeam}. Höchste Differenz: {maxDiff}.
      </p>
    </div>
  )
}

const KIND_LABEL: Record<string, string> = {
  goal: 'Tor',
  seven_m_goal: '7m-Tor',
  seven_m_miss: '7m verworfen',
  two_min: '2-min-Strafe',
  warning: 'Verwarnung',
  disqualification: 'Disqualifikation',
  timeout: 'Auszeit',
  other: '',
}

function Verlauf({ events }: { events: EventLine[] }) {
  if (events.length === 0) return null
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden">
      <h3 className="text-sm font-medium text-brand-text px-3 py-2 border-b border-brand-border-subtle">
        Ereignisse
      </h3>
      <ul className="divide-y divide-brand-border-subtle">
        {events.map((e) => (
          <li key={e.seq} className="flex items-start gap-3 px-3 py-2">
            <span className="text-xs text-brand-text-subtle w-12 shrink-0 tabular-nums">
              {gameClock(e.gameSecond)}
            </span>
            <span className="text-xs text-brand-text w-14 shrink-0 tabular-nums font-medium">
              {e.scoreHome !== null ? `${e.scoreHome}:${e.scoreGuest}` : ''}
            </span>
            <span className="text-sm text-brand-text min-w-0">
              {KIND_LABEL[e.kind] && (
                <span className="text-brand-text-muted">{KIND_LABEL[e.kind]}: </span>
              )}
              {e.playerName ?? e.rawText}
              {e.playerName && e.number !== null && (
                <span className="text-brand-text-subtle"> (#{e.number})</span>
              )}
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
}

export { sevenMeterRate }
