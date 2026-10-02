import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { X, Plus, AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useDialogA11y } from '../lib/useDialogA11y'
import { errorData, errorMessage, errorStatus } from '../lib/errors'
import { BTN_PRIMARY, BTN_SECONDARY } from '../lib/buttonStyles'
import { MODAL_TITLE, OVERLINE } from '../lib/typography'

// Abgleich einer Chat-Gruppe gegen ihre Standard-Gruppen-Herkunft
// (chat-gruppe-aktualisieren). Das Modal zeigt nur die Vorschau des Servers;
// geändert wird erst mit „OK", und der Server nimmt dann nur IDs an, die in
// seinem neu berechneten Diff stehen.

export interface GroupSource {
  groupType: 'team' | 'practice'
  refId: number
  kind: 'trainer' | 'spieler' | 'eltern' | 'alle_trainer'
}

interface SyncSource extends GroupSource {
  label: string
  total: number
  alreadyIn: number
  problem?: 'empty' | 'not_visible'
}

interface Person { id: number; name: string }

interface SyncPreview {
  sources: SyncSource[]
  add: Person[]
  remove: Person[]
  blocked: boolean
  suggestions?: SyncSource[]
}

interface Props {
  convId: number
  onClose: () => void
  onApplied: () => void
}

const sourceKey = (s: GroupSource) => `${s.groupType}:${s.refId}:${s.kind}`
const toSource = ({ groupType, refId, kind }: GroupSource): GroupSource => ({ groupType, refId, kind })

const PROBLEM_TEXT: Record<NonNullable<SyncSource['problem']>, string> = {
  empty: 'derzeit leer',
  not_visible: 'für dich nicht sichtbar',
}

