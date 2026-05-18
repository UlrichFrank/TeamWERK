import { useState, FormEvent } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import axios from 'axios'

export default function RegisterPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const navigate = useNavigate()
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)

  if (!token) return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-red-600">Ungültiger oder abgelaufener Einladungslink.</p>
    </div>
  )

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    try {
      await axios.post('/api/auth/register', { token, name, password })
      setDone(true)
      setTimeout(() => navigate('/login'), 2000)
    } catch {
      setError('Ungültiger oder abgelaufener Link.')
    }
  }

  if (done) return (
    <div className="min-h-screen flex items-center justify-center">
      <p className="text-green-600 font-medium">Registrierung erfolgreich! Weiterleitung…</p>
    </div>
  )

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm bg-white rounded-xl shadow p-8">
        <h1 className="text-2xl font-bold mb-6">Konto erstellen</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && <p className="text-red-600 text-sm">{error}</p>}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Vor- und Nachname</label>
            <input
              type="text" value={name} onChange={e => setName(e.target.value)} required
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Passwort</label>
            <input
              type="password" value={password} onChange={e => setPassword(e.target.value)} required minLength={8}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <button
            type="submit"
            className="w-full bg-brand-yellow text-black rounded-md py-2 text-sm font-semibold hover:bg-black hover:text-brand-yellow transition-colors"
          >
            Konto erstellen
          </button>
        </form>
      </div>
    </div>
  )
}
