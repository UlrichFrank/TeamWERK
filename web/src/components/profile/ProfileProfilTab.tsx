import { useState, useEffect, useRef, FormEvent } from 'react'
import { api } from '../../lib/api'
import { Member, Parent, Phone, Visibility } from '../../pages/ProfilePage'

interface Props {
  children: Member[]
  parents: Parent[]
}

export default function ProfileProfilTab({ children, parents }: Props) {
  const [address, setAddress] = useState({ street: '', zip: '', city: '', house_number: '' })
  const [phones, setPhones] = useState<Phone[]>([])
  const [visibility, setVisibility] = useState<Visibility>({ phones_visible: false, address_visible: false, photo_visible: false })
  const [photoURL, setPhotoURL] = useState('')
  const [iban, setIban] = useState('')
  const [sepaMandat, setSepaMandat] = useState(false)

  const [changed, setChanged] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  const [showAddPhone, setShowAddPhone] = useState(false)
  const [newPhone, setNewPhone] = useState({ label: '', number: '' })
  const [photoUploading, setPhotoUploading] = useState(false)
  const photoInputRef = useRef<HTMLInputElement>(null)

  const PHONE_LABEL_SUGGESTIONS = ['Privat', 'Mobil', 'Firma', 'Notfall']

  useEffect(() => {
    api.get('/profile/me').then(r => {
      setAddress({ street: r.data?.street ?? '', zip: r.data?.zip ?? '', city: r.data?.city ?? '', house_number: r.data?.house_number ?? '' })
      setPhones(r.data?.phones ?? [])
      setVisibility(r.data?.visibility ?? { phones_visible: false, address_visible: false, photo_visible: false })
      setIban(r.data?.iban ?? '')
      setSepaMandat(r.data?.sepa_mandat ?? false)
      if (r.data?.photo_url) setPhotoURL(r.data.photo_url)
    })
  }, [])

  const handleChange = () => setChanged(true)

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      await api.put('/profile/me', { street: address.street, zip: address.zip, city: address.city, house_number: address.house_number, iban })
      await api.put('/profile/visibility', visibility)
      setSaved(true)
      setChanged(false)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  const handleAddPhone = async () => {
    if (!newPhone.number) return
    try {
      const r = await api.post('/profile/phones', { label: newPhone.label, number: newPhone.number, sort_order: phones.length })
      setPhones([...phones, { id: r.data.id, label: newPhone.label, number: newPhone.number, sort_order: phones.length }])
      setNewPhone({ label: '', number: '' })
      setShowAddPhone(false)
      setChanged(true)
    } catch (err) {
      setError('Fehler beim Hinzufügen')
    }
  }

  const handleDeletePhone = async (phoneId: number) => {
    try {
      await api.delete(`/profile/phones/${phoneId}`)
      setPhones(phones.filter(p => p.id !== phoneId))
      setChanged(true)
    } catch (err) {
      setError('Fehler beim Löschen')
    }
  }

  const handlePhotoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setPhotoUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', file)
      const r = await api.post('/upload/user-photo', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
      setPhotoURL(r.data.photo_url ?? '')
      setChanged(true)
    } catch (err) {
      setError('Foto-Upload fehlgeschlagen')
    } finally {
      setPhotoUploading(false)
      if (photoInputRef.current) photoInputRef.current.value = ''
    }
  }

  return (
    <div className="space-y-6">
      {/* Profilbild */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Profilbild</h2>
        <div className="flex gap-4 items-start">
          {photoURL ? (
            <img src={photoURL} alt="Profilbild" className="w-20 h-20 rounded-full object-cover border border-gray-200" />
          ) : (
            <div className="w-20 h-20 rounded-full bg-gray-200 flex items-center justify-center text-gray-400 text-xs">Kein Bild</div>
          )}
          <div>
            <input ref={photoInputRef} type="file" accept="image/jpeg,image/png,image/webp" className="hidden" onChange={handlePhotoUpload} />
            <button
              onClick={() => photoInputRef.current?.click()}
              disabled={photoUploading}
              className="bg-brand-yellow text-black px-3 py-1.5 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
            >
              {photoUploading ? 'Hochladen…' : photoURL ? 'Bild ersetzen' : 'Bild hochladen'}
            </button>
            <p className="text-xs text-gray-400 mt-1">JPEG, PNG oder WebP, max. 5 MB</p>
          </div>
        </div>
      </div>

      {/* Kontaktinformationen */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Kontaktinformationen</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <input
              type="text"
              value={address.street}
              onChange={(e) => { setAddress({...address, street: e.target.value}); handleChange() }}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input
                type="text"
                value={address.zip}
                onChange={(e) => { setAddress({...address, zip: e.target.value}); handleChange() }}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input
                type="text"
                value={address.city}
                onChange={(e) => { setAddress({...address, city: e.target.value}); handleChange() }}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Telefonnummern</label>
            {phones.length > 0 && (
              <div className="space-y-2 mb-4">
                {phones.map(p => (
                  <div key={p.id} className="flex items-center justify-between border border-gray-100 rounded-lg px-4 py-2 text-sm">
                    <div>
                      {p.label && <span className="text-gray-400 mr-2">{p.label}:</span>}
                      <span className="font-mono">{p.number}</span>
                    </div>
                    <button onClick={() => handleDeletePhone(p.id)} className="text-xs text-gray-400 hover:text-red-600">×</button>
                  </div>
                ))}
              </div>
            )}
            {showAddPhone ? (
              <div className="border border-gray-200 rounded-lg p-3 space-y-2">
                <div className="flex gap-2">
                  <div className="flex-1">
                    <label className="block text-xs text-gray-500 mb-1">Bezeichnung</label>
                    <input
                      list="phone-label-suggestions"
                      value={newPhone.label}
                      onChange={(e) => setNewPhone({...newPhone, label: e.target.value})}
                      placeholder="z.B. Mobil"
                      className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm"
                    />
                    <datalist id="phone-label-suggestions">
                      {PHONE_LABEL_SUGGESTIONS.map(s => <option key={s} value={s} />)}
                    </datalist>
                  </div>
                  <div className="flex-1">
                    <label className="block text-xs text-gray-500 mb-1">Nummer</label>
                    <input
                      type="tel"
                      value={newPhone.number}
                      onChange={(e) => setNewPhone({...newPhone, number: e.target.value})}
                      placeholder="+49 711 …"
                      className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm"
                    />
                  </div>
                </div>
                <div className="flex gap-2">
                  <button onClick={handleAddPhone} disabled={!newPhone.number} className="bg-brand-yellow text-black px-3 py-1.5 rounded text-sm font-medium hover:bg-black hover:text-brand-yellow disabled:opacity-40">
                    Hinzufügen
                  </button>
                  <button onClick={() => { setShowAddPhone(false); setNewPhone({ label: '', number: '' }) }} className="text-sm text-gray-500 hover:text-gray-700 px-2">
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
        </div>
      </div>

      {/* Sichtbarkeit */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Sichtbarkeit für Mitglieder</h2>
        <p className="text-xs text-gray-500 mb-4">Wähle, welche Kontaktdaten andere Mitglieder sehen dürfen.</p>
        <div className="space-y-2">
          {[
            { key: 'phones_visible' as const, label: 'Telefonnummern sichtbar' },
            { key: 'address_visible' as const, label: 'Adresse sichtbar' },
            { key: 'photo_visible' as const, label: 'Profilbild sichtbar' },
          ].map(({ key, label }) => (
            <label key={key} className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={visibility[key]}
                onChange={(e) => { setVisibility({...visibility, [key]: e.target.checked}); handleChange() }}
                className="w-4 h-4 accent-brand-yellow"
              />
              <span className="text-sm text-gray-700">{label}</span>
            </label>
          ))}
        </div>
      </div>

      {/* Bankdaten */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Bankdaten</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">IBAN</label>
            <input
              type="text"
              value={iban}
              onChange={(e) => { setIban(e.target.value); handleChange() }}
              placeholder="DE00 0000 0000 0000 0000 00"
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-brand-yellow"
            />
          </div>
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={sepaMandat} disabled className="w-4 h-4 accent-brand-yellow" />
            <span className="text-sm text-gray-700">SEPA-Mandat erteilt (read-only)</span>
          </label>
        </div>
      </div>

      {/* Familie */}
      {(children.length > 0 || parents.length > 0) && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-gray-700 mb-4">Familie</h2>
          {children.length > 0 && (
            <div className="mb-4">
              <h3 className="text-sm font-medium text-gray-600 mb-2">Meine Kinder</h3>
              <div className="space-y-1">
                {children.map(c => (
                  <p key={c.id} className="text-sm text-gray-700">• {c.first_name} {c.last_name}</p>
                ))}
              </div>
            </div>
          )}
          {parents.length > 0 && (
            <div>
              <h3 className="text-sm font-medium text-gray-600 mb-2">Meine Elternteile</h3>
              <div className="space-y-1">
                {parents.map(p => (
                  <p key={p.id} className="text-sm text-gray-700">• {p.name} ({p.email})</p>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Save Button */}
      <div className="flex items-center gap-3">
        <button
          onClick={handleSave}
          disabled={!changed || saving}
          className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40"
        >
          {saving ? 'Speichern…' : 'Speichern'}
        </button>
        {saved && <span className="text-sm text-green-600">Gespeichert</span>}
        {error && <span className="text-sm text-red-600">{error}</span>}
      </div>
    </div>
  )
}
