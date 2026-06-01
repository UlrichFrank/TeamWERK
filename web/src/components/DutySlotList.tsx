import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useEscapeKey } from '../lib/useEscapeKey'
import PersonChip from './PersonChip'

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
  assignees?: PublicAssignee[]
}

interface Assignment {
  id: number
  user_name: string
  status: string
  cash_amount: number
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    assigned: 'bg-brand-yellow text-brand-black',
    fulfilled: 'bg-brand-black text-brand-white',
    cash_substitute: 'bg-brand-border-subtle text-brand-text-muted',
  }
  const label: Record<string, string> = {
    assigned: 'ausstehend', fulfilled: 'erfüllt', cash_substitute: 'Geldersatz',
  }
  return (
    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${map[status] ?? 'bg-brand-border-subtle text-brand-text-muted'}`}>
      {label[status] ?? status}
    </span>
  )
}


interface DutySlotListProps {
  slots: BoardSlot[]
  isPast: boolean
  canEdit: boolean
  onReload: () => void
  onSlotDeleted?: (id: number) => void
  onEdit?: (slotId: number) => void
}

export default function DutySlotList({ slots, isPast, canEdit, onReload, onSlotDeleted, onEdit }: DutySlotListProps) {
  const [expanded, setExpanded] = useState<number | null>(null)
  const [assignments, setAssignments] = useState<Record<number, Assignment[]>>({})
  const [cashAmount, setCashAmount] = useState<Record<number, string>>({})
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)

  useEscapeKey(deleteConfirm !== null ? () => setDeleteConfirm(null) : null)

  const claim = async (id: number) => {
    try {
      await api.post(`/duty-board/${id}/claim`)
      onReload()
    } catch {
      alert('Dieser Dienst ist bereits vergeben oder du hast ihn bereits.')
    }
  }

  const unclaim = async (id: number) => {
    try {
      await api.delete(`/duty-board/${id}/claim`)
      onReload()
    } catch {
      alert('Austragen fehlgeschlagen.')
    }
  }

  const toggleExpand = async (slotId: number) => {
    if (expanded === slotId) { setExpanded(null); return }
    if (!assignments[slotId]) {
      const r = await api.get(`/duty-slots/${slotId}/assignments`)
      setAssignments(prev => ({ ...prev, [slotId]: r.data ?? [] }))
    }
    setExpanded(slotId)
  }

  const fulfill = async (assignmentId: number, slotId: number) => {
    await api.post(`/duty-assignments/${assignmentId}/fulfill`)
    setAssignments(prev => ({
      ...prev,
      [slotId]: (prev[slotId] ?? []).map(a => a.id === assignmentId ? { ...a, status: 'fulfilled' } : a),
    }))
  }

  const cashSub = async (assignmentId: number, slotId: number) => {
    const amount = parseFloat(cashAmount[assignmentId] || '0')
    if (!amount) return
    await api.post(`/duty-assignments/${assignmentId}/cash-substitute`, { amount })
    setAssignments(prev => ({
      ...prev,
      [slotId]: (prev[slotId] ?? []).map(a =>
        a.id === assignmentId ? { ...a, status: 'cash_substitute', cash_amount: amount } : a
      ),
    }))
  }

  const deleteSlot = async (slotId: number) => {
    try {
      await api.delete(`/duty-slots/${slotId}`)
      if (expanded === slotId) setExpanded(null)
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
                    <div>
                      {s.claimed_by_me
                        ? <span className="text-brand-blue text-xs font-medium">Eingetragen</span>
                        : s.vacancies > 0
                          ? <span className="text-xs">{s.vacancies} frei</span>
                          : <span className="text-xs text-brand-text-subtle">Besetzt</span>
                      }
                    </div>
                    {s.assignees && s.assignees.length > 0 && (
                      <div className="flex flex-wrap justify-end gap-1">
                        {s.assignees.map((a, i) => <PersonChip key={i} userId={a.user_id} name={a.name} photoUrl={a.photo_url} />)}
                      </div>
                    )}
                  </div>
                </td>
                <td className="px-4 py-2.5 text-right">
                  <div className="flex items-center justify-end gap-2">
                    {s.claimed_by_me && !isPast && (
                      <button
                        onClick={() => unclaim(s.id)}
                        className="text-xs text-brand-text-muted hover:text-brand-danger transition-colors px-2 py-1 rounded border border-brand-border-subtle hover:border-brand-danger"
                      >
                        Austragen
                      </button>
                    )}
                    {!s.claimed_by_me && s.vacancies > 0 && !isPast && (
                      <button
                        onClick={() => claim(s.id)}
                        className="text-xs bg-brand-yellow text-brand-black font-medium px-2 py-1 rounded hover:bg-brand-black hover:text-brand-yellow transition-colors"
                      >
                        Eintragen
                      </button>
                    )}
                    {canEdit && onEdit && (
                      <button
                        onClick={() => onEdit(s.id)}
                        className="text-xs text-brand-text-muted hover:text-brand-text px-2 py-1 rounded hover:bg-brand-border-subtle transition-colors"
                      >
                        Bearbeiten
                      </button>
                    )}
                    {canEdit && (
                      <button
                        onClick={() => toggleExpand(s.id)}
                        className="text-xs bg-brand-border-subtle text-brand-text-muted px-2 py-1 rounded hover:bg-brand-border transition-colors"
                      >
                        {expanded === s.id ? 'Schließen' : 'Zuteilungen'}
                      </button>
                    )}
                    {canEdit && (
                      <button
                        onClick={() => handleDeleteClick(s)}
                        className="text-brand-text-subtle hover:text-brand-danger transition-colors p-1"
                        title="Slot löschen"
                        aria-label="Slot löschen"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                </td>
              </tr>

              {expanded === s.id && (
                <tr key={`${s.id}-assignments`}>
                  <td colSpan={4} className="bg-brand-surface-card px-6 py-4">
                    {!(assignments[s.id]?.length) ? (
                      <p className="text-sm text-brand-text-subtle">Keine Zuteilungen</p>
                    ) : (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="text-brand-text-muted text-xs">
                            <th className="text-left pb-2">Nutzer</th>
                            <th className="text-left pb-2">Status</th>
                            <th className="text-right pb-2">Aktionen</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-brand-border-subtle">
                          {(assignments[s.id] ?? []).map(a => (
                            <tr key={a.id}>
                              <td className="py-2 text-brand-text">{a.user_name}</td>
                              <td className="py-2">
                                <StatusBadge status={a.status} />
                                {a.status === 'cash_substitute' && a.cash_amount > 0 && (
                                  <span className="ml-2 text-xs text-brand-text-muted">{a.cash_amount.toFixed(2)} €</span>
                                )}
                              </td>
                              <td className="py-2 text-right">
                                {a.status === 'assigned' && (
                                  <div className="flex items-center justify-end gap-2">
                                    <button
                                      onClick={() => fulfill(a.id, s.id)}
                                      className="text-xs bg-brand-yellow text-brand-black px-2 py-1 rounded font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors"
                                    >
                                      Erfüllt
                                    </button>
                                    <input
                                      type="number" min="0" step="0.01" placeholder="Betrag €"
                                      value={cashAmount[a.id] ?? ''}
                                      onChange={e => setCashAmount(c => ({ ...c, [a.id]: e.target.value }))}
                                      className="w-24 border border-brand-border rounded px-2 py-1 text-xs focus:outline-none focus:ring-1 focus:ring-brand-yellow"
                                    />
                                    <button
                                      onClick={() => cashSub(a.id, s.id)}
                                      className="text-xs border border-brand-black text-brand-text px-2 py-1 rounded hover:bg-brand-yellow hover:border-brand-yellow transition-colors"
                                    >
                                      Geldersatz
                                    </button>
                                  </div>
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    )}
                  </td>
                </tr>
              )}
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
    </>
  )
}
