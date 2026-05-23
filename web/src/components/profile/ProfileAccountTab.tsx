import { useState, useEffect, FormEvent } from 'react'
import { api } from '../../lib/api'
import PasswordChangeModal from './PasswordChangeModal'
import EmailChangeModal from './EmailChangeModal'

interface Props {
  user: any
  logout: () => void
}

export default function ProfileAccountTab({ user, logout }: Props) {
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [original, setOriginal] = useState({ firstName: '', lastName: '' })
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  const [showPwModal, setShowPwModal] = useState(false)
  const [showEmailModal, setShowEmailModal] = useState(false)

  useEffect(() => {
    api.get('/profile/account').then(r => {
      setFirstName(r.data.first_name ?? '')
      setLastName(r.data.last_name ?? '')
      setOriginal({ firstName: r.data.first_name ?? '', lastName: r.data.last_name ?? '' })
    })
  }, [])

  const changed = firstName !== original.firstName || lastName !== original.lastName

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      await api.put('/profile/account', { first_name: firstName, last_name: lastName })
      setOriginal({ firstName, lastName })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Kontoangaben */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Kontoangaben</h2>
        <form onSubmit={handleSave} className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
              <input
                type="text"
                value={firstName}
                onChange={e => setFirstName(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
              <input
                type="text"
                value={lastName}
                onChange={e => setLastName(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <input
              type="email"
              value={user?.email || ''}
              disabled
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600"
            />
          </div>
        </form>
      </div>

      {/* Sicherheit */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Sicherheit</h2>
        <div className="flex gap-3 flex-wrap">
          <button
            onClick={() => setShowPwModal(true)}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
          >
            Passwort ändern
          </button>
          <button
            onClick={() => setShowEmailModal(true)}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
          >
            E-Mail ändern
          </button>
        </div>
      </div>

      {/* Save Button */}
      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={!changed || saving}
          className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
        >
          {saving ? 'Speichern…' : 'Speichern'}
        </button>
        {saved && <span className="text-sm text-green-600">Gespeichert</span>}
        {error && <span className="text-sm text-red-600">{error}</span>}
      </div>

      {/* Modals */}
      {showPwModal && <PasswordChangeModal onClose={() => setShowPwModal(false)} logout={logout} />}
      {showEmailModal && <EmailChangeModal onClose={() => setShowEmailModal(false)} />}
    </div>
  )
}
