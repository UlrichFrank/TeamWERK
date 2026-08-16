import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Trash2, BookOpen } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useEscapeKey } from '../lib/useEscapeKey'
import { useWindowedList } from '../hooks/useWindowedList'
import WindowedTableBody from './WindowedTableBody'
import PersonChip from './PersonChip'
import ActionMenu from './ActionMenu'
import DeleteReasonFields, { deletionPayload } from './DeleteReasonFields'
import { AUDIENCE_LABELS } from '../lib/constants'
import type { ProxyChild } from '../pages/DutyPage'

// Bewusst schlank: Board liefert nur Namen inline; Avatar/Kontakt lädt
// PersonChip on-demand über GET /api/users/{id}/contact (Sichtbarkeitsregeln
// wie bisher serverseitig via *_visible).
export interface PublicAssignee {
  user_id: number
  name: string
}

export interface BoardSlot {
  id: number
  duty_type: string
  duty_type_id: number
  has_instruction: boolean
  event_time: string
  slots_total: number
  vacancies: number
  claimed_by_me: boolean
  role_desc?: string
  audiences?: string[] | null
  assignees?: PublicAssignee[]
}


interface DutySlotListProps {
  slots: BoardSlot[]
  isPast: boolean
  canEdit: boolean
  onReload: () => void
  onSlotDeleted?: (id: number) => void
  onEdit?: (slotId: number) => void
  proxyChildren?: ProxyChild[]
  hideClaimActions?: boolean
  /**
   * Wird vor der Navigation zur Anleitungsseite aufgerufen, damit der Aufrufer
   * (DutyPage) den Fokus-Marker (`focus=slot-<id>`) auf der aktuellen /dienste-URL
   * hinterlegen kann — Voraussetzung dafür, dass „Zurück" später zu dieser Zeile
   * zurückscrollt (siehe openspec/changes/zurueck-position-wiederherstellen).
   * Optional, weil DutySlotList auch außerhalb von /dienste eingebunden wird
   * (SpieltagDetailModal) — dort bleibt das Verhalten unverändert (kein Fokus-Marker).
   */
  onFocusSlot?: (slotId: number) => void
}

