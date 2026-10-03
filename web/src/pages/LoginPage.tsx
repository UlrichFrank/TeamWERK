import { useState, FormEvent } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'
import PasswordInput from '../components/forms/PasswordInput'
import { BTN_PRIMARY, INPUT, LABEL } from '../lib/buttonStyles'
import { ENTRY_TITLE, PAGE_TITLE } from '../lib/typography'

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
        <h1 className={`${PAGE_TITLE} mb-1`}>TeamWERK</h1>
        <p className="text-brand-black/50 text-sm">Team Stuttgart</p>
      </div>

      {/* Login Form */}
      <div className="flex-1 flex items-center justify-center bg-brand-white sm:rounded-l-3xl sm:border-l-4 sm:border-brand-yellow">
        <div className="w-full max-w-sm px-4 sm:px-8 py-8 sm:py-0">
          {/* Mobile Logo */}
          <div className="sm:hidden flex flex-col items-center mb-8">
            <img src="/logo.svg" alt="Team Stuttgart" className="h-16 w-16 mb-4" />
            <h1 className={`${PAGE_TITLE} mb-1`}>TeamWERK</h1>
            <p className="text-brand-black/50 text-sm">Team Stuttgart</p>
          </div>

          <div className="bg-brand-surface-card rounded-xl shadow-sm border-t-4 border-brand-yellow transform-gpu p-8">
            <h2 className={`${ENTRY_TITLE} mb-6`}>Anmelden</h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</div>}
              <div>
                <label className={LABEL}>E-Mail oder Spielername</label>
                <input
                  type="text"
                  value={email}
                  onChange={e => setEmail(e.target.value)}
                  autoComplete="username"
                  required
                  className={INPUT}
                />
              </div>
              <div>
                <label className={LABEL}>Passwort</label>
                <PasswordInput
                  value={password}
                  onChange={setPassword}
                  autoComplete="current-password"
                  required
                />
              </div>
              <button
                type="submit"
                className={`w-full ${BTN_PRIMARY}`}
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
              <div>
                <Link
                  to="/datenschutz"
                  className="text-brand-text-muted hover:text-brand-text transition-colors"
                >
                  Datenschutz
                </Link>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
