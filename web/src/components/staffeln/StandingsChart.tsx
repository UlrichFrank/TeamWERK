import { useState } from 'react'
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

const W = 640
const H = 360
const PAD = { top: 16, right: 16, bottom: 32, left: 32 }

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
  const stepX = days.length > 1 ? (W - PAD.left - PAD.right) / (days.length - 1) : 0
  const stepY = maxRank > 1 ? (H - PAD.top - PAD.bottom) / (maxRank - 1) : 0

  const x = (i: number) => PAD.left + i * stepX
  const y = (rank: number) => PAD.top + (rank - 1) * stepY

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-4">
      <div className="overflow-x-auto">
        <svg
          viewBox={`0 0 ${W} ${H}`}
          className="min-w-[36rem] w-full h-auto"
          role="img"
          aria-label="Platzierungsverlauf über die Spieltage, beste Platzierung oben"
        >
          {Array.from({ length: maxRank }, (_, i) => i + 1).map((rank) => (
            <g key={rank}>
              <line
                x1={PAD.left} x2={W - PAD.right} y1={y(rank)} y2={y(rank)}
                stroke={GRID} strokeWidth={1}
              />
              <text x={4} y={y(rank) + 4} className="text-[10px]" fill={AXIS_TEXT}>{rank}</text>
            </g>
          ))}
          {days.map((d, i) => (
            <text
              key={d.date} x={x(i)} y={H - 10}
              textAnchor="middle" className="text-[10px]" fill={AXIS_TEXT}
            >
              {d.date.slice(8, 10)}.{d.date.slice(5, 7)}.
            </text>
          ))}
          {teams.map((team, ti) => {
            const own = isOwnTeam(ownTeams, team)
            const points = days
              .map((d, i) => {
                const e = d.entries.find((x) => x.team === team)
                return e ? `${x(i)},${y(e.rank)}` : null
              })
              .filter((p): p is string => p !== null)
            if (points.length === 0) return null
            // Die eigene Linie trägt die doppelte Strichstärke — dieselbe
            // Auszeichnung wie die Zeilenmarkierung in den Tabellen, nur in
            // der Form, die eine Grafik dafür hat.
            const width = own ? 4 : hover === team ? 3 : 1.5
            return (
              <polyline
                key={team}
                data-team={team}
                data-own={own || undefined}
                points={points.join(' ')}
                fill="none"
                stroke={COLORS[ti % COLORS.length]}
                strokeDasharray={DASHES[Math.floor(ti / COLORS.length) % DASHES.length]}
                strokeWidth={width}
                strokeLinejoin="round"
                opacity={hover === null || hover === team || own ? 1 : 0.25}
              />
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
