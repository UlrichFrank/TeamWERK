import { useState, FormEvent } from 'react'
import { api } from '../../lib/api'
import { useEscapeKey } from '../../lib/useEscapeKey'
import { errorStatus } from '../../lib/errors'

interface Props {
  onClose: () => void
}

export default function EmailChangeModal({ onClose }: Props) {
  useEscapeKey(onClose)
  const [emailNew, setEmailNew] = useState('')
  const [emailPw, setEmailPw] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [saving, setSaving] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')

    setSaving(true)
    try {
      await api.post('/profile/email', { new_email: emailNew, password: emailPw })
      setSuccess(true)
      setTimeout(() => onClose(), 3000)
    } catch (err) {
      const status = errorStatus(err)
      if (status === 403) setError('Passwort nicht korrekt.')
      else if (status === 409) setError('E-Mail-Adresse bereits vergeben.')
      else setError('Fehler beim Senden.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="fixed inset-0 bg-black/50" onClick={onClose}></div>

      {/* Modal */}
      <div className="relative bg-white rounded-lg shadow-lg max-w-md w-full mx-4">
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <h2 className="text-lg font-semibold">E-Mail-Adresse ändern</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            ×
          </button>
        </div>

        {success ? (
          <div className="p-6 text-center">
            <p className="text-green-600 font-medium">Bestätigungs-Mail gesendet. Bitte prüfe dein neues Postfach.</p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="p-6 space-y-4">
            {error && <p className="text-sm text-red-600">{error}</p>}

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Neue E-Mail-Adresse</label>
              <input
                type="email"
                value={emailNew}
                onChange={(e) => setEmailNew(e.target.value)}
                required
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Passwort zur Bestätigung</label>
              <input
                type="password"
                value={emailPw}
                onChange={(e) => setEmailPw(e.target.value)}
                required
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>

            <div className="flex gap-3 pt-4">
              <button
                type="button"
                onClick={onClose}
                className="flex-1 text-gray-600 hover:text-gray-900 px-4 py-2 rounded-md text-sm font-medium"
              >
                Abbrechen
              </button>
              <button
                type="submit"
                disabled={saving}
                className="flex-1 bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
              >
                {saving ? 'Senden…' : 'Bestätigungs-Mail senden'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
