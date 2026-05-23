import { useEffect, useRef, useState } from 'react'
import { MoreVertical } from 'lucide-react'

interface Action {
  label: string
  onClick: () => void
  variant?: 'default' | 'danger'
}

interface ActionMenuProps {
  actions: Action[]
}

export default function ActionMenu({ actions }: ActionMenuProps) {
  const [open, setOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    if (open) {
      document.addEventListener('mousedown', handleClickOutside)
      return () => document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [open])

  return (
    <div ref={menuRef} className="relative">
      <button
        onClick={e => { e.stopPropagation(); setOpen(!open) }}
        aria-label="Aktionen"
        className="px-2 py-1 text-brand-text-muted hover:text-brand-text transition-colors"
      >
        <MoreVertical className="w-4 h-4" />
      </button>
      {open && (
        <div className="absolute right-0 mt-1 bg-white border border-brand-border-subtle rounded shadow-lg z-20 min-w-32">
          {actions.map((action, idx) => (
            <button
              key={idx}
              onClick={() => {
                action.onClick()
                setOpen(false)
              }}
              className={`block w-full text-left px-4 py-2 text-sm ${
                action.variant === 'danger'
                  ? 'text-brand-danger hover:bg-brand-danger-light'
                  : 'text-brand-text hover:bg-brand-gray'
              } transition-colors ${idx > 0 ? 'border-t border-brand-border-subtle' : ''}`}
            >
              {action.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
