import { ChevronDown, ChevronRight, type LucideIcon } from 'lucide-react'

interface Props {
  id: string
  title: string
  icon: LucideIcon
  isOpen: boolean
  onToggle: () => void
  children: React.ReactNode
}

export default function Accordion({ id: _id, title, icon: Icon, isOpen, onToggle, children }: Props) {
  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between px-4 py-3 hover:bg-brand-border-subtle transition-colors min-h-[44px]"
      >
        <div className="flex items-center gap-2 font-semibold text-sm uppercase tracking-wider text-brand-text">
          <Icon size={18} />
          {title}
        </div>
        {isOpen
          ? <ChevronDown className="w-4 h-4 text-brand-text-muted" />
          : <ChevronRight className="w-4 h-4 text-brand-text-muted" />
        }
      </button>
      {isOpen && (
        <div className="px-4 py-3 border-t border-brand-border-subtle">
          {children}
        </div>
      )}
    </div>
  )
}
