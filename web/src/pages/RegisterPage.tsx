import { useState, FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import axios from 'axios'

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

export default function RegisterPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const navigate = useNavigate()
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)

  if (!token) return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-brand-danger">Ungültiger oder abgelaufener Einladungslink.</p>
    </div>
  )

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await axios.post('/api/auth/register', { token, first_name: firstName, last_name: lastName, password })
      setDone(true)
      setTimeout(() => navigate('/login'), 2000)
    } catch {
      setError('Ungültiger oder abgelaufener Link.')
    }
  }

  if (done) return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-brand-success font-medium">Registrierung erfolgreich! Weiterleitung…</p>
    </div>
  )

  return (
    <div className="min-h-screen flex items-center justify-center bg-white">
      <div className="w-full max-w-sm bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8">
        <h1 className="text-2xl font-bold mb-6 text-brand-text">Konto erstellen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">Vorname</label>
            <input type="text" value={firstName} onChange={e => setFirstName(e.target.value)} required className={INPUT} />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">Nachname</label>
            <input type="text" value={lastName} onChange={e => setLastName(e.target.value)} required className={INPUT} />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text mb-1">Passwort</label>
            <input type="password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8} className={INPUT} />
          </div>
          <button
            type="submit"
            className="w-full bg-brand-yellow text-brand-black rounded-md py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            Konto erstellen
          </button>
        </form>
      </div>
    </div>
  )
}
