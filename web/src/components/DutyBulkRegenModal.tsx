import { useEffect, useMemo, useRef, useState } from 'react'
import axios from 'axios'
import { AlertTriangle, Check, Trash2, X } from 'lucide-react'
import { api } from '../lib/api'
import { errorData, errorMessage } from '../lib/errors'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useAusrichterOptions } from './GameDayHostPicker'

// Massen-Regeneration der Dienst-Slots über einen wählbaren Zeitraum
// (openspec/changes/duty-bulk-regen). Jede Eingabe-Änderung löst — entprellt und mit
// AbortController gegen veraltete Antworten abgesichert — einen serverseitigen Dry-Run
// aus (POST .../preview). Es gibt bewusst KEINEN Client-Nachbau der Regen-Logik: alle
// Zahlen (created/deleted/assignments_*) kommen aus der Serverantwort, das Frontend
// wählt nur, WAS regeneriert werden soll (design.md §3).

type EventType = 'heim' | 'auswärts' | 'generisch'
type ActionKind = 'template' | 'none' | 'purge'

interface BulkAction {
  action: ActionKind
  template_id?: number
}

interface BulkRegenRow {
  game_id: number
  date: string
  time: string
  event_name: string
  event_type: EventType
  current_template_id?: number
  effective_action: ActionKind
  effective_template_id?: number
  excluded: boolean
  slots_before: { auto: number; custom: number }
  slots_after: { auto: number; custom: number }
  created: number
  deleted_auto: number
  deleted_custom: number
  assignments_kept: number
  assignments_lost: number
  conflicts: number
}

// Tages-Zwischenebene der Serverantwort (heimspieltag-ausrichter): der
// Ausrichter ist eine Eigenschaft des Spieltags, die rows sind flach je Termin.
interface BulkRegenDay {
  date: string
  stored_ausrichter_id?: number
  effective_ausrichter_id: number
  is_explicit: boolean
}

interface BulkRegenTotals {
  games: number
  created: number
  deleted: number
  custom_kept: number
  custom_deleted: number
  assignments_kept: number
  assignments_lost: number
  conflicts: number
  notified_users: number
}

export interface BulkRegenResult {
  range: { from: string; to: string }
  rows: BulkRegenRow[]
  days?: BulkRegenDay[]
  totals: BulkRegenTotals
  warnings: string[]
  applied?: boolean
}

interface Template { id: number; name: string; template_type: string }

interface Props {
  isOpen: boolean
  onClose: () => void
  onApplied: (result: BulkRegenResult) => void
}

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const SELECT_SM = 'border border-brand-border rounded-md px-2 py-1 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_PRIMARY = 'bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_DANGER = 'bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
const BTN_SECONDARY = 'px-4 py-2.5 sm:py-2 border border-brand-border rounded-md text-sm text-brand-text hover:bg-brand-surface-card disabled:opacity-40 disabled:cursor-not-allowed transition-colors'

const EVENT_TYPES: { key: EventType; label: string }[] = [
  { key: 'heim', label: 'Heimspiele' },
  { key: 'auswärts', label: 'Auswärtsspiele' },
  { key: 'generisch', label: 'Generische Events' },
]

/** Kodiert eine BulkAction als <select>-Wert: '' = „wie bisher", 'none'/'purge', sonst Vorlagen-ID. */
function actionToValue(a: BulkAction | undefined): string {
  if (!a) return ''
  if (a.action === 'template') return a.template_id != null ? String(a.template_id) : ''
  return a.action
}

function valueToAction(v: string): BulkAction | undefined {
  if (v === '') return undefined
  if (v === 'none' || v === 'purge') return { action: v }
  const id = Number(v)
  return Number.isFinite(id) ? { action: 'template', template_id: id } : undefined
}

