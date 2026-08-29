import { useEffect, useState, FormEvent } from 'react'
import { X } from 'lucide-react'
import { api, getReference } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { formatOffset, parseOffset } from '../lib/time'
import ActionMenu from '../components/ActionMenu'
import EditModal from '../components/EditModal'
import DutyInstructionEditorModal from '../components/DutyInstructionEditorModal'
import OffsetInput from '../components/OffsetInput'
import { useEscapeKey } from '../lib/useEscapeKey'
import { AUDIENCE_OPTIONS } from '../lib/constants'
import { hoursToDisplay, parseHoursInput, resolveAnchorClock, addMinutesToTime, dynamicSpanImpossible, IMPOSSIBLE_SPAN_MESSAGE } from '../lib/duration'
import { HEADER_CTRL, HEADER_PRIMARY } from '../lib/buttonStyles'

interface DutyType {
  id: number
  name: string
  hours_value: number
  default_anchor: 'start' | 'end'
  default_offset_minutes: number
  duration_mode: 'absolut' | 'dynamisch'
  end_anchor: 'start' | 'end'
  end_offset_minutes: number
  end_at_next_duty?: boolean
  same_day_behavior?: string
  same_day_variant_id?: number | null
  adjacent_day_behavior?: string
  adjacent_day_variant_id?: number | null
  audiences?: string[] | null
  // Die Liste liefert nur noch das Flag; der Volltext kommt aus dem Detail-Pfad.
  has_instruction?: boolean
  instruction_updated_at?: string
}

interface EditState {
  name: string
  hours: string
  anchor: 'start' | 'end'
  offset: string
  duration_mode: 'absolut' | 'dynamisch'
  end_anchor: 'start' | 'end'
  end_offset: string
  end_at_next_duty: boolean
  same_day_behavior: string
  same_day_variant_id: string
  adjacent_day_behavior: string
  adjacent_day_variant_id: string
  audiences: string[]
}

function toEditState(t: DutyType): EditState {
  return {
    name: t.name,
    hours: hoursToDisplay(t.hours_value),
    anchor: t.default_anchor,
    offset: formatOffset(t.default_offset_minutes),
    duration_mode: t.duration_mode ?? 'absolut',
    end_anchor: t.end_anchor ?? 'end',
    end_offset: formatOffset(t.end_offset_minutes ?? 0),
    end_at_next_duty: t.end_at_next_duty ?? false,
    same_day_behavior: t.same_day_behavior || 'normal',
    same_day_variant_id: t.same_day_variant_id ? t.same_day_variant_id.toString() : '',
    adjacent_day_behavior: t.adjacent_day_behavior || 'normal',
    adjacent_day_variant_id: t.adjacent_day_variant_id ? t.adjacent_day_variant_id.toString() : '',
    audiences: t.audiences ?? [],
  }
}

const emptyCreate = (): EditState => ({
  name: '', hours: '1h', anchor: 'start', offset: '0',
  duration_mode: 'absolut', end_anchor: 'end', end_offset: '0', end_at_next_duty: false,
  same_day_behavior: 'normal', same_day_variant_id: '',
  adjacent_day_behavior: 'normal', adjacent_day_variant_id: '',
  audiences: [],
})

// Feste Beispielwerte für die Vorschau im dynamischen Modus (5.3): eine
// beliebige, aber plausible Anstoßzeit + Spieldauer, gegen die Anker und
// Versätze gerechnet werden — nur damit ein negativer Versatz beim Pflegen
// sofort als Spanne sichtbar wird, nicht erst am nächsten Regen-Lauf.
const EXAMPLE_KICKOFF = '10:00'
const EXAMPLE_GAME_DURATION_MIN = 60

function exampleSpan(state: EditState): { start: string; end: string } {
  const exampleGameEnd = addMinutesToTime(EXAMPLE_KICKOFF, EXAMPLE_GAME_DURATION_MIN)
  return {
    start: resolveAnchorClock(state.anchor, parseOffset(state.offset), EXAMPLE_KICKOFF, exampleGameEnd),
    end: resolveAnchorClock(state.end_anchor, parseOffset(state.end_offset), EXAMPLE_KICKOFF, exampleGameEnd),
  }
}

