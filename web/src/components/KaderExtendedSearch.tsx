import { useState, useRef, useEffect, useCallback } from 'react'
import { api } from '../lib/api'

interface Suggestion {
  id: number
  name: string
  birth_year: number
  gender: string
  already_in_kader: boolean
}

interface Props {
  kaderId: number
  onMemberAdded: () => void
}

export default function KaderExtendedSearch({ kaderId, onMemberAdded }: Props) {
  const [query, setQuery] = useState('')
  const [suggestions, setSuggestions] = useState<Suggestion[]>([])
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const fetchSuggestions = useCallback(async (search: string) => {
    setLoading(true)
    try {
      const res = await api.get(`/kader/${kaderId}/extended-member-suggestions`, {
        params: { search },
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
    debounceRef.current = setTimeout(() => fetchSuggestions(query), 300)
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current) }
    // open nur als Guard gelesen; bewusst keine Dep, sonst doppelter Fetch bei Focus
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query, fetchSuggestions])

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
      await api.put(`/kader/${kaderId}`, { extended_members_add: [s.id] })
      onMemberAdded()
    } catch {
      // ignore — parent will show error if needed
    }
    setOpen(false)
    setQuery('')
  }

  const genderLabel = (g: string) => g === 'm' ? 'm' : g === 'f' ? 'w' : 'mix'

  return (
    <div ref={containerRef} className="relative">
      <input
        value={query}
        onChange={e => setQuery(e.target.value)}
        onFocus={() => fetchSuggestions(query)}
        placeholder="Mitglied für erweiterten Kader suchen…"
        className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
      />

      {open && (
        <div className="absolute z-30 left-0 right-0 mt-1 bg-white border border-brand-border-subtle rounded-lg shadow-lg max-h-48 overflow-y-auto">
          {loading && (
            <div className="px-4 py-2 text-xs text-brand-text-subtle">Suche…</div>
          )}
          {!loading && suggestions.length === 0 && (
            <div className="px-4 py-2 text-xs text-brand-text-subtle italic">Keine Vorschläge</div>
          )}
          {!loading && suggestions.map(s => (
            <button
              key={s.id}
              onMouseDown={e => { e.preventDefault(); handleSelect(s) }}
              disabled={s.already_in_kader}
              className={`w-full text-left px-4 py-2 text-sm hover:bg-brand-gray transition-colors flex items-center justify-between gap-2
                ${s.already_in_kader ? 'opacity-40 cursor-not-allowed' : ''}`}
            >
              <span className="text-brand-text">
                {s.name}{' '}
                <span className="text-brand-text-subtle text-xs">({s.birth_year}/{genderLabel(s.gender)})</span>
              </span>
              {s.already_in_kader && <span className="text-xs text-brand-text-subtle">bereits im erweiterten Kader</span>}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
