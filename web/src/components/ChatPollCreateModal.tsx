import { useId, useMemo, useRef, useState } from 'react'
import { Plus, X } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useDialogA11y } from '../lib/useDialogA11y'
import { errorMessage } from '../lib/errors'
import { BTN_PRIMARY, BTN_SECONDARY } from '../lib/buttonStyles'

const MAX_OPTIONS = 10
const MIN_OPTIONS_ON_SCREEN = 2

interface Props {
  convId: number
  onClose: () => void
  onCreated: () => void
}

// ChatPollCreateModal legt eine Umfrage in einer Gruppenkonversation an
// (spec chat-umfragen, design.md §8). Start mit 2 Feldern, „Option
// hinzufügen" bis 10, Entfernen (X) ab dem 3. Feld — die ersten beiden
// bleiben strukturell erhalten, damit immer mindestens 2 Eingaben sichtbar
// sind. Leere Felder werden beim Senden verworfen statt als leere Optionen
// mitgeschickt.
export default function ChatPollCreateModal({ convId, onClose, onCreated }: Props) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  useEscapeKey(onClose)
  useDialogA11y(dialogRef, true)
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState<string[]>(['', ''])
  const [allowMultiple, setAllowMultiple] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const trimmedQuestion = question.trim()
  const nonEmptyOptions = options.map((o) => o.trim()).filter((o) => o.length > 0)
  const distinctOptionCount = useMemo(
    () => new Set(nonEmptyOptions.map((o) => o.toLowerCase())).size,
    [nonEmptyOptions],
  )
  const canSubmit =
    trimmedQuestion.length > 0 &&
    trimmedQuestion.length <= 200 &&
    distinctOptionCount >= MIN_OPTIONS_ON_SCREEN &&
    nonEmptyOptions.length === distinctOptionCount

  const updateOption = (idx: number, value: string) => {
    setOptions((prev) => prev.map((o, i) => (i === idx ? value : o)))
  }

  const addOption = () => {
    setOptions((prev) => (prev.length >= MAX_OPTIONS ? prev : [...prev, '']))
  }

  const removeOption = (idx: number) => {
    setOptions((prev) => prev.filter((_, i) => i !== idx))
  }

  const submit = async () => {
    if (!canSubmit || loading) return
    setLoading(true)
    setError('')
    try {
      await api.post(`/chat/conversations/${convId}/polls`, {
        question: trimmedQuestion,
        options: nonEmptyOptions,
        allowMultiple,
      })
      onCreated()
    } catch (e) {
      setError(errorMessage(e, 'Umfrage konnte nicht erstellt werden'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-md max-h-[90vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between mb-4 shrink-0">
          <h2 id={titleId} className="text-lg font-bold text-brand-text">Umfrage erstellen</h2>
          <button
            onClick={onClose}
            aria-label="Schließen"
            className="p-1 rounded hover:bg-brand-border-subtle transition-colors"
          >
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto space-y-4">
          <div>
            <label className="block text-xs text-brand-text-muted mb-1">Frage</label>
            <input
              type="text"
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              maxLength={200}
              placeholder="Frage eingeben…"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>

          <div>
            <label className="block text-xs text-brand-text-muted mb-1">Optionen</label>
            <div className="space-y-2">
              {options.map((opt, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <input
                    type="text"
                    value={opt}
                    onChange={(e) => updateOption(idx, e.target.value)}
                    maxLength={100}
                    placeholder={`Option ${idx + 1}`}
                    className="flex-1 min-w-0 border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                  />
                  {idx >= 2 && (
                    <button
                      onClick={() => removeOption(idx)}
                      aria-label={`Option ${idx + 1} entfernen`}
                      className="p-1.5 rounded-md text-brand-text-muted hover:text-brand-danger hover:bg-brand-danger-light transition-colors shrink-0"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
            {options.length < MAX_OPTIONS && (
              <button
                onClick={addOption}
                className="mt-2 inline-flex items-center gap-1 text-xs font-medium text-brand-text hover:text-brand-black"
              >
                <Plus className="w-3.5 h-3.5" />
                Option hinzufügen
              </button>
            )}
          </div>

          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={allowMultiple}
              onChange={(e) => setAllowMultiple(e.target.checked)}
              className="accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">Mehrfachantworten erlauben</span>
          </label>

          <p className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            Stimmen sind für alle Mitglieder sichtbar
          </p>

          {error && <p className="text-sm text-brand-danger">{error}</p>}
        </div>

        <div className="pt-4 shrink-0 flex gap-2">
          <button onClick={onClose} className={`flex-1 ${BTN_SECONDARY}`}>
            Abbrechen
          </button>
          <button onClick={submit} disabled={!canSubmit || loading} className={`flex-1 ${BTN_PRIMARY}`}>
            {loading ? 'Erstelle…' : 'Umfrage erstellen'}
          </button>
        </div>
      </div>
    </div>
  )
}
