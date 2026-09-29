import { useState, useEffect, FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import axios from 'axios'
import PasswordInput from '../components/forms/PasswordInput'
import { BTN_PRIMARY, INPUT, LABEL } from '../lib/buttonStyles'
import { ENTRY_TITLE } from '../lib/typography'

export default function RegisterPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const navigate = useNavigate()
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)
  const [loading, setLoading] = useState(true)
  const [tokenInvalid, setTokenInvalid] = useState(false)
  const [nameReadOnly, setNameReadOnly] = useState(false)

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    if (!token) { setLoading(false); return }
    axios.get(`/api/auth/token-info?token=${token}`)
      .then(r => {
        if (r.data.first_name) {
          setFirstName(r.data.first_name)
          setLastName(r.data.last_name)
          setNameReadOnly(true)
        }
      })
      .catch(() => setTokenInvalid(true))
      .finally(() => setLoading(false))
  }, [token])

  if (!token || tokenInvalid) return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-brand-danger">Ungültiger oder abgelaufener Einladungslink.</p>
    </div>
  )

  if (loading) return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-brand-text-muted text-sm">Wird geladen…</p>
    </div>
  )

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await axios.post('/api/auth/register', { token, first_name: firstName, last_name: lastName, password: password.trim() })
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
    <div className="min-h-screen flex items-center justify-center bg-white px-4">
      <div className="w-full max-w-sm bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-8">
        <h1 className={`${ENTRY_TITLE} mb-6`}>Konto erstellen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
              {error}
            </p>
          )}
          <div>
            <label className={LABEL}>Vorname</label>
            <input
              type="text"
              value={firstName}
              onChange={e => setFirstName(e.target.value)}
              required
              readOnly={nameReadOnly}
              className={nameReadOnly ? `${INPUT} bg-brand-surface-card text-brand-text-muted cursor-default` : INPUT}
            />
          </div>
          <div>
            <label className={LABEL}>Nachname</label>
            <input
              type="text"
              value={lastName}
              onChange={e => setLastName(e.target.value)}
              required
              readOnly={nameReadOnly}
              className={nameReadOnly ? `${INPUT} bg-brand-surface-card text-brand-text-muted cursor-default` : INPUT}
            />
          </div>
          <div>
            <label className={LABEL}>Passwort</label>
            <PasswordInput
              value={password}
              onChange={setPassword}
              autoComplete="new-password"
              required
              minLength={12}
            />
            <p className="mt-1 text-xs text-brand-text-muted">Mindestens 12 Zeichen.</p>
          </div>
          <button
            type="submit"
            className={`w-full ${BTN_PRIMARY}`}
          >
            Konto erstellen
          </button>
        </form>
      </div>
    </div>
  )
}
