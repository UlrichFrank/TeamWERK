import { useEffect, useState, FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { X } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import EditModal from '../components/EditModal'
import MobileCard from '../components/MobileCard'
import { useEscapeKey } from '../lib/useEscapeKey'
import NumberSpinner from '../components/NumberSpinner'
import { BEITRAGS_KATEGORIEN, kategorieLabel } from '../lib/beitragsKategorien'

// ─── Shared styles ────────────────────────────────────────────────────────────

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_PRIMARY = 'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_SM = 'bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_DANGER_SM = 'bg-brand-danger text-white rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed'

// ─── Verein Tab ───────────────────────────────────────────────────────────────

const GLAEUBIGER_RE = /^DE\d{2}[A-Z0-9]{3}\d{11}$/

function VereinTab() {
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [glaeubigerId, setGlaeubigerId] = useState('')
  const [iban, setIban] = useState('')
  const [bic, setBic] = useState('')
  const [kontoinhaber, setKontoinhaber] = useState('')
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    if (loaded) return
    api.get('/club').then(r => {
      setName(r.data.name ?? '')
      setAddress(r.data.address ?? '')
      setGlaeubigerId(r.data.glaeubiger_id ?? '')
      setIban(r.data.iban ?? '')
      setBic(r.data.bic ?? '')
      setKontoinhaber(r.data.kontoinhaber ?? '')
      setLoaded(true)
    })
  }, [loaded])

  useLiveUpdates(event => { if (event === 'settings') setLoaded(false) })

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    const gid = glaeubigerId.replace(/\s/g, '').toUpperCase()
    if (gid && !GLAEUBIGER_RE.test(gid)) {
      setError('Gläubiger-ID hat ein ungültiges Format (z. B. DE98ZZZ09999999999).')
      return
    }
    try {
      await api.put('/club', {
        name, address,
        glaeubiger_id: gid,
        iban: iban.replace(/\s/g, '').toUpperCase(),
        bic: bic.replace(/\s/g, '').toUpperCase(),
        kontoinhaber,
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Speichern fehlgeschlagen – bitte IBAN/BIC/Gläubiger-ID prüfen.')
    }
  }

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow px-5 py-5 max-w-lg">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Vereinsname</label>
          <input value={name} onChange={e => setName(e.target.value)} className={INPUT} />
        </div>
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Adresse</label>
          <input value={address} onChange={e => setAddress(e.target.value)} className={INPUT} />
        </div>

        <div className="pt-2 border-t border-brand-border-subtle">
          <h3 className="text-sm font-semibold text-brand-text mb-3">SEPA-Stammdaten</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Gläubiger-ID</label>
              <input value={glaeubigerId} onChange={e => setGlaeubigerId(e.target.value)} placeholder="DE98ZZZ09999999999" className={INPUT} />
            </div>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Kontoinhaber</label>
              <input value={kontoinhaber} onChange={e => setKontoinhaber(e.target.value)} className={INPUT} />
            </div>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">IBAN</label>
              <input value={iban} onChange={e => setIban(e.target.value)} placeholder="DE.." className={INPUT} />
            </div>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">BIC</label>
              <input value={bic} onChange={e => setBic(e.target.value)} className={INPUT} />
            </div>
          </div>
        </div>

        {error && (
          <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</div>
        )}
        <button type="submit" className={BTN_PRIMARY}>
          {saved ? 'Gespeichert ✓' : 'Speichern'}
        </button>
      </form>
    </div>
  )
}

// ─── Saisons Tab ─────────────────────────────────────────────────────────────

interface Season {
  id: number
  name: string
  start_date: string
  end_date: string
  is_active: boolean
}

function generateSeasonOptions() {
  const now = new Date()
  const currentYear = now.getFullYear()
  const startYear = now.getMonth() < 6 ? currentYear - 1 : currentYear
  return [0, 1, 2].map(offset => {
    const year = startYear + offset
    return { year, label: `${year}/${String(year + 1).slice(-2)}` }
  })
}

