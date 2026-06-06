import { useState, useEffect, useRef, FormEvent } from 'react'
import { ChevronDown, Trash2 } from 'lucide-react'
import { api } from '../../lib/api'
import { Member, Parent, Phone, Visibility } from '../../pages/ProfilePage'
import { UserContact } from '../../pages/ChildProfilePage'
import ImageCropModal from '../ImageCropModal'

interface Props {
  children: Member[]
  parents: Parent[]
  ownMember: Member | null
  draftRefreshKey?: number
  mode?: 'own' | 'child'
  childMemberId?: string
  userContact?: UserContact | null
}

export default function ProfileProfilTab({
  children, parents, ownMember, draftRefreshKey,
  mode = 'own', childMemberId, userContact,
}: Props) {
  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [address, setAddress] = useState({ street: '', zip: '', city: '' })
  const [phones, setPhones] = useState<Phone[]>([])
  const [visibility, setVisibility] = useState<Visibility>({ phones_visible: false, address_visible: false, photo_visible: false, email_visible: false })
  const [photoURL, setPhotoURL] = useState('')
  const [profilDraft, setProfilDraft] = useState<any>(null)

  const [changed, setChanged] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  const [showAddPhone, setShowAddPhone] = useState(false)
  const [newPhone, setNewPhone] = useState({ label: '', number: '' })
  const [photoUploading, setPhotoUploading] = useState(false)
  const [cropFile, setCropFile] = useState<File | null>(null)
  const [photoDropdown, setPhotoDropdown] = useState(false)
  const photoInputRef = useRef<HTMLInputElement>(null)
  const photoDropdownRef = useRef<HTMLDivElement>(null)

  const PHONE_LABEL_SUGGESTIONS = ['Privat', 'Mobil', 'Firma', 'Notfall']

  // Own-mode: fetch from own profile endpoints
  useEffect(() => {
    if (mode === 'child') return
    api.get('/profile/me').then(r => {
      setAddress({ street: r.data?.street ?? '', zip: r.data?.zip ?? '', city: r.data?.city ?? '' })
      setPhones(r.data?.phones ?? [])
      setVisibility(r.data?.visibility ?? { phones_visible: false, address_visible: false, photo_visible: false, email_visible: false })
      if (r.data?.photo_url) setPhotoURL(r.data.photo_url)
    })
  }, [])

  useEffect(() => {
    if (mode === 'child') return
    api.get('/profile/account').then(r => {
      setFirstName(r.data.first_name ?? '')
      setLastName(r.data.last_name ?? '')
    })
  }, [])

  useEffect(() => {
    if (mode === 'child') {
      if (!childMemberId) return
      api.get(`/members/${childMemberId}/change-drafts`).then(r => {
        const drafts: any[] = r.data?.drafts ?? []
        setProfilDraft(drafts.find(d => d.field_name === 'profil') ?? null)
      }).catch(() => {})
      return
    }
    if (!ownMember) return
    api.get(`/members/${ownMember.id}/change-drafts`).then(r => {
      const drafts: any[] = r.data?.drafts ?? []
      setProfilDraft(drafts.find(d => d.field_name === 'profil') ?? null)
    }).catch(() => {})
  }, [ownMember?.id, childMemberId, draftRefreshKey])

  // Child-mode: init from userContact (wenn Kind Account hat) oder ownMember (Fallback)
  useEffect(() => {
    if (mode !== 'child') return
    if (userContact) {
      setFirstName(userContact.first_name)
      setLastName(userContact.last_name)
      setAddress({ street: userContact.street, zip: userContact.zip, city: userContact.city })
      setPhones(userContact.phones)
      setVisibility(userContact.visibility)
      setPhotoURL(ownMember?.photo_url ?? '')
    } else if (ownMember) {
      setFirstName(ownMember.first_name)
      setLastName(ownMember.last_name)
      setAddress({ street: ownMember.street ?? '', zip: ownMember.zip ?? '', city: ownMember.city ?? '' })
      setPhotoURL(ownMember.photo_url ?? '')
    }
  }, [ownMember?.id, userContact])

  useEffect(() => {
    if (!photoDropdown) return
    const handler = (e: MouseEvent) => {
      if (photoDropdownRef.current && !photoDropdownRef.current.contains(e.target as Node))
        setPhotoDropdown(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [photoDropdown])

  const childChanged = mode === 'child' && (
    userContact ? (
      firstName !== userContact.first_name ||
      lastName !== userContact.last_name ||
      address.street !== userContact.street ||
      address.zip !== userContact.zip ||
      address.city !== userContact.city ||
      visibility.phones_visible !== userContact.visibility.phones_visible ||
      visibility.address_visible !== userContact.visibility.address_visible ||
      visibility.photo_visible !== userContact.visibility.photo_visible ||
      visibility.email_visible !== userContact.visibility.email_visible
    ) : ownMember != null && (
      firstName !== ownMember.first_name ||
      lastName !== ownMember.last_name ||
      address.street !== (ownMember.street ?? '') ||
      address.zip !== (ownMember.zip ?? '') ||
      address.city !== (ownMember.city ?? '')
    )
  )
  const isChanged = mode === 'child' ? childChanged : changed

  const handleChange = () => setChanged(true)

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      if (mode === 'child' && childMemberId) {
        if (userContact) {
          await api.put(`/profile/kind/${childMemberId}/account`, {
            first_name: firstName, last_name: lastName, street: address.street, zip: address.zip, city: address.city,
          })
          await api.put(`/profile/kind/${childMemberId}/visibility`, visibility)
        }
        await api.post(`/members/${childMemberId}/change-request`, {
          field_name: 'profil',
          new_value: { first_name: firstName, last_name: lastName, street: address.street, zip: address.zip, city: address.city },
        })
        const r = await api.get(`/members/${childMemberId}/change-drafts`)
        const drafts: any[] = r.data?.drafts ?? []
        setProfilDraft(drafts.find(d => d.field_name === 'profil') ?? null)
      } else {
        await api.put('/profile/me', { first_name: firstName, last_name: lastName, street: address.street, zip: address.zip, city: address.city })
        if (ownMember) {
          await api.post(`/members/${ownMember.id}/change-request`, {
            field_name: 'profil',
            new_value: { first_name: firstName, last_name: lastName, street: address.street, zip: address.zip, city: address.city },
          })
          const r = await api.get(`/members/${ownMember.id}/change-drafts`)
          const drafts: any[] = r.data?.drafts ?? []
          setProfilDraft(drafts.find(d => d.field_name === 'profil') ?? null)
        }
        await api.put('/profile/visibility', visibility)
      }
      setSaved(true)
      setChanged(false)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  const handleAddPhone = async () => {
    if (!newPhone.number) return
    try {
      if (mode === 'child' && childMemberId) {
        const r = await api.post(`/profile/kind/${childMemberId}/phones`, {
          label: newPhone.label, number: newPhone.number, sort_order: phones.length,
        })
        setPhones([...phones, { id: r.data.id, label: newPhone.label, number: newPhone.number, sort_order: phones.length }])
      } else {
        const r = await api.post('/profile/phones', { label: newPhone.label, number: newPhone.number, sort_order: phones.length })
        setPhones([...phones, { id: r.data.id, label: newPhone.label, number: newPhone.number, sort_order: phones.length }])
      }
      setNewPhone({ label: '', number: '' })
      setShowAddPhone(false)
    } catch {
      setError('Fehler beim Hinzufügen')
    }
  }

  const handleDeletePhone = async (phoneId: number) => {
    try {
      if (mode === 'child' && childMemberId) {
        await api.delete(`/profile/kind/${childMemberId}/phones/${phoneId}`)
      } else {
        await api.delete(`/profile/phones/${phoneId}`)
      }
      setPhones(phones.filter(p => p.id !== phoneId))
    } catch {
      setError('Fehler beim Löschen')
    }
  }

  const handlePhotoSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setCropFile(file)
    if (photoInputRef.current) photoInputRef.current.value = ''
  }

  const handleCropConfirm = async (blob: Blob) => {
    setCropFile(null)
    setPhotoUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', blob, 'photo.jpg')
      const endpoint = mode === 'child' && childMemberId
        ? `/profile/kind/${childMemberId}/photo`
        : '/upload/user-photo'
      const r = await api.post(endpoint, fd)
      setPhotoURL(r.data.photo_url ?? '')
    } catch {
      setError('Foto-Upload fehlgeschlagen')
    } finally {
      setPhotoUploading(false)
    }
  }

  const handlePhotoDelete = async () => {
    setPhotoDropdown(false)
    try {
      const endpoint = mode === 'child' && childMemberId
        ? `/profile/kind/${childMemberId}/photo`
        : '/upload/user-photo'
      await api.delete(endpoint)
      setPhotoURL('')
    } catch {
      setError('Foto konnte nicht gelöscht werden')
    }
  }

  const inputCls = `w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow`

  return (
    <div className="space-y-6">
      {/* Pending draft banner */}
      {profilDraft && (
        <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
          Änderungsanfrage ausstehend — wird beim Speichern aktualisiert.{mode === 'own' && ' Zum Zurückziehen den Tab „Mitgliedsdaten" öffnen.'}
        </div>
      )}

      {/* Profilbild */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Profilbild</h2>
        <div className="flex gap-4 items-start">
          {photoURL ? (
            <img src={photoURL} alt="Profilbild" className="w-20 h-20 rounded-full object-cover border border-brand-border" />
          ) : (
            <div className="w-20 h-20 rounded-full bg-gray-200 flex items-center justify-center text-brand-text-subtle text-xs">Kein Bild</div>
          )}
          <div>
            <input ref={photoInputRef} type="file" accept="image/*" className="hidden" onChange={handlePhotoSelect} />
            <div className="space-y-1">
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
              <p className="text-xs text-brand-text-subtle">JPEG, PNG oder WebP, max. 5 MB</p>
            </div>
          </div>
        </div>
      </div>

      {/* Persönliche Daten */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Persönliche Daten</h2>
        <form onSubmit={handleSave} className="space-y-3">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Vorname</label>
              <input
                type="text"
                value={firstName}
                onChange={e => { setFirstName(e.target.value); handleChange() }}
                className={inputCls}
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Nachname</label>
              <input
                type="text"
                value={lastName}
                onChange={e => { setLastName(e.target.value); handleChange() }}
                className={inputCls}
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Straße</label>
            <input
              type="text"
              value={address.street}
              onChange={e => { setAddress({ ...address, street: e.target.value }); handleChange() }}
              className={inputCls}
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">PLZ</label>
              <input
                type="text"
                value={address.zip}
                onChange={e => { setAddress({ ...address, zip: e.target.value }); handleChange() }}
                className={inputCls}
              />
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Ort</label>
              <input
                type="text"
                value={address.city}
                onChange={e => { setAddress({ ...address, city: e.target.value }); handleChange() }}
                className={inputCls}
              />
            </div>
          </div>
        </form>
      </div>

      {/* Telefonnummern — nur wenn User-Strang vorhanden */}
      {(mode !== 'child' || userContact) && <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Telefonnummern</h2>
        <div className="space-y-3">
          {phones.length > 0 && (
            <div className="space-y-2">
              {phones.map(p => (
                <div key={p.id} className="flex items-center justify-between border border-brand-border-subtle rounded-lg px-4 py-2 text-sm">
                  <div>
                    {p.label && <span className="text-brand-text-muted mr-2">{p.label}:</span>}
                    <span className="font-mono">{p.number}</span>
                  </div>
                  <button onClick={() => handleDeletePhone(p.id)} className="text-xs text-brand-text-subtle hover:text-brand-danger">×</button>
                </div>
              ))}
            </div>
          )}
          {showAddPhone ? (
            <div className="border border-brand-border-subtle rounded-lg p-3 space-y-2">
              <div className="flex gap-2">
                <div className="flex-1">
                  <label className="block text-xs text-brand-text-muted mb-1">Bezeichnung</label>
                  <input
                    list="phone-label-suggestions"
                    value={newPhone.label}
                    onChange={e => setNewPhone({ ...newPhone, label: e.target.value })}
                    placeholder="z.B. Mobil"
                    className="w-full border border-brand-border rounded px-2 py-1.5 text-sm"
                  />
                  <datalist id="phone-label-suggestions">
                    {PHONE_LABEL_SUGGESTIONS.map(s => <option key={s} value={s} />)}
                  </datalist>
                </div>
                <div className="flex-1">
                  <label className="block text-xs text-brand-text-muted mb-1">Nummer</label>
                  <input
                    type="tel"
                    value={newPhone.number}
                    onChange={e => setNewPhone({ ...newPhone, number: e.target.value })}
                    placeholder="+49 711 …"
                    className="w-full border border-brand-border rounded px-2 py-1.5 text-sm"
                  />
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  onClick={handleAddPhone}
                  disabled={!newPhone.number}
                  className="bg-brand-yellow text-brand-black rounded-md px-3 py-1.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  Hinzufügen
                </button>
                <button
                  onClick={() => { setShowAddPhone(false); setNewPhone({ label: '', number: '' }) }}
                  className="text-sm text-brand-text-muted hover:text-brand-text px-2"
                >
                  Abbrechen
                </button>
              </div>
            </div>
          ) : (
            <button onClick={() => setShowAddPhone(true)} className="text-sm text-brand-blue underline hover:text-brand-black">
              + Nummer hinzufügen
            </button>
          )}
        </div>
      </div>}

      {/* Sichtbarkeit — nur wenn User-Strang vorhanden */}
      {(mode !== 'child' || userContact) && <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Sichtbarkeit für Mitglieder</h2>
        <p className="text-xs text-brand-text-subtle mb-4">Wähle, welche Kontaktdaten andere Mitglieder sehen dürfen.</p>
        <div className="space-y-2">
          {[
            { key: 'phones_visible' as const, label: 'Telefonnummern sichtbar' },
            { key: 'address_visible' as const, label: 'Adresse sichtbar' },
            { key: 'photo_visible' as const, label: 'Profilbild sichtbar' },
            { key: 'email_visible' as const, label: 'E-Mail-Adresse sichtbar' },
          ].map(({ key, label }) => (
            <label key={key} className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={visibility[key]}
                onChange={e => { setVisibility({ ...visibility, [key]: e.target.checked }); handleChange() }}
                className="w-4 h-4 accent-brand-yellow"
              />
              <span className="text-sm text-brand-text">{label}</span>
            </label>
          ))}
        </div>
      </div>}

      {/* Familie */}
      {(children.length > 0 || parents.length > 0) && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-brand-text-muted mb-4">Familie</h2>
          {children.length > 0 && (
            <div className="mb-4">
              <h3 className="text-sm font-medium text-brand-text-muted mb-2">Meine Kinder</h3>
              <div className="space-y-1">
                {children.map(c => (
                  <p key={c.id} className="text-sm text-brand-text">• {c.first_name} {c.last_name}</p>
                ))}
              </div>
            </div>
          )}
          {parents.length > 0 && (
            <div>
              <h3 className="text-sm font-medium text-brand-text-muted mb-2">Erziehungsberechtigte</h3>
              <div className="space-y-1">
                {parents.map(p => (
                  <p key={p.id} className="text-sm text-brand-text">• {p.name} ({p.email})</p>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Save button */}
      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={!isChanged || saving}
          className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {saving ? 'Speichern…' : 'Speichern'}
        </button>
        {saved && <span className="text-sm text-green-600">Gespeichert</span>}
        {error && <span className="text-sm text-brand-danger">{error}</span>}
      </div>

      <ImageCropModal
        file={cropFile}
        onConfirm={handleCropConfirm}
        onCancel={() => { setCropFile(null); if (photoInputRef.current) photoInputRef.current.value = '' }}
      />
    </div>
  )
}
