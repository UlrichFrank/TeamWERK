import { useEffect, useRef, useState } from 'react'
import { AlertTriangle } from 'lucide-react'
import * as pdfjsLib from 'pdfjs-dist'
import workerSrc from 'pdfjs-dist/build/pdf.worker.min.mjs?url'
import type { PDFDocumentLoadingTask } from 'pdfjs-dist'

pdfjsLib.GlobalWorkerOptions.workerSrc = workerSrc

interface Props {
  blob: Blob
}

export default function PdfRenderer({ blob }: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    let loadingTask: PDFDocumentLoadingTask | null = null

    ;(async () => {
      setLoading(true)
      setError('')
      try {
        const buf = await blob.arrayBuffer()
        if (cancelled) return
        loadingTask = pdfjsLib.getDocument({ data: buf })
        const pdfDoc = await loadingTask.promise
        if (cancelled) {
          await loadingTask.destroy()
          return
        }

        const container = containerRef.current
        if (!container) return
        while (container.firstChild) container.removeChild(container.firstChild)

        const containerWidth = container.clientWidth || 800
        for (let pageNum = 1; pageNum <= pdfDoc.numPages; pageNum++) {
          if (cancelled) break
          const page = await pdfDoc.getPage(pageNum)
          const baseViewport = page.getViewport({ scale: 1 })
          const scale = Math.min(2, containerWidth / baseViewport.width)
          const viewport = page.getViewport({ scale })

          const canvas = document.createElement('canvas')
          canvas.width = viewport.width
          canvas.height = viewport.height
          canvas.className = 'mx-auto mb-4 shadow border border-brand-border-subtle bg-brand-white max-w-full h-auto'
          const ctx = canvas.getContext('2d')
          if (!ctx) continue
          container.appendChild(canvas)
          await page.render({ canvas, canvasContext: ctx, viewport }).promise
        }

        if (!cancelled) setLoading(false)
      } catch (e) {
        if (cancelled) return
        console.error('PDF-Render-Fehler', e)
        setError('PDF konnte nicht angezeigt werden.')
        setLoading(false)
      }
    })()

    return () => {
      cancelled = true
      if (loadingTask) loadingTask.destroy()
    }
  }, [blob])

  if (error) {
    return (
      <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2 max-w-md mx-auto">
        <AlertTriangle className="w-4 h-4" />
        {error}
      </div>
    )
  }

  return (
    <div>
      {loading && <p className="text-sm text-brand-text-muted text-center py-8">PDF wird gerendert…</p>}
      <div ref={containerRef} />
    </div>
  )
}
