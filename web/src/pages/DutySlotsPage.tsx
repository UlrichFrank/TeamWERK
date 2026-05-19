import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import MobileCard from '../components/MobileCard'

interface Slot {
  id: number; event_name: string; event_date: string
  slots_total: number; slots_filled: number; duty_type: string
}

interface Assignment {
  id: number; user_name: string; status: string; cash_amount: number
}

export default function DutySlotsPage() {
  const [slots, setSlots] = useState<Slot[]>([])
  const [expanded, setExpanded] = useState<number | null>(null)
  const [assignments, setAssignments] = useState<Assignment[]>([])
  const [cashAmount, setCashAmount] = useState<Record<number, string>>({})

  useEffect(() => { api.get('/duty-slots').then(r => setSlots(r.data ?? [])) }, [])

  const loadAssignments = async (slotId: number) => {
    if (expanded === slotId) { setExpanded(null); return }
    const r = await api.get(`/duty-slots/${slotId}/assignments`)
    setAssignments(r.data ?? [])
    setExpanded(slotId)
  }

  const fulfill = async (assignmentId: number) => {
    await api.post(`/duty-assignments/${assignmentId}/fulfill`)
    setAssignments(a => a.map(x => x.id === assignmentId ? { ...x, status: 'fulfilled' } : x))
  }

  const cashSub = async (assignmentId: number) => {
    const amount = parseFloat(cashAmount[assignmentId] || '0')
    if (!amount) return
    await api.post(`/duty-assignments/${assignmentId}/cash-substitute`, { amount })
    setAssignments(a => a.map(x => x.id === assignmentId
      ? { ...x, status: 'cash_substitute', cash_amount: amount } : x))
  }

  const statusBadge = (s: string) => {
    const map: Record<string, string> = {
      pending: 'bg-brand-yellow text-brand-black',
      fulfilled: 'bg-brand-black text-brand-white',
      cash_substitute: 'bg-gray-200 text-gray-700',
    }
    const label: Record<string, string> = {
      pending: 'ausstehend', fulfilled: 'erfüllt', cash_substitute: 'Geldersatz',
    }
    return (
      <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${map[s] ?? 'bg-gray-100 text-gray-600'}`}>
        {label[s] ?? s}
      </span>
    )
  }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Dienst-Planung</h1>

      {/* Mobile: Cards with Accordion */}
      <div className="sm:hidden space-y-0">
        {slots.map(s => (
          <div key={s.id}>
            <MobileCard
              title={s.event_name}
              subtitle={`${s.event_date} · ${s.duty_type}`}
              badge={{ label: `${s.slots_filled}/${s.slots_total}`, variant: s.slots_filled >= s.slots_total ? 'green' : 'yellow' }}
            >
              <button
                onClick={() => loadAssignments(s.id)}
                className="text-xs text-brand-yellow hover:text-brand-black transition-colors font-medium"
              >
                {expanded === s.id ? '▾ Schließen' : '▸ Zuteilungen anzeigen'}
              </button>
            </MobileCard>
            {expanded === s.id && (
              <div className="bg-gray-50 px-4 py-3 text-sm border-b border-brand-black/10">
                {assignments.length === 0 ? (
                  <p className="text-gray-400 text-xs">Keine Zuteilungen</p>
                ) : (
                  <div className="space-y-2">
                    {assignments.map(a => (
                      <div key={a.id} className="py-2 border-t border-gray-200">
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-medium">{a.user_name}</span>
                          {statusBadge(a.status)}
                        </div>
                        {a.status === 'cash_substitute' && a.cash_amount > 0 && (
                          <div className="text-xs text-gray-500 mb-2">{a.cash_amount.toFixed(2)} €</div>
                        )}
                        {a.status === 'pending' && (
                          <div className="flex flex-col gap-2">
                            <button
                              onClick={() => fulfill(a.id)}
                              className="text-xs bg-brand-yellow text-black px-2 py-1.5 rounded font-medium hover:opacity-80"
                            >
                              Erfüllt
                            </button>
                            <div className="flex gap-1">
                              <input
                                type="number"
                                min="0"
                                step="0.01"
                                placeholder="€"
                                value={cashAmount[a.id] ?? ''}
                                onChange={e => setCashAmount(c => ({ ...c, [a.id]: e.target.value }))}
                                className="flex-1 border border-gray-300 rounded px-2 py-1 text-xs"
                              />
                              <button
                                onClick={() => cashSub(a.id)}
                                className="text-xs border border-gray-300 text-gray-600 px-2 py-1 rounded hover:bg-gray-100"
                              >
                                Geldersatz
                              </button>
                            </div>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Desktop: Table */}
      <div className="hidden sm:block bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-gray-500 text-xs uppercase">
            <tr>
              <th className="px-4 py-3 text-left">Event</th>
              <th className="px-4 py-3 text-left">Datum</th>
              <th className="px-4 py-3 text-left">Diensttyp</th>
              <th className="px-4 py-3 text-right">Belegt / Gesamt</th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {slots.map(s => (
              <>
                <tr key={s.id} className="hover:bg-gray-50 border-t border-gray-100">
                  <td className="px-4 py-3 font-medium">{s.event_name}</td>
                  <td className="px-4 py-3 text-gray-500">{s.event_date}</td>
                  <td className="px-4 py-3 text-gray-500">{s.duty_type}</td>
                  <td className="px-4 py-3 text-right">
                    <span className={s.slots_filled >= s.slots_total ? 'text-brand-success' : 'text-orange-500'}>
                      {s.slots_filled}/{s.slots_total}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => loadAssignments(s.id)}
                      className="text-xs text-black hover:text-brand-yellow transition-colors"
                    >
                      {expanded === s.id ? 'schließen' : 'Zuteilungen'}
                    </button>
                  </td>
                </tr>
                {expanded === s.id && (
                  <tr key={`${s.id}-detail`}>
                    <td colSpan={5} className="bg-gray-50 px-6 py-4">
                      {assignments.length === 0 ? (
                        <p className="text-sm text-gray-400">Keine Zuteilungen</p>
                      ) : (
                        <table className="w-full text-sm">
                          <thead>
                            <tr className="text-gray-500 text-xs">
                              <th className="text-left pb-2">Nutzer</th>
                              <th className="text-left pb-2">Status</th>
                              <th className="text-right pb-2">Aktionen</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-gray-100">
                            {assignments.map(a => (
                              <tr key={a.id}>
                                <td className="py-2">{a.user_name}</td>
                                <td className="py-2">
                                  {statusBadge(a.status)}
                                  {a.status === 'cash_substitute' && a.cash_amount > 0 &&
                                    <span className="ml-2 text-xs text-gray-500">{a.cash_amount.toFixed(2)} €</span>}
                                </td>
                                <td className="py-2 text-right">
                                  {a.status === 'pending' && (
                                    <div className="flex items-center justify-end gap-2">
                                      <button
                                        onClick={() => fulfill(a.id)}
                                        className="text-xs bg-brand-yellow text-black px-2 py-1 rounded font-medium hover:bg-black hover:text-brand-yellow transition-colors"
                                      >
                                        Erfüllt
                                      </button>
                                      <input
                                        type="number" min="0" step="0.01" placeholder="Betrag €"
                                        value={cashAmount[a.id] ?? ''}
                                        onChange={e => setCashAmount(c => ({ ...c, [a.id]: e.target.value }))}
                                        className="w-24 border border-gray-300 rounded px-2 py-1 text-xs"
                                      />
                                      <button
                                        onClick={() => cashSub(a.id)}
                                        className="text-xs border border-black text-black px-2 py-1 rounded hover:bg-brand-yellow hover:border-brand-yellow transition-colors"
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
    </div>
  )
}
