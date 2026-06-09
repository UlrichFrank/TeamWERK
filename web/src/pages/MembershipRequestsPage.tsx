import { useEffect, useState } from 'react'
import { api } from '../lib/api'
import { useLiveUpdates } from '../hooks/useLiveUpdates'

interface Request { id: number; first_name: string; last_name: string; email: string; comment: string; created_at: string }

export default function MembershipRequestsPage() {
  const [requests, setRequests] = useState<Request[]>([])

  const load = () => api.get('/membership-requests').then(r => setRequests(r.data ?? []))
  useEffect(() => { load() }, [])
  useLiveUpdates(event => { if (event === 'members') load() })

  const approve = async (id: number) => { await api.post(`/membership-requests/${id}/approve`); load() }
  const reject = async (id: number) => { await api.post(`/membership-requests/${id}/reject`); load() }

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Beitrittsanfragen</h1>
      {requests.length === 0 && <p className="text-brand-text-muted">Keine offenen Anfragen.</p>}
      <div className="space-y-3">
        {requests.map(r => (
          <div key={r.id} className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-4 flex items-center justify-between">
            <div>
              <div className="font-medium text-brand-text">{r.first_name} {r.last_name}</div>
              <div className="text-sm text-brand-text-muted">{r.email}</div>
              {r.comment && <div className="text-xs text-brand-text-subtle mt-0.5">{r.comment}</div>}
              <div className="text-xs text-brand-text-subtle mt-0.5">{new Date(r.created_at).toLocaleDateString('de-DE')}</div>
            </div>
            <div className="flex gap-2">
              <button onClick={() => approve(r.id)} className="bg-brand-yellow text-brand-black rounded-md px-3 py-1.5 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
                Genehmigen
              </button>
              <button onClick={() => reject(r.id)} className="bg-brand-danger text-white rounded-md px-3 py-1.5 text-sm font-medium hover:bg-brand-danger/90 transition-colors">
                Ablehnen
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
