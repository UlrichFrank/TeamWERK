import { useEffect, useState } from 'react'
import { api } from '../lib/api'

interface Slot { id: number; event_name: string; event_date: string; vacancies: number; duty_type: string; role_desc?: string }

export default function DutyBoardPage() {
  const [slots, setSlots] = useState<Slot[]>([])

  const load = () => api.get('/duty-board').then(r => setSlots(r.data ?? []))
  useEffect(() => { load() }, [])

  const claim = async (id: number) => {
    try {
      await api.post(`/duty-board/${id}/claim`)
      load()
    } catch {
      alert('Dieser Dienst ist bereits vergeben oder du hast ihn bereits.')
    }
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Dienstbörse</h1>
      {slots.length === 0 && <p className="text-gray-500">Keine offenen Dienste.</p>}
      <div className="space-y-3">
        {slots.map(s => (
          <div key={s.id} className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-4 flex items-center justify-between">
            <div>
              <div className="font-medium">{s.event_name}</div>
              <div className="text-sm text-gray-500">{s.event_date} · {s.duty_type}{s.role_desc ? ` · ${s.role_desc}` : ''}</div>
              <div className="text-xs text-gray-400 mt-0.5">{s.vacancies} Platz{s.vacancies !== 1 ? 'e' : ''} frei</div>
            </div>
            <button
              onClick={() => claim(s.id)}
              className="bg-brand-yellow text-black rounded-md px-4 py-1.5 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
            >
              Eintragen
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
