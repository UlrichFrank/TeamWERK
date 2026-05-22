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
  template_id?: number | null
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

interface Template {
  id: number
  name: string
  template_type: string
  game_duration_minutes: number
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

  // Regenerate dialog
  const [showRegen, setShowRegen] = useState(false)
  const [regenTemplates, setRegenTemplates] = useState<Template[]>([])
  const [regenTemplateID, setRegenTemplateID] = useState<number | null>(null)
  const [regenPreview, setRegenPreview] = useState<SlotPreview[]>([])
  const [regenPreviewLoading, setRegenPreviewLoading] = useState(false)
  const [regenSaving, setRegenSaving] = useState(false)
  const [regenError, setRegenError] = useState<string | null>(null)
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
    setRegenError(null)
    setRegenPreview([])
    setRegenKeptSlots(null)
    const initialTemplateID = game.template_id ?? null
    setRegenTemplateID(initialTemplateID)
    try {
      const r = await api.get('/admin/duty-templates')
      setRegenTemplates(r.data ?? [])
    } catch {
      setRegenTemplates([])
    }
    setShowRegen(true)
    if (initialTemplateID && game) {
      fetchRegenPreview(initialTemplateID, game.time)
    }
  }

  const fetchRegenPreview = async (templateID: number, gameTime: string) => {
    setRegenPreviewLoading(true)
    setRegenError(null)
    try {
      const r = await api.get(`/admin/duty-templates/${templateID}/preview?time=${gameTime}&game_id=${gameId}`)
      setRegenPreview(r.data ?? [])
    } catch {
      setRegenPreview([])
      setRegenError('Vorschau konnte nicht geladen werden.')
    } finally {
      setRegenPreviewLoading(false)
    }
  }

  const handleRegenTemplateChange = (templateID: number | null) => {
    setRegenTemplateID(templateID)
    setRegenPreview([])
    setRegenError(null)
    if (templateID && game) {
      fetchRegenPreview(templateID, game.time)
    }
  }

  const handleRegen = async () => {
    if (!regenTemplateID) {
      setRegenError('Bitte ein Template auswählen.')
      return
    }
    setRegenSaving(true)
    setRegenError(null)
    try {
      const r = await api.post(`/admin/games/${gameId}/regenerate`, { template_id: regenTemplateID })
      await loadGame()
      setRegenKeptSlots(r.data.kept_slots)
      setShowRegen(false)
    } catch (e: any) {
      const msg = e?.response?.data || 'Regenerierung fehlgeschlagen.'
      setRegenError(typeof msg === 'string' ? msg : 'Regenerierung fehlgeschlagen.')
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
                className="text-sm border rounded-md px-3 py-1.5 hover:bg-gray-50 text-gray-600"
              >
                ↺ Dienste neu generieren
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
          <div className="bg-brand-white rounded-2xl p-6 w-full max-w-md shadow-2xl max-h-[90vh] overflow-y-auto">
            <h3 className="font-bold mb-1">Dienste neu generieren</h3>
            <p className="text-sm text-gray-500 mb-4">
              Unbesetzte Dienste werden gelöscht und durch die Vorlage ersetzt.
            </p>

            <div className="mb-4">
              <label className="block text-sm font-medium mb-1">Dienstplan-Vorlage *</label>
              <select
                value={regenTemplateID ?? ''}
                onChange={e => handleRegenTemplateChange(e.target.value ? Number(e.target.value) : null)}
                className="w-full border rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-yellow"
              >
                <option value="">Auswählen…</option>
                {regenTemplates.map(t => (
                  <option key={t.id} value={t.id}>{t.name} ({t.template_type}, {t.game_duration_minutes} Min)</option>
                ))}
              </select>
            </div>

            {regenPreviewLoading && (
              <p className="text-sm text-gray-400 mb-4">Vorschau wird geladen…</p>
            )}

            {!regenPreviewLoading && regenTemplateID && regenPreview.length === 0 && !regenError && (
              <p className="text-sm text-gray-400 mb-4 italic">Keine Dienste in dieser Vorlage.</p>
            )}

            {!regenPreviewLoading && regenPreview.length > 0 && (
              <div className="space-y-1 mb-4 max-h-48 overflow-y-auto border rounded-lg p-2 bg-gray-50">
                {regenPreview.map((s, i) => (
                  <div key={i} className="flex items-center gap-2.5 px-1 py-1 text-sm">
                    <span className="font-mono font-semibold w-12 text-gray-700">{s.event_time}</span>
                    <span className="flex-1">{s.duty_type_name}</span>
                    {s.role_desc && <span className="text-xs text-gray-400">({s.role_desc})</span>}
                    <span className="text-xs text-gray-400 ml-auto">{s.slots_count}×</span>
                  </div>
                ))}
              </div>
            )}

            {regenError && (
              <p className="text-sm text-red-600 mb-3">{regenError}</p>
            )}

            <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-xs text-yellow-700 mb-4">
              Bereits belegte Dienste werden nicht überschrieben.
            </div>

            <div className="flex gap-2">
              <button onClick={() => setShowRegen(false)}
                className="flex-1 border rounded-md px-4 py-2 text-sm hover:bg-gray-50">Abbrechen</button>
              <button
                onClick={handleRegen}
                disabled={regenSaving || !regenTemplateID || regenPreviewLoading}
                className="flex-1 bg-brand-yellow text-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors disabled:opacity-50"
              >
                {regenSaving ? 'Generieren…' : 'Anwenden'}
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
