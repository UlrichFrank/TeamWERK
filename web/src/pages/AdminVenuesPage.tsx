import { useEffect, useState } from 'react'
import { Home, MapPin, Plus, Pencil, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import ActionMenu from '../components/ActionMenu'
import MapsLink from '../components/MapsLink'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

interface Venue {
  id: number
  name: string
  street: string
  city: string
  postal_code: string
  country: string
  note: string
  is_home_venue: boolean
}

interface VenueForm {
  name: string
  street: string
  city: string
  postal_code: string
  country: string
  note: string
  is_home_venue: boolean
}

const emptyForm: VenueForm = {
  name: '', street: '', city: '', postal_code: '', country: 'DE', note: '', is_home_venue: false,
}

export default function AdminVenuesPage() {
  const [venues, setVenues] = useState<Venue[]>([])
  const [loading, setLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editVenue, setEditVenue] = useState<Venue | null>(null)
  const [form, setForm] = useState<VenueForm>(emptyForm)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)

  function load() {
    api.get<Venue[]>('/admin/venues').then(r => {
      setVenues(r.data)
      setLoading(false)
    }).catch(() => setLoading(false))
  }

  useEffect(() => { load() }, [])
  useLiveUpdates((event: string) => { if (event === 'venues') load() })

  function openCreate() {
    setEditVenue(null)
    setForm(emptyForm)
    setError('')
    setShowModal(true)
  }

  function openEdit(v: Venue) {
    setEditVenue(v)
    setForm({ name: v.name, street: v.street, city: v.city, postal_code: v.postal_code, country: v.country, note: v.note, is_home_venue: v.is_home_venue })
    setError('')
    setShowModal(true)
  }

  async function handleSave() {
    if (!form.name || !form.street || !form.city || !form.postal_code) {
      setError('Name, Straße, Stadt und PLZ sind Pflichtfelder.')
      return
    }
    setSaving(true)
    setError('')
    try {
      if (editVenue) {
        await api.put(`/admin/venues/${editVenue.id}`, form)
      } else {
        await api.post('/admin/venues', form)
      }
      setShowModal(false)
      load()
    } catch {
      setError('Fehler beim Speichern.')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id: number) {
    try {
      await api.delete(`/admin/venues/${id}`)
      setDeleteConfirm(null)
      load()
    } catch {
      setError('Fehler beim Löschen.')
    }
  }

  return (
    <div className="p-4 sm:p-8 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-brand-text">Veranstaltungsorte</h1>
        <button onClick={openCreate} className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors flex items-center gap-2">
          <Plus className="w-4 h-4" />
          Neuer Ort
        </button>
      </div>

      {loading ? (
        <p className="text-brand-text-muted text-sm">Lade...</p>
      ) : venues.length === 0 ? (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
          <MapPin className="w-10 h-10 text-brand-text-muted mx-auto mb-3" />
          <p className="text-brand-text-muted text-sm">Noch keine Veranstaltungsorte angelegt.</p>
        </div>
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden sm:block bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
            <table className="w-full">
              <thead>
                <tr>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Name</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Adresse</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Hinweis</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-subtle">
                {venues.map(v => (
                  <tr key={v.id} className="hover:bg-brand-table-select transition-colors">
                    <td className="px-4 py-3 text-sm text-brand-text">
                      <div className="flex items-center gap-2">
                        {v.is_home_venue && <Home className="w-4 h-4 text-brand-yellow flex-shrink-0" aria-label="Heimhalle" />}
                        <span className="font-medium">{v.name}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-brand-text">
                      <MapsLink venue={v} />
                      <span className="block text-xs text-brand-text-muted">{v.street}, {v.postal_code} {v.city}</span>
                    </td>
                    <td className="px-4 py-3 text-sm text-brand-text-muted">{v.note}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1 justify-end">
                        <button onClick={() => openEdit(v)} aria-label="Bearbeiten" className="p-1.5 text-brand-text-muted hover:text-brand-text transition-colors">
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button onClick={() => setDeleteConfirm(v.id)} aria-label="Löschen" className="p-1.5 text-brand-text-muted hover:text-brand-danger transition-colors">
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="sm:hidden space-y-3">
            {venues.map(v => (
              <div key={v.id} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4">
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      {v.is_home_venue && <Home className="w-4 h-4 text-brand-yellow flex-shrink-0" />}
                      <span className="font-semibold text-brand-text">{v.name}</span>
                    </div>
                    <MapsLink venue={v} className="mb-1" />
                    <p className="text-xs text-brand-text-muted">{v.street}, {v.postal_code} {v.city}</p>
                    {v.note && <p className="text-xs text-brand-text-muted mt-1">{v.note}</p>}
                  </div>
                  <ActionMenu actions={[
                    { label: 'Bearbeiten', onClick: () => openEdit(v) },
                    { label: 'Löschen', onClick: () => setDeleteConfirm(v.id), variant: 'danger' },
                  ]} />
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md max-h-[90vh] overflow-y-auto">
            <h2 className="text-lg font-semibold text-brand-text mb-4">
              {editVenue ? 'Ort bearbeiten' : 'Neuer Veranstaltungsort'}
            </h2>
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
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text mb-1">Straße *</label>
                <input
                  type="text"
                  value={form.street}
                  onChange={e => setForm(f => ({ ...f, street: e.target.value }))}
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
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                </div>
                <div className="col-span-2">
                  <label className="block text-sm font-medium text-brand-text mb-1">Stadt *</label>
                  <input
                    type="text"
                    value={form.city}
                    onChange={e => setForm(f => ({ ...f, city: e.target.value }))}
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text mb-1">Land</label>
                <input
                  type="text"
                  value={form.country}
                  onChange={e => setForm(f => ({ ...f, country: e.target.value }))}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
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
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={form.is_home_venue}
                  onChange={e => setForm(f => ({ ...f, is_home_venue: e.target.checked }))}
                  className="w-4 h-4 accent-brand-yellow"
                />
                <span className="text-sm text-brand-text flex items-center gap-1">
                  <Home className="w-4 h-4" /> Als Heimhalle markieren
                </span>
              </label>
            </div>
            <div className="flex justify-end gap-2 mt-5">
              <button
                type="button"
                onClick={() => setShowModal(false)}
                className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors"
              >
                Abbrechen
              </button>
              <button
                type="button"
                onClick={handleSave}
                disabled={saving}
                className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {saving ? 'Speichern...' : 'Speichern'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete confirmation */}
      {deleteConfirm !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
            <h2 className="text-lg font-semibold text-brand-text mb-2">Ort löschen?</h2>
            <p className="text-sm text-brand-text-muted mb-5">Events die diesem Ort zugeordnet sind, verlieren ihre Ortsangabe.</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setDeleteConfirm(null)} className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors">Abbrechen</button>
              <button onClick={() => handleDelete(deleteConfirm)} className="bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors">
                Löschen
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
