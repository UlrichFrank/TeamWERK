import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'

interface DutyType { id: number; name: string; hours_value: number; cash_substitute?: number }

export default function AdminDutyTypesPage() {
  const [types, setTypes] = useState<DutyType[]>([])
  const [name, setName] = useState('')
  const [hours, setHours] = useState('1')
  const [cash, setCash] = useState('')

  const load = () => api.get('/admin/duty-types').then(r => setTypes(r.data ?? []))
  useEffect(() => { load() }, [])

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    await api.post('/admin/duty-types', {
      name,
      hours_value: parseFloat(hours),
      cash_substitute: cash ? parseFloat(cash) : null,
    })
    setName(''); setHours('1'); setCash('')
    load()
  }

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
            <button type="submit" className="bg-brand-yellow text-black rounded-md px-4 py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
              Anlegen
            </button>
          </form>
        </div>
        <div className="bg-white rounded-xl shadow overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
              <tr>
                <th className="px-4 py-3 text-left">Name</th>
                <th className="px-4 py-3 text-right">Stunden</th>
                <th className="px-4 py-3 text-right">Geldersatz</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {types.map(t => (
                <tr key={t.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium">{t.name}</td>
                  <td className="px-4 py-3 text-right">{t.hours_value.toFixed(1)}</td>
                  <td className="px-4 py-3 text-right text-gray-500">
                    {t.cash_substitute != null ? `${t.cash_substitute.toFixed(2)} €` : '–'}
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
