import { useState, FormEvent } from 'react'
import { api } from '../../lib/api'
import { useEscapeKey } from '../../lib/useEscapeKey'
import { deriveKeyFromPassword, generateSalt, unwrapKey, wrapKey } from '../../lib/crypto'

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
      let dekEncMember: string | undefined
      let memberSalt: string | undefined

      // Try to re-wrap DEK if the member has sensitive data
      try {
        const profileRes = await api.get<{ own_member?: { id: number } }>('/profile/me')
        const ownMemberId = profileRes.data?.own_member?.id
        if (ownMemberId) {
          const sensitiveRes = await api.get<{
            dek_enc_member?: string
            member_salt?: string
          }>(`/members/${ownMemberId}/sensitive`, { validateStatus: s => s < 500 })

          if (sensitiveRes.status === 200 && sensitiveRes.data?.dek_enc_member && sensitiveRes.data?.member_salt) {
            const oldKey = await deriveKeyFromPassword(pwCurrent, sensitiveRes.data.member_salt)
            const dek = await unwrapKey(sensitiveRes.data.dek_enc_member, oldKey)
            const newSalt = generateSalt()
            const newKey = await deriveKeyFromPassword(pwNew, newSalt)
            dekEncMember = await wrapKey(dek, newKey)
            memberSalt = newSalt
          }
        }
      } catch {
        // If sensitive data check fails, proceed without DEK re-wrap
      }

      await api.post('/profile/password', {
        current_password: pwCurrent,
        new_password: pwNew,
        ...(dekEncMember ? { dek_enc_member: dekEncMember, member_salt: memberSalt } : {}),
      })
      setSuccess(true)
      setTimeout(() => logout(), 2500)
    } catch (err: any) {
      const status = err.response?.status
      setError(status === 403 ? 'Aktuelles Passwort nicht korrekt.' : 'Fehler beim Speichern.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/50" onClick={onClose}></div>

      <div className="relative bg-white rounded-lg shadow-lg max-w-md w-full mx-4">
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <h2 className="text-lg font-semibold">Passwort ändern</h2>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600">×</button>
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
              <input
                type="password"
                value={pwCurrent}
                onChange={e => setPwCurrent(e.target.value)}
                required
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Neues Passwort</label>
              <input
                type="password"
                value={pwNew}
                onChange={e => setPwNew(e.target.value)}
                required
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Wiederholen</label>
              <input
                type="password"
                value={pwConfirm}
                onChange={e => setPwConfirm(e.target.value)}
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
                {saving ? 'Speichern…' : 'Passwort ändern'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
