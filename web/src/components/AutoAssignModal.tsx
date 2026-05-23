import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
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
      alert(`Auto-Assign für ${selectedIds.size} Kader abgeschlossen`)
      onDone()
    } catch {
      setError('Fehler beim Auto-Assign. Bitte erneut versuchen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-2xl border-t-4 border-brand-yellow w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="px-6 py-4 border-b border-brand-border-subtle flex items-center justify-between">
          <h2 className="font-semibold text-base text-brand-text">Auto-Assign</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6">
          {loading ? (
            <p className="text-sm text-brand-text-muted italic">Kader werden geladen…</p>
          ) : kader.length === 0 ? (
            <p className="text-sm text-brand-text-subtle italic">Keine Kader in dieser Saison vorhanden.</p>
          ) : (
            <>
              <p className="text-sm text-brand-text-muted mb-4">
                Wähle die Kader, die mit Mitgliedern nach Jahrgang und Geschlecht befüllt werden sollen:
              </p>
              <div className="space-y-2 mb-6">
                {kader.map(k => (
                  <label key={k.id} className="flex items-center gap-3 text-sm p-2 rounded hover:bg-brand-surface-card cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selectedIds.has(k.id)}
                      onChange={() => toggleKader(k.id)}
                      className="accent-brand-yellow"
                    />
                    <span className="font-medium text-brand-text">{k.age_class} {GENDER_LABEL[k.gender]}</span>
                    {k.bracket_years.length === 2 && (
                      <span className="text-brand-text-subtle text-xs">Jg. {k.bracket_years[0]}/{k.bracket_years[1]}</span>
                    )}
                  </label>
                ))}
              </div>
              {error && (
                <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger mb-4">
                  {error}
                </p>
              )}
            </>
          )}
          <div className="flex justify-end gap-2">
            <button
              onClick={onClose}
              className="px-4 py-2.5 sm:py-2 border border-brand-border rounded-md text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
            >
              Abbrechen
            </button>
            {!loading && kader.length > 0 && (
              <button
                onClick={handleConfirm}
                disabled={selectedIds.size === 0 || saving}
                className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
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
