import { useEffect, useRef, useState } from 'react'
import { ChevronDown, Home, MapPin, Pencil, Plus, Trash2 } from 'lucide-react'
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

interface ImportResult {
  imported: number
  updated: number
  skipped: number
  errors: { line: number; reason: string }[]
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

  const [showActionsMenu, setShowActionsMenu] = useState(false)
  const [showImport, setShowImport] = useState(false)
  const [importFile, setImportFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  const [importResult, setImportResult] = useState<ImportResult | null>(null)
  const [showDeleteAll, setShowDeleteAll] = useState(false)

  const actionsMenuRef = useRef<HTMLDivElement>(null)

  function load() {
    api.get<Venue[]>('/venues').then(r => {
      setVenues(r.data)
      setLoading(false)
    }).catch(() => setLoading(false))
  }

  useEffect(() => { load() }, [])
  useLiveUpdates((event: string) => { if (event === 'venues') load() })

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (actionsMenuRef.current && !actionsMenuRef.current.contains(e.target as Node)) {
        setShowActionsMenu(false)
      }
    }
    if (showActionsMenu) document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [showActionsMenu])

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
        await api.put(`/venues/${editVenue.id}`, form)
      } else {
        await api.post('/venues', form)
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
      await api.delete(`/venues/${id}`)
      setDeleteConfirm(null)
      load()
    } catch {
      setError('Fehler beim Löschen.')
    }
  }

  async function handleImport() {
    if (!importFile) return
    setImporting(true)
    try {
      const fd = new FormData()
      fd.append('file', importFile)
      const res = await api.post<ImportResult>('/venues/import', fd)
      setImportResult(res.data)
      load()
    } catch {
      alert('Import fehlgeschlagen. Bitte CSV-Format prüfen.')
    } finally {
      setImporting(false)
    }
  }

  function resetImport() {
    setShowImport(false)
    setImportFile(null)
    setImportResult(null)
  }

  async function handleDeleteAll() {
    try {
      await api.delete('/venues')
      setShowDeleteAll(false)
      load()
    } catch {
      alert('Fehler beim Löschen.')
    }
  }

  return (
    <div className="p-4 sm:p-8 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-brand-text">Veranstaltungsorte</h1>
        <div ref={actionsMenuRef} className="relative">
          <div className="flex">
            <button
              onClick={openCreate}
              className="bg-brand-yellow text-brand-black rounded-l-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors flex items-center gap-2"
            >
              <Plus className="w-4 h-4" />
              Neuer Ort
            </button>
            <button
              onClick={() => setShowActionsMenu(v => !v)}
              aria-label="Weitere Aktionen"
              className="bg-brand-yellow text-brand-black border-l border-brand-black/20 rounded-r-md px-2 py-2.5 sm:py-2 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors"
            >
              <ChevronDown className="w-4 h-4" />
            </button>
          </div>
          {showActionsMenu && (
            <div className="absolute right-0 mt-1 w-44 bg-white border border-brand-border rounded-md shadow-lg z-20 overflow-hidden">
              <button
                onClick={() => { setShowActionsMenu(false); setShowImport(true) }}
                className="w-full text-left px-4 py-2.5 text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
              >
                Import CSV
              </button>
              <button
                onClick={() => { setShowActionsMenu(false); setShowDeleteAll(true) }}
                className="w-full text-left px-4 py-2.5 text-sm text-brand-danger hover:bg-brand-danger-light transition-colors"
              >
                Alle löschen
              </button>
            </div>
          )}
        </div>
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

      {/* Import Modal */}
      {showImport && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
            <h2 className="text-lg font-semibold text-brand-text mb-4">CSV importieren</h2>

            {!importResult ? (
              <>
                <p className="text-sm text-brand-text-muted mb-4">
                  BWHV-Hallenliste hochladen. Bestehende Orte (gleicher Name) werden aktualisiert, neue werden angelegt.
                </p>
                <label className="block w-full border-2 border-dashed border-brand-border rounded-lg p-6 text-center cursor-pointer hover:border-brand-yellow transition-colors">
                  <input
                    type="file"
                    accept=".csv"
                    className="sr-only"
                    onChange={e => setImportFile(e.target.files?.[0] ?? null)}
                  />
                  {importFile ? (
                    <span className="text-sm text-brand-text font-medium">{importFile.name}</span>
                  ) : (
                    <span className="text-sm text-brand-text-muted">CSV-Datei auswählen</span>
                  )}
                </label>
                <div className="flex justify-end gap-2 mt-5">
                  <button
                    type="button"
                    onClick={resetImport}
                    className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors"
                  >
                    Abbrechen
                  </button>
                  <button
                    type="button"
                    onClick={handleImport}
                    disabled={!importFile || importing}
                    className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    {importing ? 'Importiere...' : 'Importieren'}
                  </button>
                </div>
              </>
            ) : (
              <>
                <div className="space-y-2 mb-5">
                  <p className="text-sm text-brand-text">
                    <span className="font-semibold text-brand-text">{importResult.imported}</span> neu importiert
                    {' · '}
                    <span className="font-semibold text-brand-text">{importResult.updated}</span> aktualisiert
                    {' · '}
                    <span className="font-semibold text-brand-text">{importResult.skipped}</span> übersprungen
                  </p>
                  {importResult.errors.length > 0 && (
                    <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg">
                      <p className="text-xs font-medium text-brand-danger mb-1">{importResult.errors.length} Fehler:</p>
                      <ul className="text-xs text-brand-danger space-y-0.5 max-h-32 overflow-y-auto">
                        {importResult.errors.map((e, i) => (
                          <li key={i}>Zeile {e.line}: {e.reason}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
                <div className="flex justify-end">
                  <button
                    type="button"
                    onClick={resetImport}
                    className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
                  >
                    Schließen
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {/* Delete all confirmation */}
      {showDeleteAll && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm">
            <h2 className="text-lg font-semibold text-brand-text mb-2">Alle Orte löschen?</h2>
            <p className="text-sm text-brand-text-muted mb-5">Alle Veranstaltungsorte außer der Heimhalle werden unwiderruflich gelöscht.</p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setShowDeleteAll(false)} className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors">Abbrechen</button>
              <button onClick={handleDeleteAll} className="bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors">
                Alle löschen
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
