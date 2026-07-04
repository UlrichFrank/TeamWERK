import { type ReactNode } from 'react'
import { type WindowedListResult } from '../hooks/useWindowedList'

interface WindowedTableBodyProps<T> {
  items: T[]
  window: Pick<WindowedListResult, 'start' | 'end' | 'padTop' | 'padBottom'>
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
  window: win,
  renderRow,
  colSpan,
  className,
}: WindowedTableBodyProps<T>) {
  const { start, end, padTop, padBottom } = win
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
