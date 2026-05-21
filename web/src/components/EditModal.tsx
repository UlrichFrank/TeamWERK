import { ReactNode } from 'react'

interface EditModalProps {
  isOpen: boolean
  title: string
  onClose: () => void
  onSave: () => void
  isSaving?: boolean
  children: ReactNode
}

export default function EditModal({ isOpen, title, onClose, onSave, isSaving = false, children }: EditModalProps) {
  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div
        className="fixed inset-0 bg-black/40"
        onClick={onClose}
      />
      <div className="relative bg-white rounded shadow-lg p-6 max-w-sm mx-4 w-full max-h-[90vh] overflow-y-auto">
        <h2 className="text-lg font-bold mb-4">{title}</h2>
        <div className="mb-6 space-y-4">
          {children}
        </div>
        <div className="flex gap-2 justify-end">
          <button
            onClick={onClose}
            disabled={isSaving}
            className="px-4 py-2.5 sm:py-1.5 border border-brand-black/20 rounded text-sm text-brand-black hover:bg-brand-gray disabled:opacity-50"
          >
            Abbrechen
          </button>
          <button
            onClick={onSave}
            disabled={isSaving}
            className="px-4 py-2.5 sm:py-1.5 bg-brand-yellow text-brand-black rounded text-sm font-medium hover:bg-brand-black hover:text-brand-yellow disabled:opacity-50"
          >
            {isSaving ? 'Speichert…' : 'Speichern'}
          </button>
        </div>
      </div>
    </div>
  )
}
