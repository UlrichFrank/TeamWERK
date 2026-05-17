import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Member {
  id: number; first_name: string; last_name: string
  date_of_birth: string; pass_number: string
  jersey_number?: number; position: string; status: string
}

export default function ProfilePage() {
  const { user } = useAuth()
  const [members, setMembers] = useState<Member[]>([])
  const [vehicle, setVehicle] = useState({ seats: 0, notes: '' })
  const [vehicleSaved, setVehicleSaved] = useState(false)

  useEffect(() => {
    api.get('/profile/me').then(r => setMembers(r.data ?? []))
    api.get('/profile/vehicle').then(r => setVehicle(r.data ?? { seats: 0, notes: '' }))
  }, [])

  const handleVehicleSave = async () => {
    await api.put('/profile/vehicle', vehicle)
    setVehicleSaved(true)
    setTimeout(() => setVehicleSaved(false), 2000)
  }

  const statusColor = (s: string) =>
    s === 'aktiv' ? 'bg-green-100 text-green-700' :
    s === 'verletzt' ? 'bg-yellow-100 text-yellow-700' :
    'bg-gray-100 text-gray-600'

  return (
    <div className="max-w-2xl">
      <h1 className="text-2xl font-bold mb-6">Mein Profil</h1>

      {/* User info */}
      <div className="bg-white rounded-xl shadow p-6 mb-4">
        <h2 className="font-semibold text-gray-700 mb-2">Konto</h2>
        <p className="text-sm text-gray-600">{user?.email}</p>
        <p className="text-sm text-gray-400 capitalize">{user?.role}</p>
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
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Anmerkungen</label>
            <input
              type="text"
              value={vehicle.notes}
              onChange={e => setVehicle(v => ({ ...v, notes: e.target.value }))}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
              placeholder="z.B. Hänger vorhanden"
            />
          </div>
        </div>
        <div className="mt-4 flex items-center gap-3">
          <button
            onClick={handleVehicleSave}
            className="bg-brand-blue text-white px-4 py-2 rounded-md text-sm hover:bg-brand-blue-dark"
          >
            Speichern
          </button>
          {vehicleSaved && <span className="text-sm text-green-600">Gespeichert</span>}
        </div>
      </div>
    </div>
  )
}
