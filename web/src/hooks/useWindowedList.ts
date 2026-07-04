import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'

/**
 * Leichtgewichtiges Windowing für lange Listen — kein NPM-Paket (RAM-Budget
 * VPS/Mobilgerät, Konvention „kein zusätzliches schweres Paket").
 *
 * Rechnet aus Scroll-Offset + geschätzter Zeilenhöhe + Overscan einen sichtbaren
 * Bereich [start, end) und liefert Platzhalter-Höhen (`padTop`/`padBottom`), damit
 * die Scrollbar exakt der vollen Liste entspricht. Nur die Zeilen im Fenster liegen
 * im DOM; Scrollen tauscht sie aus — kein Eintrag geht verloren, jede Zeile bleibt
 * durch Scrollen erreichbar (Invariante des Change lazy-rendering).
 *
 * Die Zeilenhöhe ist eine Schätzung (variable Höhen sind erlaubt): sie steuert nur,
 * wie viele Zeilen materialisiert werden, nicht *welche* Daten sichtbar sind.
 */

export interface WindowedListOptions {
  /** Anzahl der Einträge in der vollständigen Liste. */
  count: number
  /** Geschätzte Höhe einer Zeile in px. */
  estimatedRowHeight: number
  /** Zusätzliche Zeilen ober-/unterhalb des Viewports (Puffer gegen leere Ränder). */
  overscan?: number
  /**
   * Wird gesetzt, sobald Windowing greifen soll. Unterhalb dieser Schwelle wird
   * die ganze Liste gerendert (kleine Listen brauchen kein Windowing; hält Tests
   * und Fallback ohne Layout-Messung stabil).
   */
  threshold?: number
}

export interface WindowedListResult {
  /** Ref auf den scrollbaren Container. */
  scrollRef: React.RefObject<HTMLDivElement | null>
  /** Index der ersten zu rendernden Zeile (inkl.). */
  start: number
  /** Index nach der letzten zu rendernden Zeile (exkl.). */
  end: number
  /** Höhe des oberen Platzhalters in px. */
  padTop: number
  /** Höhe des unteren Platzhalters in px. */
  padBottom: number
  /** Ob Windowing aktiv ist (sonst wird die ganze Liste gerendert). */
  windowed: boolean
}

const DEFAULT_OVERSCAN = 6
const DEFAULT_THRESHOLD = 40

export function useWindowedList({
  count,
  estimatedRowHeight,
  overscan = DEFAULT_OVERSCAN,
  threshold = DEFAULT_THRESHOLD,
}: WindowedListOptions): WindowedListResult {
  const scrollRef = useRef<HTMLDivElement | null>(null)
  const [scrollTop, setScrollTop] = useState(0)
  const [viewportHeight, setViewportHeight] = useState(0)

  const measure = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    setScrollTop(el.scrollTop)
    setViewportHeight(el.clientHeight)
  }, [])

  // Erste Messung nach Mount (Layout steht) + Resize-Beobachtung.
  useLayoutEffect(() => {
    measure()
  }, [measure, count])

  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const onScroll = () => setScrollTop(el.scrollTop)
    el.addEventListener('scroll', onScroll, { passive: true })

    let ro: ResizeObserver | null = null
    if (typeof ResizeObserver !== 'undefined') {
      ro = new ResizeObserver(() => setViewportHeight(el.clientHeight))
      ro.observe(el)
    }
    return () => {
      el.removeEventListener('scroll', onScroll)
      ro?.disconnect()
    }
  }, [])

  // Windowing nur, wenn Liste groß genug UND der Viewport gemessen wurde.
  // Ohne gemessene Höhe (z. B. jsdom ohne Layout) fällt es auf „ganze Liste" zurück,
  // damit nichts fälschlich ausgeblendet wird.
  const windowed = count > threshold && viewportHeight > 0

  if (!windowed) {
    return { scrollRef, start: 0, end: count, padTop: 0, padBottom: 0, windowed: false }
  }

  const visibleCount = Math.ceil(viewportHeight / estimatedRowHeight)
  const rawStart = Math.floor(scrollTop / estimatedRowHeight)
  const start = Math.max(0, rawStart - overscan)
  const end = Math.min(count, rawStart + visibleCount + overscan)
  const padTop = start * estimatedRowHeight
  const padBottom = (count - end) * estimatedRowHeight

  return { scrollRef, start, end, padTop, padBottom, windowed: true }
}
