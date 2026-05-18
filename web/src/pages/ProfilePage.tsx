import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Member {
  id: number; first_name: string; last_name: string
  date_of_birth: string; pass_number: string
  jersey_number?: number; position: string; status: string
}

export default function ProfilePage() {
  const { user, logout } = useAuth()
  const [members, setMembers] = useState<Member[]>([])
  const [vehicle, setVehicle] = useState({ seats: 0, notes: '' })
  const [vehicleSaved, setVehicleSaved] = useState(false)

  // Account (name)
  const [accountName, setAccountName] = useState('')
  const [accountSaved, setAccountSaved] = useState(false)

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
    api.get('/profile/me').then(r => setMembers(r.data ?? []))
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

  const statusColor = (s: string) =>
    s === 'aktiv' ? 'bg-black text-white' :
    s === 'verletzt' ? 'bg-brand-yellow text-black' :
    'bg-gray-200 text-gray-600'

  const inputClass = 'w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow'

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold mb-6">Mein Profil</h1>

      {/* Account */}
      <div className="bg-white rounded-xl shadow p-6 mb-4">
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
            <button type="submit" className="bg-[#3E4A98] text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-[#2e3a7a] transition-colors">
              Speichern
            </button>
            {accountSaved && <span className="text-sm text-green-600">Gespeichert</span>}
          </div>
        </form>
      </div>

      {/* Password change */}
      <div className="bg-white rounded-xl shadow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">Passwort ändern</h2>
        {pwSuccess ? (
          <p className="text-sm text-green-600">Passwort geändert. Du wirst ausgeloggt…</p>
        ) : (
          <form onSubmit={handlePasswordChange} className="space-y-3">
            {pwError && <p className="text-sm text-red-600">{pwError}</p>}
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
            <button type="submit" className="bg-[#3E4A98] text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-[#2e3a7a] transition-colors">
              Passwort ändern
            </button>
          </form>
        )}
      </div>

      {/* Email change */}
      <div className="bg-white rounded-xl shadow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-4">E-Mail-Adresse ändern</h2>
        {emailSent ? (
          <p className="text-sm text-green-600">Bestätigungs-Mail gesendet. Bitte prüfe dein neues Postfach.</p>
        ) : (
          <form onSubmit={handleEmailChange} className="space-y-3">
            {emailError && <p className="text-sm text-red-600">{emailError}</p>}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Neue E-Mail-Adresse</label>
              <input type="email" value={emailNew} onChange={e => setEmailNew(e.target.value)} required className={inputClass} />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Aktuelles Passwort zur Bestätigung</label>
              <input type="password" value={emailPw} onChange={e => setEmailPw(e.target.value)} required className={inputClass} />
            </div>
            <button type="submit" className="bg-[#3E4A98] text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-[#2e3a7a] transition-colors">
              Bestätigungs-Mail senden
            </button>
          </form>
        )}
      </div>

      {/* Member profiles */}
      {members.length > 0 && (
        <div className="bg-white rounded-xl shadow p-6 mb-4">
          <h2 className="font-semibold text-gray-700 mb-4">
            {user?.role === 'elternteil' ? 'Meine Kinder' : 'Mein Spielerprofil'}
          </h2>
          <div className="space-y-4">
            {members.map(m => (
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

      {members.length === 0 && (
        <div className="bg-white rounded-xl shadow p-6 mb-4 text-sm text-gray-500">
          Kein Mitgliedsprofil verknüpft. Bitte den Administrator kontaktieren.
        </div>
      )}

      {/* Vehicle info */}
      <div className="bg-white rounded-xl shadow p-6">
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
          {vehicleSaved && <span className="text-sm text-green-600">Gespeichert</span>}
        </div>
      </div>
    </div>
  )
}