function describeBulkRegenError(err: unknown): string {
  const status = errorStatusOf(err)
  const code = errorData<{ error?: string }>(err)?.error
  if (status === 403) return 'Keine Berechtigung für die Massen-Regeneration.'
  if (code === 'range_in_past') return 'Der Zeitraum darf nicht in der Vergangenheit beginnen.'
  if (code === 'invalid_template') return 'Eine gewählte Dienst-Vorlage existiert nicht (mehr).'
  if (code === 'invalid_action') return 'Ungültiger Zustand gewählt.'
  if (code === 'unknown_ausrichter') return 'Ein gewählter Ausrichter existiert nicht (mehr).'
  if (code === 'inactive_ausrichter') return 'Ein gewählter Ausrichter ist deaktiviert.'
  if (code === 'host_override_out_of_range') return 'Ein gewählter Ausrichter gehört zu einem Tag außerhalb des Zeitraums.'
  if (code === 'invalid_host_override') return 'Ungültiges Datum für einen Ausrichter-Wechsel.'
  if (code === 'no_active_season') return 'Keine aktive Saison eingestellt.'
  if (status === 400) return 'Keine aktive Saison eingestellt.'
  return errorMessage(err, 'Der Lauf ist fehlgeschlagen.')
}

function errorStatusOf(err: unknown): number | undefined {
  return axios.isAxiosError(err) ? err.response?.status : undefined
}

/** Gruppiert die flachen Termin-Zeilen nach Datum, Reihenfolge wie geliefert (chronologisch). */
function groupRowsByDate(rows: BulkRegenRow[]): Array<[string, BulkRegenRow[]]> {
  const byDate = new Map<string, BulkRegenRow[]>()
  for (const row of rows) {
    const existing = byDate.get(row.date)
    if (existing) existing.push(row)
    else byDate.set(row.date, [row])
  }
  return Array.from(byDate.entries())
}

function formatDate(iso: string): string {
  if (iso.length < 10) return iso
  return `${iso.slice(8, 10)}.${iso.slice(5, 7)}.${iso.slice(0, 4)}`
}

