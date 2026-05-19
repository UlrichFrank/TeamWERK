import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'

interface DutyType {
  id: number
  name: string
  hours_value: number
  cash_substitute?: number
  default_anchor: 'start' | 'end'
  default_offset_minutes: number
}

interface EditState {
  name: string
  hours: string
  cash: string
  anchor: 'start' | 'end'
  offset: string
}

function toEditState(t: DutyType): EditState {
  return {
    name: t.name,
    hours: t.hours_value.toString(),
    cash: t.cash_substitute != null ? t.cash_substitute.toString() : '',
    anchor: t.default_anchor,
    offset: t.default_offset_minutes.toString(),
  }
}

export default function AdminDutyTypesPage() {
  const [types, setTypes] = useState<DutyType[]>([])
  const [name, setName] = useState('')
  const [hours, setHours] = useState('1')
  const [cash, setCash] = useState('')
  const [anchor, setAnchor] = useState<'start' | 'end'>('start')
  const [offset, setOffset] = useState('0')
  const [editId, setEditId] = useState<number | null>(null)
  const [edit, setEdit] = useState<EditState | null>(null)

  const load = () => api.get('/admin/duty-types').then(r => setTypes(r.data ?? []))
  useEffect(() => { load() }, [])

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    await api.post('/admin/duty-types', {
      name,
      hours_value: parseFloat(hours),
      cash_substitute: cash ? parseFloat(cash) : null,
      default_anchor: anchor,
      default_offset_minutes: parseInt(offset),
    })
    setName(''); setHours('1'); setCash(''); setAnchor('start'); setOffset('0')
    load()
  }

  const startEdit = (t: DutyType) => {
    setEditId(t.id)
    setEdit(toEditState(t))
  }

  const cancelEdit = () => { setEditId(null); setEdit(null) }

  const saveEdit = async (id: number) => {
    if (!edit) return
    await api.put(`/admin/duty-types/${id}`, {
      name: edit.name,
      hours_value: parseFloat(edit.hours),
      cash_substitute: edit.cash ? parseFloat(edit.cash) : null,
      default_anchor: edit.anchor,
      default_offset_minutes: parseInt(edit.offset),
    })
    setEditId(null); setEdit(null)
    load()
  }

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`Diensttyp „${name}" wirklich löschen?`)) return
    await api.delete(`/admin/duty-types/${id}`)
    load()
  }

  const inputCls = 'border border-gray-300 rounded px-2 py-1 text-sm w-full'

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Diensttypen</h1>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl shadow p-6">
          <h2 className="font-semibold mb-4">Neuer Diensttyp</h2>
          <form onSubmit={handleCreate} className="space-y-3">
            <input value={name} onChange={e => setName(e.target.value)} placeholder="Name (z.B. Kassierer)" required
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
            <input value={hours} onChange={e => setHours(e.target.value)} type="number" step="0.5" min="0.5" placeholder="Stundenwert"
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
            <input value={cash} onChange={e => setCash(e.target.value)} type="number" step="0.01" placeholder="Geldersatz in € (optional)"
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
            <div className="flex gap-2">
              <div className="flex-1">
                <label className="block text-xs text-gray-500 mb-1">Standard-Anker</label>
                <select value={anchor} onChange={e => setAnchor(e.target.value as 'start' | 'end')}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
                  <option value="start">Anpfiff</option>
                  <option value="end">Spielende</option>
                </select>
              </div>
              <div className="w-28">
                <label className="block text-xs text-gray-500 mb-1">Versatz (min)</label>
                <input value={offset} onChange={e => setOffset(e.target.value)} type="number"
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
              </div>
            </div>
            <p className="text-xs text-gray-400">
              Negative Werte = vor dem Anker (z.B. −60 = 60 min vor Anpfiff)
            </p>
            <button type="submit" className="bg-brand-yellow text-black rounded-md px-4 py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
              Anlegen
            </button>
          </form>
        </div>

        <div className="bg-white rounded-xl shadow overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
              <tr>
                <th className="px-3 py-3 text-left">Name</th>
                <th className="px-3 py-3 text-right">Stunden</th>
                <th className="px-3 py-3 text-right">Geldersatz</th>
                <th className="px-3 py-3 text-right">Anker</th>
                <th className="px-3 py-3 text-right">Versatz</th>
                <th className="px-3 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {types.map(t => editId === t.id && edit ? (
                <tr key={t.id} className="bg-blue-50">
                  <td className="px-3 py-2">
                    <input value={edit.name} onChange={e => setEdit({ ...edit, name: e.target.value })} className={inputCls} />
                  </td>
                  <td className="px-3 py-2">
                    <input value={edit.hours} onChange={e => setEdit({ ...edit, hours: e.target.value })}
                      type="number" step="0.5" min="0.5" className={inputCls + ' text-right'} />
                  </td>
                  <td className="px-3 py-2">
                    <input value={edit.cash} onChange={e => setEdit({ ...edit, cash: e.target.value })}
                      type="number" step="0.01" placeholder="–" className={inputCls + ' text-right'} />
                  </td>
                  <td className="px-3 py-2">
                    <select value={edit.anchor} onChange={e => setEdit({ ...edit, anchor: e.target.value as 'start' | 'end' })} className={inputCls}>
                      <option value="start">Anpfiff</option>
                      <option value="end">Spielende</option>
                    </select>
                  </td>
                  <td className="px-3 py-2">
                    <input value={edit.offset} onChange={e => setEdit({ ...edit, offset: e.target.value })}
                      type="number" className={inputCls + ' text-right font-mono'} />
                  </td>
                  <td className="px-3 py-2">
                    <div className="flex gap-1 justify-end">
                      <button onClick={() => saveEdit(t.id)}
                        className="text-xs bg-[#3E4A98] text-white rounded px-2 py-1 hover:bg-[#2e3a7a]">
                        Speichern
                      </button>
                      <button onClick={cancelEdit}
                        className="text-xs bg-gray-200 text-gray-700 rounded px-2 py-1 hover:bg-gray-300">
                        Abbrechen
                      </button>
                    </div>
                  </td>
                </tr>
              ) : (
                <tr key={t.id} className="hover:bg-gray-50">
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
                  <td className="px-3 py-3">
                    <div className="flex gap-1 justify-end">
                      <button onClick={() => startEdit(t)}
                        className="text-xs text-[#3E4A98] hover:underline">
                        Bearbeiten
                      </button>
                      <span className="text-gray-300">|</span>
                      <button onClick={() => handleDelete(t.id, t.name)}
                        className="text-xs text-red-500 hover:underline">
                        Löschen
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
