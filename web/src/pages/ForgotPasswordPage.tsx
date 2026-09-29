import { useState, FormEvent } from 'react'
import axios from 'axios'
import { BTN_PRIMARY, INPUT } from '../lib/buttonStyles'
import { ENTRY_TITLE } from '../lib/typography'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    await axios.post('/api/auth/forgot-password', { email: email.trim() }).catch(() => {})
    setSent(true)
  }

  if (sent) return (
    <div className="min-h-screen flex items-center justify-center bg-white px-4">
      <div className="max-w-sm bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-8 text-center">
        <p className="text-sm text-brand-text">Falls die Adresse bekannt ist, erhältst du eine E-Mail mit dem Reset-Link.</p>
      </div>
    </div>
  )

  return (
    <div className="min-h-screen flex items-center justify-center bg-white px-4">
      <div className="w-full max-w-sm bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow transform-gpu p-8">
        <h1 className={`${ENTRY_TITLE} mb-6`}>Passwort zurücksetzen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <input
            type="text" value={email} onChange={e => setEmail(e.target.value)} required placeholder="E-Mail oder Nutzername"
            className={INPUT}
          />
          <button type="submit" className={`w-full ${BTN_PRIMARY}`}>
            Link anfordern
          </button>
        </form>
      </div>
    </div>
  )
}