export default function ConversationSyncModal({ convId, onClose, onApplied }: Props) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  useEscapeKey(onClose)
  useDialogA11y(dialogRef, true)

  // null = noch nicht geladen: die erste Vorschau läuft ohne sources und
  // liefert damit die gespeicherte Herkunft.
  const [sources, setSources] = useState<GroupSource[] | null>(null)
  const [preview, setPreview] = useState<SyncPreview | null>(null)
  const [uncheckedAdd, setUncheckedAdd] = useState<Set<number>>(new Set())
  const [uncheckedRemove, setUncheckedRemove] = useState<Set<number>>(new Set())
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const requestSeq = useRef(0)

  const loadPreview = useCallback(async (next: GroupSource[] | null) => {
    const seq = ++requestSeq.current
    setLoading(true)
    setError('')
    try {
      const r = await api.post(`/chat/conversations/${convId}/sync/preview`, next === null ? {} : { sources: next })
      if (seq !== requestSeq.current) return
      const p: SyncPreview = r.data
      setPreview(p)
      setSources(p.sources.map(toSource))
      // Abwahl bleibt erhalten, solange die Person noch vorkommt.
      const addIds = new Set(p.add.map(x => x.id))
      const removeIds = new Set(p.remove.map(x => x.id))
      setUncheckedAdd(prev => new Set([...prev].filter(id => addIds.has(id))))
      setUncheckedRemove(prev => new Set([...prev].filter(id => removeIds.has(id))))
    } catch (e) {
      if (seq === requestSeq.current) setError(errorMessage(e, 'Vorschau konnte nicht geladen werden'))
    } finally {
      if (seq === requestSeq.current) setLoading(false)
    }
  }, [convId])

  useEffect(() => { void loadPreview(null) }, [loadPreview])

  function changeSources(next: GroupSource[]) {
    setNotice('')
    setSources(next)
    void loadPreview(next)
  }

  function toggle(setter: typeof setUncheckedAdd, id: number) {
    setter(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function apply() {
    if (!preview || sources === null) return
    setSaving(true)
    setError('')
    try {
      await api.post(`/chat/conversations/${convId}/sync/apply`, {
        sources,
        addUserIds: preview.add.filter(p => !uncheckedAdd.has(p.id)).map(p => p.id),
        removeUserIds: preview.remove.filter(p => !uncheckedRemove.has(p.id)).map(p => p.id),
      })
      onApplied()
      onClose()
    } catch (e) {
      const code = errorData<{ error?: string }>(e)?.error
      if (errorStatus(e) === 409 && (code === 'sync_stale' || code === 'sync_blocked')) {
        setNotice('Die Gruppe oder die Standard-Gruppen haben sich inzwischen geändert. Die Vorschau ist neu geladen, bitte prüfen.')
        await loadPreview(sources)
      } else {
        setError(errorMessage(e, 'Abgleich fehlgeschlagen'))
      }
    } finally {
      setSaving(false)
    }
  }

  const chosen = new Set((sources ?? []).map(sourceKey))
  const suggestions = (preview?.suggestions ?? []).filter(s => !chosen.has(sourceKey(s)))
  const noOrigin = !loading && preview !== null && preview.sources.length === 0
  const nothingToDo = preview !== null && preview.add.length === 0 && preview.remove.length === 0

  return (
    <div className="fixed inset-0 bg-brand-black/40 flex items-center justify-center z-[60] p-4">
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow transform-gpu p-6 w-full max-w-md max-h-[90vh] flex flex-col"
      >
        <div className="flex items-center justify-between mb-3 shrink-0">
          <h2 id={titleId} className={MODAL_TITLE}>Teilnehmer aktualisieren</h2>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-brand-border-subtle transition-colors"
            aria-label="Schließen"
          >
            <X className="w-5 h-5 text-brand-text-muted" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto space-y-4">
          <p className="text-sm text-brand-text-muted">
            Die Gruppe wird mit den gewählten Standard-Gruppen abgeglichen: Fehlende Personen
            werden ergänzt, Personen, die nicht mehr dazugehören (z.&nbsp;B. ausgetreten oder nicht
            mehr im Kader), werden entfernt. Unten kannst du die Auswahl anpassen. Geändert wird
            erst mit „OK“.
          </p>

          <section>
            <p className={`${OVERLINE} mb-2`}>Abgleich mit</p>
            {noOrigin && (
              <p className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text mb-2">
                Für diese Gruppe ist noch nicht festgelegt, aus welchen Standard-Gruppen sie besteht.
                Wähle sie einmalig aus. Die Zahlen zeigen, wie viele Personen jeweils schon in der
                Gruppe sind.
              </p>
            )}
            {preview && preview.sources.length > 0 && (
              <div className="flex flex-wrap gap-1.5 mb-2">
                {preview.sources.map(s => (
                  <span
                    key={sourceKey(s)}
                    className={`inline-flex items-center gap-1 text-xs rounded-full px-2 py-1 ${s.problem ? 'bg-brand-danger-light text-brand-danger' : 'bg-brand-yellow/20 text-brand-text'}`}
                  >
                    {s.problem && <AlertTriangle className="w-3 h-3" />}
                    {s.label}
                    <span className={s.problem ? '' : 'text-brand-text-muted'}>
                      {s.problem ? `(${PROBLEM_TEXT[s.problem]})` : `${s.alreadyIn}/${s.total}`}
                    </span>
                    <button
                      onClick={() => changeSources((sources ?? []).filter(x => sourceKey(x) !== sourceKey(s)))}
                      aria-label={`${s.label} entfernen`}
                      disabled={saving}
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
            {suggestions.length > 0 && (
              <div className="max-h-40 overflow-y-auto border border-brand-border-subtle rounded-md divide-y divide-brand-border-subtle">
                {suggestions.map(s => (
                  <button
                    key={sourceKey(s)}
                    onClick={() => changeSources([...(sources ?? []), toSource(s)])}
                    disabled={saving}
                    className="w-full flex items-center justify-between gap-2 text-left px-3 py-2 text-sm text-brand-text hover:bg-brand-table-select transition-colors disabled:opacity-50"
                  >
                    <span className="flex items-center gap-2">
                      <Plus className="w-4 h-4 text-brand-text-muted" />
                      {s.label}
                    </span>
                    <span className="text-xs text-brand-text-muted">
                      {s.alreadyIn} von {s.total} schon drin
                    </span>
                  </button>
                ))}
              </div>
            )}
          </section>

          {preview?.blocked && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              Eine Standard-Gruppe ist derzeit leer oder für dich nicht sichtbar. Entferne sie oben,
              damit der Abgleich niemanden versehentlich entfernt.
            </p>
          )}

          {preview && !preview.blocked && (
            <>
              <PersonList
                title="Hinzufügen"
                people={preview.add}
                unchecked={uncheckedAdd}
                onToggle={id => toggle(setUncheckedAdd, id)}
                disabled={saving}
              />
              <PersonList
                title="Entfernen"
                people={preview.remove}
                unchecked={uncheckedRemove}
                onToggle={id => toggle(setUncheckedRemove, id)}
                disabled={saving}
              />
              {nothingToDo && preview.sources.length > 0 && (
                <p className="text-sm text-brand-text-muted">Die Gruppe ist auf dem aktuellen Stand.</p>
              )}
            </>
          )}

          {loading && <p className="text-sm text-brand-text-muted">Lade Vorschau…</p>}
          {notice && (
            <p className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">{notice}</p>
          )}
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</p>
          )}
        </div>

        <div className="pt-4 shrink-0 flex gap-2">
          <button onClick={onClose} className={`flex-1 ${BTN_SECONDARY}`}>
            Abbrechen
          </button>
          <button
            onClick={apply}
            disabled={loading || saving || !preview || preview.blocked}
            className={`flex-1 ${BTN_PRIMARY}`}
          >
            {saving ? 'Speichere…' : 'OK'}
          </button>
        </div>
      </div>
    </div>
  )
}

function PersonList({ title, people, unchecked, onToggle, disabled }: {
  title: string
  people: Person[]
  unchecked: Set<number>
  onToggle: (id: number) => void
  disabled: boolean
}) {
  if (people.length === 0) return null
  return (
    <section>
      <p className={`${OVERLINE} mb-2`}>{title} ({people.length - people.filter(p => unchecked.has(p.id)).length} von {people.length})</p>
      <ul className="border border-brand-border-subtle rounded-md divide-y divide-brand-border-subtle">
        {people.map(p => (
          <li key={p.id}>
            <label className="flex items-center gap-2 px-3 py-2.5 text-sm text-brand-text cursor-pointer">
              <input
                type="checkbox"
                checked={!unchecked.has(p.id)}
                onChange={() => onToggle(p.id)}
                disabled={disabled}
                className="accent-brand-yellow"
              />
              <span className="truncate">{p.name}</span>
            </label>
          </li>
        ))}
      </ul>
    </section>
  )
}
