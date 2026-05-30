import { useEffect } from 'react'
import { getAccessToken } from '../lib/api'

export function useLiveUpdates(onEvent: (eventType: string) => void) {
  useEffect(() => {
    const token = getAccessToken()
    if (!token) return

    const es = new EventSource(`/api/events?token=${encodeURIComponent(token)}`)

    es.onmessage = (e) => {
      if (e.data) onEvent(e.data)
    }

    es.onerror = () => {
      if (es.readyState === EventSource.CLOSED) es.close()
    }

    return () => es.close()
  }, [])
}
