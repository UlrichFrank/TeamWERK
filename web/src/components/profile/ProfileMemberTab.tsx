import { useState, useEffect, FormEvent } from 'react'
import { api } from '../../lib/api'
import { Member, ChangeDraft } from '../../pages/ProfilePage'

interface Props {
  ownMember: Member | null
}

export default function ProfileMemberTab({ ownMember }: Props) {
  const [drafts, setDrafts] = useState<ChangeDraft[]>([])
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (ownMember) {
      api.get(`/members/${ownMember.id}/change-drafts`).then(r => {
        setDrafts(r.data?.drafts ?? [])
      })
    }
  }, [ownMember?.id])

  const getDraftForField = (fieldName: string) => drafts.find(d => d.field_name === fieldName)

  const handleCancelDraft = async (draftId: number) => {
    if (!ownMember) return
    try {
      await api.delete(`/members/${ownMember.id}/change-drafts/${draftId}`)
      setDrafts(drafts.filter(d => d.id !== draftId))
    } catch (err) {
      setError('Fehler beim Löschen der Änderungsanfrage')
    }
  }

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    if (!ownMember || drafts.length === 0) return

    setSaving(true)
    setError('')
    try {
      await api.post(`/members/${ownMember.id}/change-request`, {})
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  if (!ownMember) {
    return <div>Keine Mitgliedsdaten verfügbar</div>
  }

  const formatDate = (dateStr: string) => {
    if (!dateStr) return '–'
    return new Date(dateStr).toLocaleDateString('de-DE')
  }

  const nameDraft = getDraftForField('name')
  const addressDraft = getDraftForField('address')
  const emailDraft = getDraftForField('email')
  const ibanDraft = getDraftForField('iban')
  const photoDraft = getDraftForField('photo_url')
  const dsgvoDraft = getDraftForField('dsgvo')
  const sepaDraft = getDraftForField('sepa_mandat')

  return (
    <div className="space-y-6">
      {/* Persönliche Daten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Persönliche Daten</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
            <div className="flex items-center gap-2">
              <input type="text" value={ownMember.first_name} disabled className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
              {nameDraft && <span className="text-lg">⏳</span>}
            </div>
            {nameDraft && (
              <div className="mt-2 text-xs text-gray-600 p-2 bg-blue-50 rounded">
                Angefordert: {nameDraft.new_value?.first_name || '–'}
                <button onClick={() => handleCancelDraft(nameDraft.id)} className="ml-2 text-red-600 hover:text-red-800">Abbrechen</button>
              </div>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
            <div className="flex items-center gap-2">
              <input type="text" value={ownMember.last_name} disabled className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
              {nameDraft && <span className="text-lg">⏳</span>}
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum</label>
            <input type="text" value={formatDate(ownMember.date_of_birth)} disabled className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Passnummer</label>
            <input type="text" value={ownMember.pass_number} disabled className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Rückennummer</label>
            <input type="text" value={ownMember.jersey_number ?? '–'} disabled className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Position</label>
            <input type="text" value={ownMember.position ?? '–'} disabled className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
            <input type="text" value={ownMember.status ?? '–'} disabled className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
          </div>
        </div>
      </div>

      {/* Kontakt */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Kontakt</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Adresse</label>
            <div className="flex items-center gap-2">
              <input type="text" disabled value="Adresse (read-only)" className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
              {addressDraft && <span className="text-lg">⏳</span>}
            </div>
            {addressDraft && (
              <div className="mt-2 text-xs text-gray-600 p-2 bg-blue-50 rounded">
                Angefordert: {addressDraft.new_value?.street} {addressDraft.new_value?.house_number}, {addressDraft.new_value?.zip} {addressDraft.new_value?.city}
                <button onClick={() => handleCancelDraft(addressDraft.id)} className="ml-2 text-red-600 hover:text-red-800">Abbrechen</button>
              </div>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Telefonnummern</label>
            <div className="flex items-center gap-2">
              <input type="text" disabled value="Telefon (read-only)" className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
              {getDraftForField('phones') && <span className="text-lg">⏳</span>}
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
            <div className="flex items-center gap-2">
              <input type="text" disabled value="E-Mail (read-only)" className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600" />
              {emailDraft && <span className="text-lg">⏳</span>}
            </div>
          </div>
        </div>
      </div>

      {/* Passfoto */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Passfoto</h2>
        <div className="flex items-center gap-4">
          <div className="w-20 h-20 rounded-full bg-gray-200 flex items-center justify-center text-gray-400 text-xs">Foto</div>
          {photoDraft && <span className="text-lg">⏳</span>}
        </div>
        {photoDraft && (
          <div className="mt-2 text-xs text-gray-600 p-2 bg-blue-50 rounded">
            Foto-Änderung angefordert
            <button onClick={() => handleCancelDraft(photoDraft.id)} className="ml-2 text-red-600 hover:text-red-800">Abbrechen</button>
          </div>
        )}
      </div>

      {/* Bankdaten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Bankdaten</h2>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">IBAN</label>
          <div className="flex items-center gap-2">
            <input type="text" disabled value="IBAN (read-only)" className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm bg-gray-100 text-gray-600 font-mono" />
            {ibanDraft && <span className="text-lg">⏳</span>}
          </div>
          {ibanDraft && (
            <div className="mt-2 text-xs text-gray-600 p-2 bg-blue-50 rounded">
              Angefordert: {ibanDraft.new_value}
              <button onClick={() => handleCancelDraft(ibanDraft.id)} className="ml-2 text-red-600 hover:text-red-800">Abbrechen</button>
            </div>
          )}
        </div>
      </div>

      {/* Datenschutz */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Datenschutz</h2>
        <div className="space-y-2">
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" disabled className="w-4 h-4 accent-brand-yellow" />
            <span className="text-sm text-gray-700">Datenverarbeitung {dsgvoDraft && <span className="text-lg">⏳</span>}</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" disabled className="w-4 h-4 accent-brand-yellow" />
            <span className="text-sm text-gray-700">Datenweitergabe {dsgvoDraft && <span className="text-lg">⏳</span>}</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" disabled className="w-4 h-4 accent-brand-yellow" />
            <span className="text-sm text-gray-700">SEPA-Mandat {sepaDraft && <span className="text-lg">⏳</span>}</span>
          </label>
          {dsgvoDraft && (
            <div className="mt-2 text-xs text-gray-600 p-2 bg-blue-50 rounded">
              DSGVO-Änderung angefordert
              <button onClick={() => handleCancelDraft(dsgvoDraft.id)} className="ml-2 text-red-600 hover:text-red-800">Abbrechen</button>
            </div>
          )}
          {sepaDraft && (
            <div className="mt-2 text-xs text-gray-600 p-2 bg-blue-50 rounded">
              SEPA-Änderung angefordert
              <button onClick={() => handleCancelDraft(sepaDraft.id)} className="ml-2 text-red-600 hover:text-red-800">Abbrechen</button>
            </div>
          )}
        </div>
      </div>

      {/* Save Button */}
      {drafts.length > 0 && (
        <div className="flex items-center gap-3">
          <button
            onClick={handleSave}
            disabled={saving}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
          >
            {saving ? 'Speichern…' : 'Änderungen speichern'}
          </button>
          {saved && <span className="text-sm text-green-600">Gespeichert</span>}
          {error && <span className="text-sm text-red-600">{error}</span>}
        </div>
      )}
    </div>
  )
}
