import { useState, useEffect } from 'react'
import { api } from '../../lib/api'
import { encryptBankData } from '../../lib/bankCrypto'
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
      setBankdatenDraft(list.find(d => d.field_name === 'bankdaten') ?? null)
    }).catch(() => {})

  useEffect(() => {
    if (!ownMember) return
    loadDrafts(ownMember.id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ownMember?.id])

  if (!ownMember) return null

  const canSave = iban.trim() !== '' || accountHolder.trim() !== ''

  const handleSave = async () => {
    setSaving(true)
    setError('')
    try {
      const raw = iban.replace(/\s/g, '').toUpperCase()
      if (!raw) { setError('IBAN darf nicht leer sein.'); setSaving(false); return }
      const env = await encryptBankData({ iban: raw, account_holder: accountHolder })
      await api.post(`/members/${ownMember.id}/change-request`, {
        field_name: 'bankdaten',
        new_value: env,
      })
      setIban('')
      setAccountHolder('')
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
    } catch {
      setCancelError('Fehler beim Zurückziehen.')
    }
  }

  const formatDate = (iso: string | undefined) => {
    if (!iso) return ''
    const [y, m, day] = iso.slice(0, 10).split('-')
    return `${day}.${m}.${y}`
  }

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Bankdaten</h2>

        {/* Statusanzeige */}
        <div className="space-y-2 mb-6">
          <div className="flex items-center gap-2 text-sm">
            <span className="text-brand-text-muted w-36">Bankverbindung:</span>
            {ownMember.has_bank_data
              ? <span className="text-green-700 font-medium">hinterlegt</span>
              : <span className="text-brand-text-subtle">nicht hinterlegt</span>
            }
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="text-brand-text-muted w-36">SEPA-Mandat:</span>
            {ownMember.sepa_mandat
              ? <span className="text-green-700 font-medium">
                  hinterlegt{ownMember.sepa_mandat_date ? ` (${formatDate(ownMember.sepa_mandat_date)})` : ''}
                </span>
              : <span className="text-brand-text-subtle">nicht hinterlegt</span>
            }
          </div>
        </div>

        {/* Pending-Draft-Hinweis */}
        {bankdatenDraft && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            Änderungsanfrage ausstehend — wird vom Verein geprüft.{' '}
            <button onClick={handleCancel} className="text-brand-danger hover:underline">
              Zurückziehen
            </button>
            {cancelError && <span className="ml-2 text-brand-danger text-xs">{cancelError}</span>}
          </div>
        )}

        {/* Änderungsformular */}
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Neuer Kontoinhaber</label>
            <input
              type="text"
              value={accountHolder}
              onChange={e => setAccountHolder(e.target.value)}
              placeholder="Vor- und Nachname"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Neue IBAN</label>
            <input
              type="text"
              value={iban}
              onChange={e => {
                const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
                if (raw.length <= 22) setIban(raw)
              }}
              placeholder="DE89 3704 0044 0532 0130 00"
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text font-mono tracking-wider placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
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
