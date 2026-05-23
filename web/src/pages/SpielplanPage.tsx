import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Game {
  id: number
  date: string
  time: string
  opponent: string
  teams: Array<{ id: number; name: string }>
  event_type: string
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
  if (slotCount === 0) return 'bg-brand-error'
  if (totalCount > 0 && filledCount >= totalCount) return 'bg-brand-success'
  if (filledCount > 0) return 'bg-brand-warning'
  return 'bg-brand-error'
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

  // Day-regen dialog
  const [showDayRegen, setShowDayRegen] = useState(false)
  const [dayRegenDate, setDayRegenDate] = useState('')
  const [dayRegenLoading, setDayRegenLoading] = useState(false)
  const [dayRegenResult, setDayRegenResult] = useState<{
    games: Array<{ game_id: number; slots_created: number; kept_slots: number; skipped?: boolean }>
    conflicts: Array<{ duty_type_id: number; event_time: string; game_ids: number[] }>
  } | null>(null)
  const [dayRegenError, setDayRegenError] = useState<string | null>(null)

  // Wizard dialog
  const [showCreate, setShowCreate] = useState(false)
  const [wizardStep, setWizardStep] = useState(1)
  const [eventType, setEventType] = useState<'heim' | 'auswärts' | 'generisch' | ''>('')
  const [selectedDate, setSelectedDate] = useState('')
  const [selectedTime, setSelectedTime] = useState('15:00')
  const [selectedOpponent, setSelectedOpponent] = useState('')
  const [selectedTeamIds, setSelectedTeamIds] = useState<number[]>([])
  const [selectedTemplate, setSelectedTemplate] = useState<number | null>(null)
  const [templates, setTemplates] = useState<any[]>([])
  const [preview, setPreview] = useState<SlotPreview[]>([])
  const [selectedSlotIndices, setSelectedSlotIndices] = useState<Set<number>>(new Set())
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
    const key = g.date.slice(0, 10)
    if (!gamesByDate[key]) gamesByDate[key] = []
    gamesByDate[key].push(g)
  }

  const firstDayOfWeek = (new Date(year, month, 1).getDay() + 6) % 7
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const todayStr = padDate(now.getFullYear(), now.getMonth(), now.getDate())

  const doCreateGame = async (slots: SlotPreview[]) => {
    setCreating(true)
    setCreateError(null)
    try {
      await api.post('/admin/games', {
        date: selectedDate,
        time: selectedTime,
        opponent: selectedOpponent,
        team_ids: selectedTeamIds,
        event_type: eventType,
        template_id: selectedTemplate ?? undefined,
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
      setCreateError('Event konnte nicht angelegt werden. Ist eine aktive Saison vorhanden?')
    } finally {
      setCreating(false)
    }
  }

  const loadTemplates = async () => {
    try {
      const r = await api.get('/admin/duty-templates')
      setTemplates(r.data ?? [])
    } catch {
      setTemplates([])
    }
  }

  const handleFetchPreview = async () => {
    if (!selectedTemplate || !selectedDate || selectedTeamIds.length === 0) return
    setPreviewLoading(true)
    try {
      const dateParam = eventType === 'heim' ? `&date=${selectedDate}` : ''
      const r = await api.get(`/admin/duty-templates/${selectedTemplate}/preview?time=${selectedTime}${dateParam}`)
      const slots: SlotPreview[] = r.data ?? []
      setPreview(slots)
      setSelectedSlotIndices(new Set(slots.map((_, i) => i)))
      setWizardStep(4)
    } catch {
      setPreview([])
      setSelectedSlotIndices(new Set())
      setWizardStep(4)
    } finally {
      setPreviewLoading(false)
    }
  }

  const toggleSlot = (i: number) => {
    setSelectedSlotIndices(prev => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  const openDayRegen = (dateStr: string) => {
    setDayRegenDate(dateStr)
    setDayRegenResult(null)
    setDayRegenError(null)
    setShowDayRegen(true)
  }

  const closeDayRegen = () => {
    setShowDayRegen(false)
    setDayRegenDate('')
    setDayRegenResult(null)
    setDayRegenError(null)
  }

  const doRegenDay = async () => {
    setDayRegenLoading(true)
    setDayRegenError(null)
    setDayRegenResult(null)
    try {
      const r = await api.post(`/admin/games/regenerate-day?date=${dayRegenDate}`)
      setDayRegenResult(r.data)
      await loadGames()
    } catch {
      setDayRegenError('Generierung fehlgeschlagen. Ist eine aktive Saison vorhanden?')
    } finally {
      setDayRegenLoading(false)
    }
  }

  const closeDialog = () => {
    setShowCreate(false)
    setWizardStep(1)
    setEventType('')
    setSelectedDate('')
    setSelectedTime('15:00')
    setSelectedOpponent('')
    setSelectedTeamIds([])
    setSelectedTemplate(null)
    setPreview([])
    setSelectedSlotIndices(new Set())
    setCreateError(null)
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Spielplan</h1>
        {user && ['admin', 'vorstand', 'trainer'].includes(user.role) && (
          <button
            onClick={() => setShowCreate(true)}
            className="bg-brand-yellow text-brand-black px-4 py-2 rounded-md text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
          >
            + Event anlegen
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
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
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
            const canRegen = user && ['admin', 'vorstand', 'trainer'].includes(user.role) && dayGames.length > 0
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
                      <span className="font-semibold truncate">{g.teams.map(t => t.name).join(', ')}</span>
                    </div>
                    <div className="truncate text-gray-500 leading-tight">vs. {g.opponent || '–'}</div>
                    <div className="text-gray-400 leading-tight">{g.time}</div>
                  </button>
                ))}
                {canRegen && (
                  <button
                    onClick={() => openDayRegen(dateStr)}
                    className="w-full mt-0.5 text-center text-[10px] text-brand-blue hover:underline leading-tight py-0.5"
                  >
                    Dienste generieren
                  </button>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {!loading && monthGames.length === 0 && (
        <p className="text-gray-400 text-center mt-10 text-sm">Keine Heimspiele in diesem Monat</p>
      )}

      {/* Day Regeneration Dialog */}
      {showDayRegen && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-2xl p-6 w-full max-w-md shadow-2xl max-h-[90vh] overflow-y-auto">
            <h2 className="text-lg font-bold mb-1">Dienste generieren</h2>
            <p className="text-sm text-gray-500 mb-4">{dayRegenDate}</p>

            <div className="space-y-2 mb-4">
              {(gamesByDate[dayRegenDate] ?? []).map(g => (
                <div key={g.id} className="p-3 border rounded-lg bg-gray-50 text-sm">
                  <div className="font-semibold">{g.time} — {g.teams.map(t => t.name).join(', ')}</div>
                  <div className="text-gray-500 text-xs">vs. {g.opponent || '–'} · {g.event_type}</div>
                </div>
              ))}
            </div>

            {dayRegenResult && (
              <div className="mb-4 space-y-2">
                <div className="p-3 bg-green-50 border border-green-200 rounded-lg text-sm">
                  <div className="font-semibold text-green-800 mb-1">Generierung abgeschlossen</div>
                  {dayRegenResult.games.map(gr => (
                    <div key={gr.game_id} className="text-green-700 text-xs">
                      {gr.skipped
                        ? `Spiel #${gr.game_id}: kein Template — übersprungen`
                        : `Spiel #${gr.game_id}: ${gr.slots_created} Dienste erstellt, ${gr.kept_slots} behalten`}
                    </div>
                  ))}
                </div>
                {dayRegenResult.conflicts.length > 0 && (
                  <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm">
                    <div className="font-semibold text-red-800 mb-1">Konflikte erkannt</div>
                    <p className="text-red-700 text-xs mb-1">
                      Gleicher Diensttyp zur gleichen Zeit bei mehreren Spielen — bitte Optimierungsregeln prüfen.
                    </p>
                    {dayRegenResult.conflicts.map((c, i) => (
                      <div key={i} className="text-red-600 text-xs">
                        {c.event_time} · Diensttyp #{c.duty_type_id} bei Spielen {c.game_ids.join(', ')}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {dayRegenError && (
              <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
                {dayRegenError}
              </div>
            )}

            <div className="flex gap-2 pt-2">
              <button onClick={closeDayRegen} className="border rounded-md px-4 py-2 text-sm hover:bg-gray-50">
                {dayRegenResult ? 'Schließen' : 'Abbrechen'}
              </button>
              {!dayRegenResult && (
                <button
                  onClick={doRegenDay}
                  disabled={dayRegenLoading}
                  className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                >
                  {dayRegenLoading ? 'Generiere…' : 'Generieren'}
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Event Wizard Dialog */}
      {showCreate && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-2xl p-6 w-full max-w-md shadow-2xl max-h-[90vh] overflow-y-auto">
            {wizardStep === 1 && (
              <div>
                <h2 className="text-lg font-bold mb-6">Welche Art von Event?</h2>
                <div className="space-y-3">
                  {['heim', 'auswärts', 'generisch'].map(type => (
                    <button
                      key={type}
                      onClick={() => {
                        setEventType(type as 'heim' | 'auswärts' | 'generisch')
                        setWizardStep(2)
                      }}
                      className="w-full p-4 border-2 rounded-lg text-left hover:bg-gray-50 hover:border-brand-yellow transition-colors"
                    >
                      <div className="font-semibold">
                        {type === 'heim' && '⚽ Heimspiel'}
                        {type === 'auswärts' && '✈️ Auswärtsspiel'}
                        {type === 'generisch' && '📋 Sonstiges Event'}
                      </div>
                      <div className="text-xs text-gray-500 mt-1">
                        {type === 'heim' && 'Heimspiel gegen eine Mannschaft'}
                        {type === 'auswärts' && 'Auswärtsspiel gegen eine Mannschaft'}
                        {type === 'generisch' && 'Event für mehrere Mannschaften'}
                      </div>
                    </button>
                  ))}
                </div>
                <div className="flex gap-2 pt-4">
                  <button onClick={closeDialog} className="flex-1 border rounded-md px-4 py-2 text-sm hover:bg-gray-50">Abbrechen</button>
                </div>
              </div>
            )}

            {wizardStep === 2 && (
              <div>
                <h2 className="text-lg font-bold mb-4">Event-Details</h2>
                <div className="space-y-3">
                  <div>
                    <label className="block text-sm font-medium mb-1">Datum *</label>
                    <input type="date" value={selectedDate} onChange={e => setSelectedDate(e.target.value)}
                      className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Anstoßzeit</label>
                    <input type="time" value={selectedTime} onChange={e => setSelectedTime(e.target.value)}
                      className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
                  </div>
                  {eventType !== 'generisch' && (
                    <div>
                      <label className="block text-sm font-medium mb-1">Gegner *</label>
                      <input type="text" value={selectedOpponent} onChange={e => setSelectedOpponent(e.target.value)}
                        placeholder="Name des Gegners" className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
                    </div>
                  )}
                  {eventType === 'generisch' && (
                    <div>
                      <label className="block text-sm font-medium mb-1">Event-Name *</label>
                      <input type="text" value={selectedOpponent} onChange={e => setSelectedOpponent(e.target.value)}
                        placeholder="Name des Events" className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
                    </div>
                  )}
                  <div>
                    <label className="block text-sm font-medium mb-2">
                      {eventType === 'generisch' ? 'Mannschaften *' : 'Mannschaft *'}
                    </label>
                    {eventType === 'generisch' ? (
                      <div className="space-y-2">
                        {teams.filter(t => t.is_active).map(t => (
                          <label key={t.id} className="flex items-center gap-2">
                            <input type="checkbox" checked={selectedTeamIds.includes(t.id)}
                              onChange={e => {
                                if (e.target.checked) {
                                  setSelectedTeamIds([...selectedTeamIds, t.id])
                                } else {
                                  setSelectedTeamIds(selectedTeamIds.filter(id => id !== t.id))
                                }
                              }} className="rounded" />
                            <span className="text-sm">{t.name}</span>
                          </label>
                        ))}
                      </div>
                    ) : (
                      <select value={selectedTeamIds[0] ?? ''} onChange={e => setSelectedTeamIds(e.target.value ? [Number(e.target.value)] : [])}
                        className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow">
                        <option value="">Auswählen…</option>
                        {teams.filter(t => t.is_active).map(t => (
                          <option key={t.id} value={t.id}>{t.name}</option>
                        ))}
                      </select>
                    )}
                  </div>
                  {createError && <p className="text-brand-error text-sm">{createError}</p>}
                </div>
                <div className="flex gap-2 pt-4">
                  <button onClick={() => setWizardStep(1)} className="border rounded-md px-4 py-2 text-sm hover:bg-gray-50">← Zurück</button>
                  <button
                    onClick={() => {
                      if (selectedDate && selectedTeamIds.length > 0) {
                        loadTemplates().then(() => setWizardStep(3))
                      }
                    }}
                    disabled={!selectedDate || selectedTeamIds.length === 0}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                  >Weiter →</button>
                </div>
              </div>
            )}

            {wizardStep === 3 && (
              <div>
                <h2 className="text-lg font-bold mb-4">Dienstplan-Vorlage</h2>
                {(() => {
                  const filteredTemplates = templates.filter(t => t.template_type === eventType)
                  return filteredTemplates.length === 0 ? (
                    <div className="text-center py-6">
                      <p className="text-gray-500">Keine passende Vorlage — Event wird ohne Dienste angelegt.</p>
                    </div>
                  ) : (
                    <div className="space-y-2 mb-4">
                      {filteredTemplates.map(t => (
                        <label key={t.id} className="flex items-center gap-2 p-3 border rounded-lg hover:bg-gray-50 cursor-pointer">
                          <input type="radio" name="template" checked={selectedTemplate === t.id}
                            onChange={() => setSelectedTemplate(t.id)} className="rounded-full" />
                          <div className="flex-1">
                            <div className="font-medium text-sm">{t.name}</div>
                            <div className="text-xs text-gray-500">{t.game_duration_minutes} Min</div>
                          </div>
                        </label>
                      ))}
                    </div>
                  )
                })()}
                <div className="flex gap-2 pt-4">
                  <button onClick={() => setWizardStep(2)} className="border rounded-md px-4 py-2 text-sm hover:bg-gray-50">← Zurück</button>
                  <button
                    onClick={() => {
                      const filteredTemplates = templates.filter(t => t.template_type === eventType)
                      if (selectedTemplate) {
                        handleFetchPreview()
                      } else if (filteredTemplates.length === 0) {
                        setWizardStep(4)
                      }
                    }}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
                    disabled={previewLoading || creating}
                  >
                    {previewLoading || creating ? 'Laden…' : 'Weiter →'}
                  </button>
                </div>
              </div>
            )}

            {wizardStep === 4 && (
              <div>
                <h2 className="text-lg font-bold mb-4">Dienste bestätigen</h2>
                {preview.length === 0 ? (
                  <p className="text-sm text-gray-500 mb-4">Keine Dienste vorhanden.</p>
                ) : (
                  <>
                    <p className="text-sm text-gray-500 mb-3">
                      Dienste ({selectedSlotIndices.size} ausgewählt):
                    </p>
                    <div className="space-y-1.5 mb-4 max-h-56 overflow-y-auto">
                      {preview.map((s, i) => (
                        <label key={i} className="flex items-center gap-2.5 p-2 rounded-lg hover:bg-gray-50 cursor-pointer">
                          <input type="checkbox" checked={selectedSlotIndices.has(i)} onChange={() => toggleSlot(i)}
                            className="rounded" />
                          <span className="font-mono text-sm font-semibold w-12">{s.event_time}</span>
                          <span className="text-sm flex-1">{s.duty_type_name}</span>
                          {s.role_desc && <span className="text-xs text-gray-400">({s.role_desc})</span>}
                          <span className="text-xs text-gray-400 ml-auto">{s.slots_count}×</span>
                        </label>
                      ))}
                    </div>
                  </>
                )}
                {createError && <p className="text-brand-error text-sm mb-3">{createError}</p>}
                <div className="flex gap-2 pt-2">
                  <button onClick={() => setWizardStep(3)} className="border rounded-md px-3 py-2 text-sm hover:bg-gray-50">← Zurück</button>
                  <button
                    onClick={() => doCreateGame([])}
                    disabled={creating}
                    className="border rounded-md px-3 py-2 text-sm text-gray-500 hover:bg-gray-50 disabled:opacity-50"
                  >Ohne Dienste</button>
                  <button
                    onClick={() => doCreateGame(preview.filter((_, i) => selectedSlotIndices.has(i)))}
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
