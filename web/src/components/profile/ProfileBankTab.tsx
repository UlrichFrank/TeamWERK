import { useState, useEffect } from 'react'
import { api } from '../../lib/api'
import { Member, ChangeDraft } from '../../pages/ProfilePage'

interface Props {
  ownMember: Member | null
}

const formatIBAN = (raw: string) =>
  raw.replace(/\s/g, '').toUpperCase().match(/.{1,4}/g)?.join(' ') ?? ''

export default function ProfileBankTab({ ownMember }: Props) {
  const [drafts, setDrafts] = useState<Record<string, ChangeDraft>>({})
  const [iban, setIban] = useState('')
  const [accountHolder, setAccountHolder] = useState('')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  const [cancelErrors, setCancelErrors] = useState<Record<string, string>>({})

  const loadDrafts = (memberId: number) =>
    api.get(`/members/${memberId}/change-drafts`).then(r => {
      const list: ChangeDraft[] = r.data?.drafts ?? []
      const map: Record<string, ChangeDraft> = {}
      for (const d of list) {
        if (d.field_name === 'iban' || d.field_name === 'account_holder') map[d.field_name] = d
      }
      setDrafts(map)
      if (map.iban) setIban(map.iban.new_value ?? '')
      if (map.account_holder) setAccountHolder(map.account_holder.new_value ?? '')
    }).catch(() => {})

  useEffect(() => {
    if (!ownMember) return
    setIban(ownMember.iban ?? '')
    setAccountHolder(ownMember.account_holder ?? '')
    loadDrafts(ownMember.id)
  }, [ownMember?.id])

  if (!ownMember) return null

  const ibanDraft = drafts.iban ?? null
  const ahDraft = drafts.account_holder ?? null
  const ibanReadonly = !!ibanDraft
  const ahReadonly = !!ahDraft

  const ibanChanged = iban.replace(/\s/g, '') !== (ownMember.iban ?? '').replace(/\s/g, '')
  const ahChanged = accountHolder !== (ownMember.account_holder ?? '')
  const canSave = (!ibanReadonly && ibanChanged) || (!ahReadonly && ahChanged)

  const handleSave = async () => {
    setSaving(true)
    setError('')
    try {
      const reqs: Promise<unknown>[] = []
      if (!ibanReadonly && ibanChanged) {
        const raw = iban.replace(/\s/g, '').toUpperCase()
        if (!raw) { setError('IBAN darf nicht leer sein.'); setSaving(false); return }
        reqs.push(api.post(`/members/${ownMember.id}/change-request`, { field_name: 'iban', new_value: raw }))
      }
      if (!ahReadonly && ahChanged) {
        reqs.push(api.post(`/members/${ownMember.id}/change-request`, { field_name: 'account_holder', new_value: accountHolder }))
      }
      await Promise.all(reqs)
      await loadDrafts(ownMember.id)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Senden der Änderungsanfrage.')
    } finally {
      setSaving(false)
    }
  }

  const handleCancel = async (fieldName: string) => {
    const draft = drafts[fieldName]
    if (!draft) return
    setCancelErrors(e => ({ ...e, [fieldName]: '' }))
    try {
      await api.delete(`/members/${ownMember.id}/change-drafts/${draft.id}`)
      setDrafts(d => { const n = { ...d }; delete n[fieldName]; return n })
      if (fieldName === 'iban') setIban(ownMember.iban ?? '')
      if (fieldName === 'account_holder') setAccountHolder(ownMember.account_holder ?? '')
    } catch {
      setCancelErrors(e => ({ ...e, [fieldName]: 'Fehler beim Zurückziehen.' }))
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Bankdaten</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Kontoinhaber</label>
            {ahDraft && (
              <div className="mb-2 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
                <span className="font-medium">Ausstehend:</span>{' '}
                <span>{ahDraft.new_value ?? ''}</span>
                <button onClick={() => handleCancel('account_holder')} className="ml-3 text-brand-danger hover:underline text-sm">
                  Zurückziehen
                </button>
                {cancelErrors.account_holder && <span className="ml-2 text-brand-danger text-xs">{cancelErrors.account_holder}</span>}
              </div>
            )}
            <input
              type="text"
              value={accountHolder}
              readOnly={ahReadonly}
              onChange={e => setAccountHolder(e.target.value)}
              placeholder="Nicht hinterlegt"
              className={`w-full border border-brand-border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow ${
                ahReadonly ? 'bg-brand-surface-card text-brand-text-muted cursor-not-allowed' : 'text-brand-text'
              }`}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">IBAN</label>
            {ibanDraft && (
              <div className="mb-2 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
                <span className="font-medium">Ausstehend:</span>{' '}
                <span className="font-mono">{formatIBAN(ibanDraft.new_value ?? '')}</span>
                <button onClick={() => handleCancel('iban')} className="ml-3 text-brand-danger hover:underline text-sm">
                  Zurückziehen
                </button>
                {cancelErrors.iban && <span className="ml-2 text-brand-danger text-xs">{cancelErrors.iban}</span>}
              </div>
            )}
            <input
              type="text"
              value={ibanReadonly ? formatIBAN(iban) : iban}
              readOnly={ibanReadonly}
              onChange={e => {
                const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
                if (raw.length <= 22) setIban(raw)
              }}
              placeholder="DE89 3704 0044 0532 0130 00"
              className={`w-full border border-brand-border rounded-md px-3 py-2 text-sm font-mono tracking-wider focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow ${
                ibanReadonly ? 'bg-brand-surface-card text-brand-text-muted cursor-not-allowed' : 'text-brand-text'
              }`}
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
