import { useEffect, useState } from 'react'
import { X, AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'
import { errorStatus } from '../lib/errors'
import { DUTY_INSTRUCTION_TEMPLATE } from '../lib/dutyInstructionTemplate'
import MarkdownRenderer from './MarkdownRenderer'

interface DutyInstructionEditorModalProps {
  dutyTypeId: number
  dutyTypeName: string
  onClose: () => void
  onSaved: () => void
}

export default function DutyInstructionEditorModal({
  dutyTypeId, dutyTypeName, onClose, onSaved,
}: DutyInstructionEditorModalProps) {
  const [markdown, setMarkdown] = useState<string>('')
  const [hasChanged, setHasChanged] = useState<boolean>(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string>('')

  useEscapeKey(onClose)
  // Der Volltext liegt nicht mehr in der Typen-Liste — beim Öffnen aus dem
  // Detail-Pfad nachladen. Leerer Text → Beispiel-Template vorbelegen.
  useEffect(() => {
    let cancelled = false
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Reset beim Öffnen/Typ-Wechsel vor dem async Nachladen, kein Ableitungs-Bug
    setLoading(true)
    setHasChanged(false)
    setError('')
    api.get<{ instruction_md?: string }>(`/duty-types/${dutyTypeId}/instruction`)
      .then(res => {
        if (cancelled) return
        const md = res.data?.instruction_md ?? ''
        setMarkdown(md === '' ? DUTY_INSTRUCTION_TEMPLATE : md)
      })
      .catch(() => {
        if (!cancelled) setMarkdown(DUTY_INSTRUCTION_TEMPLATE)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [dutyTypeId])

  const save = async () => {
    setSaving(true)
    setError('')
    try {
      await api.put(`/duty-types/${dutyTypeId}/instruction`, { markdown })
      onSaved()
    } catch (e) {
      const status = errorStatus(e)
      if (status === 403) setError('Du bist nicht berechtigt, die Anleitung zu ändern.')
      else if (status === 400) setError('Der Inhalt wurde nicht akzeptiert (evtl. zu lang).')
      else setError('Speichern fehlgeschlagen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-3xl mx-4 flex flex-col max-h-[90vh]">
        <div className="flex items-center justify-between px-6 pt-6 pb-4 shrink-0 border-b border-brand-border-subtle">
          <h2 className="font-semibold text-lg text-brand-text">Anleitung: {dutyTypeName}</h2>
          <button
            onClick={onClose}
            aria-label="Schließen"
            className="text-brand-text-muted hover:text-brand-text transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="overflow-y-auto px-6 py-4 flex-1 flex flex-col gap-4">
          <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            Bilder aus dem Ordner <strong>Anleitungen</strong> unter /dokumente verlinken:
            {' '}
            <code className="font-mono text-xs">![Alt](/dokumente/datei/DATEI_ID)</code>
          </div>

          <label className="text-xs uppercase text-brand-text-muted">Markdown</label>
          <textarea
            value={markdown}
            disabled={loading}
            onChange={e => { setMarkdown(e.target.value); setHasChanged(true) }}
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text font-mono placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow min-h-64 disabled:opacity-40"
          />

          <label className="text-xs uppercase text-brand-text-muted">Vorschau</label>
          <div className="rounded-md border border-brand-border-subtle p-4 bg-brand-surface-card">
            <MarkdownRenderer markdown={markdown} />
          </div>

          {error && (
            <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger flex items-center gap-2">
              <AlertTriangle className="w-4 h-4" />{error}
            </div>
          )}
        </div>

        <div className="flex gap-2 px-6 py-4 border-t border-brand-border-subtle shrink-0">
          <button
            type="button"
            disabled={!hasChanged || saving || loading}
            onClick={save}
            className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {saving ? 'Speichere…' : 'Speichern'}
          </button>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2.5 sm:py-2 text-sm border border-brand-border rounded-md text-brand-text hover:bg-brand-surface-card transition-colors"
          >
            Abbrechen
          </button>
        </div>
      </div>
    </div>
  )
}
