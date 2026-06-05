import { useState, useEffect, FormEvent } from 'react'
import { api } from '../../lib/api'

interface Props {
  dutyReminderDays?: number | null
}

export default function ProfileMiscTab({ dutyReminderDays: initialReminder }: Props) {
  const [reminderEnabled, setReminderEnabled] = useState(false)
  const [changed, setChanged] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (initialReminder !== undefined) {
      setReminderEnabled(initialReminder !== null)
    } else {
      api.get('/profile/me').then(r => setReminderEnabled(r.data?.duty_reminder_days !== null)).catch(() => {})
    }
  }, [initialReminder])

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      await api.put('/profile/reminder-preference', { duty_reminder_days: reminderEnabled ? 2 : null })
      setSaved(true)
      setChanged(false)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Benachrichtigungen */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Benachrichtigungen</h2>
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-brand-text">Dienst-Erinnerungsmail</p>
            <p className="text-xs text-brand-text-muted mt-0.5">2 Tage vor Events mit offenen Diensten</p>
          </div>
          <button
            onClick={() => { setReminderEnabled(v => !v); setChanged(true) }}
            aria-label={reminderEnabled ? 'Erinnerungsmail deaktivieren' : 'Erinnerungsmail aktivieren'}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              reminderEnabled ? 'bg-brand-yellow' : 'bg-brand-border'
            }`}
          >
            <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
              reminderEnabled ? 'translate-x-6' : 'translate-x-1'
            }`} />
          </button>
        </div>
      </div>

      {/* Save Button */}
      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={!changed || saving}
          className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {saving ? 'Speichern…' : 'Speichern'}
        </button>
        {saved && <span className="text-sm text-brand-text-muted">Gespeichert</span>}
        {error && <span className="text-sm text-brand-danger">{error}</span>}
      </div>
    </div>
  )
}
