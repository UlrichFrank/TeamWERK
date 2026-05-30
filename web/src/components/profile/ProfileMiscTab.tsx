import { useState, useEffect, FormEvent } from 'react'
import { Minus, Plus } from 'lucide-react'
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
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Fahrzeug</h2>
        <form onSubmit={handleSave} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Sitzplätze</label>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => { setVehicle(v => ({ ...v, seats: Math.max(0, (v.seats ?? 0) - 1) })); handleChange() }}
                className="bg-brand-yellow text-brand-black rounded-md px-3 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
                aria-label="Weniger"
              >
                <Minus className="w-4 h-4" />
              </button>
              <span className="w-10 text-center text-sm font-medium text-brand-text border border-brand-border rounded-md py-2">
                {vehicle.seats ?? 0}
              </span>
              <button
                type="button"
                onClick={() => { setVehicle(v => ({ ...v, seats: Math.min(10, (v.seats ?? 0) + 1) })); handleChange() }}
                className="bg-brand-yellow text-brand-black rounded-md px-3 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
                aria-label="Mehr"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-brand-text-muted mb-1">Anmerkungen</label>
            <textarea
              value={vehicle.notes}
              onChange={(e) => { setVehicle({...vehicle, notes: e.target.value}); handleChange() }}
              placeholder="z.B. Hänger vorhanden, Fahrradträger, etc."
              className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
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
        {saved && <span className="text-sm text-brand-text-muted">Gespeichert</span>}
        {error && <span className="text-sm text-brand-danger">{error}</span>}
      </div>
    </div>
  )
}
