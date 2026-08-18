import { useEffect, useRef } from 'react'

// Gemeinsame Verbindungsführung für alle SSE-Kanäle (`/api/events`,
// `/api/chat/events`). Vorher stand sie zweimal da und beide Kopien gaben bei
// einem fatalen Verbindungsfehler endgültig auf — für die Dauer der Seite.
//
// Warum das der teuerste Fehler in dieser Datei war: `AppShell` bleibt über die
// ganze Sitzung gemountet, und eine Homescreen-PWA wird beim Wechsel in den
// Hintergrund eingefroren und später FORTGESETZT, nicht neu geladen. Ohne
// Reconnect gibt es dann keinen Weg zurück — der Chat-Zähler blieb dauerhaft auf
// dem zuletzt geladenen Wert stehen (meist 0).
//
// Arbeitsteilung mit dem Browser: Bei einem reinen Transportabbruch reconnectet
// EventSource selbst (`readyState === CONNECTING`) — da ist nichts zu tun.
// Endgültig tot ist nur `CLOSED`, was der Browser bei einer HTTP-Fehlerantwort
// setzt (401 nach abgelaufenem Refresh-Cookie, 502 beim Server-Neustart hinter
// nginx). Genau dieser Zustand wird hier aufgefangen.

const BASE_MS = 1000
const MAX_MS = 30000

/**
 * Wartezeit vor dem n-ten Wiederverbindungsversuch: 1s → 2s → 4s → 8s → 16s → 30s,
 * danach konstant. Bewusst ohne Jitter — bei dreistelligen Nutzerzahlen ist ein
 * Thundering Herd nach Server-Neustart kein reales Problem, und Jitter würde nur
 * die Tests verwackeln.
 *
 * Exportiert, damit die Folge ohne Render und ohne Fake-Timer prüfbar ist
 * (dasselbe Muster wie `createThrottledProgress` in VideoUploadPage.tsx).
 */
export function nextBackoffMs(attempt: number): number {
  if (attempt <= 0) return BASE_MS
  return Math.min(BASE_MS * 2 ** attempt, MAX_MS)
}

/**
 * Hält eine SSE-Verbindung offen, solange `identity` gesetzt ist.
 *
 * `identity` ist der Schlüssel der Nutzer-Identität (typisch `user?.id ?? null`),
 * nicht das User-Objekt: `null` heißt „nicht verbinden", ein Wechsel des Wertes
 * (Login, Logout, Start/Ende einer Impersonation) schließt die alte Verbindung
 * und baut eine neue auf. An der Objektreferenz zu hängen würde bei jedem
 * Context-Update unnötig neu verbinden.
 */
export function useEventStream(
  path: string,
  onEvent: (data: string) => void,
  identity: string | number | null,
) {
  const onEventRef = useRef(onEvent)
  useEffect(() => { onEventRef.current = onEvent })

  useEffect(() => {
    if (identity === null || identity === undefined) return

    let es: EventSource | null = null
    let timer: ReturnType<typeof setTimeout> | null = null
    let attempt = 0
    let disposed = false

    const clearTimer = () => {
      if (timer !== null) {
        clearTimeout(timer)
        timer = null
      }
    }

    const connect = () => {
      if (disposed) return
      clearTimer()
      es = new EventSource(path)
      es.onopen = () => { attempt = 0 }
      es.onmessage = (e: MessageEvent) => {
        if (e.data) onEventRef.current(e.data)
      }
      es.onerror = () => {
        // Nur der endgültige Zustand ist unser Fall — sonst läuft der
        // browsereigene Reconnect bereits.
        if (!es || es.readyState !== EventSource.CLOSED) return
        es.close()
        scheduleReconnect()
      }
    }

    const scheduleReconnect = () => {
      if (disposed || timer !== null) return
      const delay = nextBackoffMs(attempt)
      attempt += 1
      timer = setTimeout(() => {
        timer = null
        connect()
      }, delay)
    }

    // Rückkehr des Nutzers bzw. des Netzes ist ein starkes Signal, dass sich die
    // Lage geändert hat: sofort versuchen, nicht den angelaufenen Backoff
    // abwarten. Eine noch lebende Verbindung bleibt unangetastet.
    const reconnectNow = () => {
      if (disposed) return
      if (es && es.readyState !== EventSource.CLOSED) return
      attempt = 0
      clearTimer()
      connect()
    }

    const onVisibilityChange = () => {
      if (document.visibilityState === 'visible') reconnectNow()
    }

    connect()
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('online', reconnectNow)

    return () => {
      disposed = true
      clearTimer()
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.removeEventListener('online', reconnectNow)
      es?.close()
    }
  }, [path, identity])
}
