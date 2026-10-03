import { useEffect, useRef, useState } from 'react'
import { Check, ChevronDown, Trash2, X } from 'lucide-react'
import { api } from '../../lib/api'
import { CLUB_FUNCTION_OPTIONS, EXTERN_CLUB_FUNCTIONS } from '../../lib/constants'
import { useAuth } from '../../contexts/AuthContext'
import ImageCropModal from '../ImageCropModal'
import { BTN_PRIMARY, BTN_PRIMARY_SPLIT_CARET, BTN_PRIMARY_SPLIT_MAIN, INPUT, LABEL } from '../../lib/buttonStyles'
import { MENU_ITEM_DANGER, SECTION_TITLE } from '../../lib/typography'

interface Member {
  id?: number
  first_name: string
  last_name: string
  date_of_birth: string
  member_number: string
  pass_number: string
  handball_360_id: string
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
  user_id?: number
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


const STATUS_OPTIONS = ['aktiv', 'verletzt', 'pausiert', 'passiv', 'extern', 'anwaerter', 'foerderkind', 'ausgetreten']
// Nicht-beitragspflichtige Talent-Status ohne Eintrittsdatum-Zwang (kein Beitrag → kein Beitrags-relevantes join_date).
const STATUS_WITHOUT_JOIN_DATE = ['anwaerter', 'foerderkind']
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
  const [photoError, setPhotoError] = useState('')
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

  const isExtern = form.status === 'extern'
  const joinDateRequired = !STATUS_WITHOUT_JOIN_DATE.includes(form.status)

