import { useState, FormEvent } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import PasswordInput from '../components/forms/PasswordInput'

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
      const next = searchParams.get('next')
      navigate(next && next.startsWith('/') && !next.startsWith('//') ? next : '/')
    } catch {
      setError('E-Mail/Spielername oder Passwort ungültig.')
    }
  }

  return (
    <div className="min-h-screen flex flex-col sm:flex-row bg-brand-gray">
      {/* Logo Section - Hidden on Mobile */}
      <div className="hidden sm:flex flex-col justify-center items-center sm:w-56 shrink-0 px-8 py-12 text-brand-black">
        <img src="/logo.svg" alt="Team Stuttgart" className="h-20 w-20 mb-6" />
        <h1 className="text-2xl font-bold mb-1">TeamWERK</h1>
        <p className="text-brand-black/50 text-sm">Team Stuttgart</p>
      </div>

      {/* Login Form */}
      <div className="flex-1 flex items-center justify-center bg-brand-white sm:rounded-l-3xl sm:border-l-4 sm:border-brand-yellow">
        <div className="w-full max-w-sm px-4 sm:px-8 py-8 sm:py-0">
          {/* Mobile Logo */}
          <div className="sm:hidden flex flex-col items-center mb-8">
            <img src="/logo.svg" alt="Team Stuttgart" className="h-16 w-16 mb-4" />
            <h1 className="text-2xl font-bold mb-1">TeamWERK</h1>
            <p className="text-brand-black/50 text-sm">Team Stuttgart</p>
          </div>

          <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8">
            <h2 className="text-xl font-bold text-brand-black mb-6">Anmelden</h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && <p className="text-brand-danger text-sm">{error}</p>}
              <div>
                <label className="block text-sm font-medium text-brand-black mb-1">E-Mail oder Spielername</label>
                <input
                  type="text"
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                  autoComplete="username"
                  required
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-black mb-1">Passwort</label>
                <PasswordInput
                  value={password}
                  onChange={setPassword}
                  autoComplete="current-password"
                  required
                />
              </div>
              <button
                type="submit"
                className="w-full bg-brand-yellow text-brand-black rounded-md py-2.5 sm:py-2 text-sm font-semibold hover:bg-brand-black hover:text-brand-yellow transition-colors"
              >
                Anmelden
              </button>
            </form>
            <div className="mt-4 text-center text-sm space-y-1">
              <div>
                <Link
                  to="/passwort-vergessen"
                  className="text-brand-black hover:text-brand-yellow transition-colors"
                >
                  Passwort vergessen?
                </Link>
              </div>
              <div>
                <Link
                  to="/join"
                  className="text-brand-black hover:text-brand-yellow transition-colors"
                >
                  Beitrittsantrag stellen
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