/**
 * Unmögliche Spanne im aktuellen Formularzustand — dieselbe Regel, die der Server
 * anwendet (dienst-zeitmodus-strikt). Das Formular blockiert damit vor dem Request,
 * statt den 400 abzuwarten: die Meldung steht dort, wo der Fehler entsteht.
 */
function editStateSpanImpossible(state: EditState): boolean {
  return dynamicSpanImpossible(
    state.duration_mode, state.anchor, parseOffset(state.offset),
    state.end_anchor, parseOffset(state.end_offset),
  )
}

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const INPUT_SM = 'w-full border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow'

function DutyTypeForm({ state, onChange, types, excludeId }: {
  state: EditState
  onChange: (s: EditState) => void
  types: DutyType[]
  excludeId?: number
}) {
  const variantOptions = types.filter(t => t.id !== excludeId)
  const example = exampleSpan(state)
  const spanError = editStateSpanImpossible(state)
  return (
    <div className="space-y-3">
      <div>
        <label className="block text-sm font-medium text-brand-text-muted mb-1">Name</label>
        <input value={state.name} onChange={e => onChange({ ...state, name: e.target.value })}
          placeholder="z.B. Kassierer" required className={INPUT} />
      </div>
      <div>
        <label className="block text-sm font-medium text-brand-text-muted mb-1">Zeit-Modus</label>
        <div className="flex flex-col gap-1.5 sm:flex-row sm:gap-4">
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input
              type="radio"
              name="duration-mode"
              checked={state.duration_mode === 'absolut'}
              onChange={() => onChange({ ...state, duration_mode: 'absolut' })}
              className="accent-brand-yellow"
            />
            Startzeit + Dauer
          </label>
          <label className="flex items-center gap-2 text-sm cursor-pointer">
            <input
              type="radio"
              name="duration-mode"
              checked={state.duration_mode === 'dynamisch'}
              onChange={() => onChange({ ...state, duration_mode: 'dynamisch' })}
              className="accent-brand-yellow"
            />
            Startzeit + Endzeit
          </label>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Start-Anker</label>
          <select value={state.anchor} onChange={e => onChange({ ...state, anchor: e.target.value as 'start' | 'end' })} className={INPUT}>
            <option value="start">Anpfiff/Beginn</option>
            <option value="end">Abpfiff/Ende</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Start-Versatz</label>
          <OffsetInput
            value={parseOffset(state.offset)}
            onChange={v => onChange({ ...state, offset: formatOffset(v) })}
            className={INPUT}
          />
        </div>
      </div>

      {state.duration_mode === 'absolut' ? (
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Dauer</label>
          <input
            list="hours-presets"
            value={state.hours}
            onChange={e => onChange({ ...state, hours: e.target.value })}
            placeholder="z.B. 1h 30min"
            className={INPUT}
          />
          <datalist id="hours-presets">
            {['30min','45min','1h','1h 15min','1h 30min','1h 45min','2h','2h 30min','3h'].map(v => (
              <option key={v} value={v} />
            ))}
          </datalist>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">End-Anker</label>
              <select value={state.end_anchor} onChange={e => onChange({ ...state, end_anchor: e.target.value as 'start' | 'end' })} className={INPUT}>
                <option value="start">Anpfiff/Beginn</option>
                <option value="end">Abpfiff/Ende</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">End-Versatz</label>
              <OffsetInput
                value={parseOffset(state.end_offset)}
                onChange={v => onChange({ ...state, end_offset: formatOffset(v) })}
                className={INPUT}
              />
            </div>
          </div>
          {spanError ? (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {IMPOSSIBLE_SPAN_MESSAGE}
            </p>
          ) : (
            <p className="text-xs text-brand-text-subtle">
              Beispiel bei {EXAMPLE_KICKOFF} Anpfiff, {EXAMPLE_GAME_DURATION_MIN} min Spieldauer: {example.start}–{example.end}
            </p>
          )}
          {/* Ablösung (dienst-abloesung): bewusst KEIN dritter Zeit-Modus, sondern ein
              Häkchen direkt unter dem Ende, das es deckelt — die Regel ist eine Kappung
              des oben definierten Endes, kein eigenes Ende. */}
          <label className="flex items-start gap-2 text-sm cursor-pointer">
            <input
              type="checkbox"
              checked={state.end_at_next_duty}
              onChange={e => onChange({ ...state, end_at_next_duty: e.target.checked })}
              className="mt-0.5 accent-brand-yellow"
            />
            <span>
              Endet spätestens bei Ablösung
              <span className="block text-xs text-brand-text-subtle">
                Der Dienst endet, sobald der nächste gleichartige Dienst am selben Tag beginnt —
                spätestens zum oben gesetzten Ende.
              </span>
            </span>
          </label>
        </>
      )}
      <p className="text-xs text-brand-text-subtle">Versatz-Format: <code>-1h 30min</code> (vor Anker) · <code>+30min</code> (nach Anker) · <code>0</code></p>

      <div>
        <label className="block text-sm font-medium text-brand-text-muted mb-1">Zielgruppe</label>
        <p className="text-xs text-brand-text-subtle mb-2">Leer = keine Einschränkung</p>
        <div className="grid grid-cols-2 gap-1.5">
          {AUDIENCE_OPTIONS.map(o => (
            <label key={o.value} className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={state.audiences.includes(o.value)}
                onChange={e => onChange({
                  ...state,
                  audiences: e.target.checked
                    ? [...state.audiences, o.value]
                    : state.audiences.filter(a => a !== o.value),
                })}
                className="accent-brand-yellow"
              />
              {o.label}
            </label>
          ))}
        </div>
      </div>

      <div className="border-t border-brand-border-subtle pt-3 mt-1">
        <p className="text-xs font-semibold text-brand-text-muted mb-2">Spieltag-Verhalten</p>
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-brand-text-muted mb-1">Mehrere Spiele am gleichen Tag</label>
            <select value={state.same_day_behavior} onChange={e => onChange({ ...state, same_day_behavior: e.target.value })} className={INPUT_SM}>
              <option value="normal">Normal (immer)</option>
              <option value="skip">Überspringen</option>
              <option value="reduced">Reduziert</option>
            </select>
          </div>
          {state.same_day_behavior === 'reduced' && (
            <div>
              <label className="block text-xs text-brand-text-muted mb-1">Ersatz-Diensttyp</label>
              <select value={state.same_day_variant_id} onChange={e => onChange({ ...state, same_day_variant_id: e.target.value })} className={INPUT_SM}>
                <option value="">– Wählen –</option>
                {variantOptions.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
            </div>
          )}
          <div>
            <label className="block text-xs text-brand-text-muted mb-1">Spiele am Vortag / Folgetag</label>
            <select value={state.adjacent_day_behavior} onChange={e => onChange({ ...state, adjacent_day_behavior: e.target.value })} className={INPUT_SM}>
              <option value="normal">Normal (immer)</option>
              <option value="skip">Überspringen</option>
              <option value="reduced">Reduziert</option>
            </select>
          </div>
          {state.adjacent_day_behavior === 'reduced' && (
            <div>
              <label className="block text-xs text-brand-text-muted mb-1">Ersatz-Diensttyp</label>
              <select value={state.adjacent_day_variant_id} onChange={e => onChange({ ...state, adjacent_day_variant_id: e.target.value })} className={INPUT_SM}>
                <option value="">– Wählen –</option>
                {variantOptions.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default function AdminDutyTypesPage() {
  const [types, setTypes] = useState<DutyType[]>([])
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [create, setCreate] = useState<EditState>(emptyCreate())
  const [edit, setEdit] = useState<EditState | null>(null)
  const [modalId, setModalId] = useState<number | null>(null)
  const [instructionTarget, setInstructionTarget] = useState<DutyType | null>(null)

  const load = () => getReference<DutyType[]>('/duty-types').then(d => setTypes(d ?? []))
  useEffect(() => { load() }, [])
  useLiveUpdates(event => { if (event === 'duties') load() })

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    // Letzter Riegel vor dem Request: der Anlege-Knopf ist bei unmöglicher Spanne
    // ohnehin deaktiviert, aber ein Submit per Enter im Namensfeld läuft daran vorbei.
    if (editStateSpanImpossible(create)) return
    await api.post('/duty-types', {
      name: create.name,
      hours_value: parseHoursInput(create.hours),
      default_anchor: create.anchor,
      default_offset_minutes: parseOffset(create.offset),
      duration_mode: create.duration_mode,
      end_anchor: create.end_anchor,
      end_offset_minutes: parseOffset(create.end_offset),
      end_at_next_duty: create.end_at_next_duty,
      same_day_behavior: create.same_day_behavior,
      same_day_variant_id: create.same_day_variant_id ? parseInt(create.same_day_variant_id) : null,
      adjacent_day_behavior: create.adjacent_day_behavior,
      adjacent_day_variant_id: create.adjacent_day_variant_id ? parseInt(create.adjacent_day_variant_id) : null,
      audiences: create.audiences.length > 0 ? create.audiences : null,
    })
    setCreate(emptyCreate())
    setShowCreateModal(false)
    load()
  }

  useEscapeKey(showCreateModal ? () => { setShowCreateModal(false); setCreate(emptyCreate()) } : null)

  const startEdit = (t: DutyType) => { setModalId(t.id); setEdit(toEditState(t)) }
  const cancelEdit = () => { setEdit(null); setModalId(null) }

  const saveEdit = async (id: number) => {
    if (!edit || editStateSpanImpossible(edit)) return
    await api.put(`/duty-types/${id}`, {
      name: edit.name,
      hours_value: parseHoursInput(edit.hours),
      default_anchor: edit.anchor,
      default_offset_minutes: parseOffset(edit.offset),
      duration_mode: edit.duration_mode,
      end_anchor: edit.end_anchor,
      end_offset_minutes: parseOffset(edit.end_offset),
      end_at_next_duty: edit.end_at_next_duty,
      same_day_behavior: edit.same_day_behavior,
      same_day_variant_id: edit.same_day_variant_id ? parseInt(edit.same_day_variant_id) : null,
      adjacent_day_behavior: edit.adjacent_day_behavior,
      adjacent_day_variant_id: edit.adjacent_day_variant_id ? parseInt(edit.adjacent_day_variant_id) : null,
      audiences: edit.audiences.length > 0 ? edit.audiences : null,
    })
    setEdit(null); setModalId(null)
    load()
  }

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Diensttyp „${name}" wirklich löschen?`)) return
    try {
      await api.delete(`/duty-types/${id}`)
      load()
    } catch (err) {
      const ax = err as { response?: { status?: number; data?: string } }
      if (ax.response?.status === 409) {
        const msg = typeof ax.response.data === 'string' ? ax.response.data.trim() : 'Diensttyp wird noch verwendet.'
        if (confirm(`${msg}\n\nTrotzdem inkl. aller Dienste, Vorlagen-Einträge und Varianten-Verknüpfungen löschen? Das kann nicht rückgängig gemacht werden.`)) {
          try {
            await api.delete(`/duty-types/${id}?force=true`)
            load()
          } catch {
            alert('Löschen fehlgeschlagen.')
          }
        }
      } else {
        alert('Löschen fehlgeschlagen.')
      }
    }
  }

  return (
    <div>
      {/* Header */}
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0">
          <h1 className="text-2xl font-bold">Diensttypen</h1>
          <button
            onClick={() => setShowCreateModal(true)}
            className={`${HEADER_CTRL} ${HEADER_PRIMARY}`}
          >
            + Diensttyp
          </button>
        </div>
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4 flex flex-col max-h-[90vh]">
            <div className="flex items-center justify-between px-6 pt-6 pb-4 shrink-0 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-lg text-brand-text">Neuer Diensttyp</h2>
              <button
                onClick={() => { setShowCreateModal(false); setCreate(emptyCreate()) }}
                aria-label="Schließen"
                className="text-brand-text-muted hover:text-brand-text transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleCreate} className="flex flex-col flex-1 min-h-0">
              <div className="overflow-y-auto px-6 py-4 flex-1">
                <DutyTypeForm state={create} onChange={setCreate} types={types} />
              </div>
              <div className="flex gap-2 px-6 py-4 border-t border-brand-border-subtle shrink-0">
                <button
                  type="submit"
                  disabled={editStateSpanImpossible(create)}
                  className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Anlegen
                </button>
                <button
                  type="button"
                  onClick={() => { setShowCreateModal(false); setCreate(emptyCreate()) }}
                  className="px-4 py-2.5 sm:py-2 text-sm border border-brand-border rounded-md text-brand-text hover:bg-brand-surface-card transition-colors"
                >
                  Abbrechen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Table — responsive column hiding */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mt-4">
        {types.length === 0 ? (
          <p className="text-sm text-brand-text-muted italic px-4 py-6 text-center">Keine Diensttypen vorhanden.</p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr>
                <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-3 py-3 text-left">Name</th>
                <th className="hidden md:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-3 py-3 text-right">Dauer</th>
                <th className="hidden lg:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-3 py-3 text-right">Anker</th>
                <th className="hidden lg:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-3 py-3 text-right">Versatz</th>
                <th className="hidden xl:table-cell bg-brand-surface-card text-brand-text-muted text-xs uppercase px-3 py-3">Spieltag</th>
                <th className="bg-brand-surface-card px-3 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-brand-border-subtle">
              {types.map(t => (
                <tr key={t.id} className="hover:bg-brand-table-select transition-colors">
                  <td className="px-3 py-3 font-medium text-brand-text">{t.name}</td>
                  <td className="hidden md:table-cell px-3 py-3 text-right text-brand-text">{hoursToDisplay(t.hours_value)}</td>
                  <td className="hidden lg:table-cell px-3 py-3 text-right text-brand-text-muted">
                    {t.default_anchor === 'start' ? 'Anpfiff/Beginn' : 'Abpfiff/Ende'}
                  </td>
                  <td className="hidden lg:table-cell px-3 py-3 text-right font-mono text-brand-text-muted">
                    {formatOffset(t.default_offset_minutes)}
                  </td>
                  <td className="hidden xl:table-cell px-3 py-3 text-sm space-x-1">
                    {(!t.same_day_behavior || t.same_day_behavior === 'normal') && (!t.adjacent_day_behavior || t.adjacent_day_behavior === 'normal') ? (
                      <span className="text-brand-text-subtle text-xs">Normal</span>
                    ) : (
                      <>
                        {t.same_day_behavior && t.same_day_behavior !== 'normal' && (
                          <span className="text-xs bg-brand-info/10 text-brand-text px-2 py-1 rounded">
                            {t.same_day_behavior === 'skip' ? 'Über.' : 'Red.'} (Tag)
                          </span>
                        )}
                        {t.adjacent_day_behavior && t.adjacent_day_behavior !== 'normal' && (
                          <span className="text-xs bg-brand-info/10 text-brand-text px-2 py-1 rounded">
                            {t.adjacent_day_behavior === 'skip' ? 'Über.' : 'Red.'} (Adj.)
                          </span>
                        )}
                      </>
                    )}
                  </td>
                  <td className="px-3 py-3 text-right">
                    <ActionMenu actions={[
                      { label: 'Bearbeiten', onClick: () => startEdit(t) },
                      { label: 'Anleitung', onClick: () => setInstructionTarget(t) },
                      { label: 'Löschen', onClick: () => handleDelete(t.id, t.name), variant: 'danger' },
                    ]} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {edit && (
        <EditModal
          isOpen={modalId !== null}
          title={`Bearbeiten: ${edit.name}`}
          onClose={cancelEdit}
          onSave={() => modalId && saveEdit(modalId)}
          saveDisabled={editStateSpanImpossible(edit)}
        >
          <DutyTypeForm state={edit} onChange={setEdit} types={types} excludeId={modalId ?? undefined} />
        </EditModal>
      )}

      {instructionTarget && (
        <DutyInstructionEditorModal
          dutyTypeId={instructionTarget.id}
          dutyTypeName={instructionTarget.name}
          onClose={() => setInstructionTarget(null)}
          onSaved={() => { setInstructionTarget(null); load() }}
        />
      )}
    </div>
  )
}
