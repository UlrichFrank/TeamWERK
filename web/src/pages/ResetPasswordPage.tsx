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
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm bg-white rounded-xl shadow p-8">
        <h1 className="text-xl font-bold text-brand-blue mb-6">Neues Passwort setzen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <p className="text-red-600 text-sm">{error}</p>}
          <input
            type="password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8}
            placeholder="Neues Passwort"
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
          />
          <button type="submit" className="w-full bg-brand-blue text-white rounded-md py-2 text-sm font-medium">
            Passwort speichern
          </button>
        </form>
      </div>
    </div>
  )
}