export default function DutyBulkRegenModal({ isOpen, onClose, onApplied }: Props) {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [defaults, setDefaults] = useState<Partial<Record<EventType, BulkAction>>>({})
  const [overrides, setOverrides] = useState<Record<number, BulkAction>>({})
  const [excluded, setExcluded] = useState<Set<number>>(new Set())
  const [notify, setNotify] = useState(true)
  // Ausrichter-Wechsel je Spieltag (date → ausrichter_id). Wird wie jede andere
  // Eingabe erst mit apply persistiert; die Vorschau weist die Wirkung in
  // derselben Bilanz aus wie Vorlagen-Änderungen (kein zweiter Dialog).
  const [hostOverrides, setHostOverrides] = useState<Record<string, number>>({})

  const [templates, setTemplates] = useState<Template[]>([])
  const ausrichter = useAusrichterOptions(isOpen)
  const [preview, setPreview] = useState<BulkRegenResult | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState<string | null>(null)
  const [applying, setApplying] = useState(false)
  const [applyError, setApplyError] = useState<string | null>(null)

  const requestSeq = useRef(0)

  useEscapeKey(isOpen && !applying ? onClose : null)

  useEffect(() => {
    if (!isOpen) return
    api.get<Template[]>('/duty-templates').then(r => setTemplates(r.data ?? [])).catch(() => setTemplates([]))
  }, [isOpen])

  const defaultsKey = JSON.stringify(defaults)
  const overridesKey = JSON.stringify(overrides)
  const hostOverridesKey = JSON.stringify(hostOverrides)
  const excludedKey = useMemo(() => Array.from(excluded).sort((a, b) => a - b).join(','), [excluded])

  function requestBody(withNotify: boolean) {
    return {
      from: from || undefined,
      to: to || undefined,
      defaults,
      overrides: Object.entries(overrides).map(([gameId, action]) => ({ game_id: Number(gameId), ...action })),
      host_overrides: Object.entries(hostOverrides).map(([date, ausrichterId]) => ({ date, ausrichter_id: ausrichterId })),
      excluded_game_ids: Array.from(excluded),
      notify: withNotify,
    }
  }

  // Zeitraum-Änderung: Ausrichter-Wechsel außerhalb des neuen Fensters
  // verwerfen. Der Server lehnt sie mit 400 ab (host_override_out_of_range) —
  // eine stehengebliebene Auswahl aus einem verschobenen Zeitraum würde den
  // ganzen Dialog blockieren, ohne dass sichtbar wäre, woran es liegt.
  function pruneHostOverrides(nextFrom: string, nextTo: string) {
    setHostOverrides(prev => {
      const kept = Object.entries(prev).filter(([date]) =>
        (!nextFrom || date >= nextFrom) && (!nextTo || date <= nextTo))
      return kept.length === Object.keys(prev).length ? prev : Object.fromEntries(kept)
    })
  }

  // Live-Vorschau: jede Änderung an Zeitraum/Pauschalwahl/Override/Ausnahme fordert
  // entprellt (~400ms) eine neue Vorschau an; eine laufende Anfrage wird abgebrochen,
  // veraltete Antworten werden über eine monoton steigende Sequenznummer verworfen.
  useEffect(() => {
    if (!isOpen) return
    const seq = ++requestSeq.current
    const controller = new AbortController()
    const timer = setTimeout(() => {
      setPreviewLoading(true)
      setPreviewError(null)
      api.post<BulkRegenResult>('/duty-slots/bulk-regen/preview', requestBody(true), { signal: controller.signal })
        .then(r => {
          if (seq !== requestSeq.current) return
          setPreview(r.data)
          if (from === '') setFrom(r.data.range.from)
          if (to === '') setTo(r.data.range.to)
        })
        .catch(err => {
          if (axios.isCancel(err) || seq !== requestSeq.current) return
          setPreview(null)
          setPreviewError(describeBulkRegenError(err))
        })
        .finally(() => {
          if (seq === requestSeq.current) setPreviewLoading(false)
        })
    }, 400)
    return () => { clearTimeout(timer); controller.abort() }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, from, to, defaultsKey, overridesKey, hostOverridesKey, excludedKey])

  async function apply() {
    setApplying(true)
    setApplyError(null)
    try {
      const r = await api.post<BulkRegenResult>('/duty-slots/bulk-regen/apply', requestBody(notify))
      onApplied(r.data)
      onClose()
    } catch (err) {
      setApplyError(describeBulkRegenError(err))
    } finally {
      setApplying(false)
    }
  }

  if (!isOpen) return null

  const totals = preview?.totals
  const hasPurgeRows = (preview?.rows ?? []).some(r => !r.excluded && r.effective_action === 'purge')

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/40" onClick={applying ? undefined : onClose} />
      <div className="relative bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow max-w-4xl mx-4 w-full max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
          <h2 className="text-lg font-bold text-brand-text">Dienste aktualisieren</h2>
          <button onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-4 overflow-y-auto">
          {applyError && (
            <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {applyError}
            </div>
          )}
          {previewError && (
            <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {previewError}
            </div>
          )}

          <div className="flex flex-wrap items-end gap-3">
            <div>
              <label htmlFor="bulk-regen-from" className="block text-xs text-brand-text-muted mb-1">Von</label>
              <input id="bulk-regen-from" type="date" className={INPUT} value={from} onChange={e => { setFrom(e.target.value); pruneHostOverrides(e.target.value, to) }} />
            </div>
            <div>
              <label htmlFor="bulk-regen-to" className="block text-xs text-brand-text-muted mb-1">Bis</label>
              <input id="bulk-regen-to" type="date" className={INPUT} value={to} onChange={e => { setTo(e.target.value); pruneHostOverrides(from, e.target.value) }} />
            </div>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            {EVENT_TYPES.map(({ key, label }) => (
              <div key={key}>
                <label htmlFor={`bulk-regen-default-${key}`} className="block text-xs text-brand-text-muted mb-1">{label}</label>
                <select
                  id={`bulk-regen-default-${key}`}
                  className={SELECT_SM}
                  value={actionToValue(defaults[key])}
                  onChange={e => setDefaults(prev => {
                    const next = { ...prev }
                    const action = valueToAction(e.target.value)
                    if (action == null) delete next[key]
                    else next[key] = action
                    return next
                  })}
                >
                  <option value="">Wie bisher</option>
                  {templates
                    .filter(tpl => tpl.template_type === key || tpl.template_type === 'generisch')
                    .map(tpl => (
                      <option key={tpl.id} value={tpl.id}>Vorlage: {tpl.name}</option>
                    ))}
                  <option value="none">Keine Dienste anlegen</option>
                  <option value="purge">Alle Dienste löschen</option>
                </select>
              </div>
            ))}
          </div>

          {!totals && previewLoading && (
            <div className="text-sm text-brand-text-muted italic">Vorschau wird geladen…</div>
          )}

          {totals && (
            <div className="flex flex-wrap items-center gap-3 text-sm text-brand-text-muted">
              <span>{totals.games} Termine</span>
              <span>+{totals.created} / −{totals.deleted}</span>
              <span>{totals.custom_kept} handgemacht behalten</span>
              {totals.custom_deleted > 0 && <span className="text-brand-danger">{totals.custom_deleted} handgemacht gelöscht</span>}
              <span>{totals.assignments_kept} Zuweisungen erhalten</span>
              {totals.assignments_lost > 0 && (
                <span className="text-brand-danger font-medium">{totals.assignments_lost} Zuweisungen verloren</span>
              )}
              {totals.conflicts > 0 && <span className="text-brand-danger">{totals.conflicts} Konflikte</span>}
              {previewLoading && <span className="italic">aktualisiert…</span>}
            </div>
          )}

          {hasPurgeRows && !notify && (
            <div className="flex items-start gap-2 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" />
              <span>„Alle Dienste löschen" entfernt Zuweisungen unwiderruflich, und die Benachrichtigung ist abgeschaltet — Betroffene erfahren nirgends davon.</span>
            </div>
          )}

          {/* Zwei Ebenen: der Ausrichter gehört zum Tag, Vorlage/Ausnahme zum
              einzelnen Termin. Die Serverantwort liefert rows flach je Termin,
              die Tages-Klammer wird deshalb hier über row.date gebildet. */}
          <ul className="space-y-4">
            {groupRowsByDate(preview?.rows ?? []).map(([date, dayRows]) => {
              const day = preview?.days?.find(d => d.date === date)
              return (
                <li key={date} className="space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-semibold text-brand-text">{formatDate(date)}</span>
                    {day && ausrichter.length > 0 && (
                      <>
                        <span className="text-xs text-brand-text-muted">Ausrichter</span>
                        <select
                          // Tagesbezogenes Label: im Dialog stehen mehrere dieser
                          // Auswahlen untereinander, ein blankes "Ausrichter" wäre
                          // für Screenreader nicht unterscheidbar.
                          aria-label={`Ausrichter am ${formatDate(date)}`}
                          className={SELECT_SM}
                          value={hostOverrides[date] ?? day.effective_ausrichter_id}
                          onChange={e => setHostOverrides(prev => ({ ...prev, [date]: Number(e.target.value) }))}
                        >
                          {ausrichter.map(a => (
                            <option key={a.id} value={a.id}>{a.name}{a.is_default ? ' (Standard)' : ''}</option>
                          ))}
                        </select>
                        <span className="text-xs text-brand-text-subtle">
                          {hostOverrides[date] != null && hostOverrides[date] !== day.effective_ausrichter_id
                            ? 'wird geändert'
                            : day.is_explicit ? 'festgelegt' : 'geerbt'}
                        </span>
                      </>
                    )}
                  </div>
                  <ul className="space-y-2">
                    {dayRows.map(row => (
                      <BulkRegenRowItem
                        key={row.game_id}
                        row={row}
                        templates={templates}
                        overrideValue={actionToValue(overrides[row.game_id])}
                        onOverrideChange={v => setOverrides(prev => {
                          const next = { ...prev }
                          const action = valueToAction(v)
                          if (action == null) delete next[row.game_id]
                          else next[row.game_id] = action
                          return next
                        })}
                        excluded={excluded.has(row.game_id)}
                        onToggleExcluded={() => setExcluded(prev => {
                          const next = new Set(prev)
                          if (next.has(row.game_id)) next.delete(row.game_id)
                          else next.add(row.game_id)
                          return next
                        })}
                      />
                    ))}
                  </ul>
                </li>
              )
            })}
          </ul>

          <label className="flex items-center gap-2 text-sm text-brand-text">
            <input
              type="checkbox"
              className="accent-brand-yellow"
              checked={!notify}
              onChange={e => setNotify(!e.target.checked)}
            />
            Betroffene nicht benachrichtigen
          </label>
        </div>

        <div className="flex gap-2 justify-end px-6 py-4 border-t border-brand-border-subtle">
          <button onClick={onClose} disabled={applying} className={BTN_SECONDARY}>Abbrechen</button>
          <button
            onClick={apply}
            disabled={applying || previewLoading || !preview}
            className={hasPurgeRows ? BTN_DANGER : BTN_PRIMARY}
          >
            {applying ? 'Wird ausgeführt…' : 'Dienste aktualisieren'}
          </button>
        </div>
      </div>
    </div>
  )
}

