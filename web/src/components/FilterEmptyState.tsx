interface Props {
  /** Anzahl der Elemente, die auf den Textfilter passen, aber von anderen
   *  aktiven Filtern ausgeblendet werden. 0 = normale Leermeldung. */
  hiddenByOtherFilters: number
  onResetFilters: () => void
  /** Text, wenn es wirklich nichts gibt (kein weiterer Filter aktiv). */
  emptyLabel?: string
}

/**
 * Leermeldung für die gefilterten Termin-Listen.
 *
 * Der Ausgeblendet-Zähler ist der Preis der Filter-Semantik (design.md §7):
 * eine leere Seite bei aktivem Team-Chip ist korrektes Verhalten und sieht
 * trotzdem aus wie „gibt's nicht". Ohne diesen Hinweis ist der Filter still
 * unehrlich — ein Fehler, der nie gemeldet wird, weil er sich nie wie einer
 * anfühlt.
 */
export default function FilterEmptyState({ hiddenByOtherFilters, onResetFilters, emptyLabel = 'Keine Treffer.' }: Props) {
  if (hiddenByOtherFilters <= 0) {
    return <p className="text-sm text-brand-text-muted py-6 text-center">{emptyLabel}</p>
  }
  return (
    <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
      <p>
        Keine Treffer.{' '}
        {hiddenByOtherFilters === 1
          ? '1 Termin passt, wird aber von aktiven Filtern ausgeblendet.'
          : `${hiddenByOtherFilters} Termine passen, werden aber von aktiven Filtern ausgeblendet.`}
      </p>
      <button
        type="button"
        onClick={onResetFilters}
        className="mt-2 bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
      >
        Filter zurücksetzen
      </button>
    </div>
  )
}
