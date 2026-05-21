import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'
import MobileCard from '../components/MobileCard'
import EditModal from '../components/EditModal'

interface DutyType {
  id: number
  name: string
  hours_value: number
  cash_substitute?: number
  default_anchor: 'start' | 'end'
  default_offset_minutes: number
  same_day_behavior?: string
  same_day_variant_id?: number | null
  adjacent_day_behavior?: string
  adjacent_day_variant_id?: number | null
}

interface EditState {
  name: string
  hours: string
  cash: string
  anchor: 'start' | 'end'
  offset: string
  same_day_behavior: string
  same_day_variant_id: string
  adjacent_day_behavior: string
  adjacent_day_variant_id: string
}

function toEditState(t: DutyType): EditState {
  return {
    name: t.name,
    hours: t.hours_value.toString(),
    cash: t.cash_substitute != null ? t.cash_substitute.toString() : '',
    anchor: t.default_anchor,
    offset: t.default_offset_minutes.toString(),
    same_day_behavior: t.same_day_behavior || 'normal',
    same_day_variant_id: t.same_day_variant_id ? t.same_day_variant_id.toString() : '',
    adjacent_day_behavior: t.adjacent_day_behavior || 'normal',
    adjacent_day_variant_id: t.adjacent_day_variant_id ? t.adjacent_day_variant_id.toString() : '',
  }
}

