import { useEffect, useState } from 'react'
import { Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

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

interface Assignment {
  id: number
  user_name: string
  status: string
  cash_amount: number
}

const WEEKDAYS = ['So', 'Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa']

function formatDate(iso: string): string {
  const d = new Date(iso.slice(0, 10) + 'T12:00:00')
  return `${WEEKDAYS[d.getDay()]} ${String(d.getDate()).padStart(2, '0')}.${String(d.getMonth() + 1).padStart(2, '0')}.`
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

export default function DutyPage() {
  const { user } = useAuth()
  const isAdminOrTrainer = user?.role === 'admin' || user?.role === 'trainer'

  const [groups, setGroups] = useState<BoardGroup[]>([])
  const [showPast, setShowPast] = useState(false)
  const [viewMine, setViewMine] = useState(false)
  const [expanded, setExpanded] = useState<number | null>(null)
  const [assignments, setAssignments] = useState<Record<number, Assignment[]>>({})
  const [cashAmount, setCashAmount] = useState<Record<number, string>>({})
  const [deleteConfirm, setDeleteConfirm] = useState<number | null>(null)

  const load = () => {
    const url = viewMine ? '/duty-board?view=mine' : '/duty-board'
    api.get(url).then(r => setGroups(r.data ?? []))
  }

  useEffect(() => { load() }, [viewMine])

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
      setGroups(prev =>
        prev
          .map(g => ({ ...g, slots: g.slots.filter(s => s.id !== slotId) }))
          .filter(g => g.slots.length > 0)
      )
      if (expanded === slotId) setExpanded(null)
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

  const visible = groups.filter(g => showPast || !g.past)

  return (
    <div>
      <div className="flex items-center justify-between mb-4 flex-wrap gap-2">
        <h1 className="text-2xl font-bold">Dienste</h1>
        <div className="flex items-center gap-3 flex-wrap">
          {isAdminOrTrainer && (
            <div className="flex rounded-lg border border-brand-border-subtle overflow-hidden text-sm">
              <button
                onClick={() => setViewMine(false)}
                className={`px-3 py-1.5 ${!viewMine ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-text-muted hover:bg-brand-border-subtle'}`}
              >
                Alle Dienste
              </button>
              <button
                onClick={() => setViewMine(true)}
                className={`px-3 py-1.5 border-l border-brand-border-subtle ${viewMine ? 'bg-brand-yellow text-brand-black font-medium' : 'text-brand-text-muted hover:bg-brand-border-subtle'}`}
              >
                Meine Dienste
              </button>
            </div>
          )}
          <button
            onClick={() => setShowPast(p => !p)}
            className="text-sm text-brand-text-muted hover:text-brand-blue transition-colors"
          >
            {showPast ? 'Vergangene ausblenden' : 'Vergangene einblenden'}
          </button>
        </div>
      </div>

      {visible.length === 0 && (
        <p className="text-brand-text-muted">
          {groups.length === 0
            ? 'Keine Dienste für deine Mannschaften.'
            : 'Keine aktuellen Dienste. Vergangene können oben eingeblendet werden.'}
        </p>
      )}

      <div className="space-y-4">
        {visible.map((g, i) => (
          <div
            key={i}
            className={`bg-brand-surface-card rounded-xl shadow border-t-4 overflow-hidden ${g.past ? 'border-brand-border opacity-70' : 'border-brand-yellow'}`}
          >
            {/* Group header */}
            <div className="px-4 py-3 bg-brand-surface-card border-b border-brand-border-subtle flex items-center justify-between">
              <div>
                {g.game_id ? (
                  <span className="font-semibold text-sm text-brand-text">
                    {g.date ? formatDate(g.date) : ''}
                    {g.event_time ? ` · ${g.event_time} Uhr` : ''}
                    {g.opponent ? ` · vs. ${g.opponent}` : ''}
                  </span>
                ) : (
                  <span className="font-semibold text-sm text-brand-text">{g.label}</span>
                )}
              </div>
              <span className="text-xs text-brand-text-muted font-medium">{g.team_name}</span>
            </div>

            {/* Slots */}
            <table className="w-full text-sm">
              <tbody className="divide-y divide-brand-border-subtle">
                {g.slots.map(s => (
                  <>
                    <tr key={s.id}>
                      <td className="px-4 py-2.5 font-medium text-brand-text">
                        {s.duty_type}
                        {s.role_desc ? <span className="text-brand-text-subtle font-normal"> · {s.role_desc}</span> : null}
                      </td>
                      <td className="px-4 py-2.5 text-brand-text-muted w-20">{s.event_time || '—'}</td>
                      <td className="px-4 py-2.5 text-brand-text-muted w-24 text-right">
                        {s.claimed_by_me
                          ? <span className="text-brand-blue text-xs font-medium">Eingetragen</span>
                          : s.vacancies > 0
                            ? <span className="text-xs">{s.vacancies} frei</span>
                            : <span className="text-xs text-brand-text-subtle">Besetzt</span>
                        }
                      </td>
                      <td className="px-4 py-2.5 text-right">
                        <div className="flex items-center justify-end gap-2">
                          {s.claimed_by_me && !g.past && (
                            <button
                              onClick={() => unclaim(s.id)}
                              className="text-xs text-brand-text-muted hover:text-brand-danger transition-colors px-2 py-1 rounded border border-brand-border-subtle hover:border-brand-danger"
                            >
                              Austragen
                            </button>
                          )}
                          {!s.claimed_by_me && s.vacancies > 0 && !g.past && (
                            <button
                              onClick={() => claim(s.id)}
                              className="text-xs bg-brand-yellow text-brand-black font-medium px-2 py-1 rounded hover:bg-brand-black hover:text-brand-yellow transition-colors"
                            >
                              Eintragen
                            </button>
                          )}
                          {isAdminOrTrainer && (
                            <button
                              onClick={() => toggleExpand(s.id)}
                              className="text-xs bg-brand-border-subtle text-brand-text-muted px-2 py-1 rounded hover:bg-brand-border transition-colors"
                            >
                              {expanded === s.id ? 'Schließen' : 'Zuteilungen'}
                            </button>
                          )}
                          {isAdminOrTrainer && (
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

                    {/* Expanded assignments */}
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
          </div>
        ))}
      </div>

      {/* Delete confirmation modal */}
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
    </div>
  )
}
