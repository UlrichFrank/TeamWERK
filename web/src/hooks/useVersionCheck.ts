import { useEffect, useState } from 'react'
import { getAccessToken } from '../lib/api'

export function useVersionCheck(): boolean {
  const [updateAvailable, setUpdateAvailable] = useState(false)

  useEffect(() => {
    if (import.meta.env.DEV) return

    const token = getAccessToken()
    if (!token) return

    const es = new EventSource(`/api/events?token=${encodeURIComponent(token)}`)
    let knownVersion: string | null = null

    es.onmessage = (e) => {
      if (!e.data?.startsWith('__version:')) return
      const version = e.data.slice('__version:'.length)
      if (knownVersion === null) {
        knownVersion = version
      } else if (version !== knownVersion) {
        setUpdateAvailable(true)
      }
    }

    es.onerror = () => {
      if (es.readyState === EventSource.CLOSED) es.close()
    }

    return () => es.close()
  }, [])

  return updateAvailable
}
