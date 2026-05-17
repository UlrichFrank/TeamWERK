import { useState, FormEvent, useEffect } from 'react'
import { api } from '../lib/api'

interface Team { id: number; name: string }

export default function AdminUsersPage() {
  const [teams, setTeams] = useState<Team[]>([])
  const [email, setEmail] = useState('')
  const [teamID, setTeamID] = useState('')
  const [role, setRole] = useState('elternteil')
  const [sent, setSent] = useState(false)

  useEffect(() => { api.get('/admin/teams').then(r => setTeams(r.data ?? [])) }, [])

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault()
    await api.post('/auth/invite', { email, team_id: Number(teamID) || null, role })
    setSent(true)
    setEmail('')
    setTimeout(() => setSent(false), 3000)
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Nutzerverwaltung</h1>
      <div className="bg-white rounded-xl shadow p-6 max-w-md">
        <h2 className="font-semibold mb-4">Einladung versenden</h2>
        {sent && <p className="text-green-600 text-sm mb-3">Einladung gesendet ✓</p>}
        <form onSubmit={handleInvite} className="space-y-3">
          <input value={email} onChange={e => setEmail(e.target.value)} type="email" placeholder="E-Mail" required
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
          <select value={role} onChange={e => setRole(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
            <option value="elternteil">Elternteil</option>
            <option value="spieler">Spieler</option>
            <option value="trainer">Trainer</option>
            <option value="admin">Admin</option>
          </select>
          <select value={teamID} onChange={e => setTeamID(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
            <option value="">– kein Team –</option>
            {teams.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
          </select>
          <button type="submit" className="bg-brand-blue text-white rounded-md px-4 py-2 text-sm font-medium">
            Einladung senden
          </button>
        </form>
      </div>
    </div>
  )
}
