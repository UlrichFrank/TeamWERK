import { type ReactNode } from 'react'
import { useWindowedList } from '../hooks/useWindowedList'

interface WindowedRowsBase<T> {
  items: T[]
  estimatedRowHeight: number
  renderRow: (item: T, index: number) => ReactNode
  overscan?: number
  threshold?: number
  /** Zusätzliche Klassen für den Scroll-Container. */
  className?: string
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
  className,
}: WindowedRowsBase<T>) {
  const { scrollRef, start, end, padTop, padBottom } = useWindowedList({
    count: items.length,
    estimatedRowHeight,
    overscan,
    threshold,
  })

  return (
    <div ref={scrollRef} className={className} data-windowed-scroll>
      {padTop > 0 && <div style={{ height: padTop }} aria-hidden />}
      {items.slice(start, end).map((item, i) => renderRow(item, start + i))}
      {padBottom > 0 && <div style={{ height: padBottom }} aria-hidden />}
    </div>
  )
}
