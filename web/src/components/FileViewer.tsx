import { lazy, Suspense, useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { AlertTriangle, ChevronLeft, Download } from 'lucide-react'
import { api } from '../lib/api'
import { errorStatus } from '../lib/errors'

const PdfRenderer = lazy(() => import('./PdfRenderer'))

interface CommonProps {
  fallbackPath: string
}

interface FileSourceProps extends CommonProps {
  source: 'file'
  fileId: number
  /** Optional override; sonst aus Content-Disposition. */
  filename?: string
  /** Optional override; sonst aus Content-Type. */
  mimeType?: string
}

interface BlobSourceProps extends CommonProps {
  source: 'blob'
  blob: Blob
  filename: string
  mimeType: string
}

export type FileViewerProps = FileSourceProps | BlobSourceProps

interface LoadedFile {
  blob: Blob
  filename: string
  mimeType: string
}

function parseFilenameFromDisposition(disposition: string | undefined, fallback: string): string {
  if (!disposition) return fallback
  // RFC 5987 filename* hat Vorrang
  const star = disposition.match(/filename\*=(?:UTF-8'')?([^;]+)/i)
  if (star?.[1]) {
    try {
      return decodeURIComponent(star[1].replace(/(^"|"$)/g, ''))
    } catch {
      /* fallthrough */
    }
  }
  const plain = disposition.match(/filename="?([^";]+)"?/i)
  if (plain?.[1]) return plain[1]
  return fallback
}

export default function FileViewer(props: FileViewerProps) {
  const navigate = useNavigate()
  const [loaded, setLoaded] = useState<LoadedFile | null>(
    props.source === 'blob'
      ? { blob: props.blob, filename: props.filename, mimeType: props.mimeType }
      : null,
  )
  const [loading, setLoading] = useState(props.source === 'file')
  const [error, setError] = useState('')

  useEffect(() => {
    if (props.source !== 'file') return
    let cancelled = false
    setLoading(true)
    setError('')
    setLoaded(null)

    ;(async () => {
      try {
        const { data: tokenData } = await api.get<{ token: string }>(
          `/files/${props.fileId}/download-token`,
        )
        if (cancelled) return
        const res = await api.get<Blob>(`/files/${props.fileId}/download?token=${tokenData.token}`, {
          responseType: 'blob',
        })
        if (cancelled) return
        const filename = props.filename
          ?? parseFilenameFromDisposition(res.headers['content-disposition'], 'datei')
        const mimeType = props.mimeType ?? res.data.type ?? res.headers['content-type'] ?? 'application/octet-stream'
        setLoaded({ blob: res.data, filename, mimeType })
        setLoading(false)
      } catch (e) {
        if (cancelled) return
        const status = errorStatus(e)
        if (status === 403) setError('Du hast keinen Zugriff auf diese Datei.')
        else if (status === 404) setError('Datei nicht gefunden.')
        else setError('Datei konnte nicht geladen werden.')
        setLoading(false)
      }
    })()

    return () => { cancelled = true }
  }, [props])

  const blobUrl = useMemo(() => {
    if (!loaded) return ''
    return URL.createObjectURL(loaded.blob)
  }, [loaded])

  useEffect(() => {
    return () => { if (blobUrl) URL.revokeObjectURL(blobUrl) }
  }, [blobUrl])

  function goBack() {
    if (window.history.length > 1) navigate(-1)
    else navigate(props.fallbackPath, { replace: true })
  }

  const headerName = loaded?.filename ?? 'Datei wird geladen…'

  return (
    <div>
      {/* Header */}
      <div className="sticky top-0 z-10 bg-brand-white pb-3 mb-4 flex items-center justify-between gap-3 border-b border-brand-border-subtle">
        <button
          type="button"
          onClick={goBack}
          className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text"
          aria-label="Zurück"
        >
          <ChevronLeft className="w-5 h-5" />
          <span>Zurück</span>
        </button>
        <h1 className="text-base font-medium text-brand-text truncate flex-1 text-center px-2">
          {headerName}
        </h1>
        {loaded ? (
          <a
            href={blobUrl}
            download={loaded.filename}
            className="flex items-center gap-1 text-sm text-brand-text-muted hover:text-brand-text"
            aria-label="Herunterladen"
          >
            <Download className="w-5 h-5" />
          </a>
        ) : (
          <span className="w-5 h-5" aria-hidden />
        )}
      </div>

      {/* Body */}
      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2 max-w-md mx-auto">
          <AlertTriangle className="w-4 h-4" />{error}
        </div>
      )}
      {loading && !error && (
        <p className="text-sm text-brand-text-muted text-center py-8">Datei wird geladen…</p>
      )}
      {loaded && !error && <FileBody file={loaded} blobUrl={blobUrl} />}
    </div>
  )
}

function FileBody({ file, blobUrl }: { file: LoadedFile; blobUrl: string }) {
  if (file.mimeType.startsWith('image/')) {
    return (
      <div className="flex justify-center">
        <img
          src={blobUrl}
          alt={file.filename}
          className="max-w-full max-h-[80vh] object-contain"
        />
      </div>
    )
  }

  if (file.mimeType === 'application/pdf') {
    return (
      <Suspense fallback={<p className="text-sm text-brand-text-muted text-center py-8">PDF-Viewer wird geladen…</p>}>
        <PdfRenderer blob={file.blob} />
      </Suspense>
    )
  }

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 max-w-md mx-auto text-center">
      <p className="text-sm text-brand-text mb-3">
        Diese Datei kann nicht in der App angezeigt werden.
      </p>
      <p className="text-sm text-brand-text-muted mb-4 truncate">{file.filename}</p>
      <a
        href={blobUrl}
        download={file.filename}
        className="inline-flex items-center gap-2 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
      >
        <Download className="w-4 h-4" />
        Herunterladen
      </a>
    </div>
  )
}
