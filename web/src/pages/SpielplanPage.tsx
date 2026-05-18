import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Game {
  id: number
  date: string
  time: string
  opponent: string
  team_id: number
  team_name: string
  slot_count: number
  filled_count: number
  total_count: number
}

interface SlotPreview {
  duty_type_id: number
  duty_type_name: string
  event_time: string
  slots_count: number
  role_desc: string
}

interface Team {
  id: number
  name: string
  is_active: boolean
}

const WEEKDAYS = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So']
const MONTHS = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember']

function trafficColor(filledCount: number, totalCount: number, slotCount: number): string {
  if (slotCount === 0) return 'bg-red-400'
  if (totalCount > 0 && filledCount >= totalCount) return 'bg-green-500'
  if (filledCount > 0) return 'bg-yellow-400'
  return 'bg-red-400'
}

function padDate(year: number, month: number, day: number): string {
  return `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`
}

export default function SpielplanPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const now = new Date()
  const [year, setYear] = useState(now.getFullYear())
  const [month, setMonth] = useState(now.getMonth())
  const [games, setGames] = useState<Game[]>([])
  const [teams, setTeams] = useState<Team[]>([])
  const [loading, setLoading] = useState(true)

  // Create dialog
  const [showCreate, setShowCreate] = useState(false)
  const [newDate, setNewDate] = useState('')
  const [newTime, setNewTime] = useState('15:00')
  const [newOpponent, setNewOpponent] = useState('')
  const [newTeamId, setNewTeamId] = useState<number | ''>('')
  const [preview, setPreview] = useState<SlotPreview[]>([])
  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(new Set())
  const [showPreview, setShowPreview] = useState(false)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [creating, setCreating] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const loadGames = () => api.get('/games').then(r => setGames(r.data ?? []))

  useEffect(() => {
    Promise.all([loadGames(), api.get('/teams').then(r => setTeams(r.data ?? []))]).finally(() => setLoading(false))
  }, [])

  const prevMonth = () => month === 0 ? (setMonth(11), setYear(y => y - 1)) : setMonth(m => m - 1)
  const nextMonth = () => month === 11 ? (setMonth(0), setYear(y => y + 1)) : setMonth(m => m + 1)

  const monthGames = games.filter(g => {
    const y = parseInt(g.date.slice(0, 4))
    const m = parseInt(g.date.slice(5, 7)) - 1
    return y === year && m === month
  })

  const gamesByDate: Record<string, Game[]> = {}
  for (const g of monthGames) {
    if (!gamesByDate[g.date]) gamesByDate[g.date] = []
    gamesByDate[g.date].push(g)
  }

  const firstDayOfWeek = (new Date(year, month, 1).getDay() + 6) % 7
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const todayStr = padDate(now.getFullYear(), now.getMonth(), now.getDate())

  const doCreateGame = async (slots: SlotPreview[]) => {
    setCreating(true)
    setCreateError(null)
    try {
      await api.post('/admin/games', {
        date: newDate,
        time: newTime,
        opponent: newOpponent,
        team_id: newTeamId,
        slots: slots.map(s => ({
          duty_type_id: s.duty_type_id,
          event_time: s.event_time,
          slots_count: s.slots_count,
          role_desc: s.role_desc,
        })),
      })
      await loadGames()
      closeDialog()
    } catch {
      setCreateError('Spiel konnte nicht angelegt werden. Ist eine aktive Saison vorhanden?')
    } finally {
      setCreating(false)
    }
  }

  const handleFetchPreview = async () => {
    if (!newDate || !newTeamId) return
    setPreviewLoading(true)
    try {
      const r = await api.get(`/admin/game-template/preview?date=${newDate}&time=${newTime}`)
      const slots: SlotPreview[] = r.data ?? []
      if (slots.length === 0) {
        // No template configured — create directly without slots (per spec)
        await doCreateGame([])
      } else {
        setPreview(slots)
        setSelectedIndices(new Set(slots.map((_, i) => i)))
        setShowPreview(true)
      }
    } catch {
      setPreview([])
      setSelectedIndices(new Set())
      setShowPreview(true)
    } finally {
      setPreviewLoading(false)
    }
  }

  const toggleSlot = (i: number) => {
    setSelectedIndices(prev => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  const closeDialog = () => {
    setShowCreate(false)
    setShowPreview(false)
    setNewDate('')
    setNewTime('15:00')
    setNewOpponent('')
    setNewTeamId('')
    setPreview([])
    setSelectedIndices(new Set())
    setCreateError(null)
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Spielplan</h1>
        {user?.role === 'admin' && (
          <button
            onClick={() => setShowCreate(true)}
            className="bg-brand-yellow text-black px-4 py-2 rounded-md text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
          >
            + Heimspiel anlegen
          </button>
        )}
      </div>

      {/* Month navigation */}
      <div className="flex items-center gap-4 mb-4">
        <button onClick={prevMonth} className="p-2 hover:bg-gray-200 rounded-lg transition-colors">◀</button>
        <span className="text-lg font-semibold w-44 text-center">{MONTHS[month]} {year}</span>
        <button onClick={nextMonth} className="p-2 hover:bg-gray-200 rounded-lg transition-colors">▶</button>
      </div>

      {/* Calendar */}
      <div className="bg-white rounded-xl shadow overflow-hidden">
        <div className="grid grid-cols-7 bg-gray-50 border-b">
          {WEEKDAYS.map(d => (
            <div key={d} className="text-center text-xs font-semibold py-2 text-gray-500 uppercase tracking-wide">{d}</div>
          ))}
        </div>
        <div className="grid grid-cols-7">
          {Array.from({ length: firstDayOfWeek }).map((_, i) => (
            <div key={`pad-${i}`} className="min-h-[90px] border-r border-b bg-gray-50/50" />
          ))}
          {Array.from({ length: daysInMonth }).map((_, i) => {
            const day = i + 1
            const dateStr = padDate(year, month, day)
            const dayGames = gamesByDate[dateStr] ?? []
            const isToday = dateStr === todayStr
            return (
              <div key={day} className={`min-h-[90px] p-1.5 border-r border-b ${isToday ? 'bg-brand-yellow/20' : ''}`}>
                <div className={`text-xs mb-1 ${isToday ? 'font-bold' : 'text-gray-400'}`}>{day}</div>
                {dayGames.map(g => (
                  <button
                    key={g.id}
                    onClick={() => navigate(`/spielplan/${g.id}`)}
                    className="w-full text-left mb-1 p-1.5 rounded-md text-xs bg-gray-100 hover:bg-gray-200 transition-colors border border-gray-200"
                  >
                    <div className="flex items-center gap-1.5 mb-0.5">
                      <div className={`w-2 h-2 rounded-full flex-shrink-0 ${trafficColor(g.filled_count, g.total_count, g.slot_count)}`} />
                      <span className="font-semibold truncate">{g.team_name}</span>
                    </div>
                    <div className="truncate text-gray-500 leading-tight">vs. {g.opponent || '–'}</div>
                    <div className="text-gray-400 leading-tight">{g.time}</div>
                  </button>
                ))}
              </div>
            )
          })}
        </div>
      </div>

      {!loading && monthGames.length === 0 && (
        <p className="text-gray-400 text-center mt-10 text-sm">Keine Heimspiele in diesem Monat</p>
      )}

      {/* Create dialog */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-2xl p-6 w-full max-w-md shadow-2xl">
            <h2 className="text-lg font-bold mb-4">Heimspiel anlegen</h2>

            {!showPreview ? (
              <div className="space-y-3">
                <div>
                  <label className="block text-sm font-medium mb-1">Datum *</label>
                  <input type="date" value={newDate} onChange={e => setNewDate(e.target.value)}
                    className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Anstoßzeit</label>
                  <input type="time" value={newTime} onChange={e => setNewTime(e.target.value)}
                    className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Gegner</label>
                  <input type="text" value={newOpponent} onChange={e => setNewOpponent(e.target.value)}
                    placeholder="Name des Gegners" className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Mannschaft *</label>
                  <select value={newTeamId} onChange={e => setNewTeamId(Number(e.target.value))}
                    className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow">
                    <option value="">Auswählen…</option>
                    {teams.filter(t => t.is_active).map(t => (
                      <option key={t.id} value={t.id}>{t.name}</option>
                    ))}
                  </select>
                </div>
                {createError && (
                  <p className="text-red-600 text-sm mt-2">{createError}</p>
                )}
                <div className="flex gap-2 pt-2">
                  <button onClick={closeDialog}
                    className="flex-1 border rounded-md px-4 py-2 text-sm hover:bg-gray-50">Abbrechen</button>
                  <button
                    onClick={handleFetchPreview}
                    disabled={!newDate || !newTeamId || previewLoading || creating}
                    className="flex-1 bg-brand-yellow text-black rounded-md px-4 py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >
                    {previewLoading || creating ? 'Laden…' : 'Weiter →'}
                  </button>
                </div>
              </div>
            ) : (
              <div>
                <p className="text-sm text-gray-500 mb-3">
                  Dienste, die automatisch angelegt werden ({selectedIndices.size} ausgewählt):
                </p>
                <div className="space-y-1.5 mb-4 max-h-56 overflow-y-auto">
                  {preview.map((s, i) => (
                    <label key={i} className="flex items-center gap-2.5 p-2 rounded-lg hover:bg-gray-50 cursor-pointer">
                      <input type="checkbox" checked={selectedIndices.has(i)} onChange={() => toggleSlot(i)}
                        className="rounded" />
                      <span className="font-mono text-sm font-semibold w-12">{s.event_time}</span>
                      <span className="text-sm flex-1">{s.duty_type_name}</span>
                      {s.role_desc && <span className="text-xs text-gray-400">({s.role_desc})</span>}
                      <span className="text-xs text-gray-400 ml-auto">{s.slots_count}×</span>
                    </label>
                  ))}
                </div>
                {createError && (
                  <p className="text-red-600 text-sm mb-3">{createError}</p>
                )}
                <div className="flex gap-2">
                  <button onClick={() => setShowPreview(false)}
                    className="border rounded-md px-3 py-2 text-sm hover:bg-gray-50">← Zurück</button>
                  <button
                    onClick={() => doCreateGame([])}
                    disabled={creating}
                    className="border rounded-md px-3 py-2 text-sm text-gray-500 hover:bg-gray-50 disabled:opacity-50"
                  >Ohne Dienste</button>
                  <button
                    onClick={() => doCreateGame(preview.filter((_, i) => selectedIndices.has(i)))}
                    disabled={creating}
                    className="flex-1 bg-brand-yellow text-black rounded-md px-4 py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >
                    {creating ? 'Anlegen…' : 'Bestätigen'}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
