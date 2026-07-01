import { useEffect, useRef, useState } from 'react'
import { ChevronDown, Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import { CLUB_FUNCTION_OPTIONS } from '../../lib/constants'
import { useAuth } from '../../contexts/AuthContext'
import ImageCropModal from '../ImageCropModal'

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
  home_club_id?: number | null
  home_club_name?: string
  zweitspielrecht?: boolean
  street?: string
  zip?: string
  city?: string
  join_date?: string
  exit_date?: string
  photo_url?: string
  photo_visible?: boolean
}

interface Draft {
  id: number
  field_name: string
  old_value: { first_name?: string; last_name?: string; street?: string; zip?: string; city?: string; [k: string]: unknown } | null
  new_value: { first_name?: string; last_name?: string; street?: string; zip?: string; city?: string; [k: string]: unknown } | null
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


const STATUS_OPTIONS = ['aktiv', 'verletzt', 'pausiert', 'passiv', 'honorar', 'anwaerter', 'ausgetreten']
const HANDBALL_POSITIONS = ['Torwart', 'Linksaußen', 'Rechtsaußen', 'Rückraum Links', 'Rückraum Mitte', 'Rückraum Rechts', 'Kreisläufer']

export default function MemberStammdatenTab({ form, memberId, isNew, drafts, onFormChange, onDraftAccept, onDraftReject, onSave, saving, saved, error }: Props) {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'
  const photoInputRef = useRef<HTMLInputElement>(null)
  const photoDropdownRef = useRef<HTMLDivElement>(null)
  const [photoUploading, setPhotoUploading] = useState(false)
  const [cropFile, setCropFile] = useState<File | null>(null)
  const [photoDropdown, setPhotoDropdown] = useState(false)
  const [photoURL, setPhotoURL] = useState(form.photo_url || '')
  const [stammvereine, setStammvereine] = useState<{ id: number; name: string; aktiv: boolean }[]>([])

  useEffect(() => {
    api.get('/stammvereine?include_inactive=1')
      .then(r => setStammvereine(r.data.items ?? []))
      .catch(() => setStammvereine([]))
  }, [])

  useEffect(() => {
    if (form.photo_url && form.photo_url !== photoURL) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
      setPhotoURL(form.photo_url)
    }
  }, [form.photo_url, photoURL])

  useEffect(() => {
    if (!photoDropdown) return
    const handler = (e: MouseEvent) => {
      if (photoDropdownRef.current && !photoDropdownRef.current.contains(e.target as Node))
        setPhotoDropdown(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [photoDropdown])

  const togglePosition = (pos: string) => {
    const current = form.position ? form.position.split(',').filter(Boolean) : []
    const next = current.includes(pos) ? current.filter(p => p !== pos) : [...current, pos]
    onFormChange({ position: next.join(',') })
  }

  const selectedPositions = form.position ? form.position.split(',').filter(Boolean) : []
  const clubFunctions = form.club_functions ?? []
  const hasSpieler = clubFunctions.includes('spieler')

  const isHonorar = form.status === 'honorar'

  const toggleClubFunction = (fn: string) => {
    if (isHonorar && fn !== 'trainer') return
    const next = clubFunctions.includes(fn)
      ? clubFunctions.filter(f => f !== fn)
      : [...clubFunctions, fn]
    onFormChange({ club_functions: next })
  }

  const nameDraft = drafts.find(d => d.field_name === 'name')
  const profilDraft = drafts.find(d => d.field_name === 'profil')

  const handlePhotoSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || isNew || !memberId) return
    setCropFile(file)
    if (photoInputRef.current) photoInputRef.current.value = ''
  }

  const handleCropConfirm = async (blob: Blob) => {
    if (!memberId) return
    setCropFile(null)
    setPhotoUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', blob, 'photo.jpg')
      const r = await api.post(`/upload/member-photo/${memberId}`, fd)
      setPhotoURL(r.data.photo_url || '')
      onFormChange({ photo_url: r.data.photo_url || '' })
    } catch {
      // error handled by parent
    } finally {
      setPhotoUploading(false)
    }
  }

