import { useState, useEffect } from 'react'
import { api } from '../../lib/api'
import { Member, ChangeDraft } from '../../pages/ProfilePage'

interface Props {
  ownMember: Member | null
}

export default function ProfileBankTab({ ownMember }: Props) {
  const [bankdatenDraft, setBankdatenDraft] = useState<ChangeDraft | null>(null)
  const [iban, setIban] = useState('')
  const [accountHolder, setAccountHolder] = useState('')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  const [cancelError, setCancelError] = useState('')

  const loadDrafts = (memberId: number) =>
    api.get(`/members/${memberId}/change-drafts`).then(r => {
      const list: ChangeDraft[] = r.data?.drafts ?? []
      const draft = list.find(d => d.field_name === 'bankdaten') ?? null
      setBankdatenDraft(draft)
      if (draft) {
        setIban(draft.new_value?.iban ?? '')
        setAccountHolder(draft.new_value?.account_holder ?? '')
      }
    }).catch(() => {})

  useEffect(() => {
    if (!ownMember) return
    setIban(ownMember.iban ?? '')
    setAccountHolder(ownMember.account_holder ?? '')
    loadDrafts(ownMember.id)
  }, [ownMember?.id])

  if (!ownMember) return null

  const ibanChanged = iban.replace(/\s/g, '') !== (ownMember.iban ?? '').replace(/\s/g, '')
  const ahChanged = accountHolder !== (ownMember.account_holder ?? '')
  const canSave = ibanChanged || ahChanged

  const handleSave = async () => {
    setSaving(true)
    setError('')
    try {
      const raw = iban.replace(/\s/g, '').toUpperCase()
      if (ibanChanged && !raw) { setError('IBAN darf nicht leer sein.'); setSaving(false); return }
      await api.post(`/members/${ownMember.id}/change-request`, {
        field_name: 'bankdaten',
        new_value: { iban: raw || (ownMember.iban ?? ''), account_holder: accountHolder },
      })
      await loadDrafts(ownMember.id)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Senden der Änderungsanfrage.')
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = async () => {
    if (!bankdatenDraft) return
    setCancelError('')
    try {
      await api.delete(`/members/${ownMember.id}/change-drafts/${bankdatenDraft.id}`)
      setBankdatenDraft(null)
      setIban(ownMember.iban ?? '')
      setAccountHolder(ownMember.account_holder ?? '')
    } catch {
      setCancelError('Fehler beim Zurückziehen.')
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Bankdaten</h2>

        {bankdatenDraft && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            Änderungsanfrage ausstehend — wird beim Speichern aktualisiert.{' '}
            <button onClick={handleCancel} className="text-brand-danger hover:underline">
              Zurückziehen
            </button>
            {cancelError && <span className="ml-2 text-brand-danger text-xs">{cancelError}</span>}
          </div>
        )}

        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Kontoinhaber</label>
            <input
              type="text"
              value={accountHolder}
              onChange={e => setAccountHolder(e.target.value)}
              placeholder="Nicht hinterlegt"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">IBAN</label>
            <input
              type="text"
              value={iban}
              onChange={e => {
                const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
                if (raw.length <= 22) setIban(raw)
              }}
              placeholder="DE89 3704 0044 0532 0130 00"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text font-mono tracking-wider focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
            <p className="text-xs text-brand-text-subtle mt-1">Bankdaten-Änderungen müssen vom Verein übernommen werden.</p>
          </div>
        </div>

        {canSave && (
          <div className="flex items-center gap-3 mt-4">
            <button
              onClick={handleSave}
              disabled={saving}
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
