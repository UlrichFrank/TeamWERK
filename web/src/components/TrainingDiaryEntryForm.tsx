import { useState } from 'react'
import RpeScaleInfo from './RpeScaleInfo'
import {
  DIARY_KINDS,
  RETENTION_HINT,
  type DiaryEntry,
  type DiaryKind,
} from '../lib/trainingDiary'
import { todayISO } from '../lib/trainingDiary'
import { BTN_PRIMARY } from '../lib/buttonStyles'

export interface DiarySubmitPayload {
  trained_on: string
  kind: DiaryKind
  kind_custom?: string
  duration_min: number
  rpe: number
  note?: string
}

// Formular zum Erfassen und Bearbeiten einer Einheit. Die Client-Validierung
// spiegelt die Server-Regeln (Dauer 1..600, RPE 1..10, kein Zukunftsdatum,
// Freitext bei 'sonstiges' Pflicht) — der Server bleibt die Autorität, das
// Formular erspart nur den Roundtrip.
export default function TrainingDiaryEntryForm({
  initial,
  onSubmit,
  onCancel,
  onFileSelect,
  busy,
}: {
  initial?: DiaryEntry
  onSubmit: (payload: DiarySubmitPayload) => void
  onCancel?: () => void
  onFileSelect?: (file: File | null) => void
  busy?: boolean
}) {
  const [trainedOn, setTrainedOn] = useState(initial?.trained_on.slice(0, 10) ?? todayISO())
  const [kind, setKind] = useState<DiaryKind>(initial?.kind ?? 'kraft')
  const [kindCustom, setKindCustom] = useState(initial?.kind_custom ?? '')
  const [durationMin, setDurationMin] = useState(String(initial?.duration_min ?? 45))
  const [rpe, setRpe] = useState(initial?.rpe ?? 5)
  const [note, setNote] = useState(initial?.note ?? '')
  const [error, setError] = useState('')

  const customRequired = kind === 'sonstiges'
  const customMissing = customRequired && kindCustom.trim() === ''
  const duration = Number(durationMin)
  const durationInvalid = !Number.isFinite(duration) || duration < 1 || duration > 600
  const canSubmit = !busy && !customMissing && !durationInvalid

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (customMissing) {
      setError('Bitte gib an, welche Art von Training es war.')
      return
    }
    if (durationInvalid) {
      setError('Die Dauer muss zwischen 1 und 600 Minuten liegen.')
      return
    }
    if (trainedOn > todayISO()) {
      setError('Das Datum darf nicht in der Zukunft liegen.')
      return
    }
    setError('')
    onSubmit({
      trained_on: trainedOn,
      kind,
      ...(customRequired ? { kind_custom: kindCustom.trim() } : {}),
      duration_min: duration,
      rpe,
      ...(note.trim() ? { note: note.trim() } : {}),
    })
  }

  const inputClass =
    'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {error && (
        <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
          {error}
        </div>
      )}

      <div>
        <label htmlFor="diary-date" className="block text-sm font-medium text-brand-text mb-1">
          Datum
        </label>
        <input
          id="diary-date"
          type="date"
          value={trainedOn}
          max={todayISO()}
          onChange={e => setTrainedOn(e.target.value)}
          className={inputClass}
        />
      </div>

      <div>
        <label htmlFor="diary-kind" className="block text-sm font-medium text-brand-text mb-1">
          Art
        </label>
        <select
          id="diary-kind"
          value={kind}
          onChange={e => setKind(e.target.value as DiaryKind)}
          className={inputClass}
        >
          {DIARY_KINDS.map(k => (
            <option key={k.value} value={k.value}>
              {k.label}
            </option>
          ))}
        </select>
        {customRequired && (
          <input
            type="text"
            value={kindCustom}
            maxLength={60}
            placeholder="Welche Art von Training?"
            aria-label="Art des Trainings"
            onChange={e => setKindCustom(e.target.value)}
            className={`${inputClass} mt-2`}
          />
        )}
      </div>

      <div>
        <label htmlFor="diary-duration" className="block text-sm font-medium text-brand-text mb-1">
          Dauer (Minuten)
        </label>
        <input
          id="diary-duration"
          type="number"
          inputMode="numeric"
          min={1}
          max={600}
          value={durationMin}
          onChange={e => setDurationMin(e.target.value)}
          className={inputClass}
        />
      </div>

      <div>
        <span className="block text-sm font-medium text-brand-text mb-1">Intensität (RPE)</span>
        <div className="flex flex-wrap gap-1">
          {Array.from({ length: 10 }, (_, i) => i + 1).map(value => (
            <button
              key={value}
              type="button"
              aria-pressed={rpe === value}
              onClick={() => setRpe(value)}
              className={`w-10 h-11 sm:h-9 rounded-md text-sm font-medium transition-colors ${
                rpe === value
                  ? 'bg-brand-yellow text-brand-black'
                  : 'bg-brand-surface-card text-brand-text border border-brand-border hover:bg-brand-table-select'
              }`}
            >
              {value}
            </button>
          ))}
        </div>
        <RpeScaleInfo />
      </div>

      <div>
        <label htmlFor="diary-note" className="block text-sm font-medium text-brand-text mb-1">
          Notiz <span className="text-brand-text-muted font-normal">(optional)</span>
        </label>
        <textarea
          id="diary-note"
          rows={2}
          maxLength={500}
          value={note}
          onChange={e => setNote(e.target.value)}
          className={inputClass}
        />
      </div>

      {onFileSelect && (
        <div>
          <label htmlFor="diary-proof" className="block text-sm font-medium text-brand-text mb-1">
            Nachweis <span className="text-brand-text-muted font-normal">(optional)</span>
          </label>
          <input
            id="diary-proof"
            type="file"
            accept="image/jpeg,image/png,image/webp,application/pdf"
            onChange={e => onFileSelect(e.target.files?.[0] ?? null)}
            className="w-full text-sm text-brand-text"
          />
          <p className="mt-1 text-xs text-brand-text-muted">
            Kannst du auch später nachreichen. {RETENTION_HINT}
          </p>
        </div>
      )}

      <div className="flex gap-2">
        <button
          type="submit"
          disabled={!canSubmit}
          className={BTN_PRIMARY}
        >
          {initial ? 'Speichern' : 'Eintragen'}
        </button>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium text-brand-text-muted hover:text-brand-text transition-colors"
          >
            Abbrechen
          </button>
        )}
      </div>
    </form>
  )
}
