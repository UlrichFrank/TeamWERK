import { useState, useRef, useEffect, useCallback } from 'react'
import { X } from 'lucide-react'
import { api } from '../lib/api'
import BrandCheckbox from './BrandCheckbox'
import PersonChip from './PersonChip'

interface Suggestion {
  id: number
  name: string
}

interface Trainer {
  id: number
  name: string
  user_id?: number
  status?: string
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
        <div className="relative">
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            onFocus={() => fetchSuggestions(query, filterTrainer)}
            placeholder="Trainer suchen…"
            className="w-full border border-brand-border rounded-md px-3 py-2 pr-9 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
          <div className="absolute right-2 top-1/2 -translate-y-1/2">
            <BrandCheckbox
              checked={filterTrainer}
              onChange={setFilterTrainer}
              title="Nur Trainer"
            />
          </div>
        </div>

        {open && (
          <div className="absolute z-30 left-0 right-0 mt-1 bg-white border border-brand-border-subtle rounded-lg shadow-lg max-h-48 overflow-y-auto">
            {loading && <div className="px-4 py-2 text-xs text-brand-text-subtle">Suche…</div>}
            {!loading && visibleSuggestions.length === 0 && (
              <div className="px-4 py-2 text-xs text-brand-text-subtle italic">Keine Vorschläge</div>
            )}
            {!loading && visibleSuggestions.map(s => (
              <button
                key={s.id}
                onMouseDown={e => { e.preventDefault(); handleSelect(s) }}
                className="w-full text-left px-4 py-2 text-sm text-brand-text hover:bg-brand-gray transition-colors"
              >
                {s.name}
              </button>
            ))}
          </div>
        )}
      </div>

      {assignedTrainers.length > 0 && (
        <ul className="divide-y divide-brand-border-subtle mt-1">
          {assignedTrainers.map(t => (
            <li key={t.id} className="flex items-center justify-between py-2 gap-2">
              <div className="flex items-center gap-2 min-w-0">
                <PersonChip userId={t.user_id} name={t.name} />
                {t.status === 'honorar' && (
                  <span className="inline-flex rounded-full px-2 py-0.5 text-xs font-medium bg-brand-blue/10 text-brand-blue shrink-0">Honorar</span>
                )}
              </div>
              <button
                onClick={() => handleRemove(t.id)}
                disabled={busy[t.id]}
                aria-label={`${t.name} entfernen`}
                className="text-brand-text-subtle hover:text-brand-danger transition-colors disabled:opacity-40 p-1 rounded"
              >
                <X className="w-4 h-4" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
