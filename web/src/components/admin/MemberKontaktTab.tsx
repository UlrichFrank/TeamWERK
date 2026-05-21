interface Member {
  street?: string
  zip?: string
  city?: string
  email?: string
  iban?: string
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
  onSave: () => Promise<void>
  saving: boolean
  saved: boolean
  error: string
}

export default function MemberKontaktTab({ form, isNew, drafts, onFormChange, onSave, saving, saved, error }: Props) {
  const addressDraft = drafts.find(d => d.field_name === 'address')

  const handleAddressChange = (updates: any) => {
    onFormChange(updates)
  }

  return (
    <div className="space-y-6">
      {/* Adresse */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Adresse</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <div className="flex items-center gap-2">
              <input
                type="text"
                value={form.street || ''}
                onChange={e => handleAddressChange({ street: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
              {addressDraft && <span className="text-lg">⏳</span>}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                type="text"
                value={form.zip || ''}
                onChange={e => handleAddressChange({ zip: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                type="text"
                value={form.city || ''}
                onChange={e => handleAddressChange({ city: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
            </div>
          </div>
          {addressDraft && (
            <div className="mt-2 text-xs text-gray-600 p-2 bg-blue-50 rounded">
              Angefordert: {addressDraft.new_value?.street} {addressDraft.new_value?.zip} {addressDraft.new_value?.city}
            </div>
          )}
        </div>
      </div>

      {/* Kontaktdaten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Kontaktdaten</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <input
              type="email"
              value={form.email || ''}
              onChange={e => onFormChange({ email: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
        </div>
      </div>

      {/* Bankdaten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Bankdaten</h2>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">IBAN</label>
          <input
            type="text"
            value={form.iban || ''}
            onChange={e => onFormChange({ iban: e.target.value })}
            placeholder="DE__ ____ ____ ____ ____ ____"
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
          />
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