const emptyCreate = (): EditState => ({
  name: '', hours: '1', cash: '', anchor: 'start', offset: '0',
  same_day_behavior: 'normal', same_day_variant_id: '',
  adjacent_day_behavior: 'normal', adjacent_day_variant_id: '',
})

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
        <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
        <input value={state.name} onChange={e => onChange({ ...state, name: e.target.value })}
          placeholder="z.B. Kassierer" required
          className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Stundenwert</label>
          <input value={state.hours} onChange={e => onChange({ ...state, hours: e.target.value })}
            type="number" step="0.5" min="0.5"
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Geldersatz €{' '}
            <span className="text-gray-400 font-normal text-xs">(optional)</span>
          </label>
          <input value={state.cash} onChange={e => onChange({ ...state, cash: e.target.value })}
            type="number" step="0.01" placeholder="–"
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Standard-Anker</label>
          <select value={state.anchor} onChange={e => onChange({ ...state, anchor: e.target.value as 'start' | 'end' })}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
            <option value="start">Anpfiff</option>
            <option value="end">Spielende</option>
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Versatz (min)</label>
          <input value={state.offset} onChange={e => onChange({ ...state, offset: e.target.value })}
            type="number"
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
        </div>
      </div>
      <p className="text-xs text-gray-400">Negative Werte = vor dem Anker (z.B. −60 = 60 min vor Anpfiff)</p>

      <div className="border-t pt-3 mt-1">
        <p className="text-xs font-semibold text-gray-600 mb-2">Spieltag-Verhalten</p>
        <div className="space-y-3">
          <div>
            <label className="block text-xs text-gray-500 mb-1">Mehrere Spiele am gleichen Tag</label>
            <select value={state.same_day_behavior} onChange={e => onChange({ ...state, same_day_behavior: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
              <option value="normal">Normal (immer)</option>
              <option value="skip">Überspringen</option>
              <option value="reduced">Reduziert</option>
            </select>
          </div>
          {state.same_day_behavior === 'reduced' && (
            <div>
              <label className="block text-xs text-gray-500 mb-1">Ersatz-Diensttyp</label>
              <select value={state.same_day_variant_id} onChange={e => onChange({ ...state, same_day_variant_id: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
                <option value="">– Wählen –</option>
                {variantOptions.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
            </div>
          )}
          <div>
            <label className="block text-xs text-gray-500 mb-1">Spiele am Vortag / Folgetag</label>
            <select value={state.adjacent_day_behavior} onChange={e => onChange({ ...state, adjacent_day_behavior: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
              <option value="normal">Normal (immer)</option>
              <option value="skip">Überspringen</option>
              <option value="reduced">Reduziert</option>
            </select>
          </div>
          {state.adjacent_day_behavior === 'reduced' && (
            <div>
              <label className="block text-xs text-gray-500 mb-1">Ersatz-Diensttyp</label>
              <select value={state.adjacent_day_variant_id} onChange={e => onChange({ ...state, adjacent_day_variant_id: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
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
      hours_value: parseFloat(create.hours),
      cash_substitute: create.cash ? parseFloat(create.cash) : null,
      default_anchor: create.anchor,
      default_offset_minutes: parseInt(create.offset),
      same_day_behavior: create.same_day_behavior,
      same_day_variant_id: create.same_day_variant_id ? parseInt(create.same_day_variant_id) : null,
      adjacent_day_behavior: create.adjacent_day_behavior,
      adjacent_day_variant_id: create.adjacent_day_variant_id ? parseInt(create.adjacent_day_variant_id) : null,
    })
    setCreate(emptyCreate())
    setShowCreateModal(false)
    load()
  }

  const startEdit = (t: DutyType) => { setModalId(t.id); setEdit(toEditState(t)) }
  const cancelEdit = () => { setEdit(null); setModalId(null) }

  const saveEdit = async (id: number) => {
    if (!edit) return
    await api.put(`/admin/duty-types/${id}`, {
      name: edit.name,
      hours_value: parseFloat(edit.hours),
      cash_substitute: edit.cash ? parseFloat(edit.cash) : null,
      default_anchor: edit.anchor,
      default_offset_minutes: parseInt(edit.offset),
      same_day_behavior: edit.same_day_behavior,
      same_day_variant_id: edit.same_day_variant_id ? parseInt(edit.same_day_variant_id) : null,
      adjacent_day_behavior: edit.adjacent_day_behavior,
      adjacent_day_variant_id: edit.adjacent_day_variant_id ? parseInt(edit.adjacent_day_variant_id) : null,
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
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">Diensttypen</h1>
          <button
            onClick={() => setShowCreateModal(true)}
            className="text-sm bg-brand-yellow text-brand-black border border-brand-yellow rounded-md px-3 py-2.5 sm:py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors"
          >
            + Neuer Diensttyp
          </button>
        </div>
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4 flex flex-col max-h-[90vh]">
            <div className="flex items-center justify-between px-6 pt-6 pb-4 shrink-0">
              <h2 className="font-semibold text-lg">Neuer Diensttyp</h2>
              <button
                onClick={() => { setShowCreateModal(false); setCreate(emptyCreate()) }}
                className="text-gray-400 hover:text-gray-600 text-xl leading-none"
              >
                &times;
              </button>
            </div>
            <form onSubmit={handleCreate} className="flex flex-col flex-1 min-h-0">
              <div className="overflow-y-auto px-6 pb-2 flex-1">
                <DutyTypeForm state={create} onChange={setCreate} types={types} />
              </div>
              <div className="flex gap-2 px-6 py-4 border-t shrink-0">
                <button
                  type="submit"
                  className="flex-1 bg-brand-yellow text-black rounded-md px-4 py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
                >
                  Anlegen
                </button>
                <button
                  type="button"
                  onClick={() => { setShowCreateModal(false); setCreate(emptyCreate()) }}
                  className="px-4 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
                >
                  Abbrechen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Mobile: Cards */}
      <div className="sm:hidden space-y-0 mt-4">
        {types.map(t => (
          <MobileCard
            key={t.id}
            title={t.name}
            subtitle={`${t.hours_value.toFixed(1)}h${t.cash_substitute ? ` · ${t.cash_substitute.toFixed(2)}€` : ''}`}
            actions={[
              { label: 'Bearbeiten', onClick: () => startEdit(t) },
              { label: 'Löschen', onClick: () => handleDelete(t.id, t.name), variant: 'danger' },
            ]}
          >
            <div className="text-xs text-gray-500 space-y-1">
              <div>Anker: {t.default_anchor === 'start' ? 'Anpfiff' : 'Spielende'}</div>
              <div>Versatz: {t.default_offset_minutes > 0 ? `+${t.default_offset_minutes}` : t.default_offset_minutes} min</div>
            </div>
          </MobileCard>
        ))}
      </div>

      {/* Desktop: Table */}
      <div className="hidden sm:block bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mt-6">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
            <tr>
              <th className="px-3 py-3 text-left">Name</th>
              <th className="px-3 py-3 text-right">Stunden</th>
              <th className="px-3 py-3 text-right">Geldersatz</th>
              <th className="px-3 py-3 text-right">Anker</th>
              <th className="px-3 py-3 text-right">Versatz</th>
              <th className="px-3 py-3">Spieltag</th>
              <th className="px-3 py-3"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {types.map(t => (
              <tr key={t.id} className="hover:bg-brand-gray">
                <td className="px-3 py-3 font-medium">{t.name}</td>
                <td className="px-3 py-3 text-right">{t.hours_value.toFixed(1)}</td>
                <td className="px-3 py-3 text-right text-gray-500">
                  {t.cash_substitute != null ? `${t.cash_substitute.toFixed(2)} €` : '–'}
                </td>
                <td className="px-3 py-3 text-right text-gray-500">
                  {t.default_anchor === 'start' ? 'Anpfiff' : 'Spielende'}
                </td>
                <td className="px-3 py-3 text-right font-mono text-gray-500">
                  {t.default_offset_minutes > 0 ? `+${t.default_offset_minutes}` : t.default_offset_minutes}
                </td>
                <td className="px-3 py-3 text-sm space-x-1">
                  {(!t.same_day_behavior || t.same_day_behavior === 'normal') && (!t.adjacent_day_behavior || t.adjacent_day_behavior === 'normal') ? (
                    <span className="text-gray-400 text-xs">Normal</span>
                  ) : (
                    <>
                      {t.same_day_behavior && t.same_day_behavior !== 'normal' && (
                        <span className="text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded">
                          {t.same_day_behavior === 'skip' ? 'Über.' : 'Red.'} (Tag)
                        </span>
                      )}
                      {t.adjacent_day_behavior && t.adjacent_day_behavior !== 'normal' && (
                        <span className="text-xs bg-purple-100 text-purple-700 px-2 py-1 rounded">
                          {t.adjacent_day_behavior === 'skip' ? 'Über.' : 'Red.'} (Adj.)
                        </span>
                      )}
                    </>
                  )}
                </td>
                <td className="px-3 py-3">
                  <div className="flex gap-1 justify-end">
                    <button onClick={() => startEdit(t)}
                      className="text-xs bg-brand-yellow text-brand-black px-3 py-1 rounded font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
                      Bearbeiten
                    </button>
                    <button onClick={() => handleDelete(t.id, t.name)}
                      className="text-xs border border-red-300 text-red-600 px-3 py-1 rounded font-medium hover:bg-red-50 hover:border-red-400 transition-colors">
                      Löschen
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
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
