import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface GameDetail {
  id: number
  date: string
  time: string
  opponent: string
  team_id: number
  team_name: string
  season_id: number
}

interface SlotDetail {
  id: number
  duty_type_name: string
  event_time: string
  role_description: string
  slots_total: number
  slots_filled: number
}

interface DutyType {
  id: number
  name: string
}

interface SlotPreview {
  duty_type_id: number
  duty_type_name: string
  event_time: string
  slots_count: number
  role_desc: string
}

function ProgressBar({ filled, total }: { filled: number; total: number }) {
  const pct = total > 0 ? Math.round((filled / total) * 100) : 0
  const color = pct === 100 ? 'bg-brand-success' : pct > 0 ? 'bg-brand-warning' : 'bg-brand-error'
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-2 bg-gray-200 rounded-full overflow-hidden">
        <div className={`h-full rounded-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <span className="text-xs text-gray-500 w-10 text-right">{filled}/{total}</span>
    </div>
  )
}

export default function SpieltagDetailPage() {
  const { gameId } = useParams<{ gameId: string }>()
  const { user } = useAuth()
  const navigate = useNavigate()
  const canEdit = user?.role === 'admin' || user?.role === 'vorstand' || user?.role === 'trainer'

  const [game, setGame] = useState<GameDetail | null>(null)
  const [slots, setSlots] = useState<SlotDetail[]>([])
  const [dutyTypes, setDutyTypes] = useState<DutyType[]>([])
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)

  // Add slot form
  const [showAddSlot, setShowAddSlot] = useState(false)
  const [addDutyTypeId, setAddDutyTypeId] = useState<number | ''>('')
  const [addEventTime, setAddEventTime] = useState('')
  const [addSlotsTotal, setAddSlotsTotal] = useState(1)
  const [addRoleDesc, setAddRoleDesc] = useState('')
  const [addSaving, setAddSaving] = useState(false)

  // Edit slot modal
  const [editSlot, setEditSlot] = useState<SlotDetail | null>(null)
  const [editEventTime, setEditEventTime] = useState('')
  const [editSlotsTotal, setEditSlotsTotal] = useState(1)
  const [editRoleDesc, setEditRoleDesc] = useState('')
  const [editSaving, setEditSaving] = useState(false)

  // Delete slot
  const [deleteSlotId, setDeleteSlotId] = useState<number | null>(null)
  const [deleteSaving, setDeleteSaving] = useState(false)

  // Delete game
  const [showDeleteGame, setShowDeleteGame] = useState(false)
  const [deletingGame, setDeletingGame] = useState(false)

  // Regenerate flow
  const [showRegen, setShowRegen] = useState(false)
  const [regenPreview, setRegenPreview] = useState<SlotPreview[]>([])
  const [regenSelectedIndices, setRegenSelectedIndices] = useState<Set<number>>(new Set())
  const [regenLoading, setRegenLoading] = useState(false)
  const [regenSaving, setRegenSaving] = useState(false)
  const [regenKeptSlots, setRegenKeptSlots] = useState<number | null>(null)

  const loadGame = async () => {
    try {
      const r = await api.get(`/games/${gameId}`)
      setGame(r.data.game)
      setSlots(r.data.slots ?? [])
    } catch (e: any) {
      if (e?.response?.status === 404) setNotFound(true)
    }
  }

  useEffect(() => {
    Promise.all([
      loadGame(),
      canEdit ? api.get('/admin/duty-types').then(r => setDutyTypes(r.data ?? [])) : Promise.resolve(),
    ]).finally(() => setLoading(false))
  }, [gameId])

  const handleAddSlot = async () => {
    if (!addDutyTypeId || !game) return
    setAddSaving(true)
    try {
      await api.post('/duty-slots', {
        event_name: `Heimspiel vs. ${game.opponent || ''}`.trim(),
        event_date: game.date.slice(0, 10),
        event_time: addEventTime || null,
        duty_type_id: addDutyTypeId,
        role_desc: addRoleDesc,
        slots_total: addSlotsTotal,
        team_id: game.team_id,
        season_id: game.season_id,
        game_id: game.id,
      })
      await loadGame()
      setShowAddSlot(false)
      setAddDutyTypeId('')
      setAddEventTime('')
      setAddSlotsTotal(1)
      setAddRoleDesc('')
    } finally {
      setAddSaving(false)
    }
  }

  const openEditSlot = (s: SlotDetail) => {
    setEditSlot(s)
    setEditEventTime(s.event_time)
    setEditSlotsTotal(s.slots_total)
    setEditRoleDesc(s.role_description)
  }

  const handleEditSlot = async () => {
    if (!editSlot) return
    setEditSaving(true)
    try {
      await api.put(`/duty-slots/${editSlot.id}`, {
        event_name: `Heimspiel vs. ${game?.opponent || ''}`.trim(),
        event_date: game?.date.slice(0, 10),
        event_time: editEventTime || null,
        role_desc: editRoleDesc,
        slots_total: editSlotsTotal,
      })
      await loadGame()
      setEditSlot(null)
    } finally {
      setEditSaving(false)
    }
  }

  const handleDeleteSlot = async () => {
    if (deleteSlotId === null) return
    setDeleteSaving(true)
    try {
      await api.delete(`/duty-slots/${deleteSlotId}`)
      await loadGame()
      setDeleteSlotId(null)
    } finally {
      setDeleteSaving(false)
    }
  }

  const handleDeleteGame = async () => {
    if (!gameId) return
    setDeletingGame(true)
    try {
      await api.delete(`/admin/games/${gameId}`)
      navigate('/spielplan')
    } finally {
      setDeletingGame(false)
    }
  }

  const handleOpenRegen = async () => {
    if (!game) return
    setRegenLoading(true)
    setRegenKeptSlots(null)
    try {
      const r = await api.get(`/admin/game-template/preview?date=${game.date.slice(0, 10)}&time=${game.time}&game_id=${gameId}`)
      const p: SlotPreview[] = r.data ?? []
      setRegenPreview(p)
      setRegenSelectedIndices(new Set(p.map((_, i) => i)))
      setShowRegen(true)
    } finally {
      setRegenLoading(false)
    }
  }

  const toggleRegenSlot = (i: number) => {
    setRegenSelectedIndices(prev => {
      const next = new Set(prev)
      next.has(i) ? next.delete(i) : next.add(i)
      return next
    })
  }

  const handleRegen = async () => {
    setRegenSaving(true)
    try {
      const r = await api.post(`/admin/games/${gameId}/regenerate`, {
        slots: regenPreview.filter((_, i) => regenSelectedIndices.has(i)).map(s => ({
          duty_type_id: s.duty_type_id,
          event_time: s.event_time,
          slots_count: s.slots_count,
          role_desc: s.role_desc,
        })),
      })
      await loadGame()
      setRegenKeptSlots(r.data.kept_slots)
      setShowRegen(false)
    } finally {
      setRegenSaving(false)
    }
  }

  if (loading) return <div className="text-gray-400 text-sm">Laden…</div>
  if (notFound) return (
    <div className="text-center py-12">
      <p className="text-gray-500 mb-4">Spiel nicht gefunden.</p>
      <Link to="/spielplan" className="text-brand-black hover:underline text-sm">← Zurück zum Spielplan</Link>
    </div>
  )
  if (!game) return null

  const dateFormatted = new Date(game.date.slice(0, 10) + 'T12:00:00').toLocaleDateString('de-DE', {
    weekday: 'long', year: 'numeric', month: 'long', day: 'numeric',
  })

  return (
    <div className="max-w-2xl">
      <Link to="/spielplan" className="text-sm text-gray-400 hover:text-black mb-4 inline-block">
        ← Spielplan
      </Link>

      {/* Game header */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold">vs. {game.opponent || '(kein Gegner)'}</h1>
            <p className="text-gray-500 mt-1">{game.team_name}</p>
            <p className="text-gray-500 text-sm mt-1">{dateFormatted} · {game.time} Uhr</p>
          </div>
          {canEdit && (
            <div className="flex gap-2 flex-shrink-0">
              <button
                onClick={handleOpenRegen}
                disabled={regenLoading}
                className="text-sm border rounded-md px-3 py-1.5 hover:bg-gray-50 text-gray-600 disabled:opacity-50"
              >
                {regenLoading ? 'Laden…' : '↺ Dienste neu generieren'}
              </button>
              <button
                onClick={() => setShowDeleteGame(true)}
                className="text-sm border border-brand-error rounded-md px-3 py-1.5 hover:bg-red-50 text-brand-error disabled:opacity-50"
              >
                🗑 Event löschen
              </button>
            </div>
          )}
        </div>
        {regenKeptSlots !== null && regenKeptSlots > 0 && (
          <div className="mt-3 p-3 bg-brand-warning-light border border-brand-warning rounded-lg text-sm text-brand-warning">
            Hinweis: {regenKeptSlots} belegte Dienst{regenKeptSlots !== 1 ? 'e' : ''} wurden nicht überschrieben.
          </div>
        )}
      </div>

      {/* Slots */}
      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-4">
        <div className="flex items-center justify-between px-4 py-3 border-b">
          <h2 className="font-semibold">Dienste</h2>
          {canEdit && (
            <button
              onClick={() => setShowAddSlot(true)}
              className="text-sm bg-brand-yellow text-brand-black px-3 py-1.5 rounded-md font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
            >
              + Dienst hinzufügen
            </button>
          )}
        </div>

        {slots.length === 0 ? (
          <p className="text-sm text-gray-400 text-center py-8 italic">Keine Dienste für dieses Spiel angelegt</p>
        ) : (
          <div className="divide-y">
            {slots.map(s => (
              <div key={s.id} className="px-4 py-3 flex items-start gap-4">
                <div className="font-mono text-sm font-semibold w-12 flex-shrink-0 pt-0.5">
                  {s.event_time || '–'}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-sm">{s.duty_type_name}</div>
                  {s.role_description && (
                    <div className="text-xs text-gray-400 mt-0.5">{s.role_description}</div>
                  )}
                  <div className="mt-1.5">
                    <ProgressBar filled={s.slots_filled} total={s.slots_total} />
                  </div>
                </div>
                {canEdit && (
                  <div className="flex gap-1 flex-shrink-0">
                    <button
                      onClick={() => openEditSlot(s)}
                      className="text-xs text-gray-400 hover:text-black px-2 py-1 rounded hover:bg-gray-100"
                    >Bearbeiten</button>
                    <button
                      onClick={() => setDeleteSlotId(s.id)}
                      className="text-xs text-gray-400 hover:text-brand-error px-2 py-1 rounded hover:bg-red-50"
                    >Löschen</button>
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Add slot modal */}
      {showAddSlot && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-2xl p-6 w-full max-w-sm shadow-2xl">
            <h3 className="font-bold mb-4">Dienst hinzufügen</h3>
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium mb-1">Diensttyp *</label>
                <select value={addDutyTypeId} onChange={e => setAddDutyTypeId(Number(e.target.value))}
                  className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow">
                  <option value="">Auswählen…</option>
                  {dutyTypes.map(dt => <option key={dt.id} value={dt.id}>{dt.name}</option>)}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Uhrzeit</label>
                <input type="time" value={addEventTime} onChange={e => setAddEventTime(e.target.value)}
                  className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Personen</label>
                <input type="number" min={1} value={addSlotsTotal} onChange={e => setAddSlotsTotal(Number(e.target.value))}
                  className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Rollenbezeichnung</label>
                <input type="text" value={addRoleDesc} onChange={e => setAddRoleDesc(e.target.value)}
                  placeholder="z.B. Aufbau, Bewirtung…"
                  className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
              </div>
              <div className="flex gap-2 pt-1">
                <button onClick={() => setShowAddSlot(false)}
                  className="flex-1 border rounded-md px-4 py-2 text-sm hover:bg-gray-50">Abbrechen</button>
                <button onClick={handleAddSlot} disabled={!addDutyTypeId || addSaving}
                  className="flex-1 bg-brand-yellow text-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-50">
                  {addSaving ? 'Speichern…' : 'Hinzufügen'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Edit slot modal */}
      {editSlot && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-2xl p-6 w-full max-w-sm shadow-2xl">
            <h3 className="font-bold mb-4">Dienst bearbeiten</h3>
            <p className="text-sm text-gray-500 mb-3 font-medium">{editSlot.duty_type_name}</p>
            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium mb-1">Uhrzeit</label>
                <input type="time" value={editEventTime} onChange={e => setEditEventTime(e.target.value)}
                  className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Personen</label>
                <input type="number" min={1} value={editSlotsTotal} onChange={e => setEditSlotsTotal(Number(e.target.value))}
                  className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Rollenbezeichnung</label>
                <input type="text" value={editRoleDesc} onChange={e => setEditRoleDesc(e.target.value)}
                  className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow" />
              </div>
              <div className="flex gap-2 pt-1">
                <button onClick={() => setEditSlot(null)}
                  className="flex-1 border rounded-md px-4 py-2 text-sm hover:bg-gray-50">Abbrechen</button>
                <button onClick={handleEditSlot} disabled={editSaving}
                  className="flex-1 bg-brand-yellow text-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-50">
                  {editSaving ? 'Speichern…' : 'Speichern'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Delete slot confirmation */}
      {deleteSlotId !== null && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-2xl p-6 w-full max-w-sm shadow-2xl">
            <h3 className="font-bold mb-2">Dienst löschen?</h3>
            <p className="text-sm text-gray-500 mb-5">Dieser Dienst wird endgültig gelöscht.</p>
            <div className="flex gap-2">
              <button onClick={() => setDeleteSlotId(null)}
                className="flex-1 border rounded-md px-4 py-2 text-sm hover:bg-gray-50">Abbrechen</button>
              <button onClick={handleDeleteSlot} disabled={deleteSaving}
                className="flex-1 bg-brand-error hover:bg-brand-error text-brand-white rounded-md px-4 py-2 text-sm disabled:opacity-50">
                {deleteSaving ? 'Löschen…' : 'Löschen'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Regenerate dialog */}
      {showRegen && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-2xl p-6 w-full max-w-md shadow-2xl">
            <h3 className="font-bold mb-1">Dienste neu generieren</h3>
            <p className="text-sm text-gray-500 mb-3">
              Unbesetzte Dienste werden gelöscht und durch diese ersetzt:
            </p>
            {regenPreview.length === 0 ? (
              <p className="text-sm text-gray-400 mb-4 italic">Kein Template konfiguriert.</p>
            ) : (
              <div className="space-y-1.5 mb-4 max-h-56 overflow-y-auto">
                {regenPreview.map((s, i) => (
                  <label key={i} className="flex items-center gap-2.5 p-2 rounded-md hover:bg-gray-50 cursor-pointer">
                    <input type="checkbox" checked={regenSelectedIndices.has(i)} onChange={() => toggleRegenSlot(i)}
                      className="rounded" />
                    <span className="font-mono text-sm font-semibold w-12">{s.event_time}</span>
                    <span className="text-sm flex-1">{s.duty_type_name}</span>
                    {s.role_desc && <span className="text-xs text-gray-400">({s.role_desc})</span>}
                    <span className="text-xs text-gray-400 ml-auto">{s.slots_count}×</span>
                  </label>
                ))}
              </div>
            )}
            <div className="p-3 bg-brand-warning-light border border-brand-warning rounded-lg text-xs text-brand-warning mb-4">
              Bereits belegte Dienste werden nicht überschrieben.
            </div>
            <div className="flex gap-2">
              <button onClick={() => setShowRegen(false)}
                className="flex-1 border rounded-md px-4 py-2 text-sm hover:bg-gray-50">Abbrechen</button>
              <button onClick={handleRegen} disabled={regenSaving}
                className="flex-1 bg-brand-yellow text-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-50">
                {regenSaving ? 'Generieren…' : 'Bestätigen'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete game confirmation */}
      {showDeleteGame && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-2xl p-6 w-full max-w-sm shadow-2xl">
            <h3 className="font-bold mb-2">Event löschen?</h3>
            <p className="text-sm text-gray-500 mb-1">
              <strong>vs. {game.opponent || '(kein Gegner)'}</strong> ({dateFormatted})
            </p>
            <p className="text-sm text-gray-500 mb-5">
              Dieses Event und alle zugehörigen Dienste werden endgültig gelöscht.
            </p>
            <div className="flex gap-2">
              <button onClick={() => setShowDeleteGame(false)}
                className="flex-1 border rounded-md px-4 py-2 text-sm hover:bg-gray-50">Abbrechen</button>
              <button onClick={handleDeleteGame} disabled={deletingGame}
                className="flex-1 bg-brand-error hover:bg-red-700 text-brand-white rounded-md px-4 py-2 text-sm font-medium disabled:opacity-50">
                {deletingGame ? 'Löschen…' : 'Endgültig löschen'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
