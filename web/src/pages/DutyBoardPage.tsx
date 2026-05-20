import { useEffect, useState } from 'react'
import { api } from '../lib/api'

interface BoardSlot {
  id: number
  duty_type: string
  event_time: string
  slots_total: number
  vacancies: number
  claimed_by_me: boolean
  role_desc?: string
}

interface BoardGroup {
  game_id: number | null
  date: string | null
  event_time: string | null
  opponent: string | null
  team_name: string
  label: string | null
  past: boolean
  slots: BoardSlot[]
}

const WEEKDAYS = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']

function formatDate(iso: string): string {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  const day = WEEKDAYS[d.getDay()]
  return `${day} ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.`
}

export default function DutyBoardPage() {
  const [groups, setGroups] = useState<BoardGroup[]>([])
  const [showPast, setShowPast] = useState(false)

  const load = () => api.get('/duty-board').then(r => setGroups(r.data ?? []))
  useEffect(() => { load() }, [])

  const claim = async (id: number) => {
    try {
      await api.post(`/duty-board/${id}/claim`)
      load()
    } catch {
      alert('Dieser Dienst ist bereits vergeben oder du hast ihn bereits.')
    }
  }

  const unclaim = async (id: number) => {
    try {
      await api.delete(`/duty-board/${id}/claim`)
      load()
    } catch {
      alert('Austragen fehlgeschlagen.')
    }
  }

  const visible = groups.filter(g => showPast || !g.past)

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Dienstbörse</h1>
        <button
          onClick={() => setShowPast(p => !p)}
          className="text-sm text-gray-500 hover:text-brand-blue transition-colors"
        >
          {showPast ? 'Vergangene ausblenden' : 'Vergangene einblenden'}
        </button>
      </div>

      {visible.length === 0 && (
        <p className="text-gray-500">
          {groups.length === 0
            ? 'Keine Dienste für deine Mannschaften.'
            : 'Keine aktuellen Dienste. Vergangene Spieltage können oben eingeblendet werden.'}
        </p>
      )}

      <div className="space-y-4">
        {visible.map((g, i) => (
          <div
            key={i}
            className={`bg-white rounded-xl shadow border-t-4 overflow-hidden ${g.past ? 'border-gray-300 opacity-70' : 'border-brand-yellow'}`}
          >
            {/* Group header */}
            <div className="px-4 py-3 bg-gray-50 border-b border-gray-100 flex items-center justify-between">
              <div>
                {g.game_id ? (
                  <span className="font-semibold text-sm">
                    {g.date ? formatDate(g.date) : ''}
                    {g.event_time ? ` · ${g.event_time} Uhr` : ''}
                    {g.opponent ? ` · vs. ${g.opponent}` : ''}
                  </span>
                ) : (
                  <span className="font-semibold text-sm">{g.label}</span>
                )}
              </div>
              <span className="text-xs text-gray-400 font-medium">{g.team_name}</span>
            </div>

            {/* Slots table */}
            <table className="w-full text-sm">
              <tbody className="divide-y divide-gray-100">
                {g.slots.map(s => (
                  <tr key={s.id} className="px-4">
                    <td className="px-4 py-2.5 font-medium text-gray-800">
                      {s.duty_type}
                      {s.role_desc ? <span className="text-gray-400 font-normal"> · {s.role_desc}</span> : null}
                    </td>
                    <td className="px-4 py-2.5 text-gray-500 w-20">
                      {s.event_time || '—'}
                    </td>
                    <td className="px-4 py-2.5 text-gray-500 w-20 text-right">
                      {s.claimed_by_me
                        ? <span className="text-brand-blue text-xs font-medium">Eingetragen</span>
                        : s.vacancies > 0
                          ? <span className="text-xs">{s.vacancies} frei</span>
                          : <span className="text-xs text-gray-400">Besetzt</span>
                      }
                    </td>
                    <td className="px-4 py-2.5 w-28 text-right">
                      {s.claimed_by_me && !g.past && (
                        <button
                          onClick={() => unclaim(s.id)}
                          className="text-xs text-gray-400 hover:text-red-500 transition-colors px-2 py-1 rounded border border-gray-200 hover:border-red-300"
                        >
                          Austragen
                        </button>
                      )}
                      {!s.claimed_by_me && s.vacancies > 0 && !g.past && (
                        <button
                          onClick={() => claim(s.id)}
                          className="text-xs bg-brand-yellow text-black font-medium px-2 py-1 rounded hover:bg-black hover:text-brand-yellow transition-colors"
                        >
                          Eintragen
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}
      </div>
    </div>
  )
}
