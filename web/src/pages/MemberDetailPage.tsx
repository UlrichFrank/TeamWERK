import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface AddressStored {
  street: string; zip: string; city: string
}

interface Member {
  id: number; first_name: string; last_name: string
  date_of_birth: string; member_number: string; pass_number: string
  jersey_number?: number; position: string; gender: string; status: string; user_id?: number
  club_function?: string
  // Extended
  street?: string; zip?: string; city?: string
  join_date?: string; iban?: string
  photo_url?: string; photo_visible?: boolean
  dsgvo_verarbeitung?: boolean; dsgvo_verarbeitung_date?: string
  dsgvo_weitergabe?: boolean; dsgvo_weitergabe_date?: string
  sepa_mandat?: boolean; sepa_mandat_date?: string; sepa_mandat_url?: string
  address_source?: string; address_conflict?: boolean
  member_address_stored?: AddressStored
}

const GENDER_OPTIONS = [
  { value: 'm', label: 'männlich' },
  { value: 'f', label: 'weiblich' },
  { value: 'u', label: 'divers' },
]

const CLUB_FUNCTION_OPTIONS = [
  { value: '', label: '– keine –' },
  { value: 'trainer', label: 'Trainer' },
  { value: 'vorstand', label: 'Vorstand' },
  { value: 'vorstand_beisitzer', label: 'Vorstands-Beisitzer' },
]

interface User { id: number; name: string; email: string; role: string }

const STATUS_OPTIONS = ['aktiv', 'verletzt', 'pausiert', 'passiv', 'ausgetreten']
const HANDBALL_POSITIONS = ['Torwart', 'Linksaußen', 'Rechtsaußen', 'Rückraum Links', 'Rückraum Mitte', 'Rückraum Rechts', 'Kreisläufer']

