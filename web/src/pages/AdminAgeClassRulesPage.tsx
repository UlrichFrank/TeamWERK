import { useEffect, useState } from 'react'
import { api } from '../lib/api'

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

export default function AdminAgeClassRulesPage() {
  const [rules, setRules] = useState<AgeClassRule[]>([])
  const [rowStates, setRowStates] = useState<Record<string, RowState>>({})
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.get<AgeClassRule[]>('/admin/age-class-rules').then(r => {
      const data: AgeClassRule[] = Array.isArray(r.data) ? r.data : []
      setRules(data)
      const initial: Record<string, RowState> = {}
      for (const rule of data) {
        initial[rule.age_class] = {
          half: String(rule.half_duration_minutes),
          brk: String(rule.break_minutes),
          saving: false,
          error: '',
          success: false,
        }
      }
      setRowStates(initial)
      setLoading(false)
    })
  }, [])

  function updateRow(ageClass: string, field: 'half' | 'brk', value: string) {
    setRowStates(prev => ({
      ...prev,
      [ageClass]: { ...prev[ageClass], [field]: value, error: '', success: false },
    }))
  }

  async function saveRow(ageClass: string) {
    const s = rowStates[ageClass]
    const half = parseInt(s.half)
    const brk = parseInt(s.brk)
    if (!half || half <= 0 || !brk || brk <= 0) {
      setRowStates(prev => ({
        ...prev,
        [ageClass]: { ...prev[ageClass], error: 'Werte müssen größer als 0 sein.' },
      }))
      return
    }
    setRowStates(prev => ({
      ...prev,
      [ageClass]: { ...prev[ageClass], saving: true, error: '' },
    }))
    try {
      await api.put(`/admin/age-class-rules/${ageClass}`, {
        half_duration_minutes: half,
        break_minutes: brk,
      })
      setRowStates(prev => ({
        ...prev,
        [ageClass]: { ...prev[ageClass], saving: false, success: true },
      }))
    } catch {
      setRowStates(prev => ({
        ...prev,
        [ageClass]: { ...prev[ageClass], saving: false, error: 'Speichern fehlgeschlagen.' },
      }))
    }
  }

  return (
    <div className="p-8 sm:p-8 px-4 py-4 max-w-2xl">
      <h1 className="text-2xl font-bold text-brand-text mb-6">Altersklassen-Regeln</h1>

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <table className="w-full">
          <thead>
            <tr>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Klasse</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Halbzeit (min)</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Pause (min)</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Gesamt</th>
              <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-right"></th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-brand-text-muted text-sm">
                  Laden…
                </td>
              </tr>
            ) : (
              rules.map(rule => {
                const s = rowStates[rule.age_class]
                if (!s) return null
                const half = parseInt(s.half) || 0
                const brk = parseInt(s.brk) || 0
                const total = half > 0 && brk > 0 ? 2 * half + brk : '—'
                return (
                  <tr key={rule.age_class} className="border-t border-brand-border-subtle">
                    <td className="px-4 py-3 text-sm font-semibold text-brand-text">
                      {rule.age_class}-Jugend
                    </td>
                    <td className="px-4 py-3">
                      <input
                        type="number"
                        min={1}
                        value={s.half}
                        onChange={e => updateRow(rule.age_class, 'half', e.target.value)}
                        className="w-20 border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                      />
                    </td>
                    <td className="px-4 py-3">
                      <input
                        type="number"
                        min={1}
                        value={s.brk}
                        onChange={e => updateRow(rule.age_class, 'brk', e.target.value)}
                        className="w-20 border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                      />
                    </td>
                    <td className="px-4 py-3 text-sm text-brand-text-muted">
                      {total !== '—' ? `${total} min` : '—'}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex flex-col items-end gap-1">
                        <button
                          onClick={() => saveRow(rule.age_class)}
                          disabled={s.saving}
                          className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                        >
                          {s.saving ? 'Speichern…' : 'Speichern'}
                        </button>
                        {s.error && (
                          <span className="text-xs text-brand-danger">{s.error}</span>
                        )}
                        {s.success && !s.error && (
                          <span className="text-xs text-green-600">Gespeichert</span>
                        )}
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
        Gesamt-Spieldauer = 2 × Halbzeit + Pause. Wird für Heim- und Auswärtsspiele zur Slot-Zeitberechnung verwendet.
      </p>
    </div>
  )
}
