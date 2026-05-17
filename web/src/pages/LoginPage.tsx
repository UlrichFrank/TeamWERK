import { useState, FormEvent } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

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
    <div className="min-h-screen flex bg-brand-blue">
      <div className="flex flex-col justify-center items-center w-full max-w-xs px-8 py-12 text-white">
        <img src="/logo.svg" alt="Team Stuttgart" className="h-20 w-20 mb-6" />
        <h1 className="text-2xl font-bold mb-1">VereinsWerk</h1>
        <p className="text-white/60 text-sm">Team Stuttgart</p>
      </div>
      <div className="flex-1 flex items-center justify-center bg-white rounded-l-3xl">
        <div className="w-full max-w-sm px-8">
          <h2 className="text-xl font-bold text-gray-900 mb-6">Anmelden</h2>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && <p className="text-red-600 text-sm">{error}</p>}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
              <input
                type="email" value={email} onChange={e => setEmail(e.target.value)} required
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-blue"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Passwort</label>
              <input
                type="password" value={password} onChange={e => setPassword(e.target.value)} required
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-blue"
              />
            </div>
            <button
              type="submit"
              className="w-full bg-brand-blue text-white rounded-md py-2 text-sm font-medium hover:bg-brand-blue-dark"
            >
              Anmelden
            </button>
          </form>
          <div className="mt-4 text-center text-sm space-y-1">
            <div><Link to="/passwort-vergessen" className="text-brand-blue hover:underline">Passwort vergessen?</Link></div>
            <div><Link to="/beitritt" className="text-brand-blue hover:underline">Beitrittsantrag stellen</Link></div>
          </div>
        </div>
      </div>
    </div>
  )
}