function AddressConflictTooltip({ stored }: { stored: AddressStored }) {
  const [visible, setVisible] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setVisible(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  return (
    <div ref={ref} className="relative inline-block ml-1">
      <button
        type="button"
        onClick={() => setVisible(v => !v)}
        title="Adresse weicht von Mitgliedsdaten ab"
        className="text-amber-500 hover:text-amber-600 text-base leading-none"
      >
        ⚠
      </button>
      {visible && (
        <div className="absolute left-0 top-6 z-20 bg-white border border-amber-300 rounded-lg shadow-lg p-3 text-xs text-gray-700 min-w-56">
          <p className="font-semibold text-amber-600 mb-2">Adresse weicht ab</p>
          <p className="font-medium text-gray-500 mb-0.5">Am Mitglied gespeichert:</p>
          <p>{stored.street}</p>
          <p>{stored.zip} {stored.city}</p>
        </div>
      )}
    </div>
  )
}

export default function MemberDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const isNew = id === 'neu'
  const isAdmin = user?.role === 'admin'

  const [form, setForm] = useState<Omit<Member, 'id'>>({
    first_name: '', last_name: '', date_of_birth: '', member_number: '', pass_number: '',
    jersey_number: undefined, position: '', gender: 'u', status: 'aktiv', club_function: '',
    street: '', zip: '', city: '', join_date: '', iban: '',
    photo_visible: false,
    dsgvo_verarbeitung: false, dsgvo_verarbeitung_date: '',
    dsgvo_weitergabe: false, dsgvo_weitergabe_date: '',
    sepa_mandat: false, sepa_mandat_date: '',
  })
  const [addressSource, setAddressSource] = useState('')
  const [addressConflict, setAddressConflict] = useState(false)
  const [memberAddressStored, setMemberAddressStored] = useState<AddressStored | null>(null)
  const [photoURL, setPhotoURL] = useState('')
  const [sepaMandatURL, setSepaMandatURL] = useState('')
  const [users, setUsers] = useState<User[]>([])
  const [selectedParentUser, setSelectedParentUser] = useState('')
  const [linkedParents, setLinkedParents] = useState<User[]>([])
  const [selectedLinkedUser, setSelectedLinkedUser] = useState('')
  const [currentUserID, setCurrentUserID] = useState<number | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')
  const [removingParent, setRemovingParent] = useState<Record<number, boolean>>({})
  const [photoUploading, setPhotoUploading] = useState(false)
  const [sepaUploading, setSepaUploading] = useState(false)
  const photoInputRef = useRef<HTMLInputElement>(null)
  const sepaInputRef = useRef<HTMLInputElement>(null)

  const loadLinkedParents = () => {
    if (isAdmin && !isNew && id) {
      api.get(`/admin/members/${id}/parents`).then(r => setLinkedParents(r.data ?? []))
    }
  }

  useEffect(() => {
    if (isAdmin) api.get('/admin/users').then(r => setUsers(r.data.items ?? []))
    if (!isNew && id) {
      api.get(`/members/${id}`).then(r => {
        const m: Member = r.data
        setForm({
          first_name: m.first_name, last_name: m.last_name,
          date_of_birth: m.date_of_birth?.slice(0, 10) ?? '',
          member_number: m.member_number ?? '',
          pass_number: m.pass_number ?? '',
          jersey_number: m.jersey_number, position: m.position ?? '',
          gender: m.gender ?? 'u', status: m.status,
          club_function: m.club_function ?? '',
          street: m.street ?? '', zip: m.zip ?? '', city: m.city ?? '',
          join_date: m.join_date?.slice(0, 10) ?? '',
          iban: m.iban ?? '',
          photo_visible: m.photo_visible ?? false,
          dsgvo_verarbeitung: m.dsgvo_verarbeitung ?? false,
          dsgvo_verarbeitung_date: m.dsgvo_verarbeitung_date?.slice(0, 10) ?? '',
          dsgvo_weitergabe: m.dsgvo_weitergabe ?? false,
          dsgvo_weitergabe_date: m.dsgvo_weitergabe_date?.slice(0, 10) ?? '',
          sepa_mandat: m.sepa_mandat ?? false,
          sepa_mandat_date: m.sepa_mandat_date?.slice(0, 10) ?? '',
        })
        setAddressSource(m.address_source ?? '')
        setAddressConflict(m.address_conflict ?? false)
        setMemberAddressStored(m.member_address_stored ?? null)
        setPhotoURL(m.photo_url ?? '')
        setSepaMandatURL(m.sepa_mandat_url ?? '')
        setCurrentUserID(m.user_id ?? null)
        setSelectedLinkedUser(m.user_id ? String(m.user_id) : '')
      })
      loadLinkedParents()
    }
  }, [id, isNew, isAdmin])

  const handleSave = async () => {
    setSaving(true); setError('')
    try {
      const body = {
        ...form,
        jersey_number: form.jersey_number ? Number(form.jersey_number) : null,
        club_function: form.club_function || null,
      }
      if (isNew) {
        const r = await api.post('/members', body)
        navigate(`/mitglieder/${r.data.id}`, { replace: true })
      } else {
        await api.put(`/members/${id}`, body)
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
      }
    } catch {
      setError('Fehler beim Speichern.')
    } finally {
      setSaving(false)
    }
  }

  const handleStatusChange = async (status: string) => {
    setForm(f => ({ ...f, status }))
    if (!isNew && id) await api.put(`/members/${id}/status`, { status })
  }

  const handleFamilyLink = async () => {
    if (!selectedParentUser || !id) return
    try {
      await api.post('/admin/family-links', { parent_user_id: Number(selectedParentUser), member_id: Number(id) })
      setSelectedParentUser('')
      loadLinkedParents()
      setSaved(true); setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Verknüpfen.')
    }
  }

  const handleRemoveParent = async (parentUserId: number) => {
    if (!id) return
    setRemovingParent(prev => ({ ...prev, [parentUserId]: true }))
    try {
      await api.delete('/admin/family-links', { data: { parent_user_id: parentUserId, member_id: Number(id) } })
      loadLinkedParents()
    } catch {
      setError('Fehler beim Entfernen.')
    } finally {
      setRemovingParent(prev => ({ ...prev, [parentUserId]: false }))
    }
  }

  const handleLinkUser = async () => {
    if (!id) return
    await api.put(`/admin/members/${id}/user`, { user_id: selectedLinkedUser ? Number(selectedLinkedUser) : null })
    setCurrentUserID(selectedLinkedUser ? Number(selectedLinkedUser) : null)
    setSaved(true); setTimeout(() => setSaved(false), 2000)
  }

  const handlePhotoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !id) return
    setPhotoUploading(true)
    try {
      const fd = new FormData(); fd.append('file', file)
      const r = await api.post(`/upload/member-photo/${id}`, fd, { headers: { 'Content-Type': 'multipart/form-data' } })
      setPhotoURL(r.data.photo_url ?? '')
    } catch {
      setError('Foto-Upload fehlgeschlagen.')
    } finally {
      setPhotoUploading(false)
      if (photoInputRef.current) photoInputRef.current.value = ''
    }
  }

  const handleSepaUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file || !id) return
    setSepaUploading(true)
    try {
      const fd = new FormData(); fd.append('file', file)
      const r = await api.post(`/upload/sepa-mandat/${id}`, fd, { headers: { 'Content-Type': 'multipart/form-data' } })
      setSepaMandatURL(r.data.sepa_mandat_url ?? '')
    } catch {
      setError('SEPA-Upload fehlgeschlagen.')
    } finally {
      setSepaUploading(false)
      if (sepaInputRef.current) sepaInputRef.current.value = ''
    }
  }

  const togglePosition = (pos: string) => {
    const current = form.position ? form.position.split(',').filter(Boolean) : []
    const next = current.includes(pos) ? current.filter(p => p !== pos) : [...current, pos]
    setForm(f => ({ ...f, position: next.join(',') }))
  }

  const selectedPositions = form.position ? form.position.split(',').filter(Boolean) : []
  const isPassive = form.status === 'passiv'
  const canAddParent = linkedParents.length < 2

  const field = (label: string, key: keyof typeof form, type = 'text') => (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">{label}</label>
      <input
        type={type}
        value={String(form[key] ?? '')}
        onChange={e => setForm(f => ({ ...f, [key]: e.target.value }))}
        className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
      />
    </div>
  )

  const disabledField = (label: string) => (
    <div>
      <label className="block text-sm font-medium text-gray-400 mb-1">{label} <span className="font-normal">(passiv)</span></label>
      <input disabled className="w-full border border-gray-200 rounded-md px-3 py-2 text-sm bg-gray-50 text-gray-400" />
    </div>
  )

  const toggleBtn = (label: string, value: boolean, onChange: (v: boolean) => void) => (
    <label className="flex items-center gap-2 cursor-pointer">
      <input type="checkbox" checked={value} onChange={e => onChange(e.target.checked)} className="w-4 h-4 accent-brand-yellow" />
      <span className="text-sm text-gray-700">{label}</span>
    </label>
  )

  return (
    <div className="max-w-2xl">
      <div className="flex items-center gap-3 mb-6">
        <Link to="/mitglieder" className="text-sm text-gray-500 hover:text-gray-700">← Mitglieder</Link>
        <h1 className="text-2xl font-bold">{isNew ? 'Mitglied anlegen' : 'Mitglied bearbeiten'}</h1>
      </div>

      {/* Stammdaten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">Stammdaten</h2>
        <div className="grid grid-cols-2 gap-4">
          {field('Vorname', 'first_name')}
          {field('Nachname', 'last_name')}
          {field('Geburtsdatum', 'date_of_birth', 'date')}
          {field('Mitgliedsnummer', 'member_number')}
          {isPassive ? disabledField('Passnummer') : field('Passnummer', 'pass_number')}
          {isPassive ? disabledField('Rückennummer') : field('Rückennummer', 'jersey_number', 'number')}
        </div>

        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">Positionen</label>
          {isPassive ? (
            <p className="text-sm text-gray-400 italic">Nicht zutreffend (passiv)</p>
          ) : (
            <div className="flex flex-wrap gap-2">
              {HANDBALL_POSITIONS.map(pos => (
                <button
                  key={pos}
                  type="button"
                  onClick={() => togglePosition(pos)}
                  className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                    selectedPositions.includes(pos)
                      ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                      : 'text-gray-600 border-gray-300 hover:border-brand-black hover:text-brand-black'
                  }`}
                >
                  {pos}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">Geschlecht</label>
          <div className="flex gap-2">
            {GENDER_OPTIONS.map(g => (
              <button
                key={g.value}
                type="button"
                onClick={() => setForm(f => ({ ...f, gender: g.value }))}
                className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                  form.gender === g.value
                    ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                    : 'text-gray-600 border-gray-300 hover:border-brand-black hover:text-brand-black'
                }`}
              >
                {g.label}
              </button>
            ))}
          </div>
        </div>

        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
          <div className="flex gap-2 flex-wrap">
            {STATUS_OPTIONS.map(s => (
              <button
                key={s}
                onClick={() => handleStatusChange(s)}
                className={`px-3 py-1 rounded-full text-sm border ${
                  form.status === s
                    ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                    : 'text-gray-600 border-gray-300 hover:border-brand-black hover:text-brand-black'
                }`}
              >
                {s}
              </button>
            ))}
          </div>
        </div>

        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">Vereinsfunktion</label>
          <div className="flex gap-2 flex-wrap">
            {CLUB_FUNCTION_OPTIONS.map(opt => (
              <button
                key={opt.value}
                type="button"
                onClick={() => setForm(f => ({ ...f, club_function: opt.value }))}
                className={`px-3 py-1 rounded-full text-sm border transition-colors ${
                  (form.club_function ?? '') === opt.value
                    ? 'bg-brand-yellow text-brand-black border-brand-yellow'
                    : 'text-gray-600 border-gray-300 hover:border-brand-black hover:text-brand-black'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>

        {error && <p className="mt-3 text-sm text-red-500">{error}</p>}
        <div className="mt-4 flex items-center gap-3">
          <button
            onClick={handleSave}
            disabled={saving}
            className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && <span className="text-sm text-green-600">Gespeichert</span>}
        </div>
      </div>

      {/* Adresse & Kontakt */}
      {!isNew && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
          <div className="flex items-center gap-2 mb-4">
            <h2 className="font-semibold text-gray-700">Adresse & Kontakt</h2>
            {addressSource === 'user' && (
              <span className="text-xs text-gray-400 bg-gray-200 px-2 py-0.5 rounded-full">Aus Nutzerprofil</span>
            )}
            {addressConflict && memberAddressStored && (
              <AddressConflictTooltip stored={memberAddressStored} />
            )}
          </div>
          <div className="grid grid-cols-1 gap-3">
            {field('Straße', 'street')}
            <div className="grid grid-cols-3 gap-3">
              <div className="col-span-1">{field('PLZ', 'zip')}</div>
              <div className="col-span-2">{field('Ort', 'city')}</div>
            </div>
          </div>
          {isAdmin && (
            <div className="mt-3">
              {field('Eintrittsdatum', 'join_date', 'date')}
            </div>
          )}
          {isAdmin && (
            <div className="mt-3">
              <label className="block text-sm font-medium text-gray-700 mb-1">IBAN</label>
              <input
                type="text"
                value={form.iban ?? ''}
                onChange={e => setForm(f => ({ ...f, iban: e.target.value }))}
                placeholder="DE00 0000 0000 0000 0000 00"
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono"
              />
            </div>
          )}
          <div className="mt-4 flex items-center gap-3">
            <button
              onClick={handleSave}
              disabled={saving}
              className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
            >
              {saving ? 'Speichern…' : 'Speichern'}
            </button>
            {saved && <span className="text-sm text-green-600">Gespeichert</span>}
          </div>
        </div>
      )}

      {/* Passfoto */}
      {!isNew && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-4">Passfoto</h2>
          <div className="flex gap-4 items-start">
            {photoURL ? (
              <img src={photoURL} alt="Passfoto" className="w-24 h-24 rounded-lg object-cover border border-gray-200" />
            ) : (
              <div className="w-24 h-24 rounded-lg bg-gray-200 flex items-center justify-center text-gray-400 text-xs">Kein Foto</div>
            )}
            {isAdmin && (
              <div className="flex-1 space-y-3">
                <div>
                  <input ref={photoInputRef} type="file" accept="image/jpeg,image/png,image/webp" className="hidden" onChange={handlePhotoUpload} />
                  <button
                    onClick={() => photoInputRef.current?.click()}
                    disabled={photoUploading}
                    className="bg-brand-yellow text-brand-black px-3 py-1.5 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
                  >
                    {photoUploading ? 'Hochladen…' : 'Foto hochladen'}
                  </button>
                  <p className="text-xs text-gray-400 mt-1">JPEG, PNG oder WebP, max. 5 MB</p>
                </div>
                {toggleBtn(
                  'Foto für alle Mitglieder sichtbar',
                  form.photo_visible ?? false,
                  v => setForm(f => ({ ...f, photo_visible: v }))
                )}
                <button
                  onClick={handleSave}
                  disabled={saving}
                  className="text-sm text-gray-500 underline hover:text-gray-700"
                >
                  Sichtbarkeit speichern
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* DSGVO & SEPA */}
      {!isNew && isAdmin && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-4">DSGVO & SEPA</h2>
          <div className="space-y-4">
            <div className="flex items-center gap-4 flex-wrap">
              {toggleBtn('Datenverarbeitung eingewilligt', form.dsgvo_verarbeitung ?? false,
                v => setForm(f => ({ ...f, dsgvo_verarbeitung: v })))}
              {form.dsgvo_verarbeitung && (
                <div className="flex items-center gap-2">
                  <label className="text-sm text-gray-500">am</label>
                  <input
                    type="date"
                    value={form.dsgvo_verarbeitung_date ?? ''}
                    onChange={e => setForm(f => ({ ...f, dsgvo_verarbeitung_date: e.target.value }))}
                    className="border border-gray-300 rounded px-2 py-1 text-sm"
                  />
                </div>
              )}
            </div>

            <div className="flex items-center gap-4 flex-wrap">
              {toggleBtn('Datenweitergabe eingewilligt', form.dsgvo_weitergabe ?? false,
                v => setForm(f => ({ ...f, dsgvo_weitergabe: v })))}
              {form.dsgvo_weitergabe && (
                <div className="flex items-center gap-2">
                  <label className="text-sm text-gray-500">am</label>
                  <input
                    type="date"
                    value={form.dsgvo_weitergabe_date ?? ''}
                    onChange={e => setForm(f => ({ ...f, dsgvo_weitergabe_date: e.target.value }))}
                    className="border border-gray-300 rounded px-2 py-1 text-sm"
                  />
                </div>
              )}
            </div>

            <div className="flex items-center gap-4 flex-wrap">
              {toggleBtn('SEPA-Mandat erteilt', form.sepa_mandat ?? false,
                v => setForm(f => ({ ...f, sepa_mandat: v })))}
              {form.sepa_mandat && (
                <div className="flex items-center gap-2">
                  <label className="text-sm text-gray-500">am</label>
                  <input
                    type="date"
                    value={form.sepa_mandat_date ?? ''}
                    onChange={e => setForm(f => ({ ...f, sepa_mandat_date: e.target.value }))}
                    className="border border-gray-300 rounded px-2 py-1 text-sm"
                  />
                </div>
              )}
            </div>

            {form.sepa_mandat && (
              <div className="border-t border-gray-200 pt-3">
                <p className="text-sm font-medium text-gray-700 mb-2">SEPA-Dokument</p>
                <div className="flex items-center gap-3">
                  {sepaMandatURL && (
                    <a href={sepaMandatURL} target="_blank" rel="noopener noreferrer"
                      className="text-sm text-brand-blue underline">Dokument anzeigen</a>
                  )}
                  <input ref={sepaInputRef} type="file" accept="application/pdf,image/jpeg,image/png,image/webp" className="hidden" onChange={handleSepaUpload} />
                  <button
                    onClick={() => sepaInputRef.current?.click()}
                    disabled={sepaUploading}
                    className="bg-gray-200 text-gray-700 px-3 py-1.5 rounded-md text-sm hover:bg-gray-300 transition-colors disabled:opacity-40"
                  >
                    {sepaUploading ? 'Hochladen…' : sepaMandatURL ? 'Ersetzen' : 'Hochladen'}
                  </button>
                </div>
                <p className="text-xs text-gray-400 mt-1">PDF, JPEG oder PNG, max. 10 MB</p>
              </div>
            )}
          </div>

          <div className="mt-4 flex items-center gap-3">
            <button
              onClick={handleSave}
              disabled={saving}
              className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40"
            >
              {saving ? 'Speichern…' : 'Speichern'}
            </button>
            {saved && <span className="text-sm text-green-600">Gespeichert</span>}
          </div>
        </div>
      )}

      {/* Nutzer verknüpfen (Admin only, existierendes Mitglied) */}
      {isAdmin && !isNew && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-1">Nutzer verknüpfen</h2>
          <p className="text-xs text-gray-500 mb-4">Das Mitglied erhält damit Zugang zu seinem eigenen Profil.</p>
          {currentUserID && (
            <p className="text-xs text-brand-black mb-3">
              Aktuell verknüpft: {users.find(u => u.id === currentUserID)?.name ?? `User #${currentUserID}`}
            </p>
          )}
          <div className="flex gap-3 items-end">
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">Nutzer</label>
              <select
                value={selectedLinkedUser}
                onChange={e => setSelectedLinkedUser(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              >
                <option value="">– keine Verknüpfung –</option>
                {users.map(u => (
                  <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                ))}
              </select>
            </div>
            <button
              onClick={handleLinkUser}
              className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
            >
              Speichern
            </button>
          </div>
          {saved && <p className="mt-2 text-sm text-green-600">Gespeichert</p>}
        </div>
      )}

      {/* Erziehungsberechtigte (Admin only, existierendes Mitglied) */}
      {isAdmin && !isNew && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-gray-700 mb-4">Erziehungsberechtigte</h2>

          {linkedParents.length > 0 && (
            <div className="mb-4 space-y-2">
              {linkedParents.map(p => (
                <div key={p.id} className="flex items-center justify-between border border-gray-100 rounded-lg px-4 py-2 text-sm">
                  <div>
                    <span className="font-medium">{p.name}</span>
                    <span className="ml-2 text-gray-400">{p.email}</span>
                  </div>
                  <button
                    onClick={() => handleRemoveParent(p.id)}
                    disabled={removingParent[p.id]}
                    className="text-xs text-gray-400 hover:text-red-600 transition-colors disabled:opacity-40 px-2 py-1 rounded"
                    title="Verknüpfung entfernen"
                  >
                    {removingParent[p.id] ? '…' : 'Entfernen'}
                  </button>
                </div>
              ))}
            </div>
          )}

          {canAddParent ? (
            <div className="flex gap-3 items-end">
              <div className="flex-1">
                <label className="block text-sm font-medium text-gray-700 mb-1">Erziehungsberechtigten hinzufügen</label>
                <select
                  value={selectedParentUser}
                  onChange={e => setSelectedParentUser(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                >
                  <option value="">– auswählen –</option>
                  {users.filter(u => !linkedParents.some(p => p.id === u.id)).map(u => (
                    <option key={u.id} value={u.id}>{u.name} ({u.email})</option>
                  ))}
                </select>
              </div>
              <button
                onClick={handleFamilyLink}
                disabled={!selectedParentUser}
                className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
              >
                Hinzufügen
              </button>
            </div>
          ) : (
            <p className="text-xs text-gray-400 italic">Maximal zwei Erziehungsberechtigte möglich.</p>
          )}

          {saved && <p className="mt-2 text-sm text-green-600">Gespeichert</p>}
        </div>
      )}
    </div>
  )
}
