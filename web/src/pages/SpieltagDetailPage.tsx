import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { Trash2, AlertTriangle } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import DutySlotList, { BoardSlot } from '../components/DutySlotList'
import { AUDIENCE_OPTIONS } from '../lib/constants'

interface GameDetail {
  id: number
  date: string
  time: string
  end_time?: string | null
  opponent: string
  event_type?: string
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
  audiences?: string[] | null
}

interface DutyType {
  id: number
  name: string
  audiences?: string[] | null
}

interface SlotPreview {
  duty_type_id: number
  duty_type_name: string
  event_time: string
  slots_count: number
  role_desc: string
  conflict?: boolean
}

interface Template {
  id: number
  name: string
  template_type: string
  duration_minutes: number
}

const INPUT_WIZ = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'
const BTN_SECONDARY = 'border border-brand-border rounded-md px-4 py-2 text-sm text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors'

export default function SpieltagDetailPage() {
  const { gameId } = useParams<{ gameId: string }>()
  const { user } = useAuth()
  const navigate = useNavigate()
  const canEdit = user?.role === 'admin' || user?.role === 'vorstand' || user?.role === 'trainer'

  const [game, setGame] = useState<GameDetail | null>(null)
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

  const [showRegen, setShowRegen] = useState(false)
  const [regenTemplates, setRegenTemplates] = useState<Template[]>([])
  const [regenTemplateID, setRegenTemplateID] = useState<number | null>(null)
  const [regenPreview, setRegenPreview] = useState<SlotPreview[]>([])
  const [regenPreviewLoading, setRegenPreviewLoading] = useState(false)
  const [regenSaving, setRegenSaving] = useState(false)
  const [regenError, setRegenError] = useState<string | null>(null)
  const [regenKeptSlots, setRegenKeptSlots] = useState<number | null>(null)

  useEscapeKey(
    showDeleteGame ? () => setShowDeleteGame(false) :
    showRegen ? () => setShowRegen(false) :
    deleteSlotId !== null ? () => setDeleteSlotId(null) :
    editSlot ? () => setEditSlot(null) :
    showAddSlot ? () => setShowAddSlot(false) :
    null
  )

  const loadGame = async () => {
    try {
      const r = await api.get(`/kalender/${gameId}`)
      setGame(r.data.game)
      setSlots(r.data.slots ?? [])
    } catch (e: any) {
      if (e?.response?.status === 404) setNotFound(true)
    }
  }

  const loadBoard = async () => {
    try {
      const r = await api.get(`/duty-board?game_id=${gameId}`)
      const groups: any[] = r.data ?? []
      setBoardSlots(groups.length > 0 ? groups[0].slots ?? [] : [])
    } catch {
      setBoardSlots([])
    }
  }

  useEffect(() => {
    Promise.all([
      loadGame(),
      loadBoard(),
      canEdit ? api.get('/duty-types').then(r => setDutyTypes(r.data ?? [])) : Promise.resolve(),
    ]).finally(() => setLoading(false))
  }, [gameId])

  useLiveUpdates((event) => { if (event === 'duties') loadBoard() })

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
        team_id: game.team_id,
        season_id: game.season_id,
        game_id: game.id,
        audiences: addAudiences.length > 0 ? addAudiences : null,
      })
      await Promise.all([loadGame(), loadBoard()])
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
      await Promise.all([loadGame(), loadBoard()])
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
      await Promise.all([loadGame(), loadBoard()])
      setDeleteSlotId(null)
    } finally {
      setDeleteSaving(false)
    }
  }

  const handleDeleteGame = async () => {
    if (!gameId) return
    setDeletingGame(true)
    try {
      await api.delete(`/kalender/${gameId}?delete_slots=true`)
      navigate(game ? `/kalender?date=${game.date.slice(0, 10)}` : '/kalender')
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
      const r = await api.get('/duty-templates')
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
      const r = await api.get(`/duty-templates/${templateID}/preview?time=${gameTime}&game_id=${gameId}`)
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
      const r = await api.post(`/kalender/${gameId}/regenerate`, { template_id: regenTemplateID })
      await Promise.all([loadGame(), loadBoard()])
      setRegenKeptSlots(r.data.kept_slots)
      setShowRegen(false)
    } catch (e: any) {
      const msg = e?.response?.data || 'Regenerierung fehlgeschlagen.'
      setRegenError(typeof msg === 'string' ? msg : 'Regenerierung fehlgeschlagen.')
    } finally {
      setRegenSaving(false)
    }
  }

  if (loading) return <div className="text-brand-text-muted text-sm">Laden…</div>
  if (notFound) return (
    <div className="text-center py-12">
      <p className="text-brand-text-muted mb-4">Spiel nicht gefunden.</p>
      <Link to="/spielplan" className="text-brand-text hover:underline text-sm">← Zurück zum Spielplan</Link>
    </div>
  )
  if (!game) return null

  const dateFormatted = new Date(game.date.slice(0, 10) + 'T12:00:00').toLocaleDateString('de-DE', {
    weekday: 'long', year: 'numeric', month: 'long', day: 'numeric',
  })
  const isPast = new Date(game.date.slice(0, 10) + 'T23:59:59') < new Date()

  return (
    <div className="max-w-2xl">
      <Link to={`/kalender?date=${game.date.slice(0, 10)}`} className="text-sm text-brand-text-muted hover:text-brand-text mb-4 inline-block">
        ← Kalender
      </Link>

      {/* Game header */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 mb-6">
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-brand-text">
              {game.event_type === 'generisch' ? (game.opponent || '(kein Name)') : `Team vs ${game.opponent || '(kein Gegner)'}`}
            </h1>
            <p className="text-brand-text-muted mt-1">{game.team_name}</p>
            <p className="text-brand-text-muted text-sm mt-1">
              {dateFormatted} · {game.event_type === 'generisch' && game.end_time
                ? `${game.time}–${game.end_time} Uhr`
                : `${game.time} Uhr`}
            </p>
          </div>
          {canEdit && (
            <div className="flex gap-2 flex-shrink-0">
              <button
                onClick={handleOpenRegen}
                className="text-sm border border-brand-border rounded-md px-3 py-1.5 text-brand-text-muted hover:text-brand-text hover:bg-brand-border-subtle transition-colors"
              >
                ↺ Dienste neu generieren
              </button>
              <button
                onClick={() => setShowDeleteGame(true)}
                className="text-sm bg-brand-danger text-white rounded-md px-3 py-1.5 hover:bg-brand-danger/90 transition-colors flex items-center gap-1.5"
              >
                <Trash2 className="w-4 h-4" /> Event löschen
              </button>
            </div>
          )}
        </div>
        {regenKeptSlots !== null && regenKeptSlots > 0 && (
          <div className="mt-3 p-3 bg-brand-warning-light border border-brand-warning/40 rounded-lg text-sm text-brand-text">
            Hinweis: {regenKeptSlots} belegte Dienst{regenKeptSlots !== 1 ? 'e' : ''} wurden nicht überschrieben.
          </div>
        )}
      </div>

      {/* Slots */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mb-4">
        <div className="flex items-center justify-between px-4 py-3 border-b border-brand-border-subtle">
          <h2 className="font-semibold text-brand-text">Dienste</h2>
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
            onReload={() => Promise.all([loadGame(), loadBoard()])}
            onEdit={canEdit ? (id) => { const s = slots.find(x => x.id === id); if (s) openEditSlot(s) } : undefined}
          />
        )}
      </div>

      {/* Add slot modal */}
      {showAddSlot && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
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
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
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
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
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

      {/* Regenerate dialog */}
      {showRegen && (
        <div className="fixed inset-0 bg-brand-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-brand-white rounded-xl border-t-4 border-brand-yellow p-6 w-full max-w-md shadow-2xl max-h-[90vh] overflow-y-auto">
            <h3 className="font-bold mb-1 text-brand-text">Dienste neu generieren</h3>
            <p className="text-sm text-brand-text-muted mb-4">
              Unbesetzte Dienste werden gelöscht und durch die Vorlage ersetzt.
            </p>

            <div className="mb-4">
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Dienstplan-Vorlage *</label>
              <select
                value={regenTemplateID ?? ''}
                onChange={e => handleRegenTemplateChange(e.target.value ? Number(e.target.value) : null)}
                className={INPUT_WIZ}
              >
                <option value="">Auswählen…</option>
                {regenTemplates.map(t => (
                  <option key={t.id} value={t.id}>
                    {t.name} ({t.template_type}{t.template_type === 'generisch' ? `, ${t.duration_minutes} Min` : ''})
                  </option>
                ))}
              </select>
            </div>

            {regenPreviewLoading && (
              <p className="text-sm text-brand-text-subtle mb-4">Vorschau wird geladen…</p>
            )}

            {!regenPreviewLoading && regenTemplateID && regenPreview.length === 0 && !regenError && (
              <p className="text-sm text-brand-text-subtle mb-4 italic">Keine Dienste in dieser Vorlage.</p>
            )}

            {!regenPreviewLoading && regenPreview.length > 0 && (
              <>
                {regenPreview.some(s => s.conflict) && (
                  <div className="mb-3 p-2.5 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-xs text-brand-danger flex items-start gap-2">
                    <AlertTriangle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                    <span>
                      Gleicher Diensttyp zur selben Uhrzeit existiert bereits für ein anderes Spiel an diesem Tag.
                      Bitte Optimierungsregeln prüfen.
                    </span>
                  </div>
                )}
                <div className="space-y-1 mb-4 max-h-48 overflow-y-auto border border-brand-border-subtle rounded-lg p-2 bg-brand-surface-card">
                  {regenPreview.map((s, i) => (
                    <div key={i} className={`flex items-center gap-2.5 px-1 py-1 text-sm rounded ${s.conflict ? 'bg-brand-danger-light' : ''}`}>
                      <span className="font-mono font-semibold w-12 text-brand-text">{s.event_time}</span>
                      <span className={`flex-1 ${s.conflict ? 'text-brand-danger' : 'text-brand-text'}`}>{s.duty_type_name}</span>
                      {s.role_desc && <span className="text-xs text-brand-text-subtle">({s.role_desc})</span>}
                      <span className="text-xs text-brand-text-subtle">{s.slots_count}×</span>
                      {s.conflict && <span className="text-brand-danger text-xs font-medium flex items-center gap-0.5"><AlertTriangle className="w-3 h-3" /> Konflikt</span>}
                    </div>
                  ))}
                </div>
              </>
            )}

            {regenError && (
              <p className="text-sm text-brand-danger mb-3">{regenError}</p>
            )}

            <div className="p-3 bg-brand-warning-light border border-brand-warning/40 rounded-lg text-xs text-brand-text mb-4">
              Bereits belegte Dienste werden nicht überschrieben.
            </div>

            <div className="flex gap-2">
              <button onClick={() => setShowRegen(false)} className={`flex-1 ${BTN_SECONDARY}`}>Abbrechen</button>
              <button
                onClick={handleRegen}
                disabled={regenSaving || !regenTemplateID || regenPreviewLoading}
                className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-50"
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
  )
}
