import { useEffect, useRef, useState } from 'react'

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
        onClick={() => setOpen(!open)}
        className="px-2 py-1 text-brand-black/60 hover:text-brand-black"
      >
        ⋮
      </button>
      {open && (
        <div className="absolute right-0 mt-1 bg-white border border-brand-black/10 rounded shadow-lg z-20 min-w-32">
          {actions.map((action, idx) => (
            <button
              key={idx}
              onClick={() => {
                action.onClick()
                setOpen(false)
              }}
              className={`block w-full text-left px-4 py-2 text-sm ${
                action.variant === 'danger'
                  ? 'text-red-600 hover:bg-red-50'
                  : 'text-brand-black hover:bg-brand-gray'
              } transition-colors ${idx > 0 ? 'border-t border-brand-black/10' : ''}`}
            >
              {action.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
