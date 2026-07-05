import { useCallback, useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useLiveUpdates } from './useLiveUpdates'

interface MaintenanceStatus {
  enabled: boolean
  loading: boolean
}

/**
 * Lädt beim App-Start `GET /api/maintenance-status` (unauthentifiziert, damit
 * der Banner auch vor dem Login sichtbar sein kann) und abonniert SSE-Events,
 * damit ein Admin-Toggle unmittelbar bei allen anderen Sessions ankommt. Beim
 * Fehlerfall bleibt `enabled=false` (fail-open — im Zweifel darf man
 * schreiben, um App-Sperre bei Backend-Bug zu vermeiden).
 */
export function useMaintenanceStatus(): MaintenanceStatus {
  const [enabled, setEnabled] = useState(false)
  const [loading, setLoading] = useState(true)

  const refetch = useCallback(async () => {
    try {
      const res = await api.get<{ enabled: boolean }>('/maintenance-status')
      setEnabled(Boolean(res.data?.enabled))
    } catch {
      // Fail-open: bleibt beim letzten bekannten Zustand oder false.
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refetch()
  }, [refetch])

  useLiveUpdates((event) => {
    if (event === 'settings-changed') {
      refetch()
    }
  })

  return { enabled, loading }
}
