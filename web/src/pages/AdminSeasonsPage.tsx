import { useEffect, useState } from 'react'
import { api } from '../lib/api'

interface Season {
  id: number
  name: string
  start_date: string
  end_date: string
  is_active: boolean
}

export default function AdminSeasonsPage() {
  const [seasons, setSeasons] = useState<Season[]>([])
  const [loading, setLoading] = useState(true)
  const [name, setName] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = () => api.get('/admin/seasons').then(r => setSeasons(r.data ?? []))

  useEffect(() => { load().finally(() => setLoading(false)) }, [])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name || !startDate || !endDate) return
    setSaving(true)
    setError(null)
    try {
      await api.post('/admin/seasons', { name, start_date: startDate, end_date: endDate })
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

  if (loading) return <div className="text-gray-400 text-sm">Laden…</div>

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold mb-6">Saisons</h1>

      {/* Existing seasons */}
      <div className="bg-white rounded-xl shadow overflow-hidden mb-8">
        <div className="px-5 py-3 border-b">
          <h2 className="font-semibold">Vorhandene Saisons</h2>
        </div>
        {seasons.length === 0 ? (
          <p className="text-sm text-gray-400 text-center py-8 italic">Noch keine Saisons angelegt.</p>
        ) : (
          <ul className="divide-y">
            {seasons.map(s => (
              <li key={s.id} className="flex items-center justify-between px-5 py-3">
                <div>
                  <span className="font-medium text-sm">{s.name}</span>
                  <span className="text-xs text-gray-400 ml-3">{s.start_date.slice(0, 10)} – {s.end_date.slice(0, 10)}</span>
                  {s.is_active && (
                    <span className="ml-2 text-xs bg-green-100 text-green-700 px-2 py-0.5 rounded-full font-medium">aktiv</span>
                  )}
                </div>
                {!s.is_active && (
                  <button
                    onClick={() => handleActivate(s.id)}
                    className="text-xs bg-[#3E4A98] text-white px-3 py-1.5 rounded-md hover:bg-[#2e3a7a] transition-colors"
                  >
                    Aktivieren
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Create new season */}
      <div className="bg-white rounded-xl shadow p-5">
        <h2 className="font-semibold mb-4">Neue Saison anlegen</h2>
        <form onSubmit={handleCreate} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="z.B. Saison 2025/26"
              className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[#3E4A98]"
              required
            />
          </div>
          <div className="flex gap-4">
            <div className="flex-1">
              <label className="block text-sm font-medium mb-1">Startdatum</label>
              <input
                type="date"
                value={startDate}
                onChange={e => setStartDate(e.target.value)}
                className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[#3E4A98]"
                required
              />
            </div>
            <div className="flex-1">
              <label className="block text-sm font-medium mb-1">Enddatum</label>
              <input
                type="date"
                value={endDate}
                onChange={e => setEndDate(e.target.value)}
                className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[#3E4A98]"
                required
              />
            </div>
          </div>
          {error && <p className="text-red-600 text-sm">{error}</p>}
          <button
            type="submit"
            disabled={saving}
            className="bg-[#FAE806] text-black px-6 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-[#FAE806] transition-colors disabled:opacity-50"
          >
            {saving ? 'Speichern…' : 'Saison anlegen'}
          </button>
        </form>
      </div>
    </div>
  )
}
