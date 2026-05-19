import { useState, FormEvent } from 'react'
import axios from 'axios'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    await axios.post('/api/auth/forgot-password', { email }).catch(() => {})
    setSent(true)
  }

  if (sent) return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-sm bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-8 text-center">
        <p className="text-sm text-gray-700">Falls die Adresse bekannt ist, erhältst du eine E-Mail mit dem Reset-Link.</p>
      </div>
    </div>
  )

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-8">
        <h1 className="text-xl font-bold mb-6">Passwort zurücksetzen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <input
            type="email" value={email} onChange={e => setEmail(e.target.value)} required placeholder="E-Mail"
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
          />
          <button type="submit" className="w-full bg-brand-yellow text-black rounded-md py-2 text-sm font-semibold hover:bg-black hover:text-brand-yellow transition-colors">
            Link anfordern
          </button>
        </form>
      </div>
    </div>
  )
}