export default function DutySlotList({ slots, isPast, canEdit, onReload, onSlotDeleted, onEdit, proxyChildren = [], hideClaimActions = false, onFocusSlot }: DutySlotListProps) {
  const { user } = useAuth()
  // Windowing der Slot-Zeilen: bei sehr vielen Slots nur sichtbare im DOM.
  // Scroll-Quelle ist die Seite (Duty-Board scrollt als Ganzes).
  const { containerRef: slotContainerRef, start: slotStart, end: slotEnd, padTop: slotPadTop, padBottom: slotPadBottom } =
    useWindowedList({ count: slots.length, estimatedRowHeight: 52, scroll: 'window' })
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)
  const [deleteReason, setDeleteReason] = useState('')
  const [deleteSilent, setDeleteSilent] = useState(false)
  const [claimDialog, setClaimDialog] = useState<{ slotId: number; selectedUserId: number | null } | null>(null)
  const [claimLoading, setClaimLoading] = useState(false)
  const [noInstructionOpen, setNoInstructionOpen] = useState(false)

  useEscapeKey(
    deleteConfirm !== null ? () => closeDeleteConfirm()
      : claimDialog !== null ? () => setClaimDialog(null)
      : noInstructionOpen ? () => setNoInstructionOpen(false)
      : null,
  )

  const claimForUser = async (slotId: number, userId: number) => {
    setClaimLoading(true)
    try {
      await api.post(`/duty-board/${slotId}/claim`, { user_id: userId })
      setClaimDialog(null)
      onReload()
    } catch {
      alert('Dieser Dienst ist bereits vergeben oder du hast ihn bereits.')
    } finally {
      setClaimLoading(false)
    }
  }

  const handleClaimClick = (slotId: number) => {
    if (proxyChildren.length > 0 && user) {
      setClaimDialog({ slotId, selectedUserId: user.id ?? null })
    } else if (user) {
      claimForUser(slotId, user.id)
    }
  }

  const claim = async (id: number) => {
    handleClaimClick(id)
  }

  const unclaim = async (id: number) => {
    try {
      await api.delete(`/duty-board/${id}/claim`)
      onReload()
    } catch {
      alert('Austragen fehlgeschlagen.')
    }
  }

  // withReason=false ist der Direktlöschpfad für unbesetzte Slots: dort gibt es
  // niemanden zu benachrichtigen, also auch keinen Grund zu erfassen.
  const deleteSlot = async (slotId: number, withReason: boolean) => {
    try {
      await api.delete(`/duty-slots/${slotId}`,
        withReason ? { data: deletionPayload(deleteReason, deleteSilent) } : undefined)
      onSlotDeleted?.(slotId)
      onReload()
    } catch {
      alert('Löschen fehlgeschlagen.')
    }
    closeDeleteConfirm()
  }

  const closeDeleteConfirm = () => {
    setDeleteConfirm(null)
    setDeleteReason('')
    setDeleteSilent(false)
  }

  const handleDeleteClick = (slot: BoardSlot) => {
    const slotsFilled = slot.slots_total - slot.vacancies
    if (slotsFilled > 0) {
      setDeleteConfirm(slot.id)
    } else {
      deleteSlot(slot.id, false)
    }
  }

  return (
    <>
      <div ref={slotContainerRef}>
      <table className="w-full text-sm table-fixed">
        <colgroup>
          {/* Mobile enger (nur Uhrzeit-Ziffern), Desktop etwas großzügiger. */}
          <col className="w-16 sm:w-[4.5rem]" />
          <col />
          <col style={{ width: '35%' }} />
          {/* Mobile zeigt nur das Punkte-Menü (schmal) statt der Desktop-Buttons —
              schmalere Spalte hier zieht Spalte 3 (rechtsbündig) direkt ans Menü heran. */}
          <col className="w-11 sm:w-[9.5rem]" />
        </colgroup>
        <WindowedTableBody
          items={slots}
          start={slotStart}
          end={slotEnd}
          padTop={slotPadTop}
          padBottom={slotPadBottom}
          colSpan={4}
          className="divide-y divide-brand-border-subtle"
          renderRow={s => (
              <tr key={s.id} id={`duty-slot-${s.id}`}>
                <td className="pl-4 pr-1 sm:px-4 py-2.5 text-brand-text-muted whitespace-nowrap">{s.event_time || '—'}</td>
                <td className="pl-2 pr-4 sm:px-4 py-2.5 font-medium text-brand-text">
                  <span className="inline-flex items-center gap-1.5">
                    {s.duty_type}
                    {s.has_instruction ? (
                      <Link
                        to={`/dienste/anleitung/${s.duty_type_id}`}
                        aria-label="Anleitung ansehen"
                        onClick={e => { e.stopPropagation(); onFocusSlot?.(s.id) }}
                        className="text-brand-text-muted hover:text-brand-text"
                      >
                        <BookOpen className="w-4 h-4" />
                      </Link>
                    ) : (
                      <button
                        type="button"
                        aria-label="Keine Anleitung vorhanden"
                        onClick={e => { e.stopPropagation(); setNoInstructionOpen(true) }}
                        className="relative inline-flex items-center justify-center text-brand-text-subtle hover:text-brand-text-muted"
                      >
                        <BookOpen className="w-4 h-4 opacity-60" />
                        <span
                          aria-hidden
                          className="pointer-events-none absolute left-0 right-0 top-1/2 -translate-y-1/2 h-px bg-current rotate-45"
                        />
                      </button>
                    )}
                  </span>
                  {s.role_desc ? <span className="text-brand-text-subtle font-normal"> · {s.role_desc}</span> : null}
                </td>
                <td className="px-4 py-2.5 text-brand-text-muted text-right">
                  <div className="flex flex-col items-end gap-1.5">
                    {s.vacancies > 0 && <div><span className="text-xs">{s.vacancies} frei</span></div>}
                    {s.audiences && s.audiences.length > 0 && (
                      <div className="flex flex-wrap justify-end gap-1">
                        {s.audiences.map(a => (
                          <span key={a} className="text-xs bg-brand-info/10 text-brand-text px-1.5 py-0.5 rounded">
                            {AUDIENCE_LABELS[a] ?? a}
                          </span>
                        ))}
                      </div>
                    )}
                    {s.assignees && s.assignees.length > 0 && (
                      <div className="flex flex-wrap justify-end gap-1">
                        {s.assignees.map((a, i) => <PersonChip key={i} userId={a.user_id} name={a.name} />)}
                      </div>
                    )}
                  </div>
                </td>
                <td className="px-4 py-2.5 text-right">
                  {/* Desktop buttons */}
                  <div className="hidden sm:flex items-center justify-end gap-2">
                    {!hideClaimActions && s.claimed_by_me && !isPast && (
                      <button onClick={() => unclaim(s.id)} className="text-xs bg-brand-danger text-white font-medium px-2 py-1 rounded hover:bg-brand-danger/90 transition-colors">
                        Austragen
                      </button>
                    )}
                    {!hideClaimActions && !s.claimed_by_me && s.vacancies > 0 && !isPast && (
                      <button onClick={() => claim(s.id)} className="text-xs bg-brand-yellow text-brand-black font-medium px-2 py-1 rounded hover:bg-brand-black hover:text-brand-yellow transition-colors">
                        Eintragen
                      </button>
                    )}
                    {canEdit && onEdit && (
                      <button onClick={() => onEdit(s.id)} className="text-xs text-brand-text-muted hover:text-brand-text px-2 py-1 rounded hover:bg-brand-border-subtle transition-colors">
                        Bearbeiten
                      </button>
                    )}
                    {canEdit && (
                      <button onClick={() => handleDeleteClick(s)} className="text-brand-text-subtle hover:text-brand-danger transition-colors p-1" aria-label="Slot löschen">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                  {/* Mobile ActionMenu */}
                  <div className="sm:hidden">
                    {(() => {
                      const actions = [
                        ...(!hideClaimActions && !s.claimed_by_me && s.vacancies > 0 && !isPast ? [{ label: 'Eintragen', onClick: () => claim(s.id) }] : []),
                        ...(!hideClaimActions && s.claimed_by_me && !isPast ? [{ label: 'Austragen', onClick: () => unclaim(s.id), variant: 'danger' as const }] : []),
                        ...(canEdit && onEdit ? [{ label: 'Bearbeiten', onClick: () => onEdit(s.id) }] : []),
                        ...(canEdit ? [{ label: 'Löschen', onClick: () => handleDeleteClick(s), variant: 'danger' as const }] : []),
                      ]
                      return actions.length > 0 ? <ActionMenu actions={actions} /> : null
                    })()}
                  </div>
                </td>
              </tr>
          )}
        />
      </table>
      </div>

      {noInstructionOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 max-w-sm w-full mx-4">
            <h2 className="text-lg font-bold mb-2 text-brand-text">Keine Anleitung</h2>
            <p className="text-sm text-brand-text-muted mb-4">
              Für diesen Dienst gibt es noch keine Anleitung.
            </p>
            <div className="flex justify-end">
              <button
                onClick={() => setNoInstructionOpen(false)}
                className="text-sm px-4 py-2 rounded bg-brand-yellow text-brand-black font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
              >
                OK
              </button>
            </div>
          </div>
        </div>
      )}

      {deleteConfirm !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 max-w-sm w-full mx-4">
            <h2 className="text-lg font-bold mb-2 text-brand-text">Slot löschen?</h2>
            <p className="text-sm text-brand-text-muted mb-4">
              Dieser Slot hat bereits Zuteilungen. Alle Zuteilungen werden ebenfalls gelöscht.
            </p>
            <DeleteReasonFields
              reason={deleteReason}
              onReasonChange={setDeleteReason}
              silent={deleteSilent}
              onSilentChange={setDeleteSilent}
              idPrefix="duty-slot-delete"
            />
            <div className="flex justify-end gap-3">
              <button
                onClick={closeDeleteConfirm}
                className="text-sm px-4 py-2 rounded border border-brand-border text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={() => deleteSlot(deleteConfirm, true)}
                className="text-sm px-4 py-2 rounded bg-brand-danger text-white font-medium hover:bg-brand-danger/90 transition-colors"
              >
                Löschen
              </button>
            </div>
          </div>
        </div>
      )}

      {claimDialog !== null && user && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 max-w-sm w-full mx-4">
            <h2 className="text-lg font-bold mb-3 text-brand-text">Dienst übernehmen für…</h2>
            <div className="space-y-2 mb-4">
              <label className="flex items-center gap-3 p-2.5 rounded-lg border border-brand-border-subtle cursor-pointer hover:bg-brand-surface-card transition-colors">
                <input
                  type="radio"
                  name="claim-for"
                  value={user.id}
                  checked={claimDialog.selectedUserId === user.id}
                  onChange={() => setClaimDialog(d => d ? { ...d, selectedUserId: user.id } : d)}
                  className="accent-brand-yellow"
                />
                <span className="text-sm font-medium text-brand-text">Mich selbst</span>
              </label>
              {proxyChildren.map(child => (
                <label key={child.user_id} className="flex items-center gap-3 p-2.5 rounded-lg border border-brand-border-subtle cursor-pointer hover:bg-brand-surface-card transition-colors">
                  <input
                    type="radio"
                    name="claim-for"
                    value={child.user_id}
                    checked={claimDialog.selectedUserId === child.user_id}
                    onChange={() => setClaimDialog(d => d ? { ...d, selectedUserId: child.user_id } : d)}
                    className="accent-brand-yellow"
                  />
                  <span className="text-sm text-brand-text">{child.name}</span>
                </label>
              ))}
            </div>
            <div className="flex gap-2">
              <button
                disabled={claimDialog.selectedUserId === null || claimLoading}
                onClick={() => claimDialog.selectedUserId !== null && claimForUser(claimDialog.slotId, claimDialog.selectedUserId)}
                className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {claimLoading ? 'Eintragen…' : 'Eintragen'}
              </button>
              <button
                onClick={() => setClaimDialog(null)}
                className="flex-1 px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors"
              >
                Abbrechen
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
