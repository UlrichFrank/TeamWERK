import { useEffect, useRef, useState } from 'react'

export interface SearchableSelectItem {
  value: string
  label: string
  searchText: string
  likely?: boolean
}

interface Props {
  items: SearchableSelectItem[]
  value: string
  onChange: (value: string) => void
  placeholder?: string
  clearLabel?: string
  maxResults?: number
  disabled?: boolean
  className?: string
}

export default function SearchableSelect({
  items, value, onChange, placeholder, clearLabel, maxResults = 100, disabled, className,
}: Props) {
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
        setQuery('')
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const selectedItem = items.find(i => i.value === value)

  const q = query.trim().toLowerCase()
  const filtered = q === '' ? items : items.filter(i => i.searchText.includes(q))
  const likely = filtered.filter(i => i.likely)
  const rest = filtered.filter(i => !i.likely)
  const combined = [...likely, ...rest]
  const shown = combined.slice(0, maxResults)
  const likelyShown = shown.filter(i => i.likely).length
  const restShown = shown.length - likelyShown
  const truncatedCount = combined.length - shown.length

  const handleSelect = (item: SearchableSelectItem) => {
    onChange(item.value)
    setOpen(false)
    setQuery('')
  }

  const handleClear = () => {
    onChange('')
    setOpen(false)
    setQuery('')
  }

  const renderItem = (item: SearchableSelectItem) => (
    <button
      key={item.value}
      type="button"
      onMouseDown={e => { e.preventDefault(); handleSelect(item) }}
      className={`w-full text-left px-4 py-2 text-sm hover:bg-brand-table-select transition-colors ${item.value === value ? 'font-medium text-brand-text' : 'text-brand-text'}`}
    >
      {item.label}
    </button>
  )

  return (
    <div ref={containerRef} className={`relative ${className ?? ''}`}>
      <input
        type="text"
        value={open ? query : (selectedItem?.label ?? '')}
        onChange={e => { setQuery(e.target.value); setOpen(true) }}
        onFocus={() => setOpen(true)}
        onKeyDown={e => { if (e.key === 'Escape') { setOpen(false); setQuery('') } }}
        placeholder={open ? (selectedItem?.label || placeholder) : placeholder}
        disabled={disabled}
        autoComplete="off"
        className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow disabled:opacity-40 disabled:cursor-not-allowed"
      />

      {open && !disabled && (
        <div className="absolute z-30 left-0 right-0 mt-1 bg-white border border-brand-border-subtle rounded-lg shadow-lg max-h-64 overflow-y-auto">
          {clearLabel && (
            <button
              type="button"
              onMouseDown={e => { e.preventDefault(); handleClear() }}
              className="w-full text-left px-4 py-2 text-sm text-brand-text-muted italic hover:bg-brand-table-select transition-colors"
            >
              {clearLabel}
            </button>
          )}

          {shown.length === 0 && (
            <div className="px-4 py-2 text-xs text-brand-text-subtle italic">Keine Treffer</div>
          )}

          {likelyShown > 0 && (
            <div className="px-4 py-1 text-xs uppercase text-brand-text-muted">Wahrscheinliche Treffer</div>
          )}
          {shown.slice(0, likelyShown).map(renderItem)}

          {likelyShown > 0 && restShown > 0 && (
            <div className="border-t border-brand-border-subtle my-1" />
          )}
          {shown.slice(likelyShown).map(renderItem)}

          {truncatedCount > 0 && (
            <div className="px-4 py-2 text-xs text-brand-text-subtle italic">
              … und {truncatedCount} weitere. Bitte weiter eingrenzen.
            </div>
          )}
        </div>
      )}
    </div>
  )
}
