import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useEscapeKey } from '../lib/useEscapeKey'
import PersonChip from './PersonChip'
import ActionMenu from './ActionMenu'
import { AUDIENCE_LABELS } from '../lib/constants'
import type { ProxyChild } from '../pages/DutyPage'

export interface PublicAssignee {
  user_id: number
  name: string
  photo_url?: string
}

export interface BoardSlot {
  id: number
  duty_type: string
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
}

export default function DutySlotList({ slots, isPast, canEdit, onReload, onSlotDeleted, onEdit, proxyChildren = [] }: DutySlotListProps) {
  const { user } = useAuth()
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)
  const [claimDialog, setClaimDialog] = useState<{ slotId: number; selectedUserId: number | null } | null>(null)
  const [claimLoading, setClaimLoading] = useState(false)

  useEscapeKey(deleteConfirm !== null ? () => setDeleteConfirm(null) : claimDialog !== null ? () => setClaimDialog(null) : null)

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

  const deleteSlot = async (slotId: number) => {
    try {
      await api.delete(`/duty-slots/${slotId}`)
onSlotDeleted?.(slotId)
      onReload()
    } catch {
      alert('Löschen fehlgeschlagen.')
    }
    setDeleteConfirm(null)
  }

  const handleDeleteClick = (slot: BoardSlot) => {
    const slotsFilled = slot.slots_total - slot.vacancies
    if (slotsFilled > 0) {
      setDeleteConfirm(slot.id)
    } else {
      deleteSlot(slot.id)
    }
  }

  return (
    <>
      <table className="w-full text-sm">
        <tbody className="divide-y divide-brand-border-subtle">
          {slots.map(s => (
            <>
              <tr key={s.id}>
                <td className="px-4 py-2.5 font-medium text-brand-text">
                  {s.duty_type}
                  {s.role_desc ? <span className="text-brand-text-subtle font-normal"> · {s.role_desc}</span> : null}
                </td>
                <td className="px-4 py-2.5 text-brand-text-muted w-20">{s.event_time || '—'}</td>
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
                        {s.assignees.map((a, i) => <PersonChip key={i} userId={a.user_id} name={a.name} photoUrl={a.photo_url} />)}
                      </div>
                    )}
                  </div>
                </td>
                <td className="px-4 py-2.5 text-right">
                  {/* Desktop buttons */}
                  <div className="hidden sm:flex items-center justify-end gap-2">
                    {s.claimed_by_me && !isPast && (
                      <button onClick={() => unclaim(s.id)} className="text-xs bg-brand-danger text-white font-medium px-2 py-1 rounded hover:bg-brand-danger/90 transition-colors">
                        Austragen
                      </button>
                    )}
                    {!s.claimed_by_me && s.vacancies > 0 && !isPast && (
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
                        ...(!s.claimed_by_me && s.vacancies > 0 && !isPast ? [{ label: 'Eintragen', onClick: () => claim(s.id) }] : []),
                        ...(s.claimed_by_me && !isPast ? [{ label: 'Austragen', onClick: () => unclaim(s.id), variant: 'danger' as const }] : []),
                        ...(canEdit && onEdit ? [{ label: 'Bearbeiten', onClick: () => onEdit(s.id) }] : []),
                        ...(canEdit ? [{ label: 'Löschen', onClick: () => handleDeleteClick(s), variant: 'danger' as const }] : []),
                      ]
                      return actions.length > 0 ? <ActionMenu actions={actions} /> : null
                    })()}
                  </div>
                </td>
              </tr>

            </>
          ))}
        </tbody>
      </table>

      {deleteConfirm !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow p-6 max-w-sm w-full mx-4">
            <h2 className="text-lg font-bold mb-2 text-brand-text">Slot löschen?</h2>
            <p className="text-sm text-brand-text-muted mb-4">
              Dieser Slot hat bereits Zuteilungen. Alle Zuteilungen werden ebenfalls gelöscht.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="text-sm px-4 py-2 rounded border border-brand-border text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors"
              >
                Abbrechen
              </button>
              <button
                onClick={() => deleteSlot(deleteConfirm)}
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
