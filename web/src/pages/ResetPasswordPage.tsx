import { useState, FormEvent } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import axios from 'axios'

export default function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const navigate = useNavigate()
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    try {
      await axios.post('/api/auth/reset-password', { token, password })
      navigate('/login')
    } catch {
      setError('Ungültiger oder abgelaufener Link.')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-brand-surface-card">
      <div className="w-full max-w-sm bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8">
        <h1 className="text-xl font-bold mb-6 text-brand-text">Neues Passwort setzen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
          <input
            type="password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8}
            placeholder="Neues Passwort"
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
          <button type="submit" className="w-full bg-brand-yellow text-brand-black rounded-md py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
            Passwort speichern
          </button>
        </form>
      </div>
    </div>
  )
}
