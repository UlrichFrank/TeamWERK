import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { MoreVertical } from 'lucide-react'
import { MENU_ITEM, MENU_ITEM_DANGER } from '../lib/typography'

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
  const [pos, setPos] = useState({ top: 0, right: 0 })
  const buttonRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (!buttonRef.current?.contains(e.target as Node) && !menuRef.current?.contains(e.target as Node))
        setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  function toggle(e: React.MouseEvent) {
    e.stopPropagation()
    if (buttonRef.current) {
      const rect = buttonRef.current.getBoundingClientRect()
      setPos({ top: rect.bottom + 4, right: window.innerWidth - rect.right })
    }
    setOpen(o => !o)
  }

  return (
    <>
      <button
        ref={buttonRef}
        onClick={toggle}
        aria-label="Aktionen"
        className="px-2 py-1 text-brand-text-muted hover:text-brand-text transition-colors"
      >
        <MoreVertical className="w-4 h-4" />
      </button>
      {open && createPortal(
        <div
          ref={menuRef}
          style={{ position: 'fixed', top: pos.top, right: pos.right, zIndex: 9999 }}
          role="menu"
          className="bg-white border border-brand-border-subtle rounded-lg shadow-lg min-w-[140px] py-1"
        >
          {actions.map((action, idx) => (
            <button
              key={idx}
              role="menuitem"
              onClick={() => { action.onClick(); setOpen(false) }}
              className={`${action.variant === 'danger' ? MENU_ITEM_DANGER : MENU_ITEM}${
                idx > 0 ? ' border-t border-brand-border-subtle' : ''
              }`}
            >
              {action.label}
            </button>
          ))}
        </div>,
        document.body
      )}
    </>
  )
}
