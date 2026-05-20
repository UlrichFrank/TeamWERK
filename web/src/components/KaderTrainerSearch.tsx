import { useState, useRef, useEffect, useCallback } from 'react'
import { api } from '../lib/api'

interface Suggestion {
  id: number
  name: string
}

interface Trainer {
  id: number
  name: string
}

interface Props {
  assignedTrainers: Trainer[]
  onAdd: (memberId: number) => Promise<void>
  onRemove: (memberId: number) => Promise<void>
}

export default function KaderTrainerSearch({ assignedTrainers, onAdd, onRemove }: Props) {
  const [query, setQuery] = useState('')
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [filterTrainer, setFilterTrainer] = useState(true)
  const [busy, setBusy] = useState<Record<number, boolean>>({})
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const assignedIds = new Set(assignedTrainers.map(t => t.id))

  const fetchSuggestions = useCallback(async (search: string, onlyTrainers: boolean) => {
    setLoading(true)
    try {
      const params: Record<string, string | number> = { search, limit: 20 }
      if (onlyTrainers) params.club_function = 'trainer'
      const res = await api.get('/members', { params })
      const items: { id: number; first_name: string; last_name: string }[] = res.data?.items ?? []
      setSuggestions(items.map(m => ({ id: m.id, name: `${m.first_name} ${m.last_name}` })))
      setOpen(true)
    } catch {
      setSuggestions([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!query && !open) return
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => fetchSuggestions(query, filterTrainer), 300)
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current) }
  }, [query, filterTrainer, fetchSuggestions])

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSelect = async (s: Suggestion) => {
    if (assignedIds.has(s.id)) return
    setOpen(false)
    setQuery('')
    await onAdd(s.id)
  }

  const handleRemove = async (id: number) => {
    setBusy(prev => ({ ...prev, [id]: true }))
    try {
      await onRemove(id)
    } finally {
      setBusy(prev => ({ ...prev, [id]: false }))
    }
  }

  const visibleSuggestions = suggestions.filter(s => !assignedIds.has(s.id))

  return (
    <div>
      <div ref={containerRef} className="relative">
        <div className="flex gap-2 items-center">
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            onFocus={() => fetchSuggestions(query, filterTrainer)}
            placeholder="Trainer suchen…"
            className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-blue"
          />
          <label className="flex items-center gap-1 text-xs text-gray-500 whitespace-nowrap">
            <input
              type="checkbox"
              checked={filterTrainer}
              onChange={e => setFilterTrainer(e.target.checked)}
              className="accent-brand-blue"
            />
            Nur Trainer
          </label>
        </div>

        {open && (
          <div className="absolute z-30 left-0 right-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto">
            {loading && <div className="px-4 py-2 text-xs text-gray-400">Suche…</div>}
            {!loading && visibleSuggestions.length === 0 && (
              <div className="px-4 py-2 text-xs text-gray-400 italic">Keine Vorschläge</div>
            )}
            {!loading && visibleSuggestions.map(s => (
              <button
                key={s.id}
                onMouseDown={e => { e.preventDefault(); handleSelect(s) }}
                className="w-full text-left px-4 py-2 text-sm hover:bg-brand-gray transition-colors"
              >
                {s.name}
              </button>
            ))}
          </div>
        )}
      </div>

      {assignedTrainers.length > 0 && (
        <ul className="divide-y divide-gray-100 mt-1">
          {assignedTrainers.map(t => (
            <li key={t.id} className="flex items-center justify-between py-2 gap-2">
              <span className="text-sm font-medium text-brand-blue">{t.name}</span>
              <button
                onClick={() => handleRemove(t.id)}
                disabled={busy[t.id]}
                className="text-xs text-gray-400 hover:text-red-500 transition-colors disabled:opacity-40 px-1.5 py-0.5 rounded"
                title="Trainer entfernen"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
