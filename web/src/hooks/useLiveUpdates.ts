import { useEffect, useRef } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { invalidateReferenceCache } from '../lib/api'
import { useEventStream } from './useEventStream'

// Coalescing-Fenster: ein Burst gleichartiger SSE-Events (z. B. mehrere
// Broadcast-Aufrufe in einem Handler) löst genau EINEN Callback je eindeutigem
// Event-Typ aus. 300 ms ist kürzer als die menschliche „live"-Wahrnehmung und
// lang genug, um Server-Bursts zusammenzufassen. Der globale SSE-Channel hat
// ohnehin Buffer 1 mit Drop — Debounce ist semantisch unbedenklich.
const COALESCE_MS = 300

export function useLiveUpdates(onEvent: (eventType: string) => void) {
  const onEventRef = useRef(onEvent)
  useEffect(() => { onEventRef.current = onEvent })

  const { user } = useAuth()

  // Gesammelte, deduplizierte Event-Typen im aktuellen Fenster + Timer.
  // Als Refs, damit sie den Reconnect überleben: die Verbindung liegt in
  // useEventStream, das Coalescing gehört hierher.
  const pending = useRef(new Set<string>())
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => () => {
    if (timer.current !== null) clearTimeout(timer.current)
  }, [])

  // Verbindung, Reconnect und Bindung an die Identität: useEventStream.
  // SSE authentifiziert über das HttpOnly-Refresh-Cookie — kein Token in der URL.
  useEventStream('/api/events', (data: string) => {
    if (data.startsWith('__version:')) return
    // Referenz-Cache sofort verwerfen (nicht debouncen) — sonst bedient
    // getReference bis zum TTL-Ablauf veraltete Daten.
    invalidateReferenceCache(data)
    // Reload-Callback gebündelt: gleicher Typ im Fenster → ein Aufruf.
    pending.current.add(data)
    if (timer.current === null) {
      timer.current = setTimeout(() => {
        timer.current = null
        const types = Array.from(pending.current)
        pending.current.clear()
        for (const type of types) onEventRef.current(type)
      }, COALESCE_MS)
    }
  }, user?.id ?? null)
}
