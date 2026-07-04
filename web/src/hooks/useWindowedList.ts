import { useCallback, useEffect, useState } from 'react'

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
 *
 * Der Container wird über eine Callback-Ref (`containerRef`) angebunden — kein
 * `RefObject`, damit die Geometrie-Felder ohne `.current`-Zugriff während des
 * Renderings gelesen werden dürfen.
 *
 * Zwei Scroll-Modi:
 * - `scroll: 'container'` (default): `containerRef` an einen eigenen
 *   `overflow-y-auto`-Container hängen (z. B. Chat).
 * - `scroll: 'window'`: die Seite selbst scrollt; das Fenster wird relativ zur
 *   Position des Containers im Viewport berechnet (z. B. Tabellenseiten).
 */

export interface WindowedListOptions {
  /** Anzahl der Einträge in der vollständigen Liste. */
  count: number
  /** Geschätzte Höhe einer Zeile in px. */
  estimatedRowHeight: number
  /** Zusätzliche Zeilen ober-/unterhalb des Viewports (Puffer gegen leere Ränder). */
  overscan?: number
  /**
   * Ab dieser Listengröße greift Windowing. Kleinere Listen werden vollständig
   * gerendert (kein Windowing-Overhead; stabiler Fallback ohne Layout-Messung).
   */
  threshold?: number
  /** Scroll-Quelle: eigener Container (default) oder die ganze Seite. */
  scroll?: 'container' | 'window'
}

export interface WindowedListResult {
  /** Callback-Ref für den scrollbaren Container bzw. den beobachteten Wrapper. */
  containerRef: (el: HTMLDivElement | null) => void
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

interface Metrics {
  /** Scroll-Offset innerhalb der Liste (0 = oberster Eintrag im/über Viewport). */
  offset: number
  /** Sichtbare Höhe für die Liste. */
  viewport: number
}

export function useWindowedList({
  count,
  estimatedRowHeight,
  overscan = DEFAULT_OVERSCAN,
  threshold = DEFAULT_THRESHOLD,
  scroll = 'container',
}: WindowedListOptions): WindowedListResult {
  const [el, setEl] = useState<HTMLDivElement | null>(null)
  const [metrics, setMetrics] = useState<Metrics>({ offset: 0, viewport: 0 })

  const containerRef = useCallback((node: HTMLDivElement | null) => {
    setEl(node)
  }, [])

  useEffect(() => {
    if (!el) return

    const measure = () => {
      if (scroll === 'window') {
        const rect = el.getBoundingClientRect()
        // Wie weit ist der Listenkopf bereits über den oberen Viewport-Rand gescrollt?
        setMetrics({ offset: Math.max(0, -rect.top), viewport: window.innerHeight })
      } else {
        setMetrics({ offset: el.scrollTop, viewport: el.clientHeight })
      }
    }

    // Erste Messung, sobald der Container steht.
    measure()

    if (scroll === 'window') {
      window.addEventListener('scroll', measure, { passive: true })
      window.addEventListener('resize', measure)
      return () => {
        window.removeEventListener('scroll', measure)
        window.removeEventListener('resize', measure)
      }
    }

    el.addEventListener('scroll', measure, { passive: true })
    let ro: ResizeObserver | null = null
    if (typeof ResizeObserver !== 'undefined') {
      ro = new ResizeObserver(measure)
      ro.observe(el)
    }
    return () => {
      el.removeEventListener('scroll', measure)
      ro?.disconnect()
    }
  }, [el, scroll, count])

  // Windowing nur, wenn Liste groß genug UND der Viewport gemessen wurde.
  // Ohne gemessene Höhe (z. B. jsdom ohne Layout) fällt es auf „ganze Liste" zurück,
  // damit nichts fälschlich ausgeblendet wird.
  const windowed = count > threshold && metrics.viewport > 0

  if (!windowed) {
    return { containerRef, start: 0, end: count, padTop: 0, padBottom: 0, windowed: false }
  }

  const visibleCount = Math.ceil(metrics.viewport / estimatedRowHeight)
  const rawStart = Math.floor(metrics.offset / estimatedRowHeight)
  const start = Math.max(0, rawStart - overscan)
  const end = Math.min(count, rawStart + visibleCount + overscan)
  const padTop = start * estimatedRowHeight
  const padBottom = (count - end) * estimatedRowHeight

  return { containerRef, start, end, padTop, padBottom, windowed: true }
}
