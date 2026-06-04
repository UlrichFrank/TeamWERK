import { X, RefreshCw } from 'lucide-react'

interface Props {
  onReload: () => void
  onDismiss: () => void
}

export function UpdateBanner({ onReload, onDismiss }: Props) {
  return (
    <div className="fixed bottom-0 left-0 right-0 z-50 flex items-center justify-between gap-3 bg-brand-yellow px-4 py-2.5 sm:py-2 shadow-lg">
      <div className="flex items-center gap-2 text-sm font-medium text-brand-black">
        <RefreshCw className="w-4 h-4 shrink-0" />
        <span>Neue Version verfügbar</span>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={onReload}
          className="rounded-md bg-brand-black px-3 py-2.5 sm:py-1.5 text-xs font-medium text-brand-yellow hover:bg-brand-black/80 transition-colors"
        >
          Jetzt neu laden
        </button>
        <button
          onClick={onDismiss}
          aria-label="Schließen"
          className="rounded-md p-2.5 sm:p-1.5 text-brand-black hover:bg-brand-black/10 transition-colors"
        >
          <X className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}
