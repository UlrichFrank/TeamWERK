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
  street?: string
  zip?: string
  city?: string
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
  const addressDraft = drafts.find(d => d.field_name === 'address')
  const ibanDraft = drafts.find(d => d.field_name === 'iban')

  const [ibanDisplay, setIbanDisplay] = useState(formatIBAN(form.iban || ''))
  const [ibanError, setIbanError] = useState('')

  useEffect(() => {
    setIbanDisplay(formatIBAN(form.iban || ''))
  }, [form.iban])

  const handleIbanChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value.replace(/[^A-Za-z0-9]/g, '').toUpperCase()
    if (raw.length > 22) return
    const display = raw.match(/.{1,4}/g)?.join(' ') ?? raw
    setIbanDisplay(display)
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
      {/* Adresse */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Adresse</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <input
              type="text"
              value={form.street || ''}
              onChange={e => onFormChange({ street: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                type="text"
                value={form.zip || ''}
                onChange={e => onFormChange({ zip: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                type="text"
                value={form.city || ''}
                onChange={e => onFormChange({ city: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
            </div>
          </div>

          {addressDraft && (
            <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg text-xs text-gray-700">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <span>
                  <span className="font-medium text-blue-700">Angeforderte Adressänderung:</span>{' '}
                  {addressDraft.new_value?.street}, {addressDraft.new_value?.zip} {addressDraft.new_value?.city}
                </span>
                <div className="flex gap-2">
                  <button
                    onClick={() => onDraftAccept(addressDraft.id)}
                    className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 font-medium"
                  >
                    ✓ Annehmen
                  </button>
                  <button
                    onClick={() => onDraftReject(addressDraft.id)}
                    className="px-2 py-1 bg-red-100 text-red-700 rounded hover:bg-red-200 font-medium"
                  >
                    ✗ Ablehnen
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Bankdaten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Bankdaten</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Kontoinhaber</label>
            <input
              type="text"
              value={form.account_holder || ''}
              onChange={e => onFormChange({ account_holder: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">IBAN</label>
            <input
              type="text"
              value={ibanDisplay}
              onChange={handleIbanChange}
              onBlur={handleIbanBlur}
              placeholder="DE89 3704 0044 0532 0130 00"
              maxLength={42}
              className={`w-full border rounded-md px-3 py-2 text-sm font-mono tracking-wider ${ibanError ? 'border-red-400 bg-red-50' : 'border-gray-300'}`}
            />
            {ibanError && <p className="text-xs text-red-600 mt-1">{ibanError}</p>}
          </div>

          {ibanDraft && (
            <div className="p-3 bg-blue-50 border border-blue-200 rounded-lg text-xs text-gray-700">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <span>
                  <span className="font-medium text-blue-700">Angeforderte IBAN:</span>{' '}
                  <span className="font-mono">{ibanDraft.new_value}</span>
                </span>
                <div className="flex gap-2">
                  <button
                    onClick={() => onDraftAccept(ibanDraft.id)}
                    className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 font-medium"
                  >
                    ✓ Annehmen
                  </button>
                  <button
                    onClick={() => onDraftReject(ibanDraft.id)}
                    className="px-2 py-1 bg-red-100 text-red-700 rounded hover:bg-red-200 font-medium"
                  >
                    ✗ Ablehnen
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Save Button */}
      {!isNew && (
        <div className="flex items-center gap-3">
          <button
            onClick={onSave}
            disabled={saving}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow disabled:opacity-40"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && <span className="text-sm text-green-600">Gespeichert</span>}
          {error && <span className="text-sm text-red-600">{error}</span>}
        </div>
      )}
    </div>
  )
}
