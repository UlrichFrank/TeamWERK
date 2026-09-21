/**
 * Platzierungen mit Gleichstand ("1, 1, 3"): wer im Wert gleichauf liegt,
 * teilt sich den Platz, und der nächste Platz wird übersprungen — wie im
 * Sport üblich.
 *
 * `rows` muss bereits nach dem Wert sortiert sein, `key` liefert den Wert,
 * der über Gleichstand entscheidet. Er soll die ANGEZEIGTE Größe sein (die
 * gerundete Zahl, nicht der Rohwert): zwei Mannschaften mit "28,5" Toren je
 * Spiel dürfen nicht auf verschiedenen Plätzen stehen, nur weil hinter dem
 * Komma etwas abweicht, das niemand sieht.
 */
export function sharedRanks<T>(rows: T[], key: (row: T) => string | number): number[] {
  const ranks: number[] = []
  rows.forEach((row, i) => {
    ranks.push(i > 0 && key(row) === key(rows[i - 1]) ? ranks[i - 1] : i + 1)
  })
  return ranks
}
