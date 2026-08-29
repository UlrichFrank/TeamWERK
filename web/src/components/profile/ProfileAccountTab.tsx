import { useState } from 'react'
import PasswordChangeModal from './PasswordChangeModal'
import EmailChangeModal from './EmailChangeModal'
import { BTN_PRIMARY } from '../../lib/buttonStyles'

interface Props {
  user: { email?: string } | null
  logout: () => void
  recoveryEmail?: string
}

export default function ProfileAccountTab({ user, logout, recoveryEmail }: Props) {
  const [showPwModal, setShowPwModal] = useState(false)
  const [showEmailModal, setShowEmailModal] = useState(false)

  return (
    <div className="space-y-6">
      {/* Kontoangaben */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Kontoangaben</h2>
        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">E-Mail</label>
          <input
            type="email"
            value={user?.email || ''}
            disabled
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm bg-gray-100 text-brand-text-muted"
          />
        </div>
        {recoveryEmail && (
          <div className="mt-4">
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Eltern-E-Mail (Passwort-Reset)</label>
            <input
              type="email"
              value={recoveryEmail}
              disabled
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm bg-brand-surface-card text-brand-text-muted"
            />
            <p className="mt-1 text-xs text-brand-text-subtle">An diese Adresse gehen Passwort-Mails. Änderbar nur über die Eltern.</p>
          </div>
        )}
      </div>

      {/* Sicherheit */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Sicherheit</h2>
        <div className="flex gap-3 flex-wrap">
          <button
            onClick={() => setShowPwModal(true)}
            className={BTN_PRIMARY}
          >
            Passwort ändern
          </button>
          <button
            onClick={() => setShowEmailModal(true)}
            className={BTN_PRIMARY}
          >
            E-Mail ändern
          </button>
        </div>
      </div>

      {showPwModal && <PasswordChangeModal onClose={() => setShowPwModal(false)} logout={logout} />}
      {showEmailModal && <EmailChangeModal onClose={() => setShowEmailModal(false)} />}
    </div>
  )
}
