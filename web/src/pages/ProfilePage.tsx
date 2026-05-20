import { useEffect, useRef, useState, FormEvent } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Member {
  id: number; first_name: string; last_name: string
  date_of_birth: string; pass_number: string
  jersey_number?: number; position: string; status: string
}

interface Parent {
  id: number; name: string; email: string
}

interface Phone {
  id: number; label: string; number: string; sort_order: number
}

interface Visibility {
  phones_visible: boolean; address_visible: boolean; photo_visible: boolean
}

const PHONE_LABEL_SUGGESTIONS = ['Privat', 'Mobil', 'Firma', 'Notfall']

export default function ProfilePage() {
  const { user, logout } = useAuth()
  const [ownMember, setOwnMember] = useState<Member | null>(null)
  const [children, setChildren] = useState<Member[]>([])
  const [parents, setParents] = useState<Parent[]>([])
  const [vehicle, setVehicle] = useState({ seats: 0, notes: '' })
  const [vehicleSaved, setVehicleSaved] = useState(false)

  // Account (name)
  const [accountName, setAccountName] = useState('')
  const [accountSaved, setAccountSaved] = useState(false)

  // Contact data
  const [address, setAddress] = useState({ street: '', zip: '', city: '' })
  const [addressSaved, setAddressSaved] = useState(false)
  const [phones, setPhones] = useState<Phone[]>([])
  const [visibility, setVisibility] = useState<Visibility>({ phones_visible: false, address_visible: false, photo_visible: false })
  const [visSaved, setVisSaved] = useState(false)
  const [photoURL, setPhotoURL] = useState('')
  const [photoUploading, setPhotoUploading] = useState(false)
  const photoInputRef = useRef<HTMLInputElement>(null)
  // New phone form
  const [showAddPhone, setShowAddPhone] = useState(false)
  const [newPhone, setNewPhone] = useState({ label: '', number: '' })
  const [addingPhone, setAddingPhone] = useState(false)

  // Password change
  const [pwCurrent, setPwCurrent] = useState('')
  const [pwNew, setPwNew] = useState('')
  const [pwConfirm, setPwConfirm] = useState('')
  const [pwError, setPwError] = useState('')
  const [pwSuccess, setPwSuccess] = useState(false)

  // Email change
  const [emailNew, setEmailNew] = useState('')
  const [emailPw, setEmailPw] = useState('')
  const [emailError, setEmailError] = useState('')
  const [emailSent, setEmailSent] = useState(false)

  useEffect(() => {
    api.get('/profile/me').then(r => {
      setOwnMember(r.data?.own_member ?? null)
      setChildren(r.data?.children ?? [])
      setParents(r.data?.parents ?? [])
      setAddress({ street: r.data?.street ?? '', zip: r.data?.zip ?? '', city: r.data?.city ?? '' })
      setPhones(r.data?.phones ?? [])
      setVisibility(r.data?.visibility ?? { phones_visible: false, address_visible: false, photo_visible: false })
      if (r.data?.photo_url) setPhotoURL(r.data.photo_url)
    })
    api.get('/profile/vehicle').then(r => setVehicle(r.data ?? { seats: 0, notes: '' }))
    api.get('/profile/account').then(r => setAccountName(r.data.name ?? ''))
  }, [])

  const handleVehicleSave = async () => {
    await api.put('/profile/vehicle', vehicle)
    setVehicleSaved(true)
    setTimeout(() => setVehicleSaved(false), 2000)
  }

  const handleAccountSave = async (e: FormEvent) => {
    e.preventDefault()
    await api.put('/profile/account', { name: accountName })
    setAccountSaved(true)
    setTimeout(() => setAccountSaved(false), 2000)
  }

  const handlePasswordChange = async (e: FormEvent) => {
    e.preventDefault()
    setPwError('')
    if (pwNew !== pwConfirm) {
      setPwError('Die Passwörter stimmen nicht überein.')
      return
    }
    try {
      await api.post('/profile/password', { current_password: pwCurrent, new_password: pwNew })
      setPwSuccess(true)
      setTimeout(() => logout(), 2500)
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      setPwError(status === 403 ? 'Aktuelles Passwort nicht korrekt.' : 'Fehler beim Speichern.')
    }
  }

  const handleEmailChange = async (e: FormEvent) => {
    e.preventDefault()
    setEmailError('')
    try {
      await api.post('/profile/email', { new_email: emailNew, password: emailPw })
      setEmailSent(true)
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 403) setEmailError('Passwort nicht korrekt.')
      else if (status === 409) setEmailError('E-Mail-Adresse bereits vergeben.')
      else setEmailError('Fehler beim Senden.')
    }
  }

  const handleAddressSave = async () => {
    await api.put('/profile/me', address)
    setAddressSaved(true)
    setTimeout(() => setAddressSaved(false), 2000)
  }

  const handleAddPhone = async () => {
    if (!newPhone.number) return
    setAddingPhone(true)
    try {
      const r = await api.post('/profile/phones', { label: newPhone.label, number: newPhone.number, sort_order: phones.length })
      setPhones(prev => [...prev, { id: r.data.id, label: newPhone.label, number: newPhone.number, sort_order: phones.length }])
      setNewPhone({ label: '', number: '' })
      setShowAddPhone(false)
    } finally {
      setAddingPhone(false)
    }
  }

  const handleDeletePhone = async (phoneId: number) => {
    await api.delete(`/profile/phones/${phoneId}`)
    setPhones(prev => prev.filter(p => p.id !== phoneId))
  }

  const handleVisSave = async () => {
    await api.put('/profile/visibility', visibility)
    setVisSaved(true)
    setTimeout(() => setVisSaved(false), 2000)
  }

  const handlePhotoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setPhotoUploading(true)
    try {
      const fd = new FormData(); fd.append('file', file)
      const r = await api.post('/upload/user-photo', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
      setPhotoURL(r.data.photo_url ?? '')
    } finally {
      setPhotoUploading(false)
      if (photoInputRef.current) photoInputRef.current.value = ''
    }
  }

  const statusColor = (s: string) =>
    s === 'aktiv' ? 'bg-brand-black text-brand-white' :
    s === 'verletzt' ? 'bg-brand-yellow text-brand-black' :
    'bg-gray-200 text-gray-600'

  const inputClass = 'w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow'

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold mb-6">Mein Profil</h1>

      {/* Account */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">Konto</h2>
        <p className="text-sm text-gray-500 mb-4">{user?.email}</p>
        <form onSubmit={handleAccountSave} className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input
              type="text" value={accountName} onChange={e => setAccountName(e.target.value)}
              required className={inputClass}
            />
          </div>
          <div className="flex items-center gap-3">
            <button type="submit" className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
              Speichern
            </button>
            {accountSaved && <span className="text-sm text-brand-success">Gespeichert</span>}
          </div>
        </form>
      </div>

      {/* Contact data: Address */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">Adresse</h2>
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Straße</label>
            <input type="text" value={address.street} onChange={e => setAddress(a => ({ ...a, street: e.target.value }))} className={inputClass} />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">PLZ</label>
              <input type="text" value={address.zip} onChange={e => setAddress(a => ({ ...a, zip: e.target.value }))} className={inputClass} />
            </div>
            <div className="col-span-2">
              <label className="block text-sm font-medium text-gray-700 mb-1">Ort</label>
              <input type="text" value={address.city} onChange={e => setAddress(a => ({ ...a, city: e.target.value }))} className={inputClass} />
            </div>
          </div>
        </div>
        <div className="mt-4 flex items-center gap-3">
          <button onClick={handleAddressSave} className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
            Speichern
          </button>
          {addressSaved && <span className="text-sm text-green-600">Gespeichert</span>}
        </div>
      </div>

      {/* Contact data: Phone numbers */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">Telefonnummern</h2>
        {phones.length > 0 && (
          <div className="space-y-2 mb-4">
            {phones.map(p => (
              <div key={p.id} className="flex items-center justify-between border border-gray-100 rounded-lg px-4 py-2 text-sm">
                <div>
                  {p.label && <span className="text-gray-400 mr-2">{p.label}:</span>}
                  <span className="font-mono">{p.number}</span>
                </div>
                <button onClick={() => handleDeletePhone(p.id)} className="text-xs text-gray-400 hover:text-red-600 transition-colors px-2 py-1">
                  ×
                </button>
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
                  onChange={e => setNewPhone(p => ({ ...p, label: e.target.value }))}
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
                  onChange={e => setNewPhone(p => ({ ...p, number: e.target.value }))}
                  placeholder="+49 711 …"
                  className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm"
                />
              </div>
            </div>
            <div className="flex gap-2">
              <button onClick={handleAddPhone} disabled={addingPhone || !newPhone.number}
                className="bg-brand-yellow text-black px-3 py-1.5 rounded text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40">
                {addingPhone ? '…' : 'Hinzufügen'}
              </button>
              <button onClick={() => { setShowAddPhone(false); setNewPhone({ label: '', number: '' }) }}
                className="text-sm text-gray-500 hover:text-gray-700 px-2">
                Abbrechen
              </button>
            </div>
          </div>
        ) : (
          <button onClick={() => setShowAddPhone(true)}
            className="text-sm text-brand-blue underline hover:text-brand-black">
            + Nummer hinzufügen
          </button>
        )}
      </div>

      {/* Contact data: Profile photo */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">Profilbild</h2>
        <div className="flex gap-4 items-start">
          {photoURL ? (
            <img src={photoURL} alt="Profilbild" className="w-20 h-20 rounded-full object-cover border border-gray-200" />
          ) : (
            <div className="w-20 h-20 rounded-full bg-gray-200 flex items-center justify-center text-gray-400 text-xs">Kein Bild</div>
          )}
          <div>
            <input ref={photoInputRef} type="file" accept="image/jpeg,image/png,image/webp" className="hidden" onChange={handlePhotoUpload} />
            <button onClick={() => photoInputRef.current?.click()} disabled={photoUploading}
              className="bg-brand-yellow text-black px-3 py-1.5 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-40">
              {photoUploading ? 'Hochladen…' : photoURL ? 'Bild ersetzen' : 'Bild hochladen'}
            </button>
            <p className="text-xs text-gray-400 mt-1">JPEG, PNG oder WebP, max. 5 MB</p>
          </div>
        </div>
      </div>

      {/* Visibility toggles */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-1">Sichtbarkeit für Teammitglieder</h2>
        <p className="text-xs text-gray-500 mb-4">Wähle, welche Kontaktdaten andere Teammitglieder sehen dürfen.</p>
        <div className="space-y-2">
          {[
            { key: 'phones_visible' as const, label: 'Telefonnummern sichtbar' },
            { key: 'address_visible' as const, label: 'Adresse sichtbar' },
            { key: 'photo_visible' as const, label: 'Profilbild sichtbar' },
          ].map(({ key, label }) => (
            <label key={key} className="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" checked={visibility[key]} onChange={e => setVisibility(v => ({ ...v, [key]: e.target.checked }))}
                className="w-4 h-4 accent-brand-yellow" />
              <span className="text-sm text-gray-700">{label}</span>
            </label>
          ))}
        </div>
        <div className="mt-4 flex items-center gap-3">
          <button onClick={handleVisSave} className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
            Speichern
          </button>
          {visSaved && <span className="text-sm text-green-600">Gespeichert</span>}
        </div>
      </div>

      {/* Password change */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">Passwort ändern</h2>
        {pwSuccess ? (
          <p className="text-sm text-brand-success">Passwort geändert. Du wirst ausgeloggt…</p>
        ) : (
          <form onSubmit={handlePasswordChange} className="space-y-3">
            {pwError && <p className="text-sm text-brand-error">{pwError}</p>}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Aktuelles Passwort</label>
              <input type="password" value={pwCurrent} onChange={e => setPwCurrent(e.target.value)} required className={inputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Neues Passwort</label>
              <input type="password" value={pwNew} onChange={e => setPwNew(e.target.value)} required className={inputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Neues Passwort wiederholen</label>
              <input type="password" value={pwConfirm} onChange={e => setPwConfirm(e.target.value)} required className={inputClass} />
            </div>
            <button type="submit" className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
              Passwort ändern
            </button>
          </form>
        )}
      </div>

      {/* Email change */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">E-Mail-Adresse ändern</h2>
        {emailSent ? (
          <p className="text-sm text-brand-success">Bestätigungs-Mail gesendet. Bitte prüfe dein neues Postfach.</p>
        ) : (
          <form onSubmit={handleEmailChange} className="space-y-3">
            {emailError && <p className="text-sm text-brand-error">{emailError}</p>}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Neue E-Mail-Adresse</label>
              <input type="email" value={emailNew} onChange={e => setEmailNew(e.target.value)} required className={inputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Aktuelles Passwort zur Bestätigung</label>
              <input type="password" value={emailPw} onChange={e => setEmailPw(e.target.value)} required className={inputClass} />
            </div>
            <button type="submit" className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
              Bestätigungs-Mail senden
            </button>
          </form>
        )}
      </div>

      {/* Own member profile */}
      {ownMember ? (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-4">Meine Mitgliedsdaten</h2>
          <div className="border border-gray-100 rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <span className="font-medium">{ownMember.last_name}, {ownMember.first_name}</span>
              <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColor(ownMember.status)}`}>
                {ownMember.status}
              </span>
            </div>
            <div className="grid grid-cols-2 gap-x-6 text-sm text-gray-500">
              {ownMember.date_of_birth && <div><span className="text-gray-400">Geb.:</span> {ownMember.date_of_birth}</div>}
              {ownMember.pass_number && <div><span className="text-gray-400">Pass:</span> {ownMember.pass_number}</div>}
              {ownMember.jersey_number != null && <div><span className="text-gray-400">Trikot:</span> #{ownMember.jersey_number}</div>}
              {ownMember.position && <div><span className="text-gray-400">Position:</span> {ownMember.position}</div>}
            </div>
          </div>
        </div>
      ) : (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4 text-sm text-gray-500">
          Kein Mitgliedsprofil verknüpft. Bitte den Administrator kontaktieren.
        </div>
      )}

      {/* Elternteil: linked children */}
      {user?.role === 'elternteil' && children.length > 0 && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-4">Meine Kinder</h2>
          <div className="space-y-4">
            {children.map(m => (
              <div key={m.id} className="border border-gray-100 rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <span className="font-medium">{m.last_name}, {m.first_name}</span>
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColor(m.status)}`}>
                    {m.status}
                  </span>
                </div>
                <div className="grid grid-cols-2 gap-x-6 text-sm text-gray-500">
                  {m.date_of_birth && <div><span className="text-gray-400">Geb.:</span> {m.date_of_birth}</div>}
                  {m.pass_number && <div><span className="text-gray-400">Pass:</span> {m.pass_number}</div>}
                  {m.jersey_number != null && <div><span className="text-gray-400">Trikot:</span> #{m.jersey_number}</div>}
                  {m.position && <div><span className="text-gray-400">Position:</span> {m.position}</div>}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Familie: Elternteile (spieler) */}
      {user?.role === 'spieler' && parents.length > 0 && (
        <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-4">Meine Elternteile</h2>
          <div className="space-y-2">
            {parents.map(p => (
              <div key={p.id} className="flex items-center justify-between border border-gray-100 rounded-lg px-4 py-3">
                <span className="font-medium text-sm">{p.name}</span>
                <span className="text-sm text-gray-400">{p.email}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Vehicle info */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Fahrzeug</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Sitzplätze</label>
            <input
              type="number" min={0} max={9}
              value={vehicle.seats}
              onChange={e => setVehicle(v => ({ ...v, seats: Number(e.target.value) }))}
              className={inputClass}
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Anmerkungen</label>
            <input
              type="text"
              value={vehicle.notes}
              onChange={e => setVehicle(v => ({ ...v, notes: e.target.value }))}
              className={inputClass}
              placeholder="z.B. Hänger vorhanden"
            />
          </div>
        </div>
        <div className="mt-4 flex items-center gap-3">
          <button
            onClick={handleVehicleSave}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
          >
            Speichern
          </button>
          {vehicleSaved && <span className="text-sm text-brand-success">Gespeichert</span>}
        </div>
      </div>
    </div>
  )
}
