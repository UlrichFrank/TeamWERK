import { useEffect, useState, FormEvent } from 'react'
import { X } from 'lucide-react'
import { api } from '../lib/api'
import { formatOffset, parseOffset } from '../lib/time'
import ActionMenu from '../components/ActionMenu'
import EditModal from '../components/EditModal'
import { useEscapeKey } from '../lib/useEscapeKey'
import { AUDIENCE_OPTIONS } from '../lib/constants'

interface DutyType {
  id: number
  name: string
  hours_value: number
  default_anchor: 'start' | 'end'
  default_offset_minutes: number
  same_day_behavior?: string
  same_day_variant_id?: number | null
  adjacent_day_behavior?: string
  adjacent_day_variant_id?: number | null
  audiences?: string[] | null
}

interface EditState {
  name: string
  hours: string
  anchor: 'start' | 'end'
  offset: string
  same_day_behavior: string
  same_day_variant_id: string
  adjacent_day_behavior: string
  adjacent_day_variant_id: string
  audiences: string[]
}

function hoursToDisplay(h: number): string {
  const totalMins = Math.round(h * 60)
  const hrs = Math.floor(totalMins / 60)
  const mins = totalMins % 60
  if (hrs === 0) return `${mins}min`
  if (mins === 0) return `${hrs}h`
  return `${hrs}h ${mins}min`
}

function parseHoursInput(s: string): number {
  const m = s.trim().match(/^(?:(\d+)h\s*)?(?:(\d+)min)?$/)
  if (m && (m[1] || m[2])) return (parseInt(m[1] || '0')) + parseInt(m[2] || '0') / 60
  const n = parseFloat(s)
  return isNaN(n) ? 1 : n
}

function toEditState(t: DutyType): EditState {
  return {
    name: t.name,
    hours: hoursToDisplay(t.hours_value),
    anchor: t.default_anchor,
    offset: formatOffset(t.default_offset_minutes),
    same_day_behavior: t.same_day_behavior || 'normal',
    same_day_variant_id: t.same_day_variant_id ? t.same_day_variant_id.toString() : '',
    adjacent_day_behavior: t.adjacent_day_behavior || 'normal',
    adjacent_day_variant_id: t.adjacent_day_variant_id ? t.adjacent_day_variant_id.toString() : '',
    audiences: t.audiences ?? [],
  }
}

const emptyCreate = (): EditState => ({
  name: '', hours: '1h', anchor: 'start', offset: '0',
  same_day_behavior: 'normal', same_day_variant_id: '',
  adjacent_day_behavior: 'normal', adjacent_day_variant_id: '',
  audiences: [],
})

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const INPUT_SM = 'w-full border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow'

function DutyTypeForm({ state, onChange, types, excludeId }: {
  state: EditState
  onChange: (s: EditState) => void
  types: DutyType[]
  excludeId?: number
}) {
  const variantOptions = types.filter(t => t.id !== excludeId)
  return (
    <div className="space-y-3">
      <div>
        <label className="block text-sm font-medium text-brand-text-muted mb-1">Name</label>
        <input value={state.name} onChange={e => onChange({ ...state, name: e.target.value })}
          placeholder="z.B. Kassierer" required className={INPUT} />
      </div>
      <div className="grid grid-cols-2 gap-3">
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
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Standard-Anker</label>
          <select value={state.anchor} onChange={e => onChange({ ...state, anchor: e.target.value as 'start' | 'end' })} className={INPUT}>
            <option value="start">Anpfiff/Beginn</option>
            <option value="end">Abpfiff/Ende</option>
          </select>
        </div>
      </div>
      <div>
        <label className="block text-sm font-medium text-brand-text-muted mb-1">Versatz</label>
        <input value={state.offset} onChange={e => onChange({ ...state, offset: e.target.value })}
          placeholder="z.B. -1h 30min" className={INPUT} />
      </div>
      <p className="text-xs text-brand-text-subtle">Format: <code>-1h 30min</code> (vor Anker) · <code>+30min</code> (nach Anker) · <code>0</code></p>

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

  const load = () => api.get('/admin/duty-types').then(r => setTypes(r.data ?? []))
  useEffect(() => { load() }, [])

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    await api.post('/admin/duty-types', {
      name: create.name,
      hours_value: parseHoursInput(create.hours),
      default_anchor: create.anchor,
      default_offset_minutes: parseOffset(create.offset),
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
    if (!edit) return
    await api.put(`/admin/duty-types/${id}`, {
      name: edit.name,
      hours_value: parseHoursInput(edit.hours),
      default_anchor: edit.anchor,
      default_offset_minutes: parseOffset(edit.offset),
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
    await api.delete(`/admin/duty-types/${id}`)
    load()
  }

  return (
    <div>
      {/* Header */}
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0">
          <h1 className="text-2xl font-bold">Diensttypen</h1>
          <button
            onClick={() => setShowCreateModal(true)}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            + Neuer Diensttyp
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
                  className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
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
        >
          <DutyTypeForm state={edit} onChange={setEdit} types={types} excludeId={modalId ?? undefined} />
        </EditModal>
      )}
    </div>
  )
}
