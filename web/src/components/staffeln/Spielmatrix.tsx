import { useState } from 'react'
import { Activity, Home, MapPin } from 'lucide-react'
import type { MatrixCell, MatrixGame, TeamMatrix } from '../../lib/staffeln'
import { HEADER_CTRL, HEADER_NEUTRAL } from '../../lib/buttonStyles'
import { isOwnPlayer } from '../../lib/staffelHighlight'
import TorMomentumModal from './TorMomentumModal'

// Sechs Werte je Begegnung. Ohne "Blau": bwhv_player_games.blue wird konstant
// als 0 geschrieben, der Parser liest keine blauen Karten — eine Spalte aus
// lauter Strichen behauptete eine Messung (design.md §10).
//
// `mobil` markiert die Spalte, die auf schmalen Bildschirmen stehen bleibt:
// sechs Spalten mal achtzehn Begegnungen sind 108 Spalten (design.md §11).
const WERTE: { key: keyof MatrixCell; label: string; titel: string; mobil?: boolean }[] = [
  { key: 'goals', label: 'Tore', titel: 'Tore', mobil: true },
  { key: 'sevenMAttempts', label: '7m', titel: 'Siebenmeter-Versuche' },
  { key: 'sevenMGoals', label: '7m+', titel: 'Siebenmeter-Tore' },
  { key: 'twoMin', label: '2min', titel: 'Zeitstrafen' },
  { key: 'warnings', label: 'Gelb', titel: 'Verwarnungen' },
  { key: 'disq', label: 'Rot', titel: 'Disqualifikationen' },
]

const TH = 'bg-brand-surface-card text-brand-text-muted text-xs uppercase px-2 py-2 text-center font-medium'
const TD = 'px-2 py-2 text-sm text-brand-text text-center tabular-nums'
// Die Namensspalte bleibt beim waagerechten Blättern stehen — ohne sie ist
// eine Zeile nach drei Spalten nicht mehr zuzuordnen.
const STICKY = 'sticky left-0 z-10 text-left whitespace-nowrap'

/** Nur die auf Mobile sichtbaren Wertespalten tragen keine sm:-Schranke. */
const wertKlasse = (w: (typeof WERTE)[number]) => (w.mobil ? '' : 'hidden sm:table-cell')

/**
 * Spielmatrix einer Mannschaft: Zeile = Spieler, Spaltengruppe = Begegnung.
 *
 * Die Tabelle wird bewusst NICHT zum Card-Layout aufgelöst, anders als die
 * übrigen Listen auf Mobile: eine Kreuztabelle in Karten zu zerlegen hieße,
 * ihre einzige Aussage — den Vergleich über die Zeile — wegzuwerfen. Dieselbe
 * Ausnahme gilt schon für die Kreuztabelle der Staffel.
 */
