import { useId, useRef, useState, FormEvent } from 'react'
import { api } from '../../lib/api'
import { X } from 'lucide-react'
import { BTN_PRIMARY, BTN_SECONDARY, INPUT, LABEL } from '../../lib/buttonStyles'
import { useEscapeKey } from '../../lib/useEscapeKey'
import { useDialogA11y } from '../../lib/useDialogA11y'
import { errorStatus } from '../../lib/errors'
import { MODAL_TITLE } from '../../lib/typography'

interface Props {
  onClose: () => void
}

export default function EmailChangeModal({ onClose }: Props) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  useEscapeKey(onClose)
  useDialogA11y(dialogRef, true)
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
          <h2 id={titleId} className={MODAL_TITLE}>E-Mail-Adresse ändern</h2>
          <button type="button" onClick={onClose} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
            <X className="w-5 h-5" />
          </button>
        </div>

        {success ? (
          <div className="p-6 text-center">
            <p className="text-brand-success font-medium">Bestätigungs-Mail gesendet. Bitte prüfe dein neues Postfach.</p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="p-6 space-y-4">
            {error && <div className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{error}</div>}

            <div>
              <label className={LABEL}>Neue E-Mail-Adresse</label>
              <input
                type="email"
                value={emailNew}
                onChange={(e) => setEmailNew(e.target.value)}
                required
                className={INPUT}
              />
            </div>

            <div>
              <label className={LABEL}>Passwort zur Bestätigung</label>
              <input
                type="password"
                value={emailPw}
                onChange={(e) => setEmailPw(e.target.value)}
                required
                className={INPUT}
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
                {saving ? 'Senden…' : 'Bestätigungs-Mail senden'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
