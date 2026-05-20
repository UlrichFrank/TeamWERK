import { useEffect, useState } from 'react'
import { api } from '../lib/api'

interface Kader {
  id: number
  age_class: string
  gender: string
  bracket_years: number[]
}

interface Props {
  seasonId: number
  onDone: () => void
  onClose: () => void
}

const GENDER_LABEL: Record<string, string> = { m: 'männlich', f: 'weiblich', mixed: 'gemischt' }

export default function AutoAssignModal({ seasonId, onDone, onClose }: Props) {
  const [kader, setKader] = useState<Kader[]>([])
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setLoading(true)
    api.get('/admin/kader', { params: { season_id: seasonId } })
      .then(r => {
        const kadersData: Kader[] = r.data ?? []
        setKader(kadersData)
        setSelectedIds(new Set(kadersData.map(k => k.id)))
        setLoading(false)
      })
      .catch(() => {
        setError('Fehler beim Laden der Kader.')
        setLoading(false)
      })
  }, [seasonId])

  const toggleKader = (id: number) => {
    setSelectedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleConfirm = async () => {
    setSaving(true)
    setError(null)
    try {
      await api.post('/admin/kader/auto-assign', {
        kader_ids: Array.from(selectedIds),
      })
      // Show success message and reload
      const message = `Auto-Assign für ${selectedIds.size} Kader abgeschlossen`
      // Use a simple alert for now; in production, use a toast library
      alert(message)
      onDone()
    } catch (err) {
      setError('Fehler beim Auto-Assign. Bitte erneut versuchen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="px-6 py-4 border-b flex items-center justify-between">
          <h2 className="font-semibold text-base">Auto-Assign</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-700 text-xl leading-none">×</button>
        </div>

        <div className="p-6">
          {loading ? (
            <p className="text-sm text-gray-500 italic">Kader werden geladen…</p>
          ) : kader.length === 0 ? (
            <p className="text-sm text-gray-400 italic">Keine Kader in dieser Saison vorhanden.</p>
          ) : (
            <>
              <p className="text-sm text-gray-600 mb-4">
                Wähle die Kader, die mit Mitgliedern nach Jahrgang und Geschlecht befüllt werden sollen:
              </p>
              <div className="space-y-2 mb-6">
                {kader.map(k => (
                  <label key={k.id} className="flex items-center gap-3 text-sm p-2 rounded hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selectedIds.has(k.id)}
                      onChange={() => toggleKader(k.id)}
                      className="accent-brand-blue"
                    />
                    <span className="font-medium">{k.age_class} {GENDER_LABEL[k.gender]}</span>
                    {k.bracket_years.length === 2 && (
                      <span className="text-gray-400 text-xs">Jg. {k.bracket_years[0]}/{k.bracket_years[1]}</span>
                    )}
                  </label>
                ))}
              </div>
              {error && <p className="text-brand-error text-sm mb-4">{error}</p>}
            </>
          )}
          <div className="flex justify-end gap-2">
            <button onClick={onClose} className="text-sm text-gray-500 hover:text-gray-700 px-4 py-2">Abbrechen</button>
            {!loading && kader.length > 0 && (
              <button
                onClick={handleConfirm}
                disabled={selectedIds.size === 0 || saving}
                className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
              >
                {saving ? 'Auto-Assign läuft…' : 'Auto-Assign starten'}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
