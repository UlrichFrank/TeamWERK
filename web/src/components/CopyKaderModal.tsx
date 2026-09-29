import { useEffect, useId, useRef, useState } from 'react'
import { X } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useDialogA11y } from '../lib/useDialogA11y'
import { BTN_PRIMARY, BTN_SECONDARY, INPUT } from '../lib/buttonStyles'
import { GENDER_LABEL } from '../lib/teamName'
import { MODAL_TITLE } from '../lib/typography'

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

/**
 * Eine Zeile des Kopier-Dialogs: eine Kombination aus Altersklasse und
 * Geschlecht. Der Kopierer (`copyKader`) arbeitet pro Kombination und legt in
 * der Zielsaison genau eine Mannschaft an — zwei C-Jugenden der Quellsaison
 * sind deshalb eine Zeile, nicht zwei mit geteiltem Häkchen.
 */
interface SourceGroup {
  key: string
  age_class: string
  gender: string
  team_count: number
}

function groupSourceKader(kader: SourceKader[]): SourceGroup[] {
  const groups = new Map<string, SourceGroup>()
  for (const k of kader) {
    const key = `${k.age_class}|${k.gender}`
    const g = groups.get(key)
    if (g) g.team_count++
    else groups.set(key, { key, age_class: k.age_class, gender: k.gender, team_count: 1 })
  }
  return [...groups.values()]
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

export default function CopyKaderModal({ toSeasonId, toSeasonName, onDone, onClose }: Props) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  useEscapeKey(onClose)
  useDialogA11y(dialogRef, true)
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
    const res = await api.get('/kader', { params: { season_id: seasonId, limit: 200 } })
    const kader: SourceKader[] = res.data?.items ?? []
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
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-brand-black/40 p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="bg-white rounded-xl shadow-2xl border-t-4 border-brand-yellow transform-gpu w-full max-w-lg max-h-[90vh] overflow-y-auto"
      >
        <div className="px-6 py-4 border-b border-brand-border-subtle flex items-center justify-between">
          <h2 id={titleId} className={MODAL_TITLE}>Kader kopieren → {toSeasonName}</h2>
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
                  className={INPUT}
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
                  className={BTN_SECONDARY}
                >
                  Abbrechen
                </button>
                <button
                  onClick={() => setStep(2)}
                  disabled={!fromSeasonId || sourceKader.length === 0}
                  className={BTN_PRIMARY}
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
                {groupSourceKader(sourceKader).map(g => {
                  const key = g.key
                  return (
                    <div key={key} className="border border-brand-border-subtle rounded-lg p-3 space-y-2">
                      <label className="flex items-center gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={selectedKader.has(key)}
                          onChange={() => toggleKader(key)}
                          className="accent-brand-yellow"
                        />
                        <span className="font-medium text-sm text-brand-text">{g.age_class} {GENDER_LABEL[g.gender] ?? g.gender}</span>
                        {g.team_count > 1 && (
                          <span className="text-xs text-brand-text-muted">
                            {g.team_count} Mannschaften in der Quellsaison – es wird eine angelegt
                          </span>
                        )}
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
                  className={BTN_SECONDARY}
                >
                  Zurück
                </button>
                <button
                  onClick={handleConfirm}
                  disabled={selectedKader.size === 0 || saving}
                  className={BTN_PRIMARY}
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
