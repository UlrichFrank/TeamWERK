import type { ReactNode } from 'react'
import type { TeamStat, TeamStats } from '../../lib/staffeln'
import { isOwnTeam } from '../../lib/staffelHighlight'
import { sharedRanks } from '../../lib/ranking'

const TH = 'bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left'
const TD = 'px-4 py-3 text-sm text-brand-text'
const CARD = 'bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden'

/** Eine Mannschafts-Tabelle mit Titel, Erläuterung und eigenen Spalten. */
function TeamTable({
  title, hint, rows, ownTeams, columns, tieKey,
}: {
  title: string
  hint: string
  rows: TeamStat[]
  ownTeams: string[]
  /** Wert, über den Gleichstand entschieden wird — die angezeigte Größe (lib/ranking.ts). */
  tieKey: (s: TeamStat) => string | number
  columns: { label: string; cell: (s: TeamStat) => ReactNode; strong?: boolean }[]
}) {
  const ranks = sharedRanks(rows, tieKey)
  return (
    <div className={CARD}>
      <div className="px-4 py-3 border-b border-brand-border-subtle">
        <h2 className="text-sm font-medium text-brand-text">{title}</h2>
        <p className="text-xs text-brand-text-muted mt-1">{hint}</p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr>
              <th className={TH}>#</th>
              <th className={TH}>Mannschaft</th>
              {columns.map((c) => <th key={c.label} className={TH}>{c.label}</th>)}
            </tr>
          </thead>
          <tbody>
            {rows.map((s, i) => {
              const own = isOwnTeam(ownTeams, s.team)
              return (
                <tr
                  key={s.team}
                  aria-current={own || undefined}
                  className={own
                    ? 'bg-brand-table-select font-semibold'
                    : 'hover:bg-brand-table-select transition-colors'}
                >
                  <td className={TD}>{ranks[i]}</td>
                  <td className={TD}>{s.team}</td>
                  {columns.map((c) => (
                    <td key={c.label} className={`${TD}${c.strong ? ' font-medium' : ''}`}>
                      {c.cell(s)}
                    </td>
                  ))}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

const perGame = (value: number, games: number) =>
  games === 0 ? '–' : (value / games).toFixed(1).replace('.', ',')

/**
 * Torverhältnis, bester Angriff, beste Verteidigung — drei Sortierungen
 * derselben Zahlen, alle aus den Ergebnissen gebildet und deshalb auch ohne
 * einen einzigen Spielbericht gefüllt.
 */
export function GoalTablesView({ data, ownTeams }: { data: TeamStats; ownTeams: string[] }) {
  const played = data.teams.filter((s) => s.games > 0)
  if (played.length === 0) return <Empty />

  const byDiff = [...played].sort((a, b) => b.goalDiff - a.goalDiff || b.goalsFor - a.goalsFor)
  const byAttack = [...played].sort((a, b) => b.goalsFor / b.games - a.goalsFor / a.games)
  const byDefense = [...played].sort((a, b) => a.goalsAgainst / a.games - b.goalsAgainst / b.games)

  return (
    <div className="space-y-6">
      <TeamTable
        title="Torverhältnis" ownTeams={ownTeams} rows={byDiff}
        tieKey={(s) => `${s.goalDiff}|${s.goalsFor}`}
        hint="Aus den Ergebnissen aller Begegnungen — kein Spielbericht nötig."
        columns={[
          { label: 'Sp', cell: (s) => s.games },
          { label: 'Tore', cell: (s) => `${s.goalsFor}:${s.goalsAgainst}` },
          { label: 'Diff', cell: (s) => (s.goalDiff > 0 ? `+${s.goalDiff}` : s.goalDiff), strong: true },
        ]}
      />
      <TeamTable
        title="Bester Angriff" ownTeams={ownTeams} rows={byAttack}
        tieKey={(s) => perGame(s.goalsFor, s.games)}
        hint="Geworfene Tore je Spiel."
        columns={[
          { label: 'Sp', cell: (s) => s.games },
          { label: 'Tore', cell: (s) => s.goalsFor },
          { label: 'Ø/Spiel', cell: (s) => perGame(s.goalsFor, s.games), strong: true },
        ]}
      />
      <TeamTable
        title="Beste Verteidigung" ownTeams={ownTeams} rows={byDefense}
        tieKey={(s) => perGame(s.goalsAgainst, s.games)}
        hint="Erhaltene Tore je Spiel — weniger ist besser."
        columns={[
          { label: 'Sp', cell: (s) => s.games },
          { label: 'Gegentore', cell: (s) => s.goalsAgainst },
          { label: 'Ø/Spiel', cell: (s) => perGame(s.goalsAgainst, s.games), strong: true },
        ]}
      />
    </div>
  )
}

/**
 * Fair-Play-Wertung. Sie hängt an den Spielberichten und ist deshalb schlechter
 * abgedeckt als die Tor-Tabellen: eine Mannschaft ohne Bericht bleibt LEER und
 * wird nicht als straffreie Mannschaft an die Spitze sortiert.
 */
export function FairPlayView({ data, ownTeams }: { data: TeamStats; ownTeams: string[] }) {
  const rows = data.teams.filter((s) => s.fairPlayScore !== null)
  const ohne = data.teams.filter((s) => s.fairPlayScore === null)
  const w = data.fairPlayWeights
  const sorted = [...rows].sort((a, b) => (a.fairPlayScore ?? 0) - (b.fairPlayScore ?? 0))

  return (
    <div className="space-y-4">
      {sorted.length > 0 && (
        <TeamTable
          title="Fair-Play" ownTeams={ownTeams} rows={sorted}
          tieKey={(s) => s.fairPlayScore ?? 0}
          hint={`Gewichtung: Gelb ${w.yellow}, 2 min ${w.twoMin}, Rot ${w.red}, Blau ${w.blue}. ` +
            'Weniger ist besser. Grundlage sind die ausgewerteten Spielberichte, nicht alle Begegnungen.'}
          columns={[
            { label: 'Berichte', cell: (s) => s.reportGames },
            { label: 'Gelb', cell: (s) => s.yellow },
            { label: '2 min', cell: (s) => s.twoMin },
            { label: 'Rot', cell: (s) => s.red },
            { label: 'Blau', cell: (s) => s.blue },
            { label: 'Punkte', cell: (s) => s.fairPlayScore, strong: true },
          ]}
        />
      )}
      {ohne.length > 0 && (
        <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
          Ohne Wertung, weil noch kein Spielbericht ausgewertet ist:{' '}
          {ohne.map((s) => s.team).join(', ')}. Sie sind damit nicht straffrei, sondern unbekannt.
        </div>
      )}
      {sorted.length === 0 && ohne.length === 0 && <Empty />}
    </div>
  )
}

/**
 * Torverteilung: wie stark hängt eine Mannschaft an einzelnen Werfern?
 *
 * Der Gini allein ist schwer zu lesen, deshalb stehen Median und Durchschnitt
 * daneben und die Spalte trägt eine Erläuterung.
 */
export function DistributionView({ data, ownTeams }: { data: TeamStats; ownTeams: string[] }) {
  const rows = data.teams.filter((s) => s.distribution !== null)
  if (rows.length === 0) return <Empty />
  const sorted = [...rows].sort((a, b) => (a.distribution?.gini ?? 0) - (b.distribution?.gini ?? 0))

  return (
    <TeamTable
      title="Torverteilung" ownTeams={ownTeams} rows={sorted}
      tieKey={(s) => (s.distribution?.gini ?? 0).toFixed(2)}
      hint={'Der Gini-Wert misst die Ungleichverteilung der Saisontore über die Spieler: ' +
        'nahe 0 heißt gleichmäßig verteilt, ein hoher Wert heißt Abhängigkeit von wenigen Werfern. ' +
        'Grundlage sind die ausgewerteten Spielberichte.'}
      columns={[
        { label: 'Berichte', cell: (s) => s.reportGames },
        { label: 'Spieler', cell: (s) => s.distribution?.players ?? '–' },
        { label: 'Ø Tore', cell: (s) => (s.distribution?.average ?? 0).toFixed(1).replace('.', ',') },
        { label: 'Median', cell: (s) => (s.distribution?.median ?? 0).toFixed(1).replace('.', ',') },
        { label: 'Gini', cell: (s) => (s.distribution?.gini ?? 0).toFixed(2).replace('.', ','), strong: true },
      ]}
    />
  )
}

function Empty() {
  return (
    <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
      Noch keine Daten abgerufen.
    </div>
  )
}