export default function Spielmatrix({ matrix, ownPlayers }: { matrix: TeamMatrix; ownPlayers: number[] }) {
  const [ablauf, setAblauf] = useState<MatrixGame | null>(null)

  if (matrix.games.length === 0) {
    return (
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
        Für {matrix.team} ist in dieser Staffel noch keine Begegnung gespielt.
      </div>
    )
  }

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden">
      <div className="px-4 py-3 border-b border-brand-border-subtle">
        <h2 className="text-sm font-medium text-brand-text">{matrix.team}</h2>
        <p className="text-xs text-brand-text-muted mt-1">
          {matrix.games.length} {matrix.games.length === 1 ? 'Spiel' : 'Spiele'}, davon{' '}
          {matrix.reportGames} mit Spielbericht.
          {matrix.reportGames < matrix.games.length &&
            ' Spalten ohne Bericht tragen den Endstand, aber keine Einzelwerte.'}
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse">
          <thead>
            <tr>
              <th className={`${TH} ${STICKY} bg-brand-surface-card`}>Spieler</th>
              {matrix.games.map((g) => (
                <th
                  key={g.bwhvGameId}
                  colSpan={WERTE.length}
                  className={`${TH} border-l border-brand-border-subtle align-top`}
                >
                  <SpaltenKopf game={g} onAblauf={() => setAblauf(g)} />
                </th>
              ))}
              <th
                colSpan={WERTE.length + 1}
                className={`${TH} border-l border-brand-border-subtle`}
              >
                Gesamt
              </th>
            </tr>
            <tr>
              <th className={`${TH} ${STICKY} bg-brand-surface-card`} />
              {matrix.games.map((g) => (
                WERTE.map((w) => (
                  <th
                    key={`${g.bwhvGameId}-${w.key}`}
                    title={w.titel}
                    className={`${TH} ${wertKlasse(w)} ${w.key === 'goals' ? 'border-l border-brand-border-subtle' : ''}`}
                  >
                    {w.label}
                  </th>
                ))
              ))}
              {WERTE.map((w) => (
                <th
                  key={`total-${w.key}`}
                  title={w.titel}
                  className={`${TH} ${wertKlasse(w)} ${w.key === 'goals' ? 'border-l border-brand-border-subtle' : ''}`}
                >
                  {w.label}
                </th>
              ))}
              <th className={TH} title="Spiele mit Eintrag in der Mannschaftsliste">Sp</th>
            </tr>
          </thead>
          <tbody>
            {matrix.players.map((p) => {
              const own = isOwnPlayer(ownPlayers, p.playerId)
              const bg = own ? 'bg-brand-table-select' : 'bg-white'
              return (
                <tr
                  key={p.playerId}
                  aria-current={own || undefined}
                  className={`${own ? 'bg-brand-table-select font-semibold' : 'hover:bg-brand-table-select transition-colors'} border-t border-brand-border-subtle`}
                >
                  <td className={`${TD} ${STICKY} ${bg} font-medium`}>{p.name}</td>
                  {p.cells.map((c, i) => (
                    WERTE.map((w) => (
                      <td
                        key={`${matrix.games[i].bwhvGameId}-${w.key}`}
                        className={`${TD} ${wertKlasse(w)} ${w.key === 'goals' ? 'border-l border-brand-border-subtle' : ''}`}
                      >
                        {/* "–" heißt "stand nicht in der Mannschaftsliste",
                            "0" heißt "war dabei und hat nicht getroffen". */}
                        {c === null
                          ? <span className="text-brand-text-subtle">–</span>
                          : zahl(c[w.key])}
                      </td>
                    ))
                  ))}
                  <SummenZellen cell={p.total} bold />
                  <td className={`${TD} font-medium`}>{p.games}</td>
                </tr>
              )
            })}
            <tr className="border-t-2 border-brand-border font-semibold">
              <td className={`${TD} ${STICKY} bg-brand-surface-card text-left`}>Mannschaft</td>
              {matrix.gameTotals.map((c, i) => (
                WERTE.map((w) => (
                  <td
                    key={`gt-${matrix.games[i].bwhvGameId}-${w.key}`}
                    className={`${TD} ${wertKlasse(w)} ${w.key === 'goals' ? 'border-l border-brand-border-subtle' : ''}`}
                  >
                    {matrix.games[i].hasReport ? zahl(c[w.key]) : <span className="text-brand-text-subtle">–</span>}
                  </td>
                ))
              ))}
              <SummenZellen cell={matrix.total} bold />
              <td className={TD}>{matrix.reportGames}</td>
            </tr>
          </tbody>
        </table>
      </div>

      {ablauf && (
        <TorMomentumModal
          game={ablauf}
          halfDurationMinutes={matrix.halfDurationMinutes}
          onClose={() => setAblauf(null)}
        />
      )}
    </div>
  )
}

// Eine 0 bleibt eine 0 — nur die fehlende Zelle wird zum Strich. Werte ohne
// Bedeutung (keine Karte, keine Strafe) würden als Nullenteppich die Tore
// erschlagen, deshalb sind sie gedämpft statt versteckt.
function zahl(v: number) {
  return v === 0 ? <span className="text-brand-text-subtle">0</span> : v
}

function SummenZellen({ cell, bold }: { cell: MatrixCell; bold?: boolean }) {
  return (
    <>
      {WERTE.map((w) => (
        <td
          key={`sum-${w.key}`}
          className={`${TD} ${wertKlasse(w)} ${bold ? 'font-semibold' : ''} ${w.key === 'goals' ? 'border-l border-brand-border-subtle' : ''}`}
        >
          {zahl(cell[w.key])}
        </td>
      ))}
    </>
  )
}

function SpaltenKopf({ game, onAblauf }: { game: MatrixGame; onAblauf: () => void }) {
  const gegner = game.isHome ? game.guestTeam : game.homeTeam
  return (
    <div className="flex flex-col items-center gap-1 normal-case min-w-[9rem]">
      <span className="text-brand-text-muted">{formatDate(game.date)}</span>
      <span className="inline-flex items-center gap-1 text-brand-text font-semibold text-sm tabular-nums">
        {game.isHome ? <Home className="w-3 h-3" /> : <MapPin className="w-3 h-3" />}
        {game.homeGoals}:{game.guestGoals}
      </span>
      <span className="text-brand-text-muted max-w-[9rem] truncate" title={gegner}>{gegner}</span>
      {game.hasReport ? (
        <button type="button" onClick={onAblauf} className={`${HEADER_CTRL} ${HEADER_NEUTRAL}`}>
          <Activity className="w-4 h-4" /> Ablauf
        </button>
      ) : (
        <span className="text-brand-text-subtle">kein Bericht</span>
      )}
    </div>
  )
}

// SQLite liefert DATE-Felder als ISO-Timestamp; für die Anzeige zählt nur der
// Datumsanteil (docs/agent/06-gotchas.md).
function formatDate(date: string): string {
  const d = new Date(`${date.slice(0, 10)}T12:00:00`)
  return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: '2-digit' })
}
