import { ReactNode } from 'react'
import { X } from 'lucide-react'

interface EditModalProps {
  isOpen: boolean
  title: string
  onClose: () => void
  onSave: () => void
  isSaving?: boolean
  maxWidthClass?: string
  children: ReactNode
}

export default function EditModal({ isOpen, title, onClose, onSave, isSaving = false, maxWidthClass = 'max-w-sm', children }: EditModalProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={onClose} />
      <div className={`relative bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow ${maxWidthClass} mx-4 w-full max-h-[90vh] overflow-y-auto`}>
        <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
          <h2 className="text-lg font-bold text-brand-text">{title}</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>
        <div className="p-6 space-y-4">
          {children}
        </div>
        <div className="flex gap-2 justify-end px-6 py-4 border-t border-brand-border-subtle">
          <button
            onClick={onClose}
            disabled={isSaving}
            className="px-4 py-2.5 sm:py-2 border border-brand-border rounded-md text-sm text-brand-text hover:bg-brand-surface-card disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            Abbrechen
          </button>
          <button
            onClick={onSave}
            disabled={isSaving}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {isSaving ? 'Speichert…' : 'Speichern'}
          </button>
        </div>
      </div>
    </div>
  )
}
