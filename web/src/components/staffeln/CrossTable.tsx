import type { CrossTable as CrossTableData } from '../../lib/staffeln'
import { isOwnTeam } from '../../lib/staffelHighlight'

const TD = 'px-2 py-2 text-xs text-brand-text text-center whitespace-nowrap'

/**
 * Kreuztabelle einer Staffel: Heimmannschaft in der Zeile, Gastmannschaft in
 * der Spalte.
 *
 * Die Spaltenköpfe sind um 90° gedreht. Bei zehn Mannschaften sind es lange
 * Vereinsnamen; gedreht lösen sie das Problem ohne eine Abkürzungstabelle, die
 * gepflegt werden müsste. Die erste Spalte bleibt stehen, der Rest scrollt
 * waagerecht — dasselbe Muster wie der Verlauf.
 */
export default function CrossTable({
  data, ownTeams,
}: { data: CrossTableData; ownTeams: string[] }) {
  if (data.teams.length === 0) {
    return (
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
        Noch kein Spielplan abgerufen — ohne Begegnungen gibt es keine Kreuztabelle.
      </div>
    )
  }

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden">
      <p className="px-4 py-3 text-xs text-brand-text-muted border-b border-brand-border-subtle">
        Zeile = Heimmannschaft, Spalte = Gastmannschaft. Eine Zelle trägt den Endstand
        oder — solange nicht gespielt — das angesetzte Datum.
      </p>
      <div className="overflow-x-auto">
        <table className="border-collapse">
          <thead>
            <tr>
              <th className="sticky left-0 z-10 bg-brand-surface-card px-3 py-2 text-left text-xs uppercase text-brand-text-muted align-bottom">
                Heim \ Gast
              </th>
              {data.teams.map((team) => (
                <th
                  key={team}
                  scope="col"
                  className={`h-32 align-bottom px-1 text-xs text-brand-text-muted ${
                    isOwnTeam(ownTeams, team) ? 'bg-brand-table-select font-semibold' : ''
                  }`}
                  aria-current={isOwnTeam(ownTeams, team) || undefined}
                >
                  <span className="block origin-bottom-left translate-x-1/2 -rotate-90 w-6 whitespace-nowrap">
                    {team}
                  </span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.rows.map((row, r) => {
              const ownRow = isOwnTeam(ownTeams, row.team)
              return (
                <tr
                  key={row.team}
                  className={ownRow ? 'bg-brand-table-select font-semibold' : ''}
                  aria-current={ownRow || undefined}
                >
                  <th
                    scope="row"
                    className={`sticky left-0 z-10 px-3 py-2 text-left text-xs text-brand-text whitespace-nowrap border-r border-brand-border-subtle ${
                      ownRow ? 'bg-brand-table-select font-semibold' : 'bg-brand-surface-card'
                    }`}
                  >
                    {row.team}
                  </th>
                  {row.cells.map((cell, c) => {
                    // Zeile UND Spalte der eigenen Mannschaft sind markiert:
                    // in einer Matrix ist die Mannschaft auf beiden Achsen
                    // vertreten, und nur eine zu markieren versteckte ihre
                    // Auswärtsspiele.
                    const ownCol = isOwnTeam(ownTeams, data.teams[c])
                    const own = ownRow || ownCol
                    if (r === c) {
                      return <td key={c} className="bg-brand-border-subtle/40" aria-hidden="true" />
                    }
                    return (
                      <td key={c} className={`${TD} ${own ? 'bg-brand-table-select font-semibold' : ''}`}>
                        {cell === null ? (
                          <span className="text-brand-text-subtle">–</span>
                        ) : cell.played ? (
                          <span>{cell.homeGoals}:{cell.guestGoals}</span>
                        ) : (
                          <span className="text-brand-text-subtle">{shortDate(cell.date)}</span>
                        )}
                      </td>
                    )
                  })}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// Nur Tag und Monat: in der engen Zelle ist das Jahr entbehrlich, die Staffel
// läuft ohnehin über eine Saison.
function shortDate(date: string): string {
  const d = new Date(`${date.slice(0, 10)}T12:00:00`)
  return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit' })
}
