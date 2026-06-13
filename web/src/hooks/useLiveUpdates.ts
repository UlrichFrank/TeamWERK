import { useEffect, useRef } from 'react'
import { useAuth } from '../contexts/AuthContext'

export function useLiveUpdates(onEvent: (eventType: string) => void) {
  const onEventRef = useRef(onEvent)
  useEffect(() => { onEventRef.current = onEvent })

  const { user } = useAuth()

  // Reconnects whenever `user` changes (login/logout/impersonation).
  // SSE authenticates via the HttpOnly refresh-token cookie — no token in URL.
  useEffect(() => {
    if (!user) return

    const es = new EventSource('/api/events')

    es.onmessage = (e) => {
      if (e.data && !e.data.startsWith('__version:')) onEventRef.current(e.data)
    }

    es.onerror = () => {
      if (es.readyState === EventSource.CLOSED) es.close()
    }

    return () => es.close()
  }, [user])
}
