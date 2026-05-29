import { useState, useEffect } from 'react'
import { api } from '../../lib/api'
import { Member, ChangeDraft } from '../../pages/ProfilePage'

interface Props {
  ownMember: Member | null
}

const formatIBAN = (raw: string) =>
  raw.replace(/\s/g, '').toUpperCase().match(/.{1,4}/g)?.join(' ') ?? ''

export default function ProfileBankTab({ ownMember }: Props) {
  const [ibanDraft, setIbanDraft] = useState<ChangeDraft | null>(null)
  const [iban, setIban] = useState('')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  const [cancelError, setCancelError] = useState('')

  useEffect(() => {
    if (!ownMember) return
    setIban(ownMember.iban ?? '')
    api.get(`/members/${ownMember.id}/change-drafts`).then(r => {
      const drafts: ChangeDraft[] = r.data?.drafts ?? []
      const draft = drafts.find(d => d.field_name === 'iban') ?? null
      setIbanDraft(draft)
      if (draft) setIban(draft.new_value ?? '')
    }).catch(() => {})
  }, [ownMember?.id])

  if (!ownMember) return null

  const readonly = !!ibanDraft
  const currentIban = ownMember.iban ?? ''
  const ibanChanged = iban.replace(/\s/g, '') !== currentIban.replace(/\s/g, '')

  const handleSave = async () => {
    const raw = iban.replace(/\s/g, '').toUpperCase()
    if (!raw) {
      setError('IBAN darf nicht leer sein.')
      return
    }
    setSaving(true)
    setError('')
    try {
      await api.post(`/members/${ownMember.id}/change-request`, {
        field_name: 'iban',
        new_value: raw,
      })
      const r = await api.get(`/members/${ownMember.id}/change-drafts`)
      const drafts: ChangeDraft[] = r.data?.drafts ?? []
      setIbanDraft(drafts.find(d => d.field_name === 'iban') ?? null)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Senden der Änderungsanfrage.')
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = async () => {
    if (!ibanDraft) return
    setCancelError('')
    try {
      await api.delete(`/members/${ownMember.id}/change-drafts/${ibanDraft.id}`)
      setIbanDraft(null)
      setIban(ownMember.iban ?? '')
    } catch {
      setCancelError('Fehler beim Zurückziehen.')
    }
  }

  return (
    <div className="space-y-6">
      {ownMember.account_holder && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-brand-text-muted mb-4">Bankdaten</h2>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Kontoinhaber</label>
            <input
              type="text"
              value={ownMember.account_holder}
              readOnly
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text bg-brand-surface-card cursor-not-allowed"
            />
            <p className="text-xs text-brand-text-subtle mt-1">Wird vom Verein verwaltet.</p>
          </div>
        </div>
      )}

      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-1">IBAN</h2>
        <p className="text-xs text-brand-text-subtle mb-4">IBAN-Änderungen müssen vom Verein übernommen werden.</p>

        {ibanDraft && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            <span className="font-medium">Ausstehend:</span>{' '}
            <span className="font-mono">{formatIBAN(ibanDraft.new_value ?? '')}</span>
            <button
              onClick={handleCancel}
              className="ml-3 text-brand-danger hover:underline text-sm"
            >
              Zurückziehen
            </button>
            {cancelError && <span className="ml-2 text-brand-danger text-xs">{cancelError}</span>}
          </div>
        )}

        <div className="mb-4">
          <label className="block text-sm font-medium text-brand-text-muted mb-1">IBAN</label>
          <input
            type="text"
            value={readonly ? formatIBAN(iban) : iban}
            readOnly={readonly}
            onChange={e => {
              const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
              if (raw.length <= 22) setIban(raw)
            }}
            placeholder="DE89 3704 0044 0532 0130 00"
            className={`w-full border border-brand-border rounded-md px-3 py-2 text-sm font-mono tracking-wider focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow ${
              readonly ? 'bg-gray-50 text-brand-text-muted cursor-not-allowed' : 'text-brand-text'
            }`}
          />
        </div>

        {!readonly && (
          <div className="flex items-center gap-3">
            <button
              onClick={handleSave}
              disabled={saving || !ibanChanged}
              className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {saving ? 'Senden…' : 'Änderung anfordern'}
            </button>
            {saved && <span className="text-sm text-green-600">Anfrage gesendet</span>}
            {error && <span className="text-sm text-brand-danger">{error}</span>}
          </div>
        )}
      </div>
    </div>
  )
}
