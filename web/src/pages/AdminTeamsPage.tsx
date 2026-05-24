import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'
import MobileCard from '../components/MobileCard'

interface Team { id: number; name: string; age_class: string | null; gender: string; is_active: boolean }
interface AgeClassRule { age_class: string }

const GENDER_LABEL: Record<string, string> = { m: 'männlich', f: 'weiblich', mixed: 'gemischt' }
const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

export default function AdminTeamsPage() {
  const [teams, setTeams] = useState<Team[]>([])
  const [ageClasses, setAgeClasses] = useState<string[]>([])
  const [activeTab, setActiveTab] = useState<string | null>(null)

  const [name, setName] = useState('')
  const [ageClass, setAgeClass] = useState('')
  const [gender, setGender] = useState<'m' | 'f' | 'mixed'>('m')

  const [editId, setEditId] = useState<number | null>(null)
  const [editAgeClass, setEditAgeClass] = useState('')
  const [editGender, setEditGender] = useState<'m' | 'f' | 'mixed'>('m')
  const [editName, setEditName] = useState('')

  const load = () => api.get<Team[]>('/admin/teams').then(r => setTeams(r.data ?? []))

  useEffect(() => {
    load()
    api.get<AgeClassRule[]>('/admin/age-class-rules').then(r => {
      const classes = (r.data ?? []).map(rule => rule.age_class)
      setAgeClasses(classes)
      if (classes.length > 0) setActiveTab(classes[0])
    })
  }, [])

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    await api.post('/admin/teams', { name, age_class: ageClass || null, gender })
    setName(''); setAgeClass(''); setGender('m')
    load()
  }

  const startEdit = (t: Team) => {
    setEditId(t.id)
    setEditName(t.name)
    setEditAgeClass(t.age_class ?? '')
    setEditGender(t.gender as 'm' | 'f' | 'mixed')
  }

  const saveEdit = async (id: number) => {
    await api.put(`/admin/teams/${id}`, {
      name: editName,
      age_class: editAgeClass || null,
      gender: editGender,
    })
    setEditId(null)
    load()
  }

  const tabs = [...ageClasses, '__erwachsene__']
  const tabLabel = (t: string) => t === '__erwachsene__' ? 'Erwachsene' : t
  const teamsForTab = (tab: string) =>
    tab === '__erwachsene__'
      ? teams.filter(t => !t.age_class)
      : teams.filter(t => t.age_class === tab)

  const visibleTeams = activeTab ? teamsForTab(activeTab) : []

  return (
    <div className="px-4 py-4 sm:p-8 max-w-4xl">
      <h1 className="text-2xl font-bold text-brand-text mb-6">Teams</h1>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 flex-wrap">
        {tabs.map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
              activeTab === tab
                ? 'bg-brand-yellow text-brand-black'
                : 'bg-brand-surface-card text-brand-text-muted border border-brand-border hover:bg-brand-table-select'
            }`}
          >
            {tabLabel(tab)}
            <span className="ml-1.5 text-xs opacity-70">({teamsForTab(tab).length})</span>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Team-Liste */}
        <div>
          {/* Mobile */}
          <div className="sm:hidden space-y-2">
            {visibleTeams.map(t => (
              <MobileCard
                key={t.id}
                title={t.name}
                subtitle={GENDER_LABEL[t.gender]}
                badge={{ label: t.is_active ? 'aktiv' : 'inaktiv', variant: t.is_active ? 'yellow' : 'red' }}
              />
            ))}
            {visibleTeams.length === 0 && (
              <p className="text-sm text-brand-text-muted py-4">Keine Teams in dieser Altersklasse.</p>
            )}
          </div>

          {/* Desktop */}
          <div className="hidden sm:block bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Name</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Geschlecht</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-left">Status</th>
                  <th className="bg-brand-surface-card text-brand-text-muted text-xs uppercase px-4 py-3 text-right"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-brand-border-subtle">
                {visibleTeams.length === 0 && (
                  <tr><td colSpan={4} className="px-4 py-6 text-sm text-brand-text-muted">Keine Teams.</td></tr>
                )}
                {visibleTeams.map(t => (
                  <tr key={t.id} className="hover:bg-brand-table-select transition-colors">
                    {editId === t.id ? (
                      <td colSpan={4} className="px-4 py-3">
                        <div className="flex flex-wrap gap-2 items-center">
                          <input value={editName} onChange={e => setEditName(e.target.value)}
                            className="flex-1 min-w-32 border border-brand-border rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-brand-yellow" />
                          <select value={editGender} onChange={e => setEditGender(e.target.value as 'm' | 'f' | 'mixed')}
                            className="border border-brand-border rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-brand-yellow">
                            <option value="m">männlich</option>
                            <option value="f">weiblich</option>
                            <option value="mixed">gemischt</option>
                          </select>
                          <select value={editAgeClass} onChange={e => setEditAgeClass(e.target.value)}
                            className="border border-brand-border rounded px-2 py-1.5 text-sm focus:outline-none focus:ring-1 focus:ring-brand-yellow">
                            <option value="">Keine (Erwachsene)</option>
                            {ageClasses.map(ac => <option key={ac} value={ac}>{ac}</option>)}
                          </select>
                          <button onClick={() => saveEdit(t.id)}
                            className="bg-brand-yellow text-brand-black rounded px-3 py-1.5 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
                            Speichern
                          </button>
                          <button onClick={() => setEditId(null)}
                            className="text-xs text-brand-text-muted hover:text-brand-text">
                            Abbrechen
                          </button>
                        </div>
                      </td>
                    ) : (
                      <>
                        <td className="px-4 py-3 font-medium text-brand-text">{t.name}</td>
                        <td className="px-4 py-3 text-brand-text-muted">{GENDER_LABEL[t.gender]}</td>
                        <td className="px-4 py-3">
                          <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${t.is_active ? 'bg-brand-yellow text-brand-black' : 'bg-brand-border-subtle text-brand-text-muted'}`}>
                            {t.is_active ? 'aktiv' : 'inaktiv'}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right">
                          <button onClick={() => startEdit(t)}
                            className="bg-brand-yellow text-brand-black rounded-md px-3 py-1 text-xs font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
                            Bearbeiten
                          </button>
                        </td>
                      </>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Neues Team anlegen */}
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow px-4 sm:px-6 py-6 h-fit">
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
      </div>
    </div>
  )
}
