import { ReactNode, useEffect, useRef, useState } from 'react'
import { ChevronDown, Filter } from 'lucide-react'
import { getEventColors } from '../lib/eventColors'

export type EventTypeFilterEntry = [string, string, ReactNode]

interface Props {
  types: EventTypeFilterEntry[]
  active: Set<string>
  onToggle: (type: string) => void
  compact: boolean
  ariaLabel?: string
}

export default function EventTypeFilter({ types, active, onToggle, compact, ariaLabel = 'Typ-Filter' }: Props) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  if (!compact) {
    return (
      <>
        {types.map(([type, label, icon]) => (
          <button
            key={type}
            onClick={() => onToggle(type)}
            aria-label={label}
            className={`flex items-center gap-1 rounded-md px-3 py-1.5 text-xs font-medium border transition-colors shrink-0 ${
              active.has(type)
                ? getEventColors(type).filter
                : 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
            }`}
          >
            {icon}
            <span>{label}</span>
          </button>
        ))}
      </>
    )
  }

  const activeCount = types.filter(([t]) => active.has(t)).length
  const allActive = activeCount === types.length

  return (
    <div className="relative shrink-0" ref={ref}>
      <button
        onClick={() => setOpen(o => !o)}
        aria-label={ariaLabel}
        aria-expanded={open}
        className={`flex items-center gap-1 rounded-md px-2 py-1.5 text-xs font-medium border transition-colors ${
          allActive
            ? 'bg-white text-brand-text-muted border-brand-border hover:border-brand-text hover:text-brand-text'
            : 'bg-brand-yellow text-brand-black border-brand-yellow'
        }`}
      >
        <Filter className="w-3.5 h-3.5" />
        {!allActive && <span>{activeCount}/{types.length}</span>}
        <ChevronDown className="w-3.5 h-3.5" />
      </button>
      {open && (
        <div className="absolute left-0 top-full mt-1 z-20 bg-white border border-brand-border rounded-md shadow-lg py-1 min-w-[160px]">
          {types.map(([type, label, icon]) => (
            <label
              key={type}
              className="flex items-center gap-2 px-3 py-2 text-sm text-brand-text hover:bg-brand-table-select cursor-pointer"
            >
              <input
                type="checkbox"
                checked={active.has(type)}
                onChange={() => onToggle(type)}
                className="w-4 h-4 accent-brand-yellow"
              />
              <span className={getEventColors(type).pillIcon}>{icon}</span>
              <span>{label}</span>
            </label>
          ))}
        </div>
      )}
    </div>
  )
}
