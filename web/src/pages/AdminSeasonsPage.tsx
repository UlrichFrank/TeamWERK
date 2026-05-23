import { useEffect, useState } from 'react'
import { api } from '../lib/api'

interface Season {
  id: number
  name: string
  start_date: string
  end_date: string
  is_active: boolean
}

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

export default function AdminSeasonsPage() {
  const [seasons, setSeasons] = useState<Season[]>([])
  const [loading, setLoading] = useState(true)
  const [seasonPreset, setSeasonPreset] = useState('')
  const [name, setName] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = () => api.get('/admin/seasons').then(r => setSeasons(r.data ?? []))

  useEffect(() => { load().finally(() => setLoading(false)) }, [])

  const generateSeasonOptions = () => {
    const now = new Date()
    const currentYear = now.getFullYear()
    const startYear = now.getMonth() < 6 ? currentYear - 1 : currentYear
    return [0, 1, 2].map(offset => {
      const year = startYear + offset
      return { year, label: `${year}/${String(year + 1).slice(-2)}` }
    })
  }

  const handleSeasonPresetChange = (label: string) => {
    setSeasonPreset(label)
    if (label) {
      const match = label.match(/(\d{4})\//)
      if (match) {
        const year = parseInt(match[1])
        setName(label)
        setStartDate(`${year}-07-01`)
        setEndDate(`${year + 1}-06-30`)
      }
    }
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name || !startDate || !endDate) return
    setSaving(true)
    setError(null)
    try {
      await api.post('/admin/seasons', { name, start_date: startDate, end_date: endDate })
      setSeasonPreset('')
      setName('')
      setStartDate('')
      setEndDate('')
      await load()
    } catch {
      setError('Saison konnte nicht angelegt werden.')
    } finally {
      setSaving(false)
    }
  }

  const handleActivate = async (id: number) => {
    await api.put(`/admin/seasons/${id}/activate`, {})
    await load()
  }

  const handleDelete = async (id: number) => {
    if (!confirm('Saison wirklich löschen?')) return
    setDeleting(id)
    try {
      await api.delete(`/admin/seasons/${id}`)
      await load()
    } catch {
      setError('Saison konnte nicht gelöscht werden.')
    } finally {
      setDeleting(null)
    }
  }

  if (loading) return <div className="text-brand-text-muted text-sm">Laden…</div>

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold mb-6">Saisons</h1>

      {/* Existing seasons */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-8">
        <div className="px-5 py-3 border-b border-brand-border-subtle">
          <h2 className="font-semibold text-brand-text">Vorhandene Saisons</h2>
        </div>
        {seasons.length === 0 ? (
          <p className="text-sm text-brand-text-subtle text-center py-8 italic">Noch keine Saisons angelegt.</p>
        ) : (
          <ul className="divide-y divide-brand-border-subtle">
            {seasons.map(s => (
              <li key={s.id} className="flex items-center justify-between px-5 py-3 gap-3">
                <div className="flex-1">
                  <span className="font-medium text-sm text-brand-text">{s.name}</span>
                  <span className="text-xs text-brand-text-subtle ml-3">{s.start_date.slice(0, 10)} – {s.end_date.slice(0, 10)}</span>
                  {s.is_active && (
                    <span className="ml-2 text-xs bg-brand-success-light text-brand-success px-2 py-0.5 rounded-full font-medium">aktiv</span>
                  )}
                </div>
                <div className="flex gap-2">
                  {!s.is_active && (
                    <>
                      <button
                        onClick={() => handleActivate(s.id)}
                        className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
                      >
                        Aktivieren
                      </button>
                      <button
                        onClick={() => handleDelete(s.id)}
                        disabled={deleting === s.id}
                        className="bg-brand-danger text-white rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                      >
                        {deleting === s.id ? 'Löschen…' : 'Löschen'}
                      </button>
                    </>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Create new season */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-5">
        <h2 className="font-semibold mb-4 text-brand-text">Neue Saison anlegen</h2>
        <form onSubmit={handleCreate} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Saison</label>
            <select
              value={seasonPreset}
              onChange={e => handleSeasonPresetChange(e.target.value)}
              className={INPUT}
              required
            >
              <option value="">Wählen…</option>
              {generateSeasonOptions().map(opt => (
                <option key={opt.year} value={opt.label}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
          <div className="flex gap-4">
            <div className="flex-1">
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Startdatum</label>
              <input type="date" value={startDate} onChange={e => setStartDate(e.target.value)} className={INPUT} required />
            </div>
            <div className="flex-1">
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Enddatum</label>
              <input type="date" value={endDate} onChange={e => setEndDate(e.target.value)} className={INPUT} required />
            </div>
          </div>
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
          <button
            type="submit"
            disabled={saving}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {saving ? 'Speichern…' : 'Saison anlegen'}
          </button>
        </form>
      </div>
    </div>
  )
}
