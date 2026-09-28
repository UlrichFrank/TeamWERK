import { useId, useRef, useState, FormEvent } from 'react'
import { api } from '../../lib/api'
import { X } from 'lucide-react'
import { BTN_PRIMARY, BTN_SECONDARY, LABEL } from '../../lib/buttonStyles'
import { useEscapeKey } from '../../lib/useEscapeKey'
import { useDialogA11y } from '../../lib/useDialogA11y'
import { errorStatus } from '../../lib/errors'
import PasswordInput from '../forms/PasswordInput'

interface Props {
  onClose: () => void
  logout: () => void
}

export default function PasswordChangeModal({ onClose, logout }: Props) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  useEscapeKey(onClose)
  useDialogA11y(dialogRef, true)
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
      <div className="fixed inset-0 bg-brand-black/40" onClick={onClose}></div>

      {/* Modal */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="relative bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow transform-gpu max-w-md w-full mx-4 max-h-[90vh] overflow-y-auto"
      >
        <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
          <h2 id={titleId} className="text-lg font-bold text-brand-text">Passwort ändern</h2>
          <button type="button" onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {success ? (
          <div className="p-6 text-center">
            <p className="text-brand-success font-medium">Passwort geändert. Du wirst ausgeloggt…</p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="p-6 space-y-4">
            {error && <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</div>}

            <div>
              <label className={LABEL}>Aktuelles Passwort</label>
              <PasswordInput
                value={pwCurrent}
                onChange={setPwCurrent}
                autoComplete="current-password"
                autoFocus
                required
              />
            </div>

            <div>
              <label className={LABEL}>Neues Passwort</label>
              <PasswordInput
                value={pwNew}
                onChange={setPwNew}
                autoComplete="new-password"
                required
                minLength={12}
              />
              <p className="mt-1 text-xs text-brand-text-muted">Mindestens 12 Zeichen.</p>
            </div>

            <div>
              <label className={LABEL}>Wiederholen</label>
              <PasswordInput
                value={pwConfirm}
                onChange={setPwConfirm}
                autoComplete="new-password"
                required
              />
            </div>

            <div className="flex gap-2 justify-end pt-2">
              <button
                type="button"
                onClick={onClose}
                className={BTN_SECONDARY}
              >
                Abbrechen
              </button>
              <button
                type="submit"
                disabled={saving}
                className={BTN_PRIMARY}
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
