import { useEffect } from 'react'
import { X } from 'lucide-react'
import AuthImage from './AuthImage'

// Vollbild-Overlay für ein JWT-geschütztes Bild. Klick auf den Hintergrund, der
// X-Button und ESC schließen. Das Bild wird über AuthImage ein zweites Mal
// geladen (eigene Object-URL) — bewusst: die Thumbnail-Instanz revoked ihre URL
// beim Unmount, ein geteilter Blob wäre zwischen zwei Komponenten nicht sicher
// zu besitzen.
//
// Kein Pinch-Zoom im Overlay: auf 90vh skaliert reicht für Nachweise; wer echt
// zoomen will, lädt die Datei herunter und nutzt den nativen Viewer.
export default function ImageLightbox({
  url,
  alt,
  onClose,
}: {
  url: string
  alt?: string
  onClose: () => void
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={alt ?? 'Bild'}
      className="fixed inset-0 z-50 bg-black/80 flex items-center justify-center p-4"
      onClick={onClose}
    >
      <button
        type="button"
        onClick={onClose}
        aria-label="Schließen"
        className="absolute top-4 right-4 text-white hover:text-brand-yellow transition-colors"
      >
        <X className="w-7 h-7" />
      </button>
      <AuthImage
        url={url}
        alt={alt}
        className="max-h-[90vh] max-w-full object-contain rounded-lg"
      />
    </div>
  )
}
