import { useState, useEffect, FormEvent } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'

interface Team { id: number; name: string }
interface User { id: number; name: string; email: string; role: string; team_name: string }
interface Invitation { id: number; email: string; role: string; team_name: string; expires_at: string }
interface MembershipRequest { id: number; name: string; email: string; team_id?: number; status: string; created_at: string }

type Row =
  | { kind: 'user'; data: User }
  | { kind: 'invitation'; data: Invitation }
  | { kind: 'request'; data: MembershipRequest }

const ROLE_LABELS: Record<string, string> = {
  admin: 'Admin', vorstand: 'Vorstand', trainer: 'Trainer', elternteil: 'Elternteil', spieler: 'Spieler',
}
const ROLE_RANK: Record<string, number> = {
  admin: 5, vorstand: 4, trainer: 3, elternteil: 2, spieler: 1,
}
const ALL_ROLES = ['admin', 'vorstand', 'trainer', 'elternteil', 'spieler'] as const

function buildRows(users: User[], invitations: Invitation[], requests: MembershipRequest[]): Row[] {
  const pending: Row[] = [
    ...requests.map(r => ({ kind: 'request' as const, data: r })),
    ...invitations.map(i => ({ kind: 'invitation' as const, data: i })),
  ]
  const registered: Row[] = [...users]
    .sort((a, b) => a.name.localeCompare(b.name))
    .map(u => ({ kind: 'user' as const, data: u }))
  return [...pending, ...registered]
}

