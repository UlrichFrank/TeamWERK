import { useEffect, useState, FormEvent } from 'react'
import { api } from '../lib/api'

export default function AdminClubPage() {
  const [name, setName] = useState('')
  const [address, setAddress] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    api.get('/admin/club').then(r => {
      setName(r.data.name ?? '')
      setAddress(r.data.address ?? '')
    })
  }, [])

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    await api.put('/admin/club', { name, address })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Vereinseinstellungen</h1>
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 max-w-lg">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Vereinsname</label>
            <input value={name} onChange={e => setName(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Adresse</label>
            <input value={address} onChange={e => setAddress(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
          </div>
          <button type="submit" className="bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
            {saved ? 'Gespeichert ✓' : 'Speichern'}
          </button>
        </form>
      </div>
    </div>
  )
}