interface RowProps {
  row: BulkRegenRow
  templates: Template[]
  overrideValue: string
  onOverrideChange: (v: string) => void
  excluded: boolean
  onToggleExcluded: () => void
}

function BulkRegenRowItem({ row, templates, overrideValue, onOverrideChange, excluded, onToggleExcluded }: RowProps) {
  const destructive = !excluded && row.effective_action === 'purge'
  return (
    <li className={`rounded-lg border p-3 ${destructive ? 'border-brand-danger/30 bg-brand-danger-light' : 'border-brand-border-subtle bg-brand-surface-card'} ${excluded ? 'opacity-60' : ''}`}>
      <div className="flex flex-wrap items-start gap-3">
        <div className="flex-1 min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-2 text-sm text-brand-text">
            <span className="font-medium">{formatDate(row.date)} {row.time}</span>
            <span>{row.event_name}</span>
          </div>
          <div className="text-xs text-brand-text-muted">
            {row.slots_before.auto} Slots{row.slots_before.custom > 0 && ` · ${row.slots_before.custom} handgemacht`}
            {!excluded && (
              <>
                {' → '}
                +{row.created} / −{row.deleted_auto}
                {row.deleted_custom > 0 && (
                  <span className="text-brand-danger"> · {row.deleted_custom} handgemacht gelöscht</span>
                )}
                {row.assignments_kept > 0 && ` · ${row.assignments_kept} Zuweisungen erhalten`}
                {row.assignments_lost > 0 && (
                  <span className="text-brand-danger font-medium"> · {row.assignments_lost} Zuweisungen verloren</span>
                )}
                {row.conflicts > 0 && <span className="text-brand-danger"> · {row.conflicts} Konflikt{row.conflicts === 1 ? '' : 'e'}</span>}
              </>
            )}
          </div>
        </div>

        <select
          aria-label={`Zustand für ${row.event_name} am ${formatDate(row.date)}`}
          className={SELECT_SM}
          value={overrideValue}
          disabled={excluded}
          onChange={e => onOverrideChange(e.target.value)}
        >
          <option value="">Pauschalwahl</option>
          {templates
            .filter(tpl => tpl.template_type === row.event_type || tpl.template_type === 'generisch')
            .map(tpl => (
              <option key={tpl.id} value={tpl.id}>Vorlage: {tpl.name}</option>
            ))}
          <option value="none">Keine Dienste anlegen</option>
          <option value="purge">Alle Dienste löschen</option>
        </select>

        <label className="flex items-center gap-1 text-xs text-brand-text-muted whitespace-nowrap">
          <input type="checkbox" className="accent-brand-yellow" checked={excluded} onChange={onToggleExcluded} />
          Ausnehmen
        </label>

        {destructive && <Trash2 className="w-4 h-4 text-brand-danger shrink-0" aria-hidden />}
        {!destructive && !excluded && row.assignments_lost === 0 && row.conflicts === 0 && (
          <Check className="w-4 h-4 text-brand-text-subtle shrink-0" aria-hidden />
        )}
      </div>
    </li>
  )
}
