import { useEffect, useState } from 'react'
import { Plus, Trash2, X, AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Absence {
  id: number
  member_id: number
  member_name: string
  type: 'vacation' | 'injury'
  start_date: string
  end_date: string
  note: string
}

interface PreviewEvent {
  event_type: 'training' | 'game'
  event_id: number
  name: string
  date: string
}

interface ChildMember {
  id: number
  name: string
}

const TYPE_LABELS: Record<string, string> = {
  vacation: 'Urlaub',
  injury: 'Verletzung / Sportverbot',
}

export default function AbsenzenPage() {
  const { user } = useAuth()
  const [absences, setAbsences] = useState<Absence[]>([])
  const [children, setChildren] = useState<ChildMember[]>([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [form, setForm] = useState({
    member_id: 0,
    type: 'vacation',
    start_date: '',
    end_date: '',
    note: '',
  })
  const [preview, setPreview] = useState<PreviewEvent[] | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const load = () => {
    api.get('/absences').then(r => {
      setAbsences(r.data ?? [])
      setLoading(false)
    })
  }

  useEffect(() => {
    load()
    if (user?.role === 'elternteil') {
      api.get('/profile/me').then(r => {
        const kinder = r.data?.kinder ?? []
        setChildren(kinder.map((k: { id: number; first_name: string; last_name: string }) => ({
          id: k.id,
          name: `${k.first_name} ${k.last_name}`,
        })))
      })
    }
  }, [user])

  const handleSubmitPreview = async () => {
    setError('')
    if (!form.start_date || !form.end_date) {
      setError('Bitte Start- und Enddatum angeben.')
      return
    }
    if (form.start_date > form.end_date) {
      setError('Startdatum muss vor dem Enddatum liegen.')
      return
    }
    if (user?.role === 'elternteil' && !form.member_id) {
      setError('Bitte ein Kind auswählen.')
      return
    }
    setPreviewLoading(true)
    try {
      const params = new URLSearchParams({
        from: form.start_date,
        to: form.end_date,
        ...(form.member_id ? { member_id: String(form.member_id) } : {}),
      })
      const r = await api.get(`/absences/preview?${params}`)
      const events: PreviewEvent[] = r.data ?? []
      if (events.length === 0) {
        await doSave()
      } else {
        setPreview(events)
      }
    } catch {
      setError('Fehler beim Laden der Vorschau.')
    } finally {
      setPreviewLoading(false)
    }
  }

  const doSave = async () => {
    setSaving(true)
    setError('')
    try {
      const body: Record<string, unknown> = {
        type: form.type,
        start_date: form.start_date,
        end_date: form.end_date,
        note: form.note,
      }
      if (user?.role === 'elternteil' && form.member_id) {
        body.member_id = form.member_id
      }
      await api.post('/absences', body)
      setShowForm(false)
      setPreview(null)
      setForm({ member_id: 0, type: 'vacation', start_date: '', end_date: '', note: '' })
      load()
    } catch {
      setError('Fehler beim Speichern.')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: number) => {
    await api.delete(`/absences/${id}`)
    load()
  }

  return (
    <div className="p-4 sm:p-8 max-w-2xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-brand-text">Abwesenheiten</h1>
        <button
          onClick={() => { setShowForm(true); setError('') }}
          className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
        >
          <Plus className="w-4 h-4 inline mr-1" />
          Neu
        </button>
      </div>

      {loading ? (
        <p className="text-brand-text-muted text-sm">Lädt…</p>
      ) : absences.length === 0 ? (
        <p className="text-brand-text-muted text-sm">Keine Abwesenheiten eingetragen.</p>
      ) : (
        <div className="space-y-3">
          {absences.map(a => (
            <div key={a.id} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4 flex items-start justify-between gap-4">
              <div>
                {user?.role === 'elternteil' && (
                  <p className="text-xs text-brand-text-muted mb-0.5">{a.member_name}</p>
                )}
                <p className="text-sm font-medium text-brand-text">{TYPE_LABELS[a.type]}</p>
                <p className="text-sm text-brand-text-muted">
                  {a.start_date} – {a.end_date}
                </p>
                {a.note && <p className="text-xs text-brand-text-subtle mt-1">{a.note}</p>}
              </div>
              <button
                onClick={() => handleDelete(a.id)}
                className="text-brand-danger hover:text-brand-danger/70 transition-colors shrink-0 mt-0.5"
                aria-label="Abwesenheit löschen"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* New absence form modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-brand-text">Abwesenheit eintragen</h2>
              <button onClick={() => { setShowForm(false); setError('') }} aria-label="Schließen">
                <X className="w-5 h-5 text-brand-text-muted" />
              </button>
            </div>

            <div className="space-y-4">
              {user?.role === 'elternteil' && children.length > 0 && (
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Kind</label>
                  <select
                    value={form.member_id}
                    onChange={e => setForm(f => ({ ...f, member_id: Number(e.target.value) }))}
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  >
                    <option value={0}>Bitte wählen…</option>
                    {children.map(c => (
                      <option key={c.id} value={c.id}>{c.name}</option>
                    ))}
                  </select>
                </div>
              )}

              <div>
                <label className="block text-xs font-medium text-brand-text-muted mb-1">Typ</label>
                <select
                  value={form.type}
                  onChange={e => setForm(f => ({ ...f, type: e.target.value }))}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                >
                  <option value="vacation">Urlaub</option>
                  <option value="injury">Verletzung / Sportverbot</option>
                </select>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Von</label>
                  <input
                    type="date"
                    value={form.start_date}
                    onChange={e => setForm(f => ({ ...f, start_date: e.target.value }))}
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-brand-text-muted mb-1">Bis</label>
                  <input
                    type="date"
                    value={form.end_date}
                    onChange={e => setForm(f => ({ ...f, end_date: e.target.value }))}
                    className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-brand-text-muted mb-1">Notiz (optional)</label>
                <input
                  type="text"
                  value={form.note}
                  onChange={e => setForm(f => ({ ...f, note: e.target.value }))}
                  placeholder="z.B. Familienurlaub, Knieoperation…"
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>

              {error && (
                <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                  {error}
                </p>
              )}
            </div>

            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => { setShowForm(false); setError('') }}
                className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={handleSubmitPreview}
                disabled={previewLoading}
                className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {previewLoading ? 'Prüfe…' : 'Weiter'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Confirmation modal */}
      {preview && (
        <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md">
            <div className="flex items-start gap-3 mb-4">
              <AlertTriangle className="w-5 h-5 text-brand-danger shrink-0 mt-0.5" />
              <div>
                <h2 className="text-base font-semibold text-brand-text">Bestehende Zusagen werden zurückgezogen</h2>
                <p className="text-sm text-brand-text-muted mt-1">
                  Folgende Events werden automatisch abgesagt:
                </p>
              </div>
            </div>

            <ul className="space-y-1.5 mb-5 max-h-48 overflow-y-auto">
              {preview.map(ev => (
                <li key={`${ev.event_type}-${ev.event_id}`} className="flex items-center gap-2 text-sm text-brand-text">
                  <span className="text-brand-text-muted w-16 shrink-0">{ev.date}</span>
                  <span>{ev.name}</span>
                  <span className="ml-auto text-xs text-brand-text-subtle">
                    {ev.event_type === 'training' ? 'Training' : 'Spiel'}
                  </span>
                </li>
              ))}
            </ul>

            <div className="flex justify-end gap-3">
              <button
                onClick={() => setPreview(null)}
                className="px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text transition-colors"
              >
                Zurück
              </button>
              <button
                onClick={doSave}
                disabled={saving}
                className="bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {saving ? 'Speichert…' : 'Trotzdem eintragen'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