  const toggleClubFunction = (fn: string) => {
    if (isExtern && !EXTERN_CLUB_FUNCTIONS.includes(fn)) return
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
    setPhotoError('')
    try {
      const fd = new FormData()
      fd.append('file', blob, 'photo.jpg')
      const r = await api.post(`/upload/member-photo/${memberId}`, fd)
      setPhotoURL(r.data.photo_url || '')
      onFormChange({ photo_url: r.data.photo_url || '' })
    } catch (err) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 409) {
        setPhotoError('Mitglied hat keinen Account — Foto ist erst nach der Kontoanlage möglich.')
      } else {
        setPhotoError('Foto-Upload fehlgeschlagen.')
      }
    } finally {
      setPhotoUploading(false)
    }
  }

  const handlePhotoDelete = async () => {
    if (!memberId) return
    setPhotoDropdown(false)
    setPhotoError('')
    try {
      await api.delete(`/upload/member-photo/${memberId}`)
      setPhotoURL('')
      onFormChange({ photo_url: '' })
    } catch (err) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 409) {
        setPhotoError('Mitglied hat keinen Account — kein Foto zum Löschen.')
      } else {
        setPhotoError('Foto konnte nicht gelöscht werden.')
      }
    }
  }

  return (
    <div className="space-y-6">
      {/* Persönliche Daten */}
      <div className="bg-brand-surface-card rounded-xl shadow-sm border-t-4 border-brand-yellow transform-gpu p-6">
        <h2 className={`${SECTION_TITLE} mb-4`}>Persönliche Daten</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className={LABEL}>Vorname</label>
            <input
              type="text"
              value={form.first_name}
              onChange={e => onFormChange({ first_name: e.target.value })}
              className={INPUT}
            />
          </div>
          <div>
            <label className={LABEL}>Nachname</label>
            <input
              type="text"
              value={form.last_name}
              onChange={e => onFormChange({ last_name: e.target.value })}
              className={INPUT}
            />
          </div>
          {nameDraft && (
            <div className="col-span-2 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-xs text-brand-text-muted">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <span>
                  <span className="font-medium text-brand-text">Angeforderte Namensänderung:</span>{' '}
                  {nameDraft.new_value?.first_name} {nameDraft.new_value?.last_name}
                </span>
                <div className="flex gap-2">
                  <button onClick={() => onDraftAccept(nameDraft.id)} className="inline-flex items-center gap-1 px-2 py-1 bg-brand-success-light text-brand-success rounded-md hover:bg-brand-success/20 font-medium transition-colors"><Check className="w-4 h-4" />Annehmen</button>
                  <button onClick={() => onDraftReject(nameDraft.id)} className="inline-flex items-center gap-1 px-2 py-1 bg-brand-danger-light text-brand-danger rounded-md hover:bg-brand-danger/20 font-medium transition-colors"><X className="w-4 h-4" />Ablehnen</button>
                </div>
              </div>
            </div>
          )}
          {profilDraft && (
            <div className="col-span-2 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-xs text-brand-text-muted">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <div>
                  <span className="font-medium text-brand-text">Angeforderte Profiländerung:</span>
                  <span className="ml-1">{profilDraft.new_value?.first_name} {profilDraft.new_value?.last_name}</span>
                  {profilDraft.new_value?.street && (
                    <span className="ml-2 text-brand-text-muted">{profilDraft.new_value.street}, {profilDraft.new_value.zip} {profilDraft.new_value.city}</span>
                  )}
                </div>
                <div className="flex gap-2">
                  <button onClick={() => onDraftAccept(profilDraft.id)} className="inline-flex items-center gap-1 px-2 py-1 bg-brand-success-light text-brand-success rounded-md hover:bg-brand-success/20 font-medium transition-colors"><Check className="w-4 h-4" />Annehmen</button>
                  <button onClick={() => onDraftReject(profilDraft.id)} className="inline-flex items-center gap-1 px-2 py-1 bg-brand-danger-light text-brand-danger rounded-md hover:bg-brand-danger/20 font-medium transition-colors"><X className="w-4 h-4" />Ablehnen</button>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Adresse */}
        <div className="mt-4 grid grid-cols-1 gap-3">
          <div>
            <label className={LABEL}>Straße</label>
            <input
              type="text"
              value={form.street || ''}
              onChange={e => onFormChange({ street: e.target.value })}
              className={INPUT}
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className={LABEL}>PLZ</label>
              <input
                type="text"
                value={form.zip || ''}
                onChange={e => onFormChange({ zip: e.target.value })}
                className={INPUT}
              />
            </div>
            <div className="col-span-2">
              <label className={LABEL}>Ort</label>
              <input
                type="text"
                value={form.city || ''}
                onChange={e => onFormChange({ city: e.target.value })}
                className={INPUT}
              />
            </div>
          </div>
        </div>

        <div className="mt-4 grid grid-cols-2 gap-4">
          <div>
            <label className={LABEL}>Geburtsdatum</label>
            <input
              type="date"
              value={form.date_of_birth}
              onChange={e => onFormChange({ date_of_birth: e.target.value })}
              className={INPUT}
            />
          </div>
          {!isExtern && (
            <div>
              <label className={LABEL}>Mitgliedsnummer</label>
              {isNew ? (
                <p className="text-sm text-brand-text-muted px-3 py-2 border border-brand-border-subtle rounded-md bg-brand-surface-card">
                  Wird automatisch vergeben
                </p>
              ) : isAdmin ? (
                <input
                  type="text"
                  value={form.member_number}
                  onChange={e => onFormChange({ member_number: e.target.value })}
                  className={INPUT}
                />
              ) : (
                <p className="text-sm text-brand-text px-3 py-2 border border-brand-border-subtle rounded-md bg-brand-surface-card">
                  {form.member_number || '—'}
                </p>
              )}
            </div>
          )}
          <div>
            <label className={LABEL}>Handball 360 ID</label>
            <input
              type="text"
              value={form.handball_360_id}
              onChange={e => onFormChange({ handball_360_id: e.target.value })}
              className={INPUT}
            />
          </div>
          {hasSpieler && (
            <>
              <div>
                <label className={LABEL}>Passnummer</label>
                <input
                  type="text"
                  value={form.pass_number}
                  onChange={e => onFormChange({ pass_number: e.target.value })}
                  className={INPUT}
                />
              </div>
              <div>
                <label className={LABEL}>Rückennummer</label>
                <input
                  type="number"
                  value={form.jersey_number ?? ''}
                  onChange={e => onFormChange({ jersey_number: e.target.value ? parseInt(e.target.value) : undefined })}
                  className={INPUT}
                />
              </div>
            </>
          )}
        </div>

        {/* Positionen — nur für Spieler */}
        {hasSpieler && (
          <div className="mt-4">
            <label className="block text-sm font-medium text-brand-text-muted mb-2">Positionen</label>
            <div className="flex flex-wrap gap-2">
              {HANDBALL_POSITIONS.map(pos => (
                <button
                  key={pos}
                  type="button"
                  onClick={() => togglePosition(pos)}
                  className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                    selectedPositions.includes(pos)
                      ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                      : 'text-brand-text-muted border-brand-border hover:border-brand-black'
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
            <label className="block text-sm font-medium text-brand-text-muted mb-2">Geschlecht</label>
            <div className="flex gap-2">
              {GENDER_OPTIONS.map(g => (
                <button
                  key={g.value}
                  type="button"
                  onClick={() => onFormChange({ gender: g.value })}
                  className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                    form.gender === g.value
                      ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                      : 'text-brand-text-muted border-brand-border'
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
          <label className="block text-sm font-medium text-brand-text-muted mb-2">Status</label>
          <div className="flex gap-2 flex-wrap">
            {STATUS_OPTIONS.map(s => (
              <button
                key={s}
                type="button"
                onClick={() => {
                  const updates: Record<string, unknown> = { status: s }
                  if (s === 'extern') {
                    updates.club_functions = clubFunctions.filter(f => EXTERN_CLUB_FUNCTIONS.includes(f))
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
                    : 'text-brand-text-muted border-brand-border'
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
            <label className={LABEL}>
              Eintrittsdatum {joinDateRequired && <span className="text-brand-danger">*</span>}
            </label>
            <input
              type="date"
              required={joinDateRequired}
              value={form.join_date ?? ''}
              onChange={e => onFormChange({ join_date: e.target.value })}
              className={INPUT}
            />
          </div>
          {form.status === 'ausgetreten' && (
            <div>
              <label className={LABEL}>
                Austrittsdatum <span className="text-brand-danger">*</span>
              </label>
              <input
                type="date"
                required
                value={form.exit_date ?? ''}
                onChange={e => onFormChange({ exit_date: e.target.value })}
                className={INPUT}
              />
            </div>
          )}
        </div>

        {/* Vereinsfunktion */}
        <div className="mt-4">
          <label className="block text-sm font-medium text-brand-text-muted mb-2">Vereinsfunktion</label>
          <div className="flex flex-wrap gap-2">
            {CLUB_FUNCTION_OPTIONS.map(opt => {
              const disabled = isExtern && !EXTERN_CLUB_FUNCTIONS.includes(opt.value)
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

        {/* Stammverein + Zweitspielrecht — nicht für Extern-Mitglieder */}
        {!isExtern && (
          <div className="mt-4">
            <label className={LABEL}>Stammverein</label>
            <select
              value={form.home_club_id ?? ''}
              onChange={e => onFormChange({ home_club_id: e.target.value === '' ? null : Number(e.target.value) })}
              className={INPUT}
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
      <div className="bg-brand-surface-card rounded-xl shadow-sm border-t-4 border-brand-yellow transform-gpu p-6">
        <h2 className={`${SECTION_TITLE} mb-4`}>Passfoto</h2>
        <div className="flex items-center gap-4">
          {photoURL && <img src={photoURL} alt="Passfoto" className="w-20 h-20 rounded-full object-cover" />}
          {!photoURL && <div className="w-20 h-20 rounded-full bg-brand-border-subtle flex items-center justify-center text-brand-text-subtle text-xs">Kein Bild</div>}
          {!isNew && !form.user_id && (
            <div className="text-sm text-brand-text-muted">
              Foto ist erst verfügbar, sobald ein Nutzer-Konto verknüpft ist.
            </div>
          )}
          {!isNew && form.user_id && (
            <>
              <input ref={photoInputRef} type="file" accept="image/*" className="hidden" onChange={handlePhotoSelect} />
              <div ref={photoDropdownRef} className="relative inline-flex">
                <button
                  onClick={() => photoInputRef.current?.click()}
                  disabled={photoUploading}
                  className={photoURL ? BTN_PRIMARY_SPLIT_MAIN : BTN_PRIMARY}
                >
                  {photoUploading ? 'Hochladen…' : 'Bild hochladen'}
                </button>
                {photoURL && (
                  <>
                    <button
                      onClick={() => setPhotoDropdown(v => !v)}
                      disabled={photoUploading}
                      aria-label="Weitere Optionen"
                      className={BTN_PRIMARY_SPLIT_CARET}
                    >
                      <ChevronDown className="w-3.5 h-3.5" />
                    </button>
                    {photoDropdown && (
                      <div role="menu" className="absolute left-0 top-full mt-1 w-36 bg-white border border-brand-border rounded-md shadow-lg z-20">
                        <button
                          onClick={handlePhotoDelete}
                          role="menuitem"
                          className={MENU_ITEM_DANGER}
                        >
                          <Trash2 className="w-4 h-4 shrink-0" />
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
        {photoError && (
          <div className="mt-3 p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{photoError}</div>
        )}
        {form.user_id && (
          <label className="flex items-center gap-2 cursor-pointer mt-4">
            <input
              type="checkbox"
              checked={form.photo_visible || false}
              onChange={e => onFormChange({ photo_visible: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text-muted">Sichtbar für Mitglieder</span>
          </label>
        )}
      </div>

      {/* Save Button */}
      {!isNew && (
        <div className="flex items-center gap-3">
          <button
            onClick={onSave}
            disabled={saving}
            className={BTN_PRIMARY}
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && <span className="text-sm text-brand-success">Gespeichert</span>}
          {error && <span className="text-sm text-brand-danger">{error}</span>}
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
