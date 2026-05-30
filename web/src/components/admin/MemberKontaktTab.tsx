import { useEffect, useState } from 'react'

const formatIBAN = (raw: string) =>
  raw.replace(/\s/g, '').toUpperCase().match(/.{1,4}/g)?.join(' ') ?? ''

const validateIBAN = (iban: string): boolean => {
  const clean = iban.replace(/\s/g, '').toUpperCase()
  if (!/^DE[0-9]{20}$/.test(clean)) return false
  const rearranged = clean.slice(4) + clean.slice(0, 4)
  const numeric = rearranged.split('').map(c =>
    c >= 'A' ? (c.charCodeAt(0) - 55).toString() : c
  ).join('')
  let remainder = 0
  for (const ch of numeric) remainder = (remainder * 10 + parseInt(ch)) % 97
  return remainder === 1
}

interface Member {
  iban?: string
  account_holder?: string
}

interface Draft {
  id: number
  field_name: string
  old_value: any
  new_value: any
}

interface Props {
  form: Member
  isNew: boolean
  drafts: Draft[]
  onFormChange: (updates: Partial<Member>) => void
  onDraftAccept: (draftId: number) => Promise<void>
  onDraftReject: (draftId: number) => Promise<void>
  onSave: () => Promise<void>
  saving: boolean
  saved: boolean
  error: string
}

export default function MemberKontaktTab({ form, isNew, drafts, onFormChange, onDraftAccept, onDraftReject, onSave, saving, saved, error }: Props) {
  const bankdatenDraft = drafts.find(d => d.field_name === 'bankdaten') ?? null

  const [ibanDisplay, setIbanDisplay] = useState(formatIBAN(form.iban || ''))
  const [ibanError, setIbanError] = useState('')

  useEffect(() => {
    setIbanDisplay(formatIBAN(form.iban || ''))
  }, [form.iban])

  const handleIbanChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
    if (raw.length > 22) return
    setIbanDisplay(raw.match(/.{1,4}/g)?.join(' ') ?? raw)
    setIbanError('')
    onFormChange({ iban: raw })
  }

  const handleIbanBlur = () => {
    const raw = ibanDisplay.replace(/\s/g, '')
    if (raw && !validateIBAN(raw)) setIbanError('Ungültige IBAN')
    else setIbanError('')
  }

  return (
    <div className="space-y-6">
      {/* Bankdaten */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Bankdaten</h2>

        {bankdatenDraft && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            <div className="flex items-start justify-between gap-2 flex-wrap">
              <div className="space-y-1">
                <p className="font-medium mb-1">Angeforderte Bankdaten:</p>
                {bankdatenDraft.new_value?.account_holder && (
                  <p className="text-xs">
                    <span className="text-brand-text-muted">Kontoinhaber:</span>{' '}
                    {bankdatenDraft.new_value.account_holder}
                  </p>
                )}
                {bankdatenDraft.new_value?.iban && (
                  <p className="text-xs font-mono">
                    <span className="text-brand-text-muted not-italic">IBAN:</span>{' '}
                    {formatIBAN(bankdatenDraft.new_value.iban)}
                  </p>
                )}
              </div>
              <div className="flex gap-2 shrink-0">
                <button
                  onClick={() => onDraftAccept(bankdatenDraft.id)}
                  className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 text-xs font-medium"
                >
                  Annehmen
                </button>
                <button
                  onClick={() => onDraftReject(bankdatenDraft.id)}
                  className="px-2 py-1 bg-brand-danger-light text-brand-danger rounded hover:bg-red-200 text-xs font-medium"
                >
                  Ablehnen
                </button>
              </div>
            </div>
          </div>
        )}

        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Kontoinhaber</label>
            <input
              type="text"
              value={form.account_holder || ''}
              onChange={e => onFormChange({ account_holder: e.target.value })}
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">IBAN</label>
            <input
              type="text"
              value={ibanDisplay}
              onChange={handleIbanChange}
              onBlur={handleIbanBlur}
              placeholder="DE89 3704 0044 0532 0130 00"
              maxLength={42}
              className={`w-full border rounded-md px-3 py-2 text-sm font-mono tracking-wider focus:outline-none focus:ring-2 focus:ring-brand-yellow ${
                ibanError ? 'border-brand-danger bg-brand-danger-light' : 'border-brand-border'
              }`}
            />
            {ibanError && <p className="text-xs text-brand-danger mt-1">{ibanError}</p>}
          </div>
        </div>
      </div>

      {!isNew && (
        <div className="flex items-center gap-3">
          <button
            onClick={onSave}
            disabled={saving}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && <span className="text-sm text-green-600">Gespeichert</span>}
          {error && <span className="text-sm text-brand-danger">{error}</span>}
        </div>
      )}
    </div>
  )
}
