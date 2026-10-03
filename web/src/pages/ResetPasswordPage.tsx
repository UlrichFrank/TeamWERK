import { useState, FormEvent } from 'react'
import { useSearchParams, useNavigate } from 'react-router-dom'
import axios from 'axios'
import PasswordInput from '../components/forms/PasswordInput'
import { ENTRY_TITLE } from '../lib/typography'
import { BTN_PRIMARY } from '../lib/buttonStyles'

export default function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const navigate = useNavigate()
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    try {
      await axios.post('/api/auth/reset-password', { token, password: password.trim() })
      navigate('/login')
    } catch {
      setError('Ungültiger oder abgelaufener Link.')
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-white px-4">
      <div className="w-full max-w-sm bg-brand-surface-card rounded-xl shadow-sm border-t-4 border-brand-yellow transform-gpu p-8">
        <h1 className={`${ENTRY_TITLE} mb-6`}>Neues Passwort setzen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
          <PasswordInput
            value={password}
            onChange={setPassword}
            autoComplete="new-password"
            required
            minLength={12}
            placeholder="Neues Passwort"
          />
          <p className="text-xs text-brand-text-muted">Mindestens 12 Zeichen.</p>
          <button type="submit" className={`w-full ${BTN_PRIMARY}`}>
            Passwort speichern
          </button>
        </form>
      </div>
    </div>
  )
}
