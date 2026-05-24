import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'
import MobileCard from '../components/MobileCard'

interface Team { id: number; name: string; age_class: string | null; gender: string; is_active: boolean }
interface AgeClassRule { age_class: string }

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

export default function AdminTeamsPage() {
  const [teams, setTeams] = useState<Team[]>([])
  const [ageClasses, setAgeClasses] = useState<string[]>([])
  const [name, setName] = useState('')
  const [ageClass, setAgeClass] = useState('')
  const [gender, setGender] = useState<'m' | 'f' | 'mixed'>('m')

  const load = () => api.get('/admin/teams').then(r => setTeams(r.data ?? []))

  useEffect(() => {
    load()
    api.get<AgeClassRule[]>('/admin/age-class-rules').then(r => {
      setAgeClasses((r.data ?? []).map(rule => rule.age_class))
    })
  }, [])

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    await api.post('/admin/teams', { name, age_class: ageClass || null, gender })
    setName(''); setAgeClass(''); setGender('m')
    load()
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Teams</h1>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow px-4 sm:px-6 py-6">
          <h2 className="font-semibold mb-4 text-brand-text">Neues Team</h2>
          <form onSubmit={handleCreate} className="space-y-3">
            <input value={name} onChange={e => setName(e.target.value)} placeholder="Teamname" required className={INPUT} />
            <select value={ageClass} onChange={e => setAgeClass(e.target.value)} className={INPUT}>
              <option value="">Keine Altersklasse (Erwachsene)</option>
              {ageClasses.map(ac => (
                <option key={ac} value={ac}>{ac}</option>
              ))}
            </select>
            <select value={gender} onChange={e => setGender(e.target.value as 'm' | 'f' | 'mixed')} required className={INPUT}>
              <option value="m">männlich</option>
              <option value="f">weiblich</option>
              <option value="mixed">gemischt</option>
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
              subtitle={t.age_class ?? '—'}
              badge={{ label: t.is_active ? 'aktiv' : 'inaktiv', variant: t.is_active ? 'yellow' : 'red' }}
            />
          ))}
        </div>

        {/* Desktop: Table */}
        <div className="hidden sm:block bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr>
                <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Name</th>
                <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Klasse</th>
                <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Status</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-brand-border-subtle">
              {teams.map(t => (
                <tr key={t.id} className="hover:bg-brand-table-select transition-colors">
                  <td className="px-4 py-3 font-medium text-brand-text">{t.name}</td>
                  <td className="px-4 py-3 text-brand-text-muted">{t.age_class ?? '—'}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${t.is_active ? 'bg-brand-yellow text-brand-black' : 'bg-brand-border-subtle text-brand-text-muted'}`}>
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