function SaisonsTab() {
  const [seasons, setSeasons] = useState<Season[]>([])
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)

  // Create modal
  const [showCreate, setShowCreate] = useState(false)
  const [preset, setPreset] = useState('')
  const [createName, setCreateName] = useState('')
  const [createStart, setCreateStart] = useState('')
  const [createEnd, setCreateEnd] = useState('')
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  // Edit modal
  const [editId, setEditId] = useState<number | null>(null)
  const [editName, setEditName] = useState('')
  const [editStart, setEditStart] = useState('')
  const [editEnd, setEditEnd] = useState('')
  const [editActive, setEditActive] = useState(false)
  const [saving, setSaving] = useState(false)

  const [deleting, setDeleting] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = () => api.get('/seasons').then(r => setSeasons(r.data ?? []))

  useLiveUpdates(event => { if (event === 'settings') load() })

  useEffect(() => {
    if (loaded) return
    setLoading(true)
    load().finally(() => { setLoading(false); setLoaded(true) })
  }, [loaded])

  const handlePreset = (label: string) => {
    setPreset(label)
    const m = label.match(/(\d{4})\//)
    if (m) {
      const year = parseInt(m[1])
      setCreateName(label)
      setCreateStart(`${year}-07-01`)
      setCreateEnd(`${year + 1}-06-30`)
    }
  }

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    if (!createName || !createStart || !createEnd) return
    setCreating(true)
    setCreateError(null)
    try {
      await api.post('/seasons', { name: createName, start_date: createStart, end_date: createEnd })
      setShowCreate(false)
      setPreset(''); setCreateName(''); setCreateStart(''); setCreateEnd('')
      await load()
    } catch {
      setCreateError('Saison konnte nicht angelegt werden.')
    } finally {
      setCreating(false)
    }
  }

  const openEdit = (s: Season) => {
    setEditId(s.id)
    setEditName(s.name)
    setEditStart(s.start_date.slice(0, 10))
    setEditEnd(s.end_date.slice(0, 10))
    setEditActive(s.is_active)
  }

  const handleSaveEdit = async () => {
    if (!editId) return
    setSaving(true)
    try {
      await api.put(`/seasons/${editId}`, { name: editName, start_date: editStart, end_date: editEnd })
      setEditId(null)
      await load()
    } finally {
      setSaving(false)
    }
  }

  const handleActivate = async (id: number) => {
    await api.put(`/seasons/${id}/activate`, {})
    await load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Saison wirklich löschen?')) return
    setDeleting(id)
    setError(null)
    try {
      await api.delete(`/seasons/${id}`)
      await load()
    } catch {
      setError('Saison konnte nicht gelöscht werden.')
    } finally {
      setDeleting(null)
    }
  }

  useEscapeKey(showCreate ? () => setShowCreate(false) : null)

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <span className="text-sm text-brand-text-muted">{seasons.length} Saison{seasons.length !== 1 ? 'en' : ''}</span>
        <button onClick={() => setShowCreate(true)} className={BTN_PRIMARY}>
          + Saison anlegen
        </button>
      </div>

      {error && (
        <p className="mb-3 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</p>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4 flex flex-col max-h-[90vh]">
            <div className="flex items-center justify-between px-6 pt-6 pb-4 shrink-0 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-lg text-brand-text">Neue Saison</h2>
              <button onClick={() => setShowCreate(false)} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreate} className="flex flex-col flex-1 min-h-0">
              <div className="overflow-y-auto px-6 py-4 space-y-4 flex-1">
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Saison</label>
                  <select value={preset} onChange={e => handlePreset(e.target.value)} className={INPUT} required>
                    <option value="">Wählen…</option>
                    {generateSeasonOptions().map(opt => (
                      <option key={opt.year} value={opt.label}>{opt.label}</option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Name</label>
                  <input value={createName} onChange={e => setCreateName(e.target.value)} className={INPUT} required />
                </div>
                <div className="flex gap-3">
                  <div className="flex-1">
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Startdatum</label>
                    <input type="date" value={createStart} onChange={e => setCreateStart(e.target.value)} className={INPUT} required />
                  </div>
                  <div className="flex-1">
                    <label className="block text-sm font-medium text-brand-text-muted mb-1">Enddatum</label>
                    <input type="date" value={createEnd} onChange={e => setCreateEnd(e.target.value)} className={INPUT} required />
                  </div>
                </div>
                {createError && (
                  <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{createError}</p>
                )}
              </div>
              <div className="flex gap-2 px-6 py-4 border-t border-brand-border-subtle shrink-0">
                <button type="submit" disabled={creating} className={`flex-1 ${BTN_PRIMARY}`}>
                  {creating ? 'Anlegen…' : 'Anlegen'}
                </button>
                <button type="button" onClick={() => setShowCreate(false)}
                  className="px-4 py-2.5 sm:py-2 text-sm border border-brand-border rounded-md text-brand-text hover:bg-brand-surface-card transition-colors">
                  Abbrechen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      <EditModal
        isOpen={editId !== null}
        title={editActive ? `Bearbeiten: ${editName} (aktiv)` : `Bearbeiten: ${editName}`}
        onClose={() => setEditId(null)}
        onSave={handleSaveEdit}
        isSaving={saving}
      >
        {editActive && (
          <p className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            Das ist die aktive Saison. Datumsänderungen wirken sofort.
          </p>
        )}
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Name</label>
          <input value={editName} onChange={e => setEditName(e.target.value)} className={INPUT} />
        </div>
        <div className="flex gap-3">
          <div className="flex-1">
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Startdatum</label>
            <input type="date" value={editStart} onChange={e => setEditStart(e.target.value)} className={INPUT} />
          </div>
          <div className="flex-1">
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Enddatum</label>
            <input type="date" value={editEnd} onChange={e => setEditEnd(e.target.value)} className={INPUT} />
          </div>
        </div>
      </EditModal>

      {/* Mobile: Cards */}
      <div className="sm:hidden space-y-0">
        {loading ? (
          <div className="text-sm text-brand-text-muted py-4">Laden…</div>
        ) : seasons.length === 0 ? (
          <p className="text-sm text-brand-text-subtle text-center py-8 italic">Noch keine Saisons angelegt.</p>
        ) : (
          seasons.map(s => (
            <MobileCard
              key={s.id}
              title={s.name}
              subtitle={`${s.start_date.slice(0, 10)} – ${s.end_date.slice(0, 10)}`}
              badge={s.is_active ? { label: 'aktiv', variant: 'green' } : undefined}
              actions={[
                { label: 'Bearbeiten', onClick: () => openEdit(s) },
                ...(!s.is_active ? [
                  { label: 'Aktivieren', onClick: () => handleActivate(s.id) },
                  { label: 'Löschen', onClick: () => handleDelete(s.id), variant: 'danger' as const },
                ] : []),
              ]}
            />
          ))
        )}
      </div>

      {/* Desktop: Table */}
      <div className="hidden sm:block bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        {loading ? (
          <div className="px-5 py-8 text-sm text-brand-text-muted text-center">Laden…</div>
        ) : seasons.length === 0 ? (
          <p className="text-sm text-brand-text-subtle text-center py-8 italic">Noch keine Saisons angelegt.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr>
                <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Name</th>
                <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Zeitraum</th>
                <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Status</th>
                <th className="bg-brand-surface-card px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-brand-border-subtle">
              {seasons.map(s => (
                <tr key={s.id} className="hover:bg-brand-table-select transition-colors">
                  <td className="px-4 py-3 font-medium text-brand-text">{s.name}</td>
                  <td className="px-4 py-3 text-brand-text-muted text-xs">
                    {s.start_date.slice(0, 10)} – {s.end_date.slice(0, 10)}
                  </td>
                  <td className="px-4 py-3">
                    {s.is_active && (
                      <span className="text-xs bg-brand-success-light text-brand-success px-2 py-0.5 rounded-full font-medium">aktiv</span>
                    )}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1 justify-end">
                      <button onClick={() => openEdit(s)} className={BTN_SM}>Bearbeiten</button>
                      {!s.is_active && (
                        <>
                          <button onClick={() => handleActivate(s.id)} className={BTN_SM}>Aktivieren</button>
                          <button
                            onClick={() => handleDelete(s.id)}
                            disabled={deleting === s.id}
                            className={BTN_DANGER_SM}
                          >
                            {deleting === s.id ? 'Löschen…' : 'Löschen'}
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

// ─── Altersklassen Tab ────────────────────────────────────────────────────────

interface AgeClassRule {
  age_class: string
  half_duration_minutes: number
  break_minutes: number
}

interface RowState {
  half: string
  brk: string
  saving: boolean
  error: string
  success: boolean
}

function AltersklassenTab() {
  const [rules, setRules] = useState<AgeClassRule[]>([])
  const [rowStates, setRowStates] = useState<Record<string, RowState>>({})
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)

  useEffect(() => {
    if (loaded) return
    setLoading(true)
    api.get<AgeClassRule[]>('/age-class-rules').then(r => {
      const data: AgeClassRule[] = Array.isArray(r.data) ? r.data : []
      setRules(data)
      const initial: Record<string, RowState> = {}
      for (const rule of data) {
        initial[rule.age_class] = { half: String(rule.half_duration_minutes), brk: String(rule.break_minutes), saving: false, error: '', success: false }
      }
      setRowStates(initial)
    }).finally(() => { setLoading(false); setLoaded(true) })
  }, [loaded])

  function updateRow(ageClass: string, field: 'half' | 'brk', value: string) {
    setRowStates(prev => ({ ...prev, [ageClass]: { ...prev[ageClass], [field]: value, error: '', success: false } }))
  }

  async function saveRow(ageClass: string) {
    const s = rowStates[ageClass]
    const half = parseInt(s.half)
    const brk = parseInt(s.brk)
    if (!half || half <= 0 || !brk || brk <= 0) {
      setRowStates(prev => ({ ...prev, [ageClass]: { ...prev[ageClass], error: 'Werte müssen > 0 sein.' } }))
      return
    }
    setRowStates(prev => ({ ...prev, [ageClass]: { ...prev[ageClass], saving: true, error: '' } }))
    try {
      await api.put(`/age-class-rules/${ageClass}`, { half_duration_minutes: half, break_minutes: brk })
      setRowStates(prev => ({ ...prev, [ageClass]: { ...prev[ageClass], saving: false, success: true } }))
    } catch {
      setRowStates(prev => ({ ...prev, [ageClass]: { ...prev[ageClass], saving: false, error: 'Speichern fehlgeschlagen.' } }))
    }
  }

  return (
    <div>
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <table className="w-full">
          <thead>
            <tr>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Klasse</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Halbzeit (min)</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Pause (min)</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Gesamt</th>
              <th className="bg-brand-surface-card px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-brand-text-muted text-sm">Laden…</td></tr>
            ) : (
              rules.map(rule => {
                const s = rowStates[rule.age_class]
                if (!s) return null
                const half = parseInt(s.half) || 0
                const brk = parseInt(s.brk) || 0
                const total = half > 0 && brk > 0 ? 2 * half + brk : '—'
                return (
                  <tr key={rule.age_class} className="border-t border-brand-border-subtle">
                    <td className="px-4 py-3 text-sm font-semibold text-brand-text">{rule.age_class}</td>
                    <td className="px-4 py-3">
                      <NumberSpinner value={parseInt(s.half) || 1} min={1} step={5} onChange={v => updateRow(rule.age_class, 'half', String(v))} />
                    </td>
                    <td className="px-4 py-3">
                      <NumberSpinner value={parseInt(s.brk) || 1} min={1} step={5} onChange={v => updateRow(rule.age_class, 'brk', String(v))} />
                    </td>
                    <td className="px-4 py-3 text-sm text-brand-text-muted">
                      {total !== '—' ? `${total} min` : '—'}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex flex-col items-end gap-1">
                        <button onClick={() => saveRow(rule.age_class)} disabled={s.saving} className={BTN_SM}>
                          {s.saving ? 'Speichern…' : 'Speichern'}
                        </button>
                        {s.error && <span className="text-xs text-brand-danger">{s.error}</span>}
                        {s.success && !s.error && <span className="text-xs text-brand-success">Gespeichert</span>}
                      </div>
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>
      <p className="mt-4 text-sm text-brand-text-muted">
        Gesamt-Spieldauer = 2 × Halbzeit + Pause. Wird für Slot-Zeitberechnung verwendet.
      </p>
    </div>
  )
}

// ─── Beiträge Tab ───────────────────────────────────────────────────────────────

interface BeitragsSatz {
  id: number
  kategorie: string
  betrag_cent: number
  valid_from: string
}

function BeitraegeTab() {
  const [saetze, setSaetze] = useState<BeitragsSatz[]>([])
  const [loaded, setLoaded] = useState(false)
  const [forms, setForms] = useState<Record<string, { datum: string; betrag: string }>>({})
  const [error, setError] = useState<string | null>(null)

  const load = () => {
    api.get('/fee-rates').then(r => {
      setSaetze(r.data.items ?? [])
      setLoaded(true)
    })
  }
  useEffect(() => { if (!loaded) load() }, [loaded])
  useLiveUpdates(event => { if (event === 'beitragssatz-changed') load() })

  const add = async (kategorie: string) => {
    setError(null)
    const f = forms[kategorie] ?? { datum: '', betrag: '' }
    const betrag = parseFloat((f.betrag || '').replace(',', '.'))
    if (!f.datum || isNaN(betrag) || betrag <= 0) {
      setError('Bitte gültiges Datum und einen Betrag > 0 angeben.')
      return
    }
    await api.post('/fee-rates', {
      kategorie,
      betrag_cent: Math.round(betrag * 100),
      valid_from: f.datum,
    })
    setForms({ ...forms, [kategorie]: { datum: '', betrag: '' } })
    load()
  }

  return (
    <div className="space-y-6 max-w-2xl">
      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</div>
      )}
      {BEITRAGS_KATEGORIEN.map(kat => {
        const rows = saetze
          .filter(s => s.kategorie === kat)
          .sort((a, b) => b.valid_from.slice(0, 10).localeCompare(a.valid_from.slice(0, 10)))
        const f = forms[kat] ?? { datum: '', betrag: '' }
        return (
          <div key={kat} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow px-5 py-4">
            <h3 className="text-sm font-semibold text-brand-text mb-3">{kategorieLabel(kat)}</h3>
            <table className="w-full text-sm mb-3">
              <thead>
                <tr className="text-brand-text-muted text-xs uppercase text-left">
                  <th className="py-1">Gültig ab</th>
                  <th className="py-1 text-right">Betrag</th>
                </tr>
              </thead>
              <tbody>
                {rows.length === 0 && (
                  <tr><td colSpan={2} className="py-2 text-brand-text-muted">Noch kein Satz hinterlegt.</td></tr>
                )}
                {rows.map(s => (
                  <tr key={s.id} className="border-t border-brand-border-subtle">
                    <td className="py-1.5 text-brand-text">{s.valid_from.slice(0, 10)}</td>
                    <td className="py-1.5 text-right text-brand-text">{(s.betrag_cent / 100).toFixed(2)} €</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="flex flex-wrap gap-2 items-end">
              <input
                type="date"
                value={f.datum}
                onChange={e => setForms({ ...forms, [kat]: { ...f, datum: e.target.value } })}
                className={`${INPUT} w-auto`}
              />
              <input
                type="text"
                inputMode="decimal"
                placeholder="Betrag in €"
                value={f.betrag}
                onChange={e => setForms({ ...forms, [kat]: { ...f, betrag: e.target.value } })}
                className={`${INPUT} w-32`}
              />
              <button type="button" onClick={() => add(kat)} className={BTN_SM}>Hinzufügen</button>
            </div>
          </div>
        )
      })}
    </div>
  )
}

// ─── Page ─────────────────────────────────────────────────────────────────────

type Tab = 'verein' | 'saisons' | 'altersklassen' | 'beitraege'
// Sichtbarkeit pro Tab über Capabilities (nie über role/clubFunctions direkt):
//   Kassierer      → manage_club + manage_fees      → Verein, Beiträge
//   Vorstand/Admin → zusätzlich manage_seasons      → alle vier Tabs
const TABS: { id: Tab; label: string; cap: string }[] = [
  { id: 'verein', label: 'Verein', cap: 'manage_club' },
  { id: 'saisons', label: 'Saisons', cap: 'manage_seasons' },
  { id: 'altersklassen', label: 'Altersklassen', cap: 'manage_seasons' },
  { id: 'beitraege', label: 'Beiträge', cap: 'manage_fees' },
]

export default function AdminSettingsPage() {
  const { hasCapability } = useAuth()
  const [searchParams, setSearchParams] = useSearchParams()
  const visibleTabs = TABS.filter(t => hasCapability(t.cap))
  const rawTab = searchParams.get('tab')
  const activeTab: Tab = visibleTabs.find(t => t.id === rawTab)?.id ?? visibleTabs[0]?.id ?? 'verein'

  const setTab = (id: Tab) => setSearchParams({ tab: id }, { replace: true })

  return (
    <div className="max-w-3xl">
      <h1 className="text-2xl font-bold text-brand-text mb-6">Einstellungen</h1>

      {/* Tab bar */}
      <div className="flex gap-1 border-b border-brand-border-subtle mb-6">
        {visibleTabs.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors -mb-px ${
              activeTab === t.id
                ? 'border-brand-yellow text-brand-text'
                : 'border-transparent text-brand-text-muted hover:text-brand-text'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {activeTab === 'verein' && <VereinTab />}
      {activeTab === 'saisons' && <SaisonsTab />}
      {activeTab === 'altersklassen' && <AltersklassenTab />}
      {activeTab === 'beitraege' && <BeitraegeTab />}
    </div>
  )
}
