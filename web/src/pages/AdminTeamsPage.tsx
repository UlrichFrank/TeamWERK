import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'
import MobileCard from '../components/MobileCard'

interface Team { id: number; name: string; age_class: string; gender: string; is_active: boolean }

const AGE_CLASS_OPTIONS = [
  { label: 'A-Jugend männlich', ageClass: 'A-Jugend', gender: 'm' as const },
  { label: 'A-Jugend weiblich', ageClass: 'A-Jugend', gender: 'f' as const },
  { label: 'B-Jugend männlich', ageClass: 'B-Jugend', gender: 'm' as const },
  { label: 'B-Jugend weiblich', ageClass: 'B-Jugend', gender: 'f' as const },
  { label: 'C-Jugend männlich', ageClass: 'C-Jugend', gender: 'm' as const },
  { label: 'C-Jugend weiblich', ageClass: 'C-Jugend', gender: 'f' as const },
  { label: 'D-Jugend gemischt', ageClass: 'D-Jugend', gender: 'mixed' as const },
]

export default function AdminTeamsPage() {
  const [teams, setTeams] = useState<Team[]>([])
  const [name, setName] = useState('')
  const [ageClass, setAgeClass] = useState('')
  const [gender, setGender] = useState<'m' | 'f' | 'mixed'>('m')
  const [ageGenderPreset, setAgeGenderPreset] = useState('')

  const load = () => api.get('/admin/teams').then(r => setTeams(r.data ?? []))
  useEffect(() => { load() }, [])

  const handleAgeGenderChange = (value: string) => {
    setAgeGenderPreset(value)
    const option = AGE_CLASS_OPTIONS.find(opt => `${opt.ageClass}|${opt.gender}` === value)
    if (option) {
      setAgeClass(option.ageClass)
      setGender(option.gender)
    }
  }

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    await api.post('/admin/teams', { name, age_class: ageClass, gender })
    setName(''); setAgeClass(''); setAgeGenderPreset('')
    load()
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Teams</h1>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow px-4 sm:px-6 py-6">
          <h2 className="font-semibold mb-4">Neues Team</h2>
          <form onSubmit={handleCreate} className="space-y-3">
            <input value={name} onChange={e => setName(e.target.value)} placeholder="Teamname" required
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
            <select value={ageGenderPreset} onChange={e => handleAgeGenderChange(e.target.value)} required
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
              <option value="">Altersklasse wählen…</option>
              {AGE_CLASS_OPTIONS.map(opt => (
                <option key={`${opt.ageClass}|${opt.gender}`} value={`${opt.ageClass}|${opt.gender}`}>
                  {opt.label}
                </option>
              ))}
            </select>
            <button type="submit" className="w-full sm:w-auto bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
              Team anlegen
            </button>
          </form>
        </div>

        {/* Mobile: Cards */}
        <div className="sm:hidden space-y-0">
          {teams.map(t => (
            <MobileCard
              key={t.id}
              title={t.name}
              subtitle={t.age_class}
              badge={{ label: t.is_active ? 'aktiv' : 'inaktiv', variant: t.is_active ? 'yellow' : 'red' }}
            />
          ))}
        </div>

        {/* Desktop: Table */}
        <div className="hidden sm:block bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
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
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${t.is_active ? 'bg-brand-yellow text-brand-black' : 'bg-gray-100 text-gray-500'}`}>
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
