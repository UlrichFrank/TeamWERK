import { useRef, useState } from 'react'
import { api } from '../../lib/api'

interface Member {
  dsgvo_verarbeitung?: boolean
  dsgvo_verarbeitung_date?: string
  dsgvo_weitergabe?: boolean
  dsgvo_weitergabe_date?: string
  sepa_mandat?: boolean
  sepa_mandat_date?: string
  sepa_mandat_url?: string
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

export default function MemberDatenschutzTab({ form, isNew, drafts, onFormChange, onSave, saving, saved, error }: Props) {
  const sepaInputRef = useRef<HTMLInputElement>(null)
  const [sepaUploading, setSepaUploading] = useState(false)

  const dsgvoDraft = drafts.find(d => d.field_name === 'dsgvo')
  const sepaDraft = drafts.find(d => d.field_name === 'sepa_mandat')

  const handleSepaUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || isNew) return
    setSepaUploading(true)
    try {
      // Upload logic would go here
      // For now, just mark as uploaded
      onFormChange({ sepa_mandat: true })
    } catch {
      // Error handling
    } finally {
      setSepaUploading(false)
      if (sepaInputRef.current) sepaInputRef.current.value = ''
    }
  }

  return (
    <div className="space-y-6">
      {/* DSGVO */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Datenschutz (DSGVO)</h2>
        <div className="space-y-3">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={form.dsgvo_verarbeitung || false}
              onChange={e => onFormChange({ dsgvo_verarbeitung: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-gray-700">Datenverarbeitung eingewilligt</span>
            {dsgvoDraft && <span className="text-lg">⏳</span>}
          </label>
          {form.dsgvo_verarbeitung_date && (
            <p className="text-xs text-gray-500">seit {form.dsgvo_verarbeitung_date}</p>
          )}

          <label className="flex items-center gap-2 cursor-pointer mt-4">
            <input
              type="checkbox"
              checked={form.dsgvo_weitergabe || false}
              onChange={e => onFormChange({ dsgvo_weitergabe: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-gray-700">Datenweitergabe eingewilligt</span>
            {dsgvoDraft && <span className="text-lg">⏳</span>}
          </label>
          {form.dsgvo_weitergabe_date && (
            <p className="text-xs text-gray-500">seit {form.dsgvo_weitergabe_date}</p>
          )}

          {dsgvoDraft && (
            <div className="mt-3 text-xs text-gray-600 p-2 bg-blue-50 rounded">
              DSGVO-Änderung angefordert
            </div>
          )}
        </div>
      </div>

      {/* SEPA-Mandat */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">SEPA-Mandat</h2>
        <div className="space-y-3">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={form.sepa_mandat || false}
              onChange={e => onFormChange({ sepa_mandat: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-gray-700">Mandat erteilt</span>
            {sepaDraft && <span className="text-lg">⏳</span>}
          </label>
          {form.sepa_mandat_date && (
            <p className="text-xs text-gray-500">seit {form.sepa_mandat_date}</p>
          )}

          {!isNew && (
            <div className="mt-4">
              <label className="block text-sm font-medium text-gray-700 mb-1">Mandat-Dokument</label>
              <input ref={sepaInputRef} type="file" accept=".pdf" className="hidden" onChange={handleSepaUpload} />
              <button
                onClick={() => sepaInputRef.current?.click()}
                disabled={sepaUploading}
                className="bg-brand-yellow text-black px-3 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow disabled:opacity-40"
              >
                {sepaUploading ? 'Hochladen…' : 'Dokument hochladen'}
              </button>
            </div>
          )}

          {sepaDraft && (
            <div className="mt-3 text-xs text-gray-600 p-2 bg-blue-50 rounded">
              SEPA-Änderung angefordert
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
