import { useState, FormEvent } from 'react'
import axios from 'axios'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    await axios.post('/api/auth/forgot-password', { email: email.trim() }).catch(() => {})
    setSent(true)
  }

  if (sent) return (
    <div className="min-h-screen flex items-center justify-center bg-brand-surface-card">
      <div className="max-w-sm bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
        <p className="text-sm text-brand-text">Falls die Adresse bekannt ist, erhältst du eine E-Mail mit dem Reset-Link.</p>
      </div>
    </div>
  )

  return (
    <div className="min-h-screen flex items-center justify-center bg-brand-surface-card">
      <div className="w-full max-w-sm bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-8">
        <h1 className="text-xl font-bold mb-6 text-brand-text">Passwort zurücksetzen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <input
            type="text" value={email} onChange={e => setEmail(e.target.value)} required placeholder="E-Mail oder Nutzername"
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
          <button type="submit" className="w-full bg-brand-yellow text-brand-black rounded-md py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
            Link anfordern
          </button>
        </form>
      </div>
    </div>
  )
}
