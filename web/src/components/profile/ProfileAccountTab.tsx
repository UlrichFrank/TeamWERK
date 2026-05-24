import { useState } from 'react'
import PasswordChangeModal from './PasswordChangeModal'
import EmailChangeModal from './EmailChangeModal'

interface Props {
  user: any
  logout: () => void
}

export default function ProfileAccountTab({ user, logout }: Props) {
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
      </div>

      {/* Sicherheit */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Sicherheit</h2>
        <div className="flex gap-3 flex-wrap">
          <button
            onClick={() => setShowPwModal(true)}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            Passwort ändern
          </button>
          <button
            onClick={() => setShowEmailModal(true)}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
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
