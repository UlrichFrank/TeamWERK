import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'

interface Team { id: number; name: string; age_class: string; gender: string; is_active: boolean }

export default function AdminTeamsPage() {
  const [teams, setTeams] = useState<Team[]>([])
  const [name, setName] = useState('')
  const [ageClass, setAgeClass] = useState('')
  const [gender, setGender] = useState<'m' | 'f' | 'mixed'>('m')

  const load = () => api.get('/admin/teams').then(r => setTeams(r.data ?? []))
  useEffect(() => { load() }, [])

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    await api.post('/admin/teams', { name, age_class: ageClass, gender })
    setName(''); setAgeClass('')
    load()
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Teams</h1>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl shadow p-6">
          <h2 className="font-semibold mb-4">Neues Team</h2>
          <form onSubmit={handleCreate} className="space-y-3">
            <input value={name} onChange={e => setName(e.target.value)} placeholder="Teamname" required
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
            <input value={ageClass} onChange={e => setAgeClass(e.target.value)} placeholder="Altersklasse (z.B. A-Jugend)" required
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
            <select value={gender} onChange={e => setGender(e.target.value as 'm' | 'f' | 'mixed')}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
              <option value="m">Männlich</option>
              <option value="f">Weiblich</option>
              <option value="mixed">Gemischt</option>
            </select>
            <button type="submit" className="bg-brand-blue text-white rounded-md px-4 py-2 text-sm font-medium">
              Team anlegen
            </button>
          </form>
        </div>
        <div className="bg-white rounded-xl shadow overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
              <tr>
                <th className="px-4 py-3 text-left">Name</th>
                <th className="px-4 py-3 text-left">Klasse</th>
                <th className="px-4 py-3 text-left">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {teams.map(t => (
                <tr key={t.id} className="hover:bg-gray-50">
                  <td className="px-4 py-3 font-medium">{t.name}</td>
                  <td className="px-4 py-3 text-gray-500">{t.age_class}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${t.is_active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}`}>
                      {t.is_active ? 'aktiv' : 'inaktiv'}
                    </span>
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
