import { useEffect, useState } from 'react'
import { X } from 'lucide-react'
import { api } from '../lib/api'
import { formatTeamList } from '../lib/teamName'
import { useEscapeKey } from '../lib/useEscapeKey'
import { errorStatus } from '../lib/errors'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import DutySlotList, { BoardSlot } from './DutySlotList'
import { AUDIENCE_OPTIONS } from '../lib/constants'

interface GameDetail {
  id: number
  date: string
  time: string
  end_time?: string | null
  opponent: string
  event_type?: string
  team_id: number
  teams?: Array<{ id: number; name: string; display_short?: string; display_long?: string }>
  team_display_long_csv?: string
  season_id: number
  template_id?: number | null
  can_edit?: boolean
}

interface SlotDetail {
  id: number
  duty_type_name: string
  event_time: string
  role_description: string
  slots_total: number
  slots_filled: number
  audiences?: string[] | null
}

interface DutyType {
  id: number
  name: string
  audiences?: string[] | null
}

interface Props {
  gameId: number
  onClose: () => void
  onChanged?: () => void
  onDeleted?: () => void
}

const INPUT_WIZ = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_SECONDARY = 'border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors'

export default function SpieltagDetailModal({ gameId, onClose, onChanged, onDeleted }: Props) {
  const [game, setGame] = useState<GameDetail | null>(null)
  const canEdit = game?.can_edit === true
  const [slots, setSlots] = useState<SlotDetail[]>([])
  const [boardSlots, setBoardSlots] = useState<BoardSlot[]>([])
  const [dutyTypes, setDutyTypes] = useState<DutyType[]>([])
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)

  const [showAddSlot, setShowAddSlot] = useState(false)
  const [addDutyTypeId, setAddDutyTypeId] = useState<number | ''>('')
  const [addEventTime, setAddEventTime] = useState('')
  const [addSlotsTotal, setAddSlotsTotal] = useState(1)
  const [addAudiences, setAddAudiences] = useState<string[]>([])
  const [addSaving, setAddSaving] = useState(false)

  const [editSlot, setEditSlot] = useState<SlotDetail | null>(null)
  const [editEventTime, setEditEventTime] = useState('')
  const [editSlotsTotal, setEditSlotsTotal] = useState(1)
  const [editAudiences, setEditAudiences] = useState<string[]>([])
  const [editSaving, setEditSaving] = useState(false)

  const [deleteSlotId, setDeleteSlotId] = useState<number | null>(null)
  const [deleteSaving, setDeleteSaving] = useState(false)

  const [showDeleteGame, setShowDeleteGame] = useState(false)
  const [deletingGame, setDeletingGame] = useState(false)

  useEscapeKey(
    showDeleteGame ? () => setShowDeleteGame(false) :
    deleteSlotId !== null ? () => setDeleteSlotId(null) :
    editSlot ? () => setEditSlot(null) :
    showAddSlot ? () => setShowAddSlot(false) :
    onClose
  )

  const loadGame = async (): Promise<GameDetail | null> => {
    try {
      const r = await api.get(`/games/${gameId}`)
      setGame(r.data.game)
      setSlots(r.data.slots ?? [])
      return r.data.game as GameDetail
    } catch (e) {
      if (errorStatus(e) === 404) setNotFound(true)
      return null
    }
  }

  const loadBoard = async (mayEdit: boolean) => {
    try {
      const params = mayEdit ? `game_id=${gameId}&audience=all` : `game_id=${gameId}`
      const r = await api.get(`/duty-board?${params}`)
      const groups: Array<{ slots?: BoardSlot[] }> = r.data ?? []
      setBoardSlots(groups.length > 0 ? groups[0].slots ?? [] : [])
    } catch {
      setBoardSlots([])
    }
  }

  useEffect(() => {
    (async () => {
      const g = await loadGame()
      const mayEdit = g?.can_edit === true
      await Promise.all([
        loadBoard(mayEdit),
        mayEdit ? api.get('/duty-types').then(r => setDutyTypes(r.data ?? [])) : Promise.resolve(),
      ])
      setLoading(false)
    })()
    // loadGame/loadBoard sind stabile Closures; nur Neuladen bei Wechsel des Spiels gewünscht
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameId])

  useLiveUpdates((event) => { if (event === 'duties') loadBoard(canEdit) })

  const reloadAfterMutation = async () => {
    await Promise.all([loadGame(), loadBoard(canEdit)])
    onChanged?.()
  }

  const handleAddSlot = async () => {
    if (!addDutyTypeId || !game) return
    setAddSaving(true)
    try {
      await api.post('/duty-slots', {
        event_name: game.event_type === 'generisch'
          ? (game.opponent || 'Event')
          : `${game.event_type === 'heim' ? 'Heimspiel' : 'Auswärtsspiel'} Team vs ${game.opponent || ''}`.trim(),
        event_date: game.date.slice(0, 10),
        event_time: addEventTime || null,
        duty_type_id: addDutyTypeId,
        slots_total: addSlotsTotal,
        team_id: game.teams?.[0]?.id ?? game.team_id,
        season_id: game.season_id,
        game_id: game.id,
        audiences: addAudiences.length > 0 ? addAudiences : null,
      })
      await reloadAfterMutation()
      setShowAddSlot(false)
      setAddDutyTypeId('')
      setAddEventTime('')
      setAddSlotsTotal(1)
      setAddAudiences([])
    } finally {
      setAddSaving(false)
    }
  }

  const openEditSlot = (s: SlotDetail) => {
    setEditSlot(s)
    setEditEventTime(s.event_time)
    setEditSlotsTotal(s.slots_total)
    setEditAudiences(s.audiences ?? [])
  }

  const handleEditSlot = async () => {
    if (!editSlot) return
    setEditSaving(true)
    try {
      await api.put(`/duty-slots/${editSlot.id}`, {
        event_name: game?.event_type === 'generisch'
          ? (game?.opponent || 'Event')
          : `${game?.event_type === 'heim' ? 'Heimspiel' : 'Auswärtsspiel'} Team vs ${game?.opponent || ''}`.trim(),
        event_date: game?.date.slice(0, 10),
        event_time: editEventTime || null,
        slots_total: editSlotsTotal,
        audiences: editAudiences.length > 0 ? editAudiences : null,
      })
      await reloadAfterMutation()
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
      await reloadAfterMutation()
      setDeleteSlotId(null)
    } finally {
      setDeleteSaving(false)
    }
  }

  const handleDeleteGame = async () => {
    setDeletingGame(true)
    try {
      await api.delete(`/games/${gameId}`)
      onDeleted?.()
      onClose()
    } finally {
      setDeletingGame(false)
    }
  }

  const dateFormatted = game ? new Date(game.date.slice(0, 10) + 'T12:00:00').toLocaleDateString('de-DE', {
    weekday: 'long', year: 'numeric', month: 'long', day: 'numeric',
  }) : ''
  const isPast = game ? new Date(game.date.slice(0, 10) + 'T23:59:59') < new Date() : false

  return (
    <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-start justify-between mb-4">
          <div className="min-w-0">
            {loading ? (
              <h2 className="text-lg font-bold text-brand-text">Laden…</h2>
            ) : notFound ? (
              <h2 className="text-lg font-bold text-brand-text">Spiel nicht gefunden</h2>
            ) : game ? (
              <>
                <h2 className="text-lg font-bold text-brand-text truncate">
                  {game.event_type === 'generisch' ? (game.opponent || '(kein Name)') : `Team vs ${game.opponent || '(kein Gegner)'}`}
                </h2>
                <p className="text-brand-text-muted text-sm mt-0.5">{game.team_display_long_csv || (game.teams ? formatTeamList(game.teams, 'long') : '')}</p>
                <p className="text-brand-text-muted text-sm mt-0.5">
                  {dateFormatted} · {game.event_type === 'generisch' && game.end_time
                    ? `${game.time}–${game.end_time} Uhr`
                    : `${game.time} Uhr`}
                </p>
              </>
            ) : null}
          </div>
          <div className="flex items-center gap-2 flex-shrink-0 ml-2">
            <button onClick={onClose} className="p-1 rounded hover:bg-brand-border-subtle transition-colors" aria-label="Schließen">
              <X className="w-5 h-5 text-brand-text-muted" />
            </button>
          </div>
        </div>

        {!loading && !notFound && game && (
          <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-4">
            <div className="flex items-center justify-between px-4 py-3 border-b border-brand-border-subtle">
              <h3 className="font-semibold text-brand-text">Dienste</h3>
              {canEdit && (
                <button
                  onClick={() => setShowAddSlot(true)}
                  className="text-sm bg-brand-yellow text-brand-black px-3 py-1.5 rounded-md font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
                >
                  + Dienst hinzufügen
                </button>
              )}
            </div>

            {boardSlots.length === 0 ? (
              <p className="text-sm text-brand-text-subtle text-center py-8 italic">Keine Dienste für dieses Spiel angelegt</p>
            ) : (
              <DutySlotList
                slots={boardSlots}
                isPast={isPast}
                canEdit={canEdit}
                onReload={reloadAfterMutation}
                onEdit={canEdit ? (id) => { const s = slots.find(x => x.id === id); if (s) openEditSlot(s) } : undefined}
              />
            )}
          </div>
        )}

        {/* Add slot modal */}
        {showAddSlot && (
          <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-[60] p-4">
            <div className="bg-brand-white rounded-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm shadow-2xl">
              <h3 className="font-bold mb-4 text-brand-text">Dienst hinzufügen</h3>
              <div className="space-y-3">
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Diensttyp *</label>
                  <select value={addDutyTypeId} onChange={e => {
                    const dtId = Number(e.target.value)
                    setAddDutyTypeId(dtId)
                    const dt = dutyTypes.find(d => d.id === dtId)
                    setAddAudiences(dt?.audiences ?? [])
                  }} className={INPUT_WIZ}>
                    <option value="">Auswählen…</option>
                    {dutyTypes.map(dt => <option key={dt.id} value={dt.id}>{dt.name}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Uhrzeit</label>
                  <input type="time" value={addEventTime} onChange={e => setAddEventTime(e.target.value)} className={INPUT_WIZ} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Personen</label>
                  <input type="number" min={1} value={addSlotsTotal} onChange={e => setAddSlotsTotal(Number(e.target.value))} className={INPUT_WIZ} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Zielgruppe <span className="text-brand-text-subtle text-xs font-normal">(leer = keine Einschränkung)</span></label>
                  <div className="grid grid-cols-2 gap-1.5 mt-1">
                    {AUDIENCE_OPTIONS.map(o => (
                      <label key={o.value} className="flex items-center gap-2 text-sm cursor-pointer">
                        <input
                          type="checkbox"
                          checked={addAudiences.includes(o.value)}
                          onChange={e => setAddAudiences(prev =>
                            e.target.checked ? [...prev, o.value] : prev.filter(a => a !== o.value)
                          )}
                          className="accent-brand-yellow"
                        />
                        {o.label}
                      </label>
                    ))}
                  </div>
                </div>
                <div className="flex gap-2 pt-1">
                  <button onClick={() => setShowAddSlot(false)} className={`flex-1 ${BTN_SECONDARY}`}>Abbrechen</button>
                  <button onClick={handleAddSlot} disabled={!addDutyTypeId || addSaving}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50">
                    {addSaving ? 'Speichern…' : 'Hinzufügen'}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Edit slot modal */}
        {editSlot && (
          <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-[60] p-4">
            <div className="bg-brand-white rounded-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm shadow-2xl">
              <h3 className="font-bold mb-4 text-brand-text">Dienst bearbeiten</h3>
              <p className="text-sm text-brand-text-muted mb-3 font-medium">{editSlot.duty_type_name}</p>
              <div className="space-y-3">
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Uhrzeit</label>
                  <input type="time" value={editEventTime} onChange={e => setEditEventTime(e.target.value)} className={INPUT_WIZ} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Personen</label>
                  <input type="number" min={1} value={editSlotsTotal} onChange={e => setEditSlotsTotal(Number(e.target.value))} className={INPUT_WIZ} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Zielgruppe <span className="text-brand-text-subtle text-xs font-normal">(leer = keine Einschränkung)</span></label>
                  <div className="grid grid-cols-2 gap-1.5 mt-1">
                    {AUDIENCE_OPTIONS.map(o => (
                      <label key={o.value} className="flex items-center gap-2 text-sm cursor-pointer">
                        <input
                          type="checkbox"
                          checked={editAudiences.includes(o.value)}
                          onChange={e => setEditAudiences(prev =>
                            e.target.checked ? [...prev, o.value] : prev.filter(a => a !== o.value)
                          )}
                          className="accent-brand-yellow"
                        />
                        {o.label}
                      </label>
                    ))}
                  </div>
                </div>
                <div className="flex gap-2 pt-1">
                  <button onClick={() => setEditSlot(null)} className={`flex-1 ${BTN_SECONDARY}`}>Abbrechen</button>
                  <button onClick={handleEditSlot} disabled={editSaving}
                    className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50">
                    {editSaving ? 'Speichern…' : 'Speichern'}
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Delete slot confirmation */}
        {deleteSlotId !== null && (
          <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-[60] p-4">
            <div className="bg-brand-white rounded-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm shadow-2xl">
              <h3 className="font-bold mb-2 text-brand-text">Dienst löschen?</h3>
              <p className="text-sm text-brand-text-muted mb-5">Dieser Dienst wird endgültig gelöscht.</p>
              <div className="flex gap-2">
                <button onClick={() => setDeleteSlotId(null)} className={`flex-1 ${BTN_SECONDARY}`}>Abbrechen</button>
                <button onClick={handleDeleteSlot} disabled={deleteSaving}
                  className="flex-1 bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-50">
                  {deleteSaving ? 'Löschen…' : 'Löschen'}
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Delete game confirmation */}
        {showDeleteGame && game && (
          <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-[60] p-4">
            <div className="bg-brand-white rounded-xl border-t-4 border-brand-yellow p-6 w-full max-w-sm shadow-2xl">
              <h3 className="font-bold mb-2 text-brand-text">Spiel löschen?</h3>
              <p className="text-sm text-brand-text-muted mb-1">
                <strong>{game.event_type === 'generisch' ? (game.opponent || '(kein Name)') : `Team vs ${game.opponent || '(kein Gegner)'}`}</strong> ({dateFormatted})
              </p>
              <p className="text-sm text-brand-text-muted mb-4">
                Dieses Spiel wird endgültig gelöscht.{slots.length > 0 && ` Dabei werden auch ${slots.length} ${slots.length === 1 ? 'verknüpfter Dienst' : 'verknüpfte Dienste'} gelöscht.`}
              </p>
              <div className="flex gap-2">
                <button onClick={() => setShowDeleteGame(false)} className={`flex-1 ${BTN_SECONDARY}`}>Abbrechen</button>
                <button onClick={handleDeleteGame} disabled={deletingGame}
                  className="flex-1 bg-brand-danger text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-50">
                  {deletingGame ? 'Löschen…' : 'Endgültig löschen'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
