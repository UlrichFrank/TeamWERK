import { useEffect, useState } from 'react'
import { api } from '../lib/api'

interface Season {
  id: number
  name: string
  is_active: boolean
}

interface SourceKader {
  id: number
  age_class: string
  gender: string
  member_count: number
}

interface Assignment {
  age_class: string
  gender: string
  member_source: string
}

interface Props {
  toSeasonId: number
  toSeasonName: string
  onDone: () => void
  onClose: () => void
}

const GENDER_LABEL: Record<string, string> = { m: 'männlich', f: 'weiblich', mixed: 'gemischt' }

export default function CopyKaderModal({ toSeasonId, toSeasonName, onDone, onClose }: Props) {
  const [step, setStep] = useState(1)
  const [seasons, setSeasons] = useState<Season[]>([])
  const [fromSeasonId, setFromSeasonId] = useState<number | ''>('')
  const [sourceKader, setSourceKader] = useState<SourceKader[]>([])
  const [selectedKader, setSelectedKader] = useState<Set<string>>(new Set())
  const [emptyOnly, setEmptyOnly] = useState<Set<string>>(new Set()) // "A-Jugend|m" → only structure
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.get('/admin/seasons').then(r => {
      setSeasons((r.data ?? []).filter((s: Season) => !s.is_active))
    })
  }, [])

  const handleSelectSeason = async (seasonId: number) => {
    setFromSeasonId(seasonId)
    const res = await api.get('/admin/kader', { params: { season_id: seasonId } })
    const kader: SourceKader[] = res.data ?? []
    setSourceKader(kader)
    const keys = new Set(kader.map(k => `${k.age_class}|${k.gender}`))
    setSelectedKader(keys)
    setEmptyOnly(new Set())
  }

  const toggleKader = (key: string) => {
    setSelectedKader(prev => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const toggleEmptyOnly = (key: string) => {
    setEmptyOnly(prev => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  const handleConfirm = async () => {
    setSaving(true)
    setError(null)
    const assignmentList: Assignment[] = Array.from(selectedKader).map(key => {
      const [ageClass, gender] = key.split('|')
      const memberSource = emptyOnly.has(key) ? 'empty' : 'smart-copy'
      return { age_class: ageClass, gender, member_source: memberSource }
    })
    try {
      await api.post('/admin/kader/copy-from-season', {
        from_season_id: fromSeasonId,
        to_season_id: toSeasonId,
        assignments: assignmentList,
      })
      onDone()
    } catch {
      setError('Fehler beim Kopieren. Bitte erneut versuchen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="px-6 py-4 border-b flex items-center justify-between">
          <h2 className="font-semibold text-base">Kader kopieren → {toSeasonName}</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-700 text-xl leading-none">×</button>
        </div>

        <div className="p-6">
          {/* Step indicator */}
          <div className="flex gap-2 mb-6 text-xs">
            {[1, 2].map(s => (
              <div key={s} className={`flex-1 h-1 rounded-full ${s <= step ? 'bg-brand-yellow' : 'bg-gray-200'}`} />
            ))}
          </div>

          {/* Step 1: Season selection */}
          {step === 1 && (
            <div className="space-y-4">
              <p className="text-sm text-gray-600">Aus welcher Saison sollen die Kader kopiert werden?</p>
              {seasons.length === 0 ? (
                <p className="text-sm text-gray-400 italic">Keine anderen Saisons vorhanden.</p>
              ) : (
                <select
                  value={fromSeasonId}
                  onChange={e => handleSelectSeason(Number(e.target.value))}
                  className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-blue"
                >
                  <option value="">Saison wählen…</option>
                  {seasons.map(s => (
                    <option key={s.id} value={s.id}>{s.name}</option>
                  ))}
                </select>
              )}
              <div className="flex justify-end gap-2">
                <button onClick={onClose} className="text-sm text-gray-500 hover:text-gray-700 px-4 py-2">Abbrechen</button>
                <button
                  onClick={() => setStep(2)}
                  disabled={!fromSeasonId || sourceKader.length === 0}
                  className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
                >
                  Weiter
                </button>
              </div>
            </div>
          )}

          {/* Step 2: Select which Kader to copy and set options */}
          {step === 2 && (
            <div className="space-y-4">
              <p className="text-sm text-gray-600">Welche Kader sollen kopiert werden?</p>
              <div className="space-y-3">
                {sourceKader.map(k => {
                  const key = `${k.age_class}|${k.gender}`
                  return (
                    <div key={key} className="border rounded-lg p-3 space-y-2">
                      <label className="flex items-center gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={selectedKader.has(key)}
                          onChange={() => toggleKader(key)}
                          className="accent-brand-blue"
                        />
                        <span className="font-medium text-sm">{k.age_class} {GENDER_LABEL[k.gender]}</span>
                      </label>
                      {selectedKader.has(key) && (
                        <label className="flex items-center gap-3 cursor-pointer ml-6 text-sm">
                          <input
                            type="checkbox"
                            checked={emptyOnly.has(key)}
                            onChange={() => toggleEmptyOnly(key)}
                            className="accent-brand-blue"
                          />
                          <span className="text-gray-600">Nur Struktur (keine Mitglieder)</span>
                        </label>
                      )}
                    </div>
                  )
                })}
              </div>
              {error && <p className="text-brand-error text-sm">{error}</p>}
              <div className="flex justify-between gap-2">
                <button onClick={() => setStep(1)} className="text-sm text-gray-500 hover:text-gray-700 px-4 py-2">Zurück</button>
                <button
                  onClick={handleConfirm}
                  disabled={selectedKader.size === 0 || saving}
                  className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
                >
                  {saving ? 'Anlegen…' : 'Kader anlegen'}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
