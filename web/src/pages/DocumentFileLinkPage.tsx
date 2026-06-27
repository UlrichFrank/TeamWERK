import { useEffect, useState } from 'react'
import { Link, Navigate, useParams } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { errorStatus } from '../lib/errors'
import { useMediaQuery } from '../lib/useMediaQuery'

// Deep-Link-Kompatibilität für `/dokumente/datei/:fileId`:
//   Mobile → In-App-Viewer-Route (nutzt Blob-Download → nativer Viewer).
//   Desktop → location.replace auf die Download-URL (nativer Browser-PDF-Viewer
//             im selben Tab; Browser-Chrome bietet Zurück-Navigation).
export default function DocumentFileLinkPage() {
  const { fileId } = useParams()
  const isMobile = useMediaQuery('(max-width: 639px)')
  const [error, setError] = useState('')

  useEffect(() => {
    if (!fileId || isMobile) return
    let cancelled = false
    ;(async () => {
      try {
        const { data } = await api.get<{ token: string }>(`/files/${fileId}/download-token`)
        if (cancelled) return
        window.location.replace(`/api/files/${fileId}/download?token=${data.token}`)
      } catch (e) {
        if (cancelled) return
        const status = errorStatus(e)
        if (status === 403) setError('Du hast keinen Zugriff auf diese Datei.')
        else if (status === 404) setError('Datei nicht gefunden.')
        else setError('Datei konnte nicht geöffnet werden.')
      }
    })()
    return () => { cancelled = true }
  }, [fileId, isMobile])

  if (!fileId) return <Navigate to="/dokumente" replace />
  if (isMobile) return <Navigate to={`/dokumente/anzeigen/${fileId}`} replace />

  if (error) {
    return (
      <div className="max-w-md">
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4 flex items-center gap-2">
          <AlertTriangle className="w-4 h-4" />{error}
        </div>
        <Link to="/dokumente" className="text-sm text-brand-text-muted hover:text-brand-text">
          Zurück zu Dokumente
        </Link>
      </div>
    )
  }

  return <p className="text-sm text-brand-text-muted">Datei wird geöffnet…</p>
}
