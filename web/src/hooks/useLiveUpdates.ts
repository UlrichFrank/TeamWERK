import { useEffect, useRef } from 'react'
import { useAuth } from '../contexts/AuthContext'
import { invalidateReferenceCache } from '../lib/api'

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

  // Reconnects whenever `user` changes (login/logout/impersonation).
  // SSE authenticates via the HttpOnly refresh-token cookie — no token in URL.
  useEffect(() => {
    if (!user) return

    const es = new EventSource('/api/events')

    // Gesammelte, deduplizierte Event-Typen im aktuellen Fenster + Timer.
    const pending = new Set<string>()
    let timer: ReturnType<typeof setTimeout> | null = null

    const flush = () => {
      timer = null
      const types = Array.from(pending)
      pending.clear()
      for (const type of types) onEventRef.current(type)
    }

    es.onmessage = (e) => {
      if (!e.data || e.data.startsWith('__version:')) return
      // Referenz-Cache sofort verwerfen (nicht debouncen) — sonst bedient
      // getReference bis zum TTL-Ablauf veraltete Daten.
      invalidateReferenceCache(e.data)
      // Reload-Callback gebündelt: gleicher Typ im Fenster → ein Aufruf.
      pending.add(e.data)
      if (timer === null) timer = setTimeout(flush, COALESCE_MS)
    }

    es.onerror = () => {
      if (es.readyState === EventSource.CLOSED) es.close()
    }

    return () => {
      if (timer !== null) clearTimeout(timer)
      es.close()
    }
  }, [user])
}
