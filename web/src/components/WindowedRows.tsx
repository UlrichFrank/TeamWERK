import { type ReactNode } from 'react'
import { useWindowedList } from '../hooks/useWindowedList'

interface WindowedRowsBase<T> {
  items: T[]
  estimatedRowHeight: number
  renderRow: (item: T, index: number) => ReactNode
  overscan?: number
  threshold?: number
  /** Scroll-Quelle: eigener Container (default) oder die ganze Seite. */
  scroll?: 'container' | 'window'
  /** Zusätzliche Klassen für den Scroll-Container. */
  className?: string
  /** Wird innerhalb des Containers nach dem unteren Platzhalter gerendert
   *  (z. B. ein Anker-Ref für Auto-Scroll-ans-Ende). */
  footer?: ReactNode
}

/**
 * Windowing-Wrapper für lange Listen. Rendert nur die im Viewport (+ Puffer)
 * liegenden Zeilen in den DOM und hält die Scrollbar über Platzhalter korrekt.
 *
 * Zwei Varianten, weil `<tr>`-Platzhalter nur als `<tr>` in einem `<tbody>` gültig
 * sind, Flex-Listen aber `<div>`-Platzhalter brauchen:
 *
 * - `WindowedRows` (dieser Default): Flex/Block-Liste, Platzhalter sind `<div>`.
 * - `WindowedTableBody`: rendert ein `<tbody>` mit `<tr>`-Platzhaltern.
 *
 * Beide teilen sich `useWindowedList`. Keine externe Windowing-Library
 * (RAM-Budget; siehe Change lazy-rendering).
 */
export default function WindowedRows<T>({
  items,
  estimatedRowHeight,
  renderRow,
  overscan,
  threshold,
  scroll,
  className,
  footer,
}: WindowedRowsBase<T>) {
  const { containerRef, start, end, padTop, padBottom } = useWindowedList({
    count: items.length,
    estimatedRowHeight,
    overscan,
    threshold,
    scroll,
  })

  return (
    <div ref={containerRef} className={className} data-windowed-scroll>
      {padTop > 0 && <div style={{ height: padTop }} aria-hidden />}
      {items.slice(start, end).map((item, i) => renderRow(item, start + i))}
      {padBottom > 0 && <div style={{ height: padBottom }} aria-hidden />}
      {footer}
    </div>
  )
}
