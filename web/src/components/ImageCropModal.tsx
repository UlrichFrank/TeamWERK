import { useCallback, useEffect, useRef, useState } from 'react'
import { X, ZoomIn } from 'lucide-react'

const CANVAS_SIZE = 320
const EXPORT_SIZE = 600
const RADIUS = 148
const CX = CANVAS_SIZE / 2
const CY = CANVAS_SIZE / 2

function clampOffset(ox: number, oy: number, scale: number, imgW: number, imgH: number) {
  return {
    x: Math.min(CX - RADIUS, Math.max(CX + RADIUS - imgW * scale, ox)),
    y: Math.min(CY - RADIUS, Math.max(CY + RADIUS - imgH * scale, oy)),
  }
}

interface Props {
  file: File | null
  onConfirm: (blob: Blob) => void
  onCancel: () => void
}

export default function ImageCropModal({ file, onConfirm, onCancel }: Props) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const imgRef = useRef<HTMLImageElement | null>(null)
  // Use refs for all interactive state to avoid stale closures in touch handlers
  const offsetRef = useRef({ x: 0, y: 0 })
  const zoomRef = useRef(1)
  const initialScaleRef = useRef(1)
  const naturalRef = useRef({ w: 0, h: 0 })
  const dragRef = useRef<{ startX: number; startY: number; startOX: number; startOY: number } | null>(null)
  const pinchRef = useRef<{ dist: number; zoom: number; midX: number; midY: number } | null>(null)

  const [loaded, setLoaded] = useState(false)
  const [loadError, setLoadError] = useState(false)
  const [zoomDisplay, setZoomDisplay] = useState(1)
  const [, forceRedraw] = useState(0)

  const redraw = useCallback(() => forceRedraw(n => n + 1), [])

  // Load image from file
  useEffect(() => {
    if (!file) return
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    setLoaded(false)
    setLoadError(false)
    zoomRef.current = 1
    setZoomDisplay(1)
    const url = URL.createObjectURL(file)
    const img = new Image()
    img.onload = () => {
      imgRef.current = img
      naturalRef.current = { w: img.naturalWidth, h: img.naturalHeight }
      const minScale = Math.max((RADIUS * 2) / img.naturalWidth, (RADIUS * 2) / img.naturalHeight)
      initialScaleRef.current = minScale
      offsetRef.current = {
        x: CX - (img.naturalWidth * minScale) / 2,
        y: CY - (img.naturalHeight * minScale) / 2,
      }
      setLoaded(true)
    }
    img.onerror = () => setLoadError(true)
    img.src = url
    return () => URL.revokeObjectURL(url)
  }, [file])

  // Draw canvas on every render (reads from refs, no deps needed)
  useEffect(() => {
    const canvas = canvasRef.current
    const img = imgRef.current
    if (!canvas || !img || !loaded) return
    const ctx = canvas.getContext('2d')!
    const scale = initialScaleRef.current * zoomRef.current
    const { x: ox, y: oy } = offsetRef.current
    ctx.clearRect(0, 0, CANVAS_SIZE, CANVAS_SIZE)
    ctx.drawImage(img, ox, oy, img.naturalWidth * scale, img.naturalHeight * scale)
    // Dark overlay outside circle using evenodd fill
    ctx.save()
    ctx.fillStyle = 'rgba(0,0,0,0.45)'
    ctx.beginPath()
    ctx.rect(0, 0, CANVAS_SIZE, CANVAS_SIZE)
    ctx.arc(CX, CY, RADIUS, 0, Math.PI * 2, true)
    ctx.fill('evenodd')
    ctx.restore()
    // Circle border
    ctx.save()
    ctx.strokeStyle = 'rgba(255,255,255,0.85)'
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.arc(CX, CY, RADIUS, 0, Math.PI * 2)
    ctx.stroke()
    ctx.restore()
  })

  // Non-passive touch listeners (must be added imperatively for e.preventDefault to work)
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || !loaded) return

    const onTouchStart = (e: TouchEvent) => {
      e.preventDefault()
      if (e.touches.length === 1) {
        const t = e.touches[0]
        dragRef.current = { startX: t.clientX, startY: t.clientY, startOX: offsetRef.current.x, startOY: offsetRef.current.y }
        pinchRef.current = null
      } else if (e.touches.length === 2) {
        dragRef.current = null
        const dx = e.touches[1].clientX - e.touches[0].clientX
        const dy = e.touches[1].clientY - e.touches[0].clientY
        const rect = canvas.getBoundingClientRect()
        const cs = CANVAS_SIZE / rect.width
        pinchRef.current = {
          dist: Math.hypot(dx, dy),
          zoom: zoomRef.current,
          midX: ((e.touches[0].clientX + e.touches[1].clientX) / 2 - rect.left) * cs,
          midY: ((e.touches[0].clientY + e.touches[1].clientY) / 2 - rect.top) * cs,
        }
      }
    }

    const onTouchMove = (e: TouchEvent) => {
      e.preventDefault()
      const rect = canvas.getBoundingClientRect()
      const cs = CANVAS_SIZE / rect.width
      const { w, h } = naturalRef.current
      const scale = initialScaleRef.current * zoomRef.current

      if (e.touches.length === 1 && dragRef.current) {
        const t = e.touches[0]
        offsetRef.current = clampOffset(
          dragRef.current.startOX + (t.clientX - dragRef.current.startX) * cs,
          dragRef.current.startOY + (t.clientY - dragRef.current.startY) * cs,
          scale, w, h,
        )
        redraw()
      } else if (e.touches.length === 2 && pinchRef.current) {
        const dx = e.touches[1].clientX - e.touches[0].clientX
        const dy = e.touches[1].clientY - e.touches[0].clientY
        const newDist = Math.hypot(dx, dy)
        const newZoom = Math.min(3, Math.max(1, pinchRef.current.zoom * (newDist / pinchRef.current.dist)))
        const newScale = initialScaleRef.current * newZoom
        const { midX, midY } = pinchRef.current
        // Zoom around pinch midpoint: keep image point under finger fixed
        const newOffset = clampOffset(
          midX - (midX - offsetRef.current.x) * (newScale / scale),
          midY - (midY - offsetRef.current.y) * (newScale / scale),
          newScale, w, h,
        )
        zoomRef.current = newZoom
        offsetRef.current = newOffset
        setZoomDisplay(newZoom)
        redraw()
      }
    }

    const onTouchEnd = () => { dragRef.current = null; pinchRef.current = null }

    canvas.addEventListener('touchstart', onTouchStart, { passive: false })
    canvas.addEventListener('touchmove', onTouchMove, { passive: false })
    canvas.addEventListener('touchend', onTouchEnd)
    return () => {
      canvas.removeEventListener('touchstart', onTouchStart)
      canvas.removeEventListener('touchmove', onTouchMove)
      canvas.removeEventListener('touchend', onTouchEnd)
    }
  }, [loaded, redraw])

  // Stop drag when mouse is released outside canvas
  useEffect(() => {
    const stop = () => { dragRef.current = null }
    window.addEventListener('mouseup', stop)
    return () => window.removeEventListener('mouseup', stop)
  }, [])

  const handleMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    dragRef.current = { startX: e.clientX, startY: e.clientY, startOX: offsetRef.current.x, startOY: offsetRef.current.y }
  }

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!dragRef.current) return
    const rect = e.currentTarget.getBoundingClientRect()
    const cs = CANVAS_SIZE / rect.width
    const scale = initialScaleRef.current * zoomRef.current
    const { w, h } = naturalRef.current
    offsetRef.current = clampOffset(
      dragRef.current.startOX + (e.clientX - dragRef.current.startX) * cs,
      dragRef.current.startOY + (e.clientY - dragRef.current.startY) * cs,
      scale, w, h,
    )
    redraw()
  }

  const handleSlider = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newZoom = parseFloat(e.target.value)
    const oldScale = initialScaleRef.current * zoomRef.current
    const newScale = initialScaleRef.current * newZoom
    const { w, h } = naturalRef.current
    // Zoom around canvas center
    offsetRef.current = clampOffset(
      CX - (CX - offsetRef.current.x) * (newScale / oldScale),
      CY - (CY - offsetRef.current.y) * (newScale / oldScale),
      newScale, w, h,
    )
    zoomRef.current = newZoom
    setZoomDisplay(newZoom)
  }

  const handleConfirm = () => {
    const img = imgRef.current
    if (!img) return
    const scale = initialScaleRef.current * zoomRef.current
    const { x: ox, y: oy } = offsetRef.current
    const ec = document.createElement('canvas')
    ec.width = EXPORT_SIZE
    ec.height = EXPORT_SIZE
    const ctx = ec.getContext('2d')!
    // Map circle region in canvas-space back to source image coordinates
    const cropX = (CX - RADIUS - ox) / scale
    const cropY = (CY - RADIUS - oy) / scale
    const cropSize = (RADIUS * 2) / scale
    ctx.drawImage(img, cropX, cropY, cropSize, cropSize, 0, 0, EXPORT_SIZE, EXPORT_SIZE)
    ec.toBlob(blob => { if (blob) onConfirm(blob) }, 'image/jpeg', 0.85)
  }

  if (!file) return null

  if (loadError) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
        <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 max-w-sm w-full mx-4">
          <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4">
            Bild konnte nicht geladen werden.
          </p>
          <button
            onClick={onCancel}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            Schließen
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" onClick={onCancel}>
      <div
        className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full"
        style={{ maxWidth: `${CANVAS_SIZE + 48}px` }}
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-semibold text-brand-text">Bildausschnitt wählen</h2>
          <button onClick={onCancel} className="text-brand-text-muted hover:text-brand-text" aria-label="Schließen">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="relative bg-brand-text rounded-lg overflow-hidden mb-4">
          <canvas
            ref={canvasRef}
            width={CANVAS_SIZE}
            height={CANVAS_SIZE}
            className="cursor-grab active:cursor-grabbing select-none touch-none block w-full"
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
          />
          {!loaded && (
            <div className="absolute inset-0 flex items-center justify-center">
              <div className="w-8 h-8 border-2 border-brand-yellow border-t-transparent rounded-full animate-spin" />
            </div>
          )}
        </div>

        <div className="flex items-center gap-3 mb-5">
          <ZoomIn className="w-4 h-4 text-brand-text-muted flex-shrink-0" />
          <input
            type="range" min={1} max={3} step={0.01}
            value={zoomDisplay}
            onChange={handleSlider}
            disabled={!loaded}
            className="flex-1 accent-brand-yellow"
          />
        </div>

        <div className="flex gap-3">
          <button
            onClick={onCancel}
            className="flex-1 border border-brand-border rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium text-brand-text hover:bg-brand-surface-card transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={handleConfirm}
            disabled={!loaded}
            className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            Hochladen
          </button>
        </div>
      </div>
    </div>
  )
}
