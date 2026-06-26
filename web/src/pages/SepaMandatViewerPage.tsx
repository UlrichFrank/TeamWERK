import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { AlertTriangle, ChevronLeft, Lock } from 'lucide-react'
import { api } from '../lib/api'
import { decryptFile } from '../lib/bankCrypto'
import { errorStatus } from '../lib/errors'
import { useVault } from '../contexts/VaultContext'
import FileViewer from '../components/FileViewer'

export default function SepaMandatViewerPage() {
  const { memberId } = useParams()
  const navigate = useNavigate()
  const { privateKey } = useVault()
  const fallbackPath = memberId ? `/mitglieder/${memberId}` : '/mitglieder'

  const [blob, setBlob] = useState<Blob | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!memberId || !privateKey) return
    let cancelled = false
    setLoading(true)
    setError('')
    setBlob(null)

    ;(async () => {
      try {
        const { data: tokenData } = await api.get<{ token: string; dek_enc: string }>(
          `/members/${memberId}/sepa-mandat/download-token`,
        )
        if (cancelled) return
        const res = await api.get<ArrayBuffer>(
          `/members/${memberId}/sepa-mandat/download?token=${tokenData.token}`,
          { responseType: 'arraybuffer' },
        )
        if (cancelled) return
        const plain = await decryptFile(new Uint8Array(res.data), tokenData.dek_enc, privateKey)
        if (cancelled) return
        setBlob(new Blob([plain as BlobPart], { type: 'application/pdf' }))
        setLoading(false)
      } catch (e) {
        if (cancelled) return
        const status = errorStatus(e)
        if (status === 403) setError('Du hast keinen Zugriff auf dieses Mandat.')
        else if (status === 404) setError('Kein Mandat hinterlegt.')
        else setError('Entschlüsselung fehlgeschlagen — falscher Tresor-Inhalt?')
        setLoading(false)
      }
    })()

    return () => { cancelled = true }
  }, [memberId, privateKey])

  function goBack() {
    if (window.history.length > 1) navigate(-1)
    else navigate(fallbackPath, { replace: true })
  }

  if (!memberId) {
    return <p className="text-sm text-brand-danger">Ungültige Mitglieds-ID.</p>
  }

  // Vault gesperrt → Hinweis, kein Decrypt-Versuch
  if (!privateKey) {
    return (
      <div>
        <div className="sticky top-0 z-10 bg-brand-white pb-3 mb-4 flex items-center gap-3 border-b border-brand-border-subtle">
          <button
            type="button"
            onClick={goBack}
            className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text"
            aria-label="Zurück"
          >
            <ChevronLeft className="w-5 h-5" />
            <span>Zurück</span>
          </button>
          <h1 className="text-base font-medium text-brand-text">SEPA-Mandat</h1>
        </div>
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 max-w-md mx-auto text-center">
          <Lock className="w-8 h-8 text-brand-text-muted mx-auto mb-3" />
          <p className="text-sm text-brand-text mb-2">Bankdaten-Tresor gesperrt.</p>
          <p className="text-sm text-brand-text-muted">
            Zum Anzeigen den Tresor entsperren (Menü „Tresor"), dann zurück und erneut auf „Mandat öffnen" klicken.
          </p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div>
        <div className="sticky top-0 z-10 bg-brand-white pb-3 mb-4 flex items-center gap-3 border-b border-brand-border-subtle">
          <button
            type="button"
            onClick={goBack}
            className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text"
            aria-label="Zurück"
          >
            <ChevronLeft className="w-5 h-5" />
            <span>Zurück</span>
          </button>
          <h1 className="text-base font-medium text-brand-text">SEPA-Mandat</h1>
        </div>
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2 max-w-md mx-auto">
          <AlertTriangle className="w-4 h-4" />{error}
        </div>
      </div>
    )
  }

  if (loading || !blob) {
    return (
      <div>
        <div className="sticky top-0 z-10 bg-brand-white pb-3 mb-4 flex items-center gap-3 border-b border-brand-border-subtle">
          <button
            type="button"
            onClick={goBack}
            className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text"
            aria-label="Zurück"
          >
            <ChevronLeft className="w-5 h-5" />
            <span>Zurück</span>
          </button>
          <h1 className="text-base font-medium text-brand-text">SEPA-Mandat</h1>
        </div>
        <p className="text-sm text-brand-text-muted text-center py-8">Mandat wird entschlüsselt…</p>
      </div>
    )
  }

  return (
    <FileViewer
      source="blob"
      blob={blob}
      filename="sepa-mandat.pdf"
      mimeType="application/pdf"
      fallbackPath={fallbackPath}
    />
  )
}
