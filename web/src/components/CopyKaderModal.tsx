import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'

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
  useEscapeKey(onClose)
  const [step, setStep] = useState(1)
  const [seasons, setSeasons] = useState<Season[]>([])
  const [fromSeasonId, setFromSeasonId] = useState<number | ''>('')
  const [sourceKader, setSourceKader] = useState<SourceKader[]>([])
  const [selectedKader, setSelectedKader] = useState<Set<string>>(new Set())
  const [emptyOnly, setEmptyOnly] = useState<Set<string>>(new Set())
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.get('/seasons').then(r => {
      setSeasons((r.data ?? []).filter((s: Season) => !s.is_active))
    })
  }, [])

  const handleSelectSeason = async (seasonId: number) => {
    setFromSeasonId(seasonId)
    const res = await api.get('/kader', { params: { season_id: seasonId } })
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
      await api.post('/kader/copy-from-season', {
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
      <div className="bg-white rounded-xl shadow-2xl border-t-4 border-brand-yellow w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div className="px-6 py-4 border-b border-brand-border-subtle flex items-center justify-between">
          <h2 className="font-semibold text-base text-brand-text">Kader kopieren → {toSeasonName}</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6">
          {/* Step indicator */}
          <div className="flex gap-2 mb-6 text-xs">
            {[1, 2].map(s => (
              <div key={s} className={`flex-1 h-1 rounded-full ${s <= step ? 'bg-brand-yellow' : 'bg-brand-border-subtle'}`} />
            ))}
          </div>

          {/* Step 1: Season selection */}
          {step === 1 && (
            <div className="space-y-4">
              <p className="text-sm text-brand-text-muted">Aus welcher Saison sollen die Kader kopiert werden?</p>
              {seasons.length === 0 ? (
                <p className="text-sm text-brand-text-subtle italic">Keine anderen Saisons vorhanden.</p>
              ) : (
                <select
                  value={fromSeasonId}
                  onChange={e => handleSelectSeason(Number(e.target.value))}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                >
                  <option value="">Saison wählen…</option>
                  {seasons.map(s => (
                    <option key={s.id} value={s.id}>{s.name}</option>
                  ))}
                </select>
              )}
              <div className="flex justify-end gap-2">
                <button
                  onClick={onClose}
                  className="px-4 py-2.5 sm:py-2 border border-brand-border rounded-md text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
                >
                  Abbrechen
                </button>
                <button
                  onClick={() => setStep(2)}
                  disabled={!fromSeasonId || sourceKader.length === 0}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Weiter
                </button>
              </div>
            </div>
          )}

          {/* Step 2: Select which Kader to copy and set options */}
          {step === 2 && (
            <div className="space-y-4">
              <p className="text-sm text-brand-text-muted">Welche Kader sollen kopiert werden?</p>
              <div className="space-y-3">
                {sourceKader.map(k => {
                  const key = `${k.age_class}|${k.gender}`
                  return (
                    <div key={key} className="border border-brand-border-subtle rounded-lg p-3 space-y-2">
                      <label className="flex items-center gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={selectedKader.has(key)}
                          onChange={() => toggleKader(key)}
                          className="accent-brand-yellow"
                        />
                        <span className="font-medium text-sm text-brand-text">{k.age_class} {GENDER_LABEL[k.gender]}</span>
                      </label>
                      {selectedKader.has(key) && (
                        <label className="flex items-center gap-3 cursor-pointer ml-6 text-sm">
                          <input
                            type="checkbox"
                            checked={emptyOnly.has(key)}
                            onChange={() => toggleEmptyOnly(key)}
                            className="accent-brand-yellow"
                          />
                          <span className="text-brand-text-muted">Nur Struktur (keine Mitglieder)</span>
                        </label>
                      )}
                    </div>
                  )
                })}
              </div>
              {error && (
                <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                  {error}
                </p>
              )}
              <div className="flex justify-between gap-2">
                <button
                  onClick={() => setStep(1)}
                  className="px-4 py-2.5 sm:py-2 border border-brand-border rounded-md text-sm text-brand-text hover:bg-brand-surface-card transition-colors"
                >
                  Zurück
                </button>
                <button
                  onClick={handleConfirm}
                  disabled={selectedKader.size === 0 || saving}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
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
