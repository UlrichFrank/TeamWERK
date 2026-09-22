import { useMemo } from 'react'
import type { EventLine } from '../../lib/staffeln'
import { torereihe, halbzeitGrenze, halbzeitMinuten, achsen } from '../../lib/torverlauf'
import type { Halbzeit, Situation, Tor } from '../../lib/torverlauf'

// Die Palette steht als Zahlenwert und nicht als Klasse, weil sie an
// SVG-Attribute (fill) geht — eine Tailwind-Klasse ließe sich dafür nicht zur
// Bauzeit erzeugen. Es sind exakt die brand-Tokens aus tailwind.config.js:
// brand-blue, brand-text-subtle, brand-warning.
const FARBE: Record<Situation, string> = {
  fuehrung: '#3E4A98',
  gleichstand: '#9CA3AF',
  rueckstand: '#F59E0B',
}
const BESCHRIFTUNG: Record<Situation, string> = {
  fuehrung: 'in Führung',
  gleichstand: 'unentschieden',
  rueckstand: 'im Rückstand',
}
// brand-border-subtle (Achsen) und brand-text-muted (Beschriftung).
const ACHSE = '#E5E7EB'
const ACHSE_TEXT = '#6B7280'

const W = 640
const H = 120
const PAD = { top: 26, right: 16, bottom: 18, left: 52 }

/**
 * Tor-Momentum einer Begegnung: je Halbzeit eine Zeitachse, je Tor ein Kreis
 * auf der Achse der werfenden Mannschaft.
 *
 * Die GRÖSSE zeigt den Lauf (wievieltes Tor in Folge ohne Gegentreffer), die
 * FARBE die Spielsituation nach diesem Tor. Beides sind unabhängige Aussagen,
 * sodass die Grafik auch ohne Farbunterscheidung etwas zeigt; jeder Kreis
 * trägt zusätzlich ein <title> mit Minute, Schütze und Spielstand.
 *
 * Bewusst Inline-SVG statt <canvas> wie im Vorbild: ein Kreis ist ein
 * <circle>, der Tooltip ein <title>, und Tastaturfokus gibt es geschenkt —
 * statt Trefferprüfung, devicePixelRatio-Skalierung und eigenem Tooltip von
 * Hand (design.md §9).
 */
export default function TorMomentum({
  events, homeTeam, guestTeam, homeGoalsHt, guestGoalsHt, halfDurationMinutes,
}: {
  events: EventLine[]
  homeTeam: string
  guestTeam: string
  homeGoalsHt: number | null
  guestGoalsHt: number | null
  halfDurationMinutes: number | null
}) {
  const halbzeiten = useMemo(() => {
    const tore = torereihe(events)
    if (tore.length === 0) return []
    const grenze = halbzeitGrenze(tore, homeGoalsHt, guestGoalsHt)
    return achsen(tore, grenze, halbzeitMinuten(tore, grenze, halfDurationMinutes))
  }, [events, homeGoalsHt, guestGoalsHt, halfDurationMinutes])

  if (halbzeiten.length === 0) {
    return (
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
        Zu dieser Begegnung sind keine Tore im Spielverlauf erfasst.
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {halbzeiten.map((hz) => (
        <Achse key={hz.nummer} hz={hz} homeTeam={homeTeam} guestTeam={guestTeam} einzeln={halbzeiten.length === 1} />
      ))}
      <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-brand-text-muted">
        {(Object.keys(FARBE) as Situation[]).map((s) => (
          <li key={s} className="inline-flex items-center gap-1">
            <span className="inline-block w-3 h-3 rounded-full" style={{ backgroundColor: FARBE[s] }} />
            {BESCHRIFTUNG[s]}
          </li>
        ))}
        <li className="text-brand-text-subtle">Größe = Tore in Folge</li>
      </ul>
    </div>
  )
}

function Achse({ hz, homeTeam, guestTeam, einzeln }: {
  hz: Halbzeit
  homeTeam: string
  guestTeam: string
  einzeln: boolean
}) {
  const spanne = Math.max(hz.bisMinute - hz.vonMinute, 1)
  const x = (t: Tor) => PAD.left + ((t.gameSecond / 60 - hz.vonMinute) / spanne) * (W - PAD.left - PAD.right)
  const yHeim = PAD.top + (H - PAD.top - PAD.bottom) / 4
  const yGast = PAD.top + (3 * (H - PAD.top - PAD.bottom)) / 4

  // Minutenraster im Fünfertakt, damit die Beschriftung die Spielminute des
  // GESAMTEN Spiels nennt und nicht bei der zweiten Halbzeit wieder bei 0
  // beginnt — die Spielzeit läuft kumulativ.
  const marken: number[] = []
  for (let m = hz.vonMinute; m <= hz.bisMinute; m += 5) marken.push(m)

  return (
    <div className="bg-white rounded-lg border border-brand-border-subtle p-2">
      <h4 className="text-xs font-medium text-brand-text mb-1">
        {einzeln ? 'Spielverlauf' : `Halbzeit ${hz.nummer}`}
        <span className="text-brand-text-muted font-normal">
          {' '}({hz.vonMinute}.–{hz.bisMinute}. Minute)
        </span>
      </h4>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="w-full h-28"
        role="img"
        aria-label={`Tore je Minute, ${homeTeam} oben, ${guestTeam} unten`}
      >
        <line x1={PAD.left} y1={yHeim} x2={W - PAD.right} y2={yHeim} stroke={ACHSE} strokeWidth="1" />
        <line x1={PAD.left} y1={yGast} x2={W - PAD.right} y2={yGast} stroke={ACHSE} strokeWidth="1" />
        {marken.map((m) => (
          <text
            key={m}
            x={PAD.left + ((m - hz.vonMinute) / spanne) * (W - PAD.left - PAD.right)}
            y={PAD.top - 10}
            textAnchor="middle"
            fontSize="11"
            fill={ACHSE_TEXT}
          >
            {m}'
          </text>
        ))}
        <text x={PAD.left - 8} y={yHeim + 4} textAnchor="end" fontSize="11" fill={ACHSE_TEXT}>Heim</text>
        <text x={PAD.left - 8} y={yGast + 4} textAnchor="end" fontSize="11" fill={ACHSE_TEXT}>Gast</text>
        {hz.tore.map((t) => (
          <circle
            key={t.seq}
            cx={x(t)}
            cy={t.side === 'home' ? yHeim : yGast}
            r={4 + t.lauf * 1.5}
            fill={FARBE[t.situation]}
            stroke="#181310"
            strokeWidth="1"
            tabIndex={0}
          >
            <title>{torBeschriftung(t, homeTeam, guestTeam)}</title>
          </circle>
        ))}
      </svg>
    </div>
  )
}

/**
 * Die Beschriftung eines Kreises. Sie trägt alles, was die Grafik über Farbe
 * und Größe sagt, noch einmal in Worten — Farbe ist damit nicht der einzige
 * Träger der Aussage.
 */
function torBeschriftung(t: Tor, homeTeam: string, guestTeam: string): string {
  const minute = Math.floor(t.gameSecond / 60)
  const sekunde = String(t.gameSecond % 60).padStart(2, '0')
  const teile = [
    `${minute}:${sekunde}`,
    `${t.scoreHome}:${t.scoreGuest}`,
    t.scorer || (t.side === 'home' ? homeTeam : guestTeam),
  ]
  if (t.sevenMeter) teile.push('Siebenmeter')
  if (t.lauf > 1) teile.push(`${t.lauf}. Tor in Folge`)
  teile.push(BESCHRIFTUNG[t.situation])
  return teile.join(' · ')
}
