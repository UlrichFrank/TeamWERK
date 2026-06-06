import { useEffect, useState } from 'react'

interface VersionCheckResult {
  updateAvailable: boolean
  version: string | null
}

export function useVersionCheck(): VersionCheckResult {
  const [updateAvailable, setUpdateAvailable] = useState(false)
  const [version, setVersion] = useState<string | null>(null)

  useEffect(() => {
    if (import.meta.env.DEV) return

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

    return () => es.close()
  }, [])

  return { updateAvailable, version }
}
