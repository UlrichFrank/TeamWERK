import { useEffect, useRef, type RefObject } from 'react'
import { useLocation, useNavigationType } from 'react-router-dom'

/**
 * Scroll-Position pro History-Eintrag merken und bei „Zurück" wiederherstellen.
 *
 * Der Fokus-Mechanismus (`?focus=…` in TerminePage/DutyPage) deckt nur den Fall ab,
 * dass der Nutzer eine Karte *angeklickt* hat. Wer nur scrollt und dann weiter
 * navigiert, hat beim Zurück keinen Marker in der URL.
 *
 * **Warum das überhaupt nötig ist — und wo nicht:** Der Scroll-Container der App ist
 * das `<main>` in AppShell (`overflow-auto`), nicht das Dokument. Desktop-Chromium und
 * Desktop-WebKit stellen die Position solcher verschachtelten Scroller bei
 * History-Navigation von sich aus wieder her (hier nachgemessen, u. a. mit 4,5 s
 * API-Latenz und über einen zwischenzeitlichen Reload hinweg) — dort ist dieser Hook
 * ein No-Op, der im ersten Frame den bereits richtigen Wert setzt. In der
 * **iOS-Homescreen-PWA** (Standalone) passiert das nicht, und genau dort ist der Fehler
 * gemeldet worden. Deshalb übernimmt der Hook die Aufgabe selbst, statt sich auf den
 * Browser zu verlassen. Native `history.scrollRestoration` und React Routers
 * `<ScrollRestoration>` helfen beide nicht: sie meinen den Dokument-Scroller.
 *
 * Ablauf:
 * - Während der Nutzer scrollt, wird die Position in einer Closure mitgeführt und
 *   **beim Verlassen** des History-Eintrags (Effect-Cleanup) in den `sessionStorage`
 *   geschrieben. Bewusst aus der Closure statt aus dem DOM: `main.scrollTop` ist zum
 *   Cleanup-Zeitpunkt schon am (evtl. kürzeren) Inhalt der Zielseite geklemmt.
 * - Bei POP (Zurück/Vorwärts) wird die Position wiederhergestellt, und zwar über
 *   mehrere Frames: die Liste hängt an einem API-Call, ist beim Mount also noch leer
 *   und lässt sich gar nicht so weit scrollen. Der Versuch läuft bis zum Treffer,
 *   längstens `RESTORE_BUDGET_MS`, und bricht bei jeder Nutzereingabe sofort ab.
 * - Bei PUSH/REPLACE auf einen **anderen Pfad** beginnt die neue Seite oben. Reine
 *   Query-Änderungen (Filter, `?focus=`, `?date=`) erzeugen zwar ebenfalls einen neuen
 *   History-Eintrag, dürfen die Position aber nicht zurücksetzen.
 */

const STORAGE_KEY = 'scroll-positions'
/** Ältere Einträge fliegen raus; eine Session sammelt sonst unbegrenzt Keys. */
const MAX_ENTRIES = 50
/**
 * Obergrenze fürs Nachfassen, bis die Liste hoch genug für die Zielposition ist.
 * Großzügig, weil sie nur die *langsame* Verbindung abdecken muss (Handy im
 * Hallenfunkloch — der Fall, in dem die verlorene Position am meisten nervt). Der
 * Versuch endet beim Treffer und bei jeder Nutzereingabe, läuft also praktisch nie
 * bis zum Ende.
 */
const RESTORE_BUDGET_MS = 15000

interface StoredPosition {
  top: number
  /** Pfad + Query des Eintrags — Schutz gegen wiederverwendete History-Keys. */
  url: string
}

function readAll(): Record<string, StoredPosition> {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : null
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function savePosition(key: string, entry: StoredPosition) {
  try {
    const all = readAll()
    delete all[key] // neu einfügen, damit der Eintrag ans Ende der Insertion-Order rutscht
    all[key] = entry
    const keys = Object.keys(all)
    for (const k of keys.slice(0, Math.max(0, keys.length - MAX_ENTRIES))) delete all[k]
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(all))
  } catch {
    // sessionStorage kann im Privatmodus werfen — Scroll-Restore ist kein kritischer Pfad.
  }
}

function readPosition(key: string, url: string): number | null {
  const entry = readAll()[key]
  if (!entry || entry.url !== url || typeof entry.top !== 'number' || entry.top <= 0) return null
  return entry.top
}

export function useScrollRestoration(ref: RefObject<HTMLElement | null>) {
  const location = useLocation()
  const navigationType = useNavigationType()
  const { key, pathname, search } = location
  const url = pathname + search
  // Vorheriger Pfad, um Seitenwechsel von reinen Query-Änderungen zu unterscheiden.
  // Start = aktueller Pfad, damit der erste Mount nicht als Wechsel zählt.
  const prevPathname = useRef(pathname)

  // Position des aktuellen History-Eintrags mitschreiben und beim Verlassen sichern.
  useEffect(() => {
    const el = ref.current
    if (!el) return
    let top = el.scrollTop
    const onScroll = () => { top = el.scrollTop }
    // iOS kann die PWA jederzeit einfrieren/beenden, ohne dass ein Cleanup läuft —
    // beim Wegblenden zusätzlich sichern, damit die Position den Wechsel überlebt.
    const onHide = () => savePosition(key, { top, url })
    el.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('pagehide', onHide)
    document.addEventListener('visibilitychange', onHide)
    return () => {
      el.removeEventListener('scroll', onScroll)
      window.removeEventListener('pagehide', onHide)
      document.removeEventListener('visibilitychange', onHide)
      savePosition(key, { top, url })
    }
  }, [ref, key, url])

  // Wiederherstellen (POP) bzw. Zurücksetzen (neuer Pfad).
  useEffect(() => {
    const el = ref.current
    if (!el) return

    const pathChanged = pathname !== prevPathname.current
    prevPathname.current = pathname

    if (navigationType !== 'POP') {
      // Nur ein echter Seitenwechsel beginnt oben; Filter-/Fokus-Parameter auf
      // derselben Seite lassen die Position, wo sie ist.
      if (pathChanged) el.scrollTop = 0
      return
    }

    // Der Fokus-Mechanismus ist genauer (er scrollt zur konkreten Karte und hebt sie
    // hervor) — beide gleichzeitig würden sich um die Scroll-Position streiten.
    if (new URLSearchParams(search).has('focus')) return

    const target = readPosition(key, url)
    if (target === null) return

    let frame = 0
    let done = false
    const stop = () => {
      if (done) return
      done = true
      cancelAnimationFrame(frame)
      el.removeEventListener('wheel', stop)
      el.removeEventListener('touchstart', stop)
      el.removeEventListener('mousedown', stop)
      window.removeEventListener('keydown', stop)
    }
    const deadline = Date.now() + RESTORE_BUDGET_MS
    const step = () => {
      if (done) return
      el.scrollTop = target
      // Treffer (Inhalt ist hoch genug) oder Budget aufgebraucht → fertig.
      if (Math.abs(el.scrollTop - target) < 1 || Date.now() > deadline) {
        stop()
        return
      }
      frame = requestAnimationFrame(step)
    }
    // Jede echte Eingabe gewinnt gegen den Restore — Wheel/Touch fürs Scrollen,
    // mousedown für den Scrollbar-Griff, keydown für Pfeiltasten/Bild-ab.
    el.addEventListener('wheel', stop, { passive: true })
    el.addEventListener('touchstart', stop, { passive: true })
    el.addEventListener('mousedown', stop)
    window.addEventListener('keydown', stop)
    frame = requestAnimationFrame(step)

    return stop
  }, [ref, key, pathname, search, url, navigationType])
}
