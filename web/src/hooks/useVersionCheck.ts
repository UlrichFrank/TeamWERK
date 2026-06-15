import { useEffect, useState } from 'react'
import { useAuth } from '../contexts/AuthContext'

interface VersionCheckResult {
  updateAvailable: boolean
  version: string | null
}

// In DEV liefern wir einen sichtbaren Platzhalter, damit der Sidebar-Versionslink
// auch lokal funktioniert. Keine SSE-Verbindung.
const DEV_VERSION = 'dev'

export function useVersionCheck(): VersionCheckResult {
  const { user } = useAuth()
  const [updateAvailable, setUpdateAvailable] = useState(false)
  const [version, setVersion] = useState<string | null>(
    import.meta.env.DEV ? DEV_VERSION : null,
  )

  // Reconnects whenever `user` changes (login/logout/impersonation).
  // SSE authenticates via the HttpOnly refresh-token cookie — no token in URL.
  useEffect(() => {
    if (import.meta.env.DEV) return
    if (!user) {
      setVersion(null)
      setUpdateAvailable(false)
      return
    }

    const es = new EventSource('/api/events')
    let knownVersion: string | null = null

    es.onmessage = (e) => {
      if (!e.data?.startsWith('__version:')) return
      const v = e.data.slice('__version:'.length)
      if (knownVersion === null) {
        knownVersion = v
        setVersion(v)
      } else if (v !== knownVersion) {
        setUpdateAvailable(true)
      }
    }

    // EventSource auto-reconnects on transport errors; don't close in onerror.

    return () => es.close()
  }, [user])

  return { updateAvailable, version }
}
