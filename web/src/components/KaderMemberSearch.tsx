import { useState, useRef, useEffect, useCallback } from 'react'
import { api } from '../lib/api'
import BrandCheckbox from './BrandCheckbox'

interface Suggestion {
  id: number
  name: string
  birth_year: number
  gender: string
  reason: string
  already_in_kader: boolean
}

interface Props {
  kaderId: number
  onMemberAdded: () => void
  filterByAgeBracket?: boolean
  birthYears?: number[]
}

export default function KaderMemberSearch({ kaderId, onMemberAdded, filterByAgeBracket = true }: Props) {
  const [query, setQuery] = useState('')
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [filterAge, setFilterAge] = useState(filterByAgeBracket)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const fetchSuggestions = useCallback(async (search: string, filter: boolean) => {
    setLoading(true)
    try {
      const res = await api.get(`/admin/kader/${kaderId}/member-suggestions`, {
        params: { search, filter_age_bracket: filter },
      })
      setSuggestions(res.data.suggestions ?? [])
      setOpen(true)
    } catch {
      setSuggestions([])
    } finally {
      setLoading(false)
    }
  }, [kaderId])

  useEffect(() => {
    if (!query && !open) return
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => fetchSuggestions(query, filterAge), 300)
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current) }
  }, [query, filterAge, fetchSuggestions])

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
    if (s.already_in_kader) return
    try {
      await api.put(`/admin/kader/${kaderId}`, { members_add: [s.id], members_remove: [] })
      onMemberAdded()
    } catch {
      // ignore — parent will show error if needed
    }
    setOpen(false)
  }

  const genderLabel = (g: string) => g === 'm' ? 'm' : g === 'f' ? 'w' : 'mix'

  return (
    <div ref={containerRef} className="relative">
      <div className="flex gap-2 items-center">
        <input
          value={query}
          onChange={e => setQuery(e.target.value)}
          onFocus={() => fetchSuggestions(query, filterAge)}
          placeholder="Mitglied suchen…"
          className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-blue"
        />
        <BrandCheckbox
          checked={filterAge}
          onChange={setFilterAge}
          label="Jahrgang filtern"
        />
      </div>

      {open && (
        <div className="absolute z-30 left-0 right-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-48 overflow-y-auto">
          {loading && (
            <div className="px-4 py-2 text-xs text-gray-400">Suche…</div>
          )}
          {!loading && suggestions.length === 0 && (
            <div className="px-4 py-2 text-xs text-gray-400 italic">Keine Vorschläge</div>
          )}
          {!loading && suggestions.map(s => (
            <button
              key={s.id}
              onMouseDown={e => { e.preventDefault(); handleSelect(s) }}
              disabled={s.already_in_kader}
              className={`w-full text-left px-4 py-2 text-sm hover:bg-brand-gray transition-colors flex items-center justify-between gap-2
                ${s.already_in_kader ? 'opacity-40 cursor-not-allowed' : ''}`}
            >
              <span>
                {s.name}{' '}
                <span className="text-gray-400 text-xs">({s.birth_year}/{genderLabel(s.gender)})</span>
              </span>
              {s.already_in_kader && <span className="text-xs text-gray-400">bereits im Kader</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
