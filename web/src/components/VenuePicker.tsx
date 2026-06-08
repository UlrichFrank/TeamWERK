import { useEffect, useRef, useState } from 'react'
import { ChevronDown, MapPin, Plus, X } from 'lucide-react'
import { api } from '../lib/api'

export interface Venue {
  id: number
  name: string
  street: string
  city: string
  postal_code: string
  country: string
  note: string
  is_home_venue: boolean
}

interface VenuePickerProps {
  value: number | null
  onChange: (venueId: number | null) => void
  disabled?: boolean
}

interface NewVenueForm {
  name: string
  street: string
  city: string
  postal_code: string
  note: string
}

const emptyForm: NewVenueForm = { name: '', street: '', city: '', postal_code: '', note: '' }

export default function VenuePicker({ value, onChange, disabled }: VenuePickerProps) {
  const [venues, setVenues] = useState<Venue[]>([])
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const [showModal, setShowModal] = useState(false)
  const [form, setForm] = useState<NewVenueForm>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    api.get<Venue[]>('/venues').then(r => setVenues(r.data)).catch(() => {})
  }, [])

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (!containerRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const selected = venues.find(v => v.id === value) ?? null

  const filtered = venues.filter(v => {
    const q = search.toLowerCase()
    return v.name.toLowerCase().includes(q) || v.city.toLowerCase().includes(q)
  })

  function handleSelect(id: number | null) {
    onChange(id)
    setOpen(false)
    setSearch('')
  }

  async function handleCreate() {
    if (!form.name || !form.street || !form.city || !form.postal_code) {
      setError('Name, Straße, Stadt und PLZ sind Pflichtfelder.')
      return
    }
    setSaving(true)
    setError('')
    try {
      const res = await api.post<{ id: number }>('/venues', form)
      const newVenue: Venue = { id: res.data.id, ...form, country: 'DE', is_home_venue: false }
      setVenues(vs => [...vs, newVenue])
      onChange(res.data.id)
      setShowModal(false)
      setForm(emptyForm)
    } catch {
      setError('Fehler beim Speichern.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <div ref={containerRef} className="relative">
        <button
          type="button"
          disabled={disabled}
          onClick={() => setOpen(o => !o)}
          className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text text-left flex items-center justify-between focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow disabled:opacity-50"
        >
          {selected ? (
            <span className="flex items-center gap-1.5">
              <MapPin className="w-3.5 h-3.5 text-brand-text-muted flex-shrink-0" />
              <span>{selected.name}, {selected.city}</span>
            </span>
          ) : (
            <span className="text-brand-text-subtle">Ort wählen...</span>
          )}
          <ChevronDown className="w-4 h-4 text-brand-text-muted flex-shrink-0" />
        </button>

        {open && (
          <div className="absolute z-50 top-full left-0 right-0 mt-1 bg-white border border-brand-border-subtle rounded-lg shadow-lg">
            <div className="p-2">
              <input
                autoFocus
                type="text"
                placeholder="Suchen..."
                value={search}
                onChange={e => setSearch(e.target.value)}
                className="w-full border border-brand-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
              />
            </div>
            <div className="max-h-48 overflow-y-auto">
              {selected && (
                <button
                  type="button"
                  onClick={() => handleSelect(null)}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm text-brand-text-muted hover:bg-brand-surface-card transition-colors"
                >
                  <X className="w-3.5 h-3.5" />
                  Kein Ort
                </button>
              )}
              {filtered.length === 0 && (
                <p className="px-3 py-2 text-sm text-brand-text-subtle">Keine Ergebnisse</p>
              )}
              {filtered.map(v => (
                <button
                  key={v.id}
                  type="button"
                  onClick={() => handleSelect(v.id)}
                  className={`w-full text-left px-3 py-2 text-sm transition-colors flex items-start gap-2 ${v.id === value ? 'bg-brand-yellow/10 text-brand-text' : 'hover:bg-brand-surface-card text-brand-text'}`}
                >
                  <MapPin className="w-3.5 h-3.5 mt-0.5 flex-shrink-0 text-brand-text-muted" />
                  <span>
                    <span className="font-medium">{v.name}</span>
                    <span className="text-brand-text-muted"> · {v.city}</span>
                  </span>
                </button>
              ))}
            </div>
            <div className="border-t border-brand-border-subtle">
              <button
                type="button"
                onClick={() => { setOpen(false); setShowModal(true) }}
                className="w-full flex items-center gap-2 px-3 py-2 text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
              >
                <Plus className="w-4 h-4" />
                Neuen Ort anlegen
              </button>
            </div>
          </div>
        )}
      </div>

      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold text-brand-text mb-4">Neuen Ort anlegen</h2>
            {error && (
              <p className="mb-3 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</p>
            )}
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-brand-text mb-1">Name *</label>
                <input
                  type="text"
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  placeholder="z.B. Porsche-Arena"
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text mb-1">Straße *</label>
                <input
                  type="text"
                  value={form.street}
                  onChange={e => setForm(f => ({ ...f, street: e.target.value }))}
                  placeholder="Musterstraße 1"
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              <div className="grid grid-cols-3 gap-2">
                <div>
                  <label className="block text-sm font-medium text-brand-text mb-1">PLZ *</label>
                  <input
                    type="text"
                    value={form.postal_code}
                    onChange={e => setForm(f => ({ ...f, postal_code: e.target.value }))}
                    placeholder="70372"
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                </div>
                <div className="col-span-2">
                  <label className="block text-sm font-medium text-brand-text mb-1">Stadt *</label>
                  <input
                    type="text"
                    value={form.city}
                    onChange={e => setForm(f => ({ ...f, city: e.target.value }))}
                    placeholder="Stuttgart"
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text mb-1">Hinweis</label>
                <input
                  type="text"
                  value={form.note}
                  onChange={e => setForm(f => ({ ...f, note: e.target.value }))}
                  placeholder="z.B. Parkhaus P3 empfohlen"
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 mt-5">
              <button
                type="button"
                onClick={() => { setShowModal(false); setForm(emptyForm); setError('') }}
                className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors"
              >
                Abbrechen
              </button>
              <button
                type="button"
                onClick={handleCreate}
                disabled={saving}
                className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {saving ? 'Speichern...' : 'Anlegen & auswählen'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
