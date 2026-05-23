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
    <div className="border border-black/10 rounded-lg overflow-hidden">
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between px-4 py-3 bg-brand-gray hover:bg-brand-gray/80 transition-colors min-h-[44px]"
      >
        <div className="flex items-center gap-2 font-semibold text-sm uppercase tracking-wider">
          <Icon size={18} />
          {title}
        </div>
        {isOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
      </button>
      {isOpen && (
        <div className="bg-white px-4 py-3">
          {children}
        </div>
      )}
    </div>
  )
}