export default function AdminUsersPage() {
  const { user: self } = useAuth()
  const [teams, setTeams] = useState<Team[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [requests, setRequests] = useState<MembershipRequest[]>([])
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteTeamID, setInviteTeamID] = useState('')
  const [inviteRole, setInviteRole] = useState('elternteil')
  const [sent, setSent] = useState(false)
  const [inviteError, setInviteError] = useState('')

  const reload = () => Promise.all([
    api.get('/admin/users').then(r => setUsers(r.data ?? [])),
    api.get('/admin/invitations').then(r => setInvitations(r.data ?? [])),
    api.get('/admin/membership-requests').then(r => setRequests(r.data ?? [])),
  ])

  useEffect(() => {
    api.get('/admin/teams').then(r => setTeams(r.data ?? []))
    reload()
  }, [])

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault()
    setInviteError('')
    try {
      await api.post('/auth/invite', { email: inviteEmail, team_id: Number(inviteTeamID) || null, role: inviteRole })
      setSent(true)
      setInviteEmail('')
      setInvitations(prev => [...prev, { id: Date.now(), email: inviteEmail, role: inviteRole, team_name: '', expires_at: '' }])
      setTimeout(() => { setSent(false); reload() }, 3000)
    } catch {
      setInviteError('Einladung konnte nicht gesendet werden. Bitte E-Mail-Konfiguration prüfen.')
    }
  }

  const handleDeleteUser = async (u: User) => {
    if (!window.confirm(`Nutzer „${u.name}" (${u.email}) wirklich löschen?`)) return
    await api.delete(`/admin/users/${u.id}`)
    setUsers(prev => prev.filter(x => x.id !== u.id))
  }

  const handleDeleteInvitation = async (inv: Invitation) => {
    if (!window.confirm(`Einladung für ${inv.email} widerrufen?`)) return
    await api.delete(`/admin/invitations/${inv.id}`)
    setInvitations(prev => prev.filter(x => x.id !== inv.id))
  }

  const handleApproveRequest = async (req: MembershipRequest) => {
    await api.post(`/admin/membership-requests/${req.id}/approve`)
    setRequests(prev => prev.filter(x => x.id !== req.id))
    reload()
  }

  const handleRejectRequest = async (req: MembershipRequest) => {
    await api.post(`/admin/membership-requests/${req.id}/reject`)
    setRequests(prev => prev.filter(x => x.id !== req.id))
  }

  const handleDeleteRequest = async (req: MembershipRequest) => {
    if (!window.confirm(`Beitrittsanfrage von ${req.name} löschen?`)) return
    await api.delete(`/admin/membership-requests/${req.id}`)
    setRequests(prev => prev.filter(x => x.id !== req.id))
  }

  const handleRoleChange = async (u: User, newRole: string) => {
    await api.put(`/admin/users/${u.id}/role`, { role: newRole })
    setUsers(prev => prev.map(x => x.id === u.id ? { ...x, role: newRole } : x))
  }

  const allowedRoles = (callerRole: string) =>
    ALL_ROLES.filter(r => ROLE_RANK[r] <= (ROLE_RANK[callerRole] ?? 0))

  const rows = buildRows(users, invitations, requests)
  const total = users.length + invitations.length + requests.length

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Nutzerverwaltung</h1>

      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 max-w-md mb-8">
        <h2 className="font-semibold mb-4">Einladung versenden</h2>
        {sent && <p className="text-brand-success text-sm mb-3">Einladung gesendet ✓</p>}
        {inviteError && <p className="text-brand-error text-sm mb-3">{inviteError}</p>}
        <form onSubmit={handleInvite} className="space-y-3">
          <input value={inviteEmail} onChange={e => setInviteEmail(e.target.value)} type="email" placeholder="E-Mail" required
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
          <select value={inviteRole} onChange={e => setInviteRole(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
            <option value="elternteil">Elternteil</option>
            <option value="spieler">Spieler</option>
            <option value="trainer">Trainer</option>
            <option value="vorstand">Vorstand</option>
            <option value="admin">Admin</option>
          </select>
          <select value={inviteTeamID} onChange={e => setInviteTeamID(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
            <option value="">– kein Team –</option>
            {teams.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
          </select>
          <button type="submit" className="bg-brand-yellow text-black rounded-md px-4 py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
            Einladung senden
          </button>
        </form>
      </div>

      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="font-semibold">Alle Einträge ({total})</h2>
        </div>
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-gray-500 uppercase text-xs">
            <tr>
              <th className="px-6 py-3 text-left">Name / E-Mail</th>
              <th className="px-6 py-3 text-left">E-Mail</th>
              <th className="px-6 py-3 text-left">Status / Rolle</th>
              <th className="px-6 py-3 text-left">Team</th>
              <th className="px-6 py-3 text-left"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {rows.map((row) => {
              if (row.kind === 'request') {
                const req = row.data
                return (
                  <tr key={`req-${req.id}`} className="hover:bg-gray-50">
                    <td className="px-6 py-3 font-medium">{req.name}</td>
                    <td className="px-6 py-3 text-gray-600">{req.email}</td>
                    <td className="px-6 py-3">
                      <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-brand-yellow text-black">Anfrage</span>
                    </td>
                    <td className="px-6 py-3 text-gray-600">–</td>
                    <td className="px-6 py-3 text-right space-x-3">
                      <button onClick={() => handleApproveRequest(req)} className="text-xs text-brand-success hover:text-brand-success">Genehmigen</button>
                      <button onClick={() => handleRejectRequest(req)} className="text-xs text-gray-600 hover:text-gray-900">Ablehnen</button>
                      <button onClick={() => handleDeleteRequest(req)} className="text-xs text-brand-error hover:text-brand-error">Löschen</button>
                    </td>
                  </tr>
                )
              }
              if (row.kind === 'invitation') {
                const inv = row.data
                return (
                  <tr key={`inv-${inv.id}`} className="hover:bg-gray-50">
                    <td className="px-6 py-3 text-gray-500 italic">{inv.email}</td>
                    <td className="px-6 py-3 text-gray-400">–</td>
                    <td className="px-6 py-3">
                      <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-gray-200 text-gray-700">Einladung</span>
                    </td>
                    <td className="px-6 py-3 text-gray-600">{inv.team_name || '–'}</td>
                    <td className="px-6 py-3 text-right">
                      <button onClick={() => handleDeleteInvitation(inv)} className="text-xs text-brand-error hover:text-brand-error">Löschen</button>
                    </td>
                  </tr>
                )
              }
              // kind === 'user'
              const u = row.data
              const callerRank = ROLE_RANK[self?.role ?? ''] ?? 0
              const canEdit = self?.id !== u.id && (ROLE_RANK[u.role] ?? 0) <= callerRank
              return (
                <tr key={`user-${u.id}`} className="hover:bg-gray-50">
                  <td className="px-6 py-3 font-medium">{u.name}</td>
                  <td className="px-6 py-3 text-gray-600">{u.email}</td>
                  <td className="px-6 py-3">
                    {canEdit ? (
                      <select
                        value={u.role}
                        onChange={e => handleRoleChange(u, e.target.value)}
                        className="border border-gray-300 rounded px-2 py-0.5 text-xs"
                      >
                        {allowedRoles(self?.role ?? '').map(r => (
                          <option key={r} value={r}>{ROLE_LABELS[r]}</option>
                        ))}
                      </select>
                    ) : (
                      <span className="text-xs text-gray-500">{ROLE_LABELS[u.role] ?? u.role}</span>
                    )}
                  </td>
                  <td className="px-6 py-3 text-gray-600">{u.team_name}</td>
                  <td className="px-6 py-3 text-right">
                    <button
                      onClick={() => handleDeleteUser(u)}
                      disabled={self?.id === u.id}
                      className="text-xs text-brand-error hover:text-brand-error disabled:opacity-30 disabled:cursor-not-allowed"
                    >
                      Löschen
                    </button>
                  </td>
                </tr>
              )
            })}
            {rows.length === 0 && (
              <tr>
                <td colSpan={5} className="px-6 py-6 text-center text-gray-400">Keine Einträge vorhanden</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
