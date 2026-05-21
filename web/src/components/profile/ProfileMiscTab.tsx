import { useState, useEffect, FormEvent } from 'react'
import { api } from '../../lib/api'

interface Vehicle {
  seats: number | null
  notes: string
}

export default function ProfileMiscTab() {
  const [vehicle, setVehicle] = useState<Vehicle>({ seats: null, notes: '' })
  const [changed, setChanged] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.get('/profile/vehicle').then(r => {
      if (r.data) {
        setVehicle({
          seats: r.data.seats ?? null,
          notes: r.data.notes ?? ''
        })
      }
    })
  }, [])

  const handleChange = () => setChanged(true)

  const handleSave = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError('')
    try {
      await api.put('/profile/vehicle', {
        seats: vehicle.seats,
        notes: vehicle.notes
      })
      setSaved(true)
      setChanged(false)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      setError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Fahrzeug */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-gray-700 mb-4">Fahrzeug</h2>
        <form onSubmit={handleSave} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Sitzplätze</label>
            <input
              type="number"
              min="0"
              max="10"
              value={vehicle.seats ?? ''}
              onChange={(e) => { setVehicle({...vehicle, seats: e.target.value ? parseInt(e.target.value) : null}); handleChange() }}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Anmerkungen</label>
            <textarea
              value={vehicle.notes}
              onChange={(e) => { setVehicle({...vehicle, notes: e.target.value}); handleChange() }}
              placeholder="z.B. Hänger vorhanden, Fahrradträger, etc."
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              rows={3}
            />
          </div>
        </form>
      </div>

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
