import { useState, FormEvent } from 'react'
import { api } from '../../lib/api'
import { useEscapeKey } from '../../lib/useEscapeKey'
import { errorStatus } from '../../lib/errors'
import PasswordInput from '../forms/PasswordInput'

interface Props {
  onClose: () => void
  logout: () => void
}

export default function PasswordChangeModal({ onClose, logout }: Props) {
  useEscapeKey(onClose)
  const [pwCurrent, setPwCurrent] = useState('')
  const [pwNew, setPwNew] = useState('')
  const [pwConfirm, setPwConfirm] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState(false)
  const [saving, setSaving] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')

    if (pwNew !== pwConfirm) {
      setError('Die Passwörter stimmen nicht überein.')
      return
    }

    setSaving(true)
    try {
      await api.post('/profile/password', { current_password: pwCurrent, new_password: pwNew })
      setSuccess(true)
      setTimeout(() => logout(), 2500)
    } catch (err) {
      const status = errorStatus(err)
      setError(status === 403 ? 'Aktuelles Passwort nicht korrekt.' : 'Fehler beim Speichern.')
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
          <h2 className="text-lg font-semibold">Passwort ändern</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
            ×
          </button>
        </div>

        {success ? (
          <div className="p-6 text-center">
            <p className="text-green-600 font-medium">Passwort geändert. Du wirst ausgeloggt…</p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="p-6 space-y-4">
            {error && <p className="text-sm text-red-600">{error}</p>}

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Aktuelles Passwort</label>
              <PasswordInput
                value={pwCurrent}
                onChange={setPwCurrent}
                autoComplete="current-password"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Neues Passwort</label>
              <PasswordInput
                value={pwNew}
                onChange={setPwNew}
                autoComplete="new-password"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Wiederholen</label>
              <PasswordInput
                value={pwConfirm}
                onChange={setPwConfirm}
                autoComplete="new-password"
                required
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
                {saving ? 'Speichern…' : 'Passwort ändern'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
