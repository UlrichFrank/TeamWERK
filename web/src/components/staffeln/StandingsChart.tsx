import { useState } from 'react'
import { gespieltePunkte } from '../../lib/staffeln'
import type { ProgressionDay } from '../../lib/staffeln'
import { isOwnTeam } from '../../lib/staffelHighlight'

// Die Palette steht hier als Zahlenwert und nicht als Klasse, weil sie an
// SVG-Attribute (stroke, fill) geht und je Mannschaft wechselt — eine
// Tailwind-Klasse ließe sich dafür nicht zur Bauzeit erzeugen. Es sind exakt
// die brand-Tokens aus tailwind.config.js, keine Rohfarben:
// brand-black, brand-blue, brand-green, brand-danger, brand-info,
// brand-text-subtle.
const COLORS = ['#181310', '#3E4A98', '#6EB42E', '#C0253A', '#3B82F6', '#9CA3AF']
// brand-border-subtle (Gitterlinien) und brand-text-muted (Achsenbeschriftung).
const GRID = '#E5E7EB'
const AXIS_TEXT = '#6B7280'

// Bei mehr Mannschaften als Farben wird rotiert — und zusätzlich die
// Strichführung variiert, damit die Zuordnung ohne Farbsehen möglich bleibt.
const DASHES = ['', '6 3', '2 3']

// Feste Pixelmaße statt viewBox-Skalierung: der Zeilenabstand entspricht einer
// Tabellenzeile (px-2 py-2 text-sm ≈ 36 px, siehe Spielmatrix), der
// Spaltenabstand demselben Wert — so bleibt die Kurve so dicht wie die Tabelle
// darunter, statt mit der Fensterbreite aufzublähen. Viele Spieltage scrollen
// waagerecht.
const STEP = 36
const PAD = { top: 12, right: 16, bottom: 30, left: 28 }

/**
 * Platzierungsverlauf als Inline-SVG — eine Polyline je Mannschaft.
 *
 * Bewusst ohne Chart-Bibliothek: ein Rang-Verlauf ist eine Polyline, und
 * recharts wären ~95 kB gzip im PWA-Cache für eine Abhängigkeit, die sonst
 * niemand im Projekt nutzt (design.md §8). Legende und Hervorhebung bei Hover
 * sind selbst gebaut.
 *
 * Die Y-Achse ist INVERTIERT: Rang 1 steht oben. Ein Verlauf, in dem der
 * Tabellenführer unten läuft, liest sich falsch herum.
 */
export default function StandingsChart({
  days, ownTeams,
}: { days: ProgressionDay[]; ownTeams: string[] }) {
  const [hover, setHover] = useState<string | null>(null)

  if (days.length === 0) {
    return (
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
        Noch keine gespielte Begegnung — der Verlauf beginnt mit dem ersten Ergebnis.
      </div>
    )
  }

  const teams = Array.from(new Set(days.flatMap((d) => d.entries.map((e) => e.team)))).sort()
  const maxRank = Math.max(...days.flatMap((d) => d.entries.map((e) => e.rank)))
  const W = PAD.left + (days.length - 1) * STEP + PAD.right
  const H = PAD.top + (maxRank - 1) * STEP + PAD.bottom

  const x = (i: number) => PAD.left + i * STEP
  const y = (rank: number) => PAD.top + (rank - 1) * STEP

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-4">
      <div className="overflow-x-auto">
        <svg
          width={W}
          height={H}
          viewBox={`0 0 ${W} ${H}`}
          className="block"
          role="img"
          aria-label="Platzierungsverlauf über die Spieltage, beste Platzierung oben"
        >
          {Array.from({ length: maxRank }, (_, i) => i + 1).map((rank) => (
            <g key={rank}>
              <line
                x1={PAD.left} x2={W - PAD.right} y1={y(rank)} y2={y(rank)}
                stroke={GRID} strokeWidth={1}
              />
              <text x={PAD.left - 10} y={y(rank) + 4} textAnchor="end" className="text-[10px]" fill={AXIS_TEXT}>{rank}</text>
            </g>
          ))}
          {days.map((d, i) => (
            <text
              key={d.date} x={x(i)} y={H - 8}
              textAnchor="middle" className="text-[10px]" fill={AXIS_TEXT}
            >
              {kurzDatum(d.date)}
            </text>
          ))}
          {teams.map((team, ti) => {
            const own = isOwnTeam(ownTeams, team)
            const punkte = gespieltePunkte(days, team)
            if (punkte.length === 0) return null
            const color = COLORS[ti % COLORS.length]
            const faded = hover !== null && hover !== team && !own
            // Die eigene Linie trägt die doppelte Strichstärke — dieselbe
            // Auszeichnung wie die Zeilenmarkierung in den Tabellen, nur in
            // der Form, die eine Grafik dafür hat.
            const width = own ? 4 : hover === team ? 3 : 1.5
            return (
              <g key={team} data-team={team} data-own={own || undefined} opacity={faded ? 0.25 : 1}>
                {/* Eine Polyline aus einem Punkt zeichnet nichts — am ersten
                    Spieltag steht deshalb nur der Punkt, keine Linie. */}
                {punkte.length > 1 && (
                  <polyline
                    points={punkte.map((p) => `${x(p.day)},${y(p.rank)}`).join(' ')}
                    fill="none"
                    stroke={color}
                    strokeDasharray={DASHES[Math.floor(ti / COLORS.length) % DASHES.length]}
                    strokeWidth={width}
                    strokeLinejoin="round"
                  />
                )}
                {punkte.map((p) => (
                  <circle
                    key={p.day}
                    cx={x(p.day)}
                    cy={y(p.rank)}
                    r={own ? 4.5 : 3.5}
                    fill={color}
                    onMouseEnter={() => setHover(team)}
                    onMouseLeave={() => setHover(null)}
                  >
                    <title>{`${team} · ${kurzDatum(days[p.day].date)} · Platz ${p.rank}`}</title>
                  </circle>
                ))}
              </g>
            )
          })}
        </svg>
      </div>
      <ul className="flex flex-wrap gap-x-4 gap-y-1 mt-3">
        {teams.map((team, ti) => {
          const own = isOwnTeam(ownTeams, team)
          return (
            <li
              key={team}
              onMouseEnter={() => setHover(team)}
              onMouseLeave={() => setHover(null)}
              aria-current={own || undefined}
              className={`inline-flex items-center gap-1 text-xs cursor-default ${
                own ? 'bg-brand-table-select font-semibold px-1 rounded' : 'text-brand-text-muted'
              }`}
            >
              <span
                className="inline-block w-4 h-0.5 shrink-0"
                style={{ backgroundColor: COLORS[ti % COLORS.length] }}
                aria-hidden="true"
              />
              {team}
            </li>
          )
        })}
      </ul>
    </div>
  )
}

function kurzDatum(date: string): string {
  return `${date.slice(8, 10)}.${date.slice(5, 7)}.`
}
