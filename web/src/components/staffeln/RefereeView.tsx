import { AlertTriangle } from 'lucide-react'
import type { RefereeStat } from '../../lib/staffeln'

const TH = 'bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left'
const TD = 'px-4 py-3 text-sm text-brand-text'

/**
 * Schiedsrichter-Rangliste.
 *
 * Die Zahlen sind die Strafen DES SPIELS, nicht eine Bewertung der Person —
 * das steht ausdrücklich über der Tabelle. Technisch verhindern lässt sich die
 * Fehllesart ("der pfeift streng") nicht.
 */
export default function RefereeView({ rows }: { rows: RefereeStat[] }) {
  if (rows.length === 0) {
    return (
      <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
        Noch keine Schiedsrichter erfasst. Berichte, die vor dieser Auswertung abgerufen
        wurden, tragen die Namen nur als ungetrennte Zeile — sie erscheinen hier erst nach
        einer erneuten Auswertung.
      </div>
    )
  }
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu overflow-hidden">
      <div className="px-4 py-3 border-b border-brand-border-subtle">
        <h2 className="text-sm font-medium text-brand-text">Schiedsrichter</h2>
        <p className="text-xs text-brand-text-muted mt-1">
          Gezählt sind die Strafen des Spiels — beide Mannschaften zusammen. Das ist keine
          Bewertung der Person.
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr>
              <th className={TH}>Name</th>
              <th className={TH}>Spiele</th>
              <th className={TH}>2 min</th>
              <th className={TH}>Gelb</th>
              <th className={TH}>Rot</th>
              <th className={TH}>Blau</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.name} className="hover:bg-brand-table-select transition-colors">
                <td className={TD}>
                  <span className="inline-flex items-center gap-1">
                    {r.name}
                    {r.uncertain && (
                      <span
                        className="inline-flex items-center gap-1 text-xs text-brand-text-muted"
                        title="Der Bericht führte beide Namen in einer Spalte; die Trennung ist geraten."
                      >
                        <AlertTriangle className="w-4 h-4" aria-hidden="true" />
                        Trennung unsicher
                      </span>
                    )}
                  </span>
                </td>
                <td className={`${TD} font-medium`}>{r.games}</td>
                <td className={TD}>{r.twoMin}</td>
                <td className={TD}>{r.yellow}</td>
                <td className={TD}>{r.red}</td>
                <td className={TD}>{r.blue}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
