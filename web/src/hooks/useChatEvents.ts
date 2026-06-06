import { useEffect, useRef } from 'react'
import { getAccessToken } from '../lib/api'

export function useChatEvents(onEvent: (eventType: string) => void) {
  const onEventRef = useRef(onEvent)
  useEffect(() => { onEventRef.current = onEvent })

  useEffect(() => {
    const token = getAccessToken()
    if (!token) return

    const es = new EventSource(`/api/chat/events?token=${encodeURIComponent(token)}`)

    es.onmessage = (e) => {
      if (e.data) onEventRef.current(e.data)
    }

    es.onerror = () => {
      if (es.readyState === EventSource.CLOSED) es.close()
    }

    return () => es.close()
  }, [])
}
