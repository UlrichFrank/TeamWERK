import { useEffect, useRef, useState } from 'react'
import { api } from '../../lib/api'
import { CLUB_FUNCTION_OPTIONS } from '../../lib/constants'

interface Member {
  id?: number
  first_name: string
  last_name: string
  date_of_birth: string
  member_number: string
  pass_number: string
  jersey_number?: number
  position: string
  gender: string
  status: string
  club_functions?: string[]
  home_club?: string
  street?: string
  zip?: string
  city?: string
  photo_url?: string
  photo_visible?: boolean
}

interface Draft {
  id: number
  field_name: string
  old_value: any
  new_value: any
}

interface Props {
  form: Member
  memberId?: number
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

const GENDER_OPTIONS = [
  { value: 'm', label: 'männlich' },
  { value: 'f', label: 'weiblich' },
  { value: 'u', label: 'divers' },
]


const STATUS_OPTIONS = ['aktiv', 'verletzt', 'pausiert', 'passiv', 'ausgetreten']
const HANDBALL_POSITIONS = ['Torwart', 'Linksaußen', 'Rechtsaußen', 'Rückraum Links', 'Rückraum Mitte', 'Rückraum Rechts', 'Kreisläufer']

export default function MemberStammdatenTab({ form, memberId, isNew, drafts, onFormChange, onDraftAccept, onDraftReject, onSave, saving, saved, error }: Props) {
  const photoInputRef = useRef<HTMLInputElement>(null)
  const [photoUploading, setPhotoUploading] = useState(false)
  const [photoURL, setPhotoURL] = useState(form.photo_url || '')

  useEffect(() => {
    if (form.photo_url && form.photo_url !== photoURL) {
      setPhotoURL(form.photo_url)
    }
  }, [form.photo_url, photoURL])

  const togglePosition = (pos: string) => {
    const current = form.position ? form.position.split(',').filter(Boolean) : []
    const next = current.includes(pos) ? current.filter(p => p !== pos) : [...current, pos]
    onFormChange({ position: next.join(',') })
  }

  const selectedPositions = form.position ? form.position.split(',').filter(Boolean) : []
  const clubFunctions = form.club_functions ?? []
  const hasSpieler = clubFunctions.includes('spieler')

  const toggleClubFunction = (fn: string) => {
    const next = clubFunctions.includes(fn)
      ? clubFunctions.filter(f => f !== fn)
      : [...clubFunctions, fn]
    onFormChange({ club_functions: next })
  }

  const nameDraft = drafts.find(d => d.field_name === 'name')
  const profilDraft = drafts.find(d => d.field_name === 'profil')

  const handlePhotoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || isNew || !memberId) return
    setPhotoUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', file)
      const r = await api.post(`/upload/member-photo/${memberId}`, fd)
      setPhotoURL(r.data.photo_url || '')
      onFormChange({ photo_url: r.data.photo_url || '' })
    } catch {
      // error handled by parent
    } finally {
      setPhotoUploading(false)
      if (photoInputRef.current) photoInputRef.current.value = ''
    }
  }

  return (
    <div className="space-y-6">
      {/* Persönliche Daten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Persönliche Daten</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Vorname</label>
            <input
              type="text"
              value={form.first_name}
              onChange={e => onFormChange({ first_name: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Nachname</label>
            <input
              type="text"
              value={form.last_name}
              onChange={e => onFormChange({ last_name: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          {nameDraft && (
            <div className="col-span-2 p-3 bg-blue-50 border border-blue-200 rounded-lg text-xs text-gray-700">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <span>
                  <span className="font-medium text-blue-700">Angeforderte Namensänderung:</span>{' '}
                  {nameDraft.new_value?.first_name} {nameDraft.new_value?.last_name}
                </span>
                <div className="flex gap-2">
                  <button onClick={() => onDraftAccept(nameDraft.id)} className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 font-medium">✓ Annehmen</button>
                  <button onClick={() => onDraftReject(nameDraft.id)} className="px-2 py-1 bg-red-100 text-red-700 rounded hover:bg-red-200 font-medium">✗ Ablehnen</button>
                </div>
              </div>
            </div>
          )}
          {profilDraft && (
            <div className="col-span-2 p-3 bg-blue-50 border border-blue-200 rounded-lg text-xs text-gray-700">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <div>
                  <span className="font-medium text-blue-700">Angeforderte Profiländerung:</span>
                  <span className="ml-1">{profilDraft.new_value?.first_name} {profilDraft.new_value?.last_name}</span>
                  {profilDraft.new_value?.street && (
                    <span className="ml-2 text-gray-500">{profilDraft.new_value.street}, {profilDraft.new_value.zip} {profilDraft.new_value.city}</span>
                  )}
                </div>
                <div className="flex gap-2">
                  <button onClick={() => onDraftAccept(profilDraft.id)} className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 font-medium">✓ Annehmen</button>
                  <button onClick={() => onDraftReject(profilDraft.id)} className="px-2 py-1 bg-red-100 text-red-700 rounded hover:bg-red-200 font-medium">✗ Ablehnen</button>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Adresse */}
        <div className="mt-4 grid grid-cols-1 gap-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <input
              type="text"
              value={form.street || ''}
              onChange={e => onFormChange({ street: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                type="text"
                value={form.zip || ''}
                onChange={e => onFormChange({ zip: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                type="text"
                value={form.city || ''}
                onChange={e => onFormChange({ city: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
            </div>
          </div>
        </div>

        <div className="mt-4 grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Geburtsdatum</label>
            <input
              type="date"
              value={form.date_of_birth}
              onChange={e => onFormChange({ date_of_birth: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Mitgliedsnummer</label>
            <input
              type="text"
              value={form.member_number}
              onChange={e => onFormChange({ member_number: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          {hasSpieler && (
            <>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Passnummer</label>
                <input
                  type="text"
                  value={form.pass_number}
                  onChange={e => onFormChange({ pass_number: e.target.value })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Rückennummer</label>
                <input
                  type="number"
                  value={form.jersey_number ?? ''}
                  onChange={e => onFormChange({ jersey_number: e.target.value ? parseInt(e.target.value) : undefined })}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>
            </>
          )}
        </div>

        {/* Positionen — nur für Spieler */}
        {hasSpieler && (
          <div className="mt-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">Positionen</label>
            <div className="flex flex-wrap gap-2">
              {HANDBALL_POSITIONS.map(pos => (
                <button
                  key={pos}
                  type="button"
                  onClick={() => togglePosition(pos)}
                  className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                    selectedPositions.includes(pos)
                      ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                      : 'text-gray-600 border-gray-300 hover:border-brand-black'
                  }`}
                >
                  {pos}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Geschlecht — nur für Spieler */}
        {hasSpieler && (
          <div className="mt-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">Geschlecht</label>
            <div className="flex gap-2">
              {GENDER_OPTIONS.map(g => (
                <button
                  key={g.value}
                  type="button"
                  onClick={() => onFormChange({ gender: g.value })}
                  className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                    form.gender === g.value
                      ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                      : 'text-gray-600 border-gray-300'
                  }`}
                >
                  {g.label}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Status */}
        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">Status</label>
          <div className="flex gap-2 flex-wrap">
            {STATUS_OPTIONS.map(s => (
              <button
                key={s}
                type="button"
                onClick={() => onFormChange({ status: s })}
                className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                  form.status === s
                    ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                    : 'text-gray-600 border-gray-300'
                }`}
              >
                {s}
              </button>
            ))}
          </div>
        </div>

        {/* Vereinsfunktion */}
        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">Vereinsfunktion</label>
          <div className="flex flex-wrap gap-2">
            {CLUB_FUNCTION_OPTIONS.map(opt => (
              <label key={opt.value} className="flex items-center gap-2 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={clubFunctions.includes(opt.value)}
                  onChange={() => toggleClubFunction(opt.value)}
                  className="w-4 h-4 accent-brand-yellow"
                />
                <span className="text-sm text-brand-text">{opt.label}</span>
              </label>
            ))}
          </div>
        </div>

        {/* Stammverein */}
        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">Stammverein</label>
          <input
            type="text"
            value={form.home_club ?? ''}
            onChange={e => onFormChange({ home_club: e.target.value })}
            placeholder="z. B. TV Cannstatt"
            className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
          />
        </div>
      </div>

      {/* Foto */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Passfoto</h2>
        <div className="flex items-center gap-4">
          {photoURL && <img src={photoURL} alt="Passfoto" className="w-20 h-20 rounded-full object-cover" />}
          {!photoURL && <div className="w-20 h-20 rounded-full bg-gray-200 flex items-center justify-center text-gray-400 text-xs">Kein Bild</div>}
          {!isNew && (
            <>
              <input ref={photoInputRef} type="file" accept="image/jpeg,image/png,image/webp" className="hidden" onChange={handlePhotoUpload} />
              <button
                onClick={() => photoInputRef.current?.click()}
                disabled={photoUploading}
                className="bg-brand-yellow text-black px-3 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow disabled:opacity-40"
              >
                {photoUploading ? 'Hochladen…' : 'Foto hochladen'}
              </button>
            </>
          )}
        </div>
        <label className="flex items-center gap-2 cursor-pointer mt-4">
          <input
            type="checkbox"
            checked={form.photo_visible || false}
            onChange={e => onFormChange({ photo_visible: e.target.checked })}
            className="w-4 h-4 accent-brand-yellow"
          />
          <span className="text-sm text-gray-700">Sichtbar für Mitglieder</span>
        </label>
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
