import { useState, FormEvent, useEffect } from 'react'
import axios from 'axios'

interface Team { id: number; name: string }

export default function RequestMembershipPage() {
  const [teams, setTeams] = useState<Team[]>([])
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [teamID, setTeamID] = useState('')
  const [sent, setSent] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    axios.get('/api/teams').then(r => setTeams(r.data)).catch(() => {})
  }, [])

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await axios.post('/api/auth/request-membership', {
        name,
        email,
        team_id: teamID ? Number(teamID) : null,
      })
      setSent(true)
    } catch {
      setError('Fehler beim Senden. Bitte versuche es erneut.')
    }
  }

  if (sent) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="w-full max-w-sm bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
          <h2 className="text-xl font-bold mb-2">Antrag gesendet!</h2>
          <p className="text-sm text-gray-600">
            Dein Antrag wurde weitergeleitet. Du erhältst eine E-Mail sobald er bearbeitet wurde.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-8">
        <h1 className="text-2xl font-bold mb-1">Beitrittsantrag</h1>
        <p className="text-sm text-gray-500 mb-6">Team Stuttgart – TeamWERK</p>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <p className="text-brand-error text-sm">{error}</p>}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Vor- und Nachname</label>
            <input
              type="text" value={name} onChange={e => setName(e.target.value)} required
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <input
              type="email" value={email} onChange={e => setEmail(e.target.value)} required
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Mannschaft <span className="text-gray-400 font-normal">(optional)</span>
            </label>
            <select
              value={teamID} onChange={e => setTeamID(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            >
              <option value="">– keine Angabe –</option>
              {teams.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
            </select>
          </div>
          <button
            type="submit"
            className="w-full bg-brand-yellow text-black rounded-md py-2 text-sm font-semibold hover:bg-black hover:text-brand-yellow transition-colors"
          >
            Antrag absenden
          </button>
        </form>
      </div>
    </div>
  )
}