  const handlePhotoDelete = async () => {
    if (!memberId) return
    setPhotoDropdown(false)
    try {
      await api.delete(`/upload/member-photo/${memberId}`)
      setPhotoURL('')
      onFormChange({ photo_url: '' })
    } catch {
      // error handled by parent
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
          {!isHonorar && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Mitgliedsnummer</label>
              {isNew ? (
                <p className="text-sm text-brand-text-muted px-3 py-2 border border-brand-border-subtle rounded-md bg-brand-surface-card">
                  Wird automatisch vergeben
                </p>
              ) : isAdmin ? (
                <input
                  type="text"
                  value={form.member_number}
                  onChange={e => onFormChange({ member_number: e.target.value })}
                  className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                />
              ) : (
                <p className="text-sm text-brand-text px-3 py-2 border border-brand-border-subtle rounded-md bg-brand-surface-card">
                  {form.member_number || '—'}
                </p>
              )}
            </div>
          )}
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
                onClick={() => {
                  const updates: Record<string, unknown> = { status: s }
                  if (s === 'honorar') {
                    updates.club_functions = clubFunctions.filter(f => f === 'trainer')
                    updates.member_number = ''
                    updates.pass_number = ''
                    updates.home_club = ''
                    updates.home_club_id = null
                  }
                  onFormChange(updates)
                }}
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

        {/* Eintritts-/Austrittsdatum (steuern die Beitrags-Halbierung im Beitragslauf) */}
        <div className="mt-4 grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Eintrittsdatum <span className="text-brand-danger">*</span>
            </label>
            <input
              type="date"
              required
              value={form.join_date ?? ''}
              onChange={e => onFormChange({ join_date: e.target.value })}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          {form.status === 'ausgetreten' && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Austrittsdatum <span className="text-brand-danger">*</span>
              </label>
              <input
                type="date"
                required
                value={form.exit_date ?? ''}
                onChange={e => onFormChange({ exit_date: e.target.value })}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              />
            </div>
          )}
        </div>

        {/* Vereinsfunktion */}
        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">Vereinsfunktion</label>
          <div className="flex flex-wrap gap-2">
            {CLUB_FUNCTION_OPTIONS.map(opt => {
              const disabled = isHonorar && opt.value !== 'trainer'
              return (
                <label key={opt.value} className={`flex items-center gap-2 select-none ${disabled ? 'opacity-30 cursor-not-allowed' : 'cursor-pointer'}`}>
                  <input
                    type="checkbox"
                    checked={clubFunctions.includes(opt.value)}
                    onChange={() => toggleClubFunction(opt.value)}
                    disabled={disabled}
                    className="w-4 h-4 accent-brand-yellow"
                  />
                  <span className="text-sm text-brand-text">{opt.label}</span>
                </label>
              )
            })}
          </div>
        </div>

        {/* Stammverein + Zweitspielrecht — nicht für Honorar-Mitglieder */}
        {!isHonorar && (
          <div className="mt-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">Stammverein</label>
            <select
              value={form.home_club_id ?? ''}
              onChange={e => onFormChange({ home_club_id: e.target.value === '' ? null : Number(e.target.value) })}
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
            >
              <option value="">Kein Stammverein</option>
              {stammvereine
                .filter(v => v.aktiv || v.id === form.home_club_id)
                .map(v => (
                  <option key={v.id} value={v.id}>
                    {v.name}{v.aktiv ? '' : ' (deaktiviert)'}
                  </option>
                ))}
            </select>
            <label className="flex items-center gap-2 cursor-pointer mt-2">
              <input
                type="checkbox"
                checked={form.zweitspielrecht || false}
                onChange={e => onFormChange({ zweitspielrecht: e.target.checked })}
                className="w-4 h-4 accent-brand-yellow"
              />
              <span className="text-sm text-brand-text">Zweitspielrecht</span>
            </label>
          </div>
        )}
      </div>

      {/* Foto */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Passfoto</h2>
        <div className="flex items-center gap-4">
          {photoURL && <img src={photoURL} alt="Passfoto" className="w-20 h-20 rounded-full object-cover" />}
          {!photoURL && <div className="w-20 h-20 rounded-full bg-gray-200 flex items-center justify-center text-gray-400 text-xs">Kein Bild</div>}
          {!isNew && (
            <>
              <input ref={photoInputRef} type="file" accept="image/*" className="hidden" onChange={handlePhotoSelect} />
              <div ref={photoDropdownRef} className="relative inline-flex">
                <button
                  onClick={() => photoInputRef.current?.click()}
                  disabled={photoUploading}
                  className={`bg-brand-yellow text-brand-black px-3 py-1.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed ${photoURL ? 'rounded-l-md border-r border-brand-black/20' : 'rounded-md'}`}
                >
                  {photoUploading ? 'Hochladen…' : 'Bild hochladen'}
                </button>
                {photoURL && (
                  <>
                    <button
                      onClick={() => setPhotoDropdown(v => !v)}
                      disabled={photoUploading}
                      aria-label="Weitere Optionen"
                      className="bg-brand-yellow text-brand-black rounded-r-md px-2 py-1.5 hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      <ChevronDown className="w-3.5 h-3.5" />
                    </button>
                    {photoDropdown && (
                      <div className="absolute left-0 top-full mt-1 w-36 bg-white border border-brand-border rounded-md shadow-lg z-20">
                        <button
                          onClick={handlePhotoDelete}
                          className="w-full text-left px-4 py-2.5 text-xs text-brand-danger hover:bg-brand-danger-light transition-colors flex items-center gap-2"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                          Bild löschen
                        </button>
                      </div>
                    )}
                  </>
                )}
              </div>
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

      <ImageCropModal
        file={cropFile}
        onConfirm={handleCropConfirm}
        onCancel={() => { setCropFile(null); if (photoInputRef.current) photoInputRef.current.value = '' }}
      />
    </div>
  )
}
