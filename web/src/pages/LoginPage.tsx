import { useState, FormEvent } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(
    searchParams.get('error') === 'invalid_token'
      ? 'Der Bestätigungslink ist ungültig oder abgelaufen.'
      : ''
  )

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await login(email, password)
      navigate('/')
    } catch {
      setError('E-Mail oder Passwort ungültig.')
    }
  }

  return (
    <div className="min-h-screen flex bg-brand-gray">
      <div className="flex flex-col justify-center items-center w-full max-w-xs px-8 py-12 text-black">
        <img src="/logo.svg" alt="Team Stuttgart" className="h-20 w-20 mb-6" />
        <h1 className="text-2xl font-bold mb-1">TeamWERK</h1>
        <p className="text-black/50 text-sm">Team Stuttgart</p>
      </div>
      <div className="flex-1 flex items-center justify-center bg-white rounded-l-3xl border-l-4 border-brand-yellow">
        <div className="w-full max-w-sm px-8">
          <h2 className="text-xl font-bold text-gray-900 mb-6">Anmelden</h2>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && <p className="text-red-600 text-sm">{error}</p>}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
              <input
                type="email" value={email} onChange={e => setEmail(e.target.value)} required
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Passwort</label>
              <input
                type="password" value={password} onChange={e => setPassword(e.target.value)} required
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>
            <button
              type="submit"
              className="w-full bg-brand-yellow text-black rounded-md py-2 text-sm font-semibold hover:bg-black hover:text-brand-yellow transition-colors"
            >
              Anmelden
            </button>
          </form>
          <div className="mt-4 text-center text-sm space-y-1">
            <div><Link to="/passwort-vergessen" className="text-black hover:text-brand-yellow transition-colors">Passwort vergessen?</Link></div>
            <div><Link to="/beitritt" className="text-black hover:text-brand-yellow transition-colors">Beitrittsantrag stellen</Link></div>
          </div>
        </div>
      </div>
    </div>
  )
}
