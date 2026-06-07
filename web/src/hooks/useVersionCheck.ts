import { useEffect, useState } from 'react'
import { getAccessToken } from '../lib/api'

interface VersionCheckResult {
  updateAvailable: boolean
  version: string | null
  updateDescription: string
}

export function useVersionCheck(): VersionCheckResult {
  const [updateAvailable, setUpdateAvailable] = useState(false)
  const [version, setVersion] = useState<string | null>(null)
  const [updateDescription, setUpdateDescription] = useState('')

  useEffect(() => {
    if (import.meta.env.DEV) return

    const token = getAccessToken()
    if (!token) return

    const es = new EventSource(`/api/events?token=${encodeURIComponent(token)}`)
    let knownVersion: string | null = null

    es.onmessage = async (e) => {
      if (!e.data?.startsWith('__version:')) return
      const v = e.data.slice('__version:'.length)
      if (knownVersion === null) {
        knownVersion = v
        setVersion(v)
      } else if (v !== knownVersion) {
        setUpdateAvailable(true)
        try {
          const res = await fetch(`/changes.json?v=${v}`)
          if (res.ok) {
            const data = await res.json()
            setUpdateDescription(data.description ?? '')
          }
        } catch {}
      }
    }

    return () => es.close()
  }, [])

  return { updateAvailable, version, updateDescription }
}
