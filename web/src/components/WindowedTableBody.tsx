import { type ReactNode } from 'react'

interface WindowedTableBodyProps<T> {
  items: T[]
  start: number
  end: number
  padTop: number
  padBottom: number
  renderRow: (item: T, index: number) => ReactNode
  /** Spaltenzahl für die Platzhalter-Zeilen (colSpan). */
  colSpan: number
  className?: string
}

/**
 * `<tbody>` für virtualisierte Tabellen. Der scrollbare Container + der
 * `useWindowedList`-Hook liegen beim Elternteil (das die `<table>` umschließt);
 * hier werden nur die sichtbaren `<tr>` plus obere/untere Platzhalter-`<tr>`
 * gerendert. `<tr>`-Platzhalter statt `<div>`, weil nur `<tr>` in `<tbody>` gültig
 * ist. Kein Eintrag wird ausgeblendet — Scrollen tauscht die Zeilen aus.
 */
export default function WindowedTableBody<T>({
  items,
  start,
  end,
  padTop,
  padBottom,
  renderRow,
  colSpan,
  className,
}: WindowedTableBodyProps<T>) {
  return (
    <tbody className={className}>
      {padTop > 0 && (
        <tr aria-hidden style={{ height: padTop }}>
          <td colSpan={colSpan} style={{ padding: 0, border: 0 }} />
        </tr>
      )}
      {items.slice(start, end).map((item, i) => renderRow(item, start + i))}
      {padBottom > 0 && (
        <tr aria-hidden style={{ height: padBottom }}>
          <td colSpan={colSpan} style={{ padding: 0, border: 0 }} />
        </tr>
      )}
    </tbody>
  )
}
