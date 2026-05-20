import { useState, useEffect, FormEvent } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { usePagination } from '../lib/usePagination'
import MobileCard from '../components/MobileCard'
import Pagination from '../components/Pagination'

interface Team { id: number; name: string }
interface User { id: number; name: string; email: string; role: string; team_name: string }
interface Invitation { id: number; email: string; role: string; team_name: string; expires_at: string }
interface MembershipRequest { id: number; name: string; email: string; team_id?: number; status: string; created_at: string }

const ROLE_LABELS: Record<string, string> = {
  admin: 'Admin', vorstand: 'Vorstand', trainer: 'Trainer', elternteil: 'Elternteil', spieler: 'Spieler',
}
const ROLE_RANK: Record<string, number> = {
  admin: 5, vorstand: 4, trainer: 3, elternteil: 2, spieler: 1,
}
const ALL_ROLES = ['admin', 'vorstand', 'trainer', 'elternteil', 'spieler'] as const

export default function AdminUsersPage() {
  const { user: self } = useAuth()
  const [teams, setTeams] = useState<Team[]>([])
  const { items: users, setSearch, total, currentPage, totalPages, goToPage } = usePagination<User>('/admin/users')
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [requests, setRequests] = useState<MembershipRequest[]>([])
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteTeamID, setInviteTeamID] = useState('')
  const [inviteRole, setInviteRole] = useState('elternteil')
  const [sent, setSent] = useState(false)
  const [inviteError, setInviteError] = useState('')

  const loadInvitationsAndRequests = () => Promise.all([
    api.get('/admin/invitations').then(r => setInvitations(r.data ?? [])),
    api.get('/admin/membership-requests').then(r => setRequests(r.data ?? [])),
  ])

  useEffect(() => {
    api.get('/admin/teams').then(r => setTeams(r.data ?? []))
    loadInvitationsAndRequests()
  }, [])

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault()
    setInviteError('')
    try {
      await api.post('/auth/invite', { email: inviteEmail, team_id: Number(inviteTeamID) || null, role: inviteRole })
      setSent(true)
      setInviteEmail('')
      setInvitations(prev => [...prev, { id: Date.now(), email: inviteEmail, role: inviteRole, team_name: '', expires_at: '' }])
      setTimeout(() => { setSent(false); loadInvitationsAndRequests() }, 3000)
    } catch {
      setInviteError('Einladung konnte nicht gesendet werden. Bitte E-Mail-Konfiguration prüfen.')
    }
  }

  const handleDeleteUser = async (u: User) => {
    if (!window.confirm(`Nutzer „${u.name}" (${u.email}) wirklich löschen?`)) return
    await api.delete(`/admin/users/${u.id}`)
  }

  const handleDeleteInvitation = async (inv: Invitation) => {
    if (!window.confirm(`Einladung für ${inv.email} widerrufen?`)) return
    await api.delete(`/admin/invitations/${inv.id}`)
    setInvitations(prev => prev.filter(x => x.id !== inv.id))
  }

  const handleApproveRequest = async (req: MembershipRequest) => {
    await api.post(`/admin/membership-requests/${req.id}/approve`)
    setRequests(prev => prev.filter(x => x.id !== req.id))
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
  }

  const allowedRoles = (callerRole: string) =>
    ALL_ROLES.filter(r => ROLE_RANK[r] <= (ROLE_RANK[callerRole] ?? 0))

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Nutzerverwaltung</h1>

      <div className="bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow p-6 max-w-md mb-8 px-4 sm:px-6">
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
          <button type="submit" className="w-full sm:w-auto bg-brand-yellow text-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors">
            Einladung senden
          </button>
        </form>
      </div>

      {/* Pending requests and invitations */}
      {(requests.length > 0 || invitations.length > 0) && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-3">Ausstehende Anfragen & Einladungen</h2>
          <div className="sm:hidden space-y-0">
            {requests.map(req => (
              <MobileCard
                key={`req-${req.id}`}
                title={req.name}
                subtitle={req.email}
                badge={{ label: 'Anfrage', variant: 'yellow' }}
                actions={[
                  {
                    label: 'Genehmigen',
                    onClick: () => handleApproveRequest(req),
                  },
                  {
                    label: 'Ablehnen',
                    onClick: () => handleRejectRequest(req),
                  },
                  {
                    label: 'Löschen',
                    onClick: () => handleDeleteRequest(req),
                    variant: 'danger',
                  },
                ]}
              />
            ))}
            {invitations.map(inv => (
              <MobileCard
                key={`inv-${inv.id}`}
                title={inv.email}
                subtitle={inv.team_name || '–'}
                badge={{ label: 'Einladung', variant: 'red' }}
                actions={[
                  {
                    label: 'Löschen',
                    onClick: () => handleDeleteInvitation(inv),
                    variant: 'danger',
                  },
                ]}
              />
            ))}
          </div>

          <div className="hidden sm:block bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
            <table className="w-full text-sm">
              <tbody className="divide-y divide-gray-100">
                {requests.map(req => (
                  <tr key={`req-${req.id}`} className="hover:bg-brand-gray">
                    <td className="px-6 py-3 font-medium">{req.name}</td>
                    <td className="px-6 py-3 text-gray-600">{req.email}</td>
                    <td className="px-6 py-3"><span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-brand-yellow">Anfrage</span></td>
                    <td className="px-6 py-3 text-right">
                      <div className="flex gap-1 justify-end">
                        <button onClick={() => handleApproveRequest(req)} className="text-xs bg-brand-yellow text-brand-black px-3 py-1 rounded font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">Genehmigen</button>
                        <button onClick={() => handleRejectRequest(req)} className="text-xs border border-gray-300 text-gray-700 px-3 py-1 rounded font-medium hover:border-gray-500 transition-colors">Ablehnen</button>
                        <button onClick={() => handleDeleteRequest(req)} className="text-xs border border-red-300 text-red-600 px-3 py-1 rounded font-medium hover:bg-red-50 hover:border-red-400 transition-colors">Löschen</button>
                      </div>
                    </td>
                  </tr>
                ))}
                {invitations.map(inv => (
                  <tr key={`inv-${inv.id}`} className="hover:bg-brand-gray">
                    <td className="px-6 py-3 text-gray-500 italic">{inv.email}</td>
                    <td className="px-6 py-3 text-gray-400">–</td>
                    <td className="px-6 py-3"><span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-gray-200">Einladung</span></td>
                    <td className="px-6 py-3 text-right">
                      <button onClick={() => handleDeleteInvitation(inv)} className="text-xs border border-red-300 text-red-600 px-3 py-1 rounded font-medium hover:bg-red-50 hover:border-red-400 transition-colors">Löschen</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Registered users */}
      <div>
        <div className="mb-3 sm:mb-0">
          <input
            type="search"
            placeholder="Nutzer suchen…"
            onChange={e => setSearch(e.target.value)}
            className="w-full sm:w-64 border border-gray-300 rounded-md px-3 py-2 text-sm"
          />
        </div>

        {/* Mobile: Cards */}
        <div className="sm:hidden space-y-0 mt-4">
          {users.map(u => {
            const callerRank = ROLE_RANK[self?.role ?? ''] ?? 0
            const canEdit = self?.id !== u.id && (ROLE_RANK[u.role] ?? 0) <= callerRank
            return (
              <MobileCard
                key={`user-${u.id}`}
                title={u.name}
                subtitle={u.email}
                badge={{ label: ROLE_LABELS[u.role], variant: 'blue' }}
                actions={[
                  ...(canEdit ? [{
                    label: 'Rolle ändern',
                    onClick: () => {
                      const newRole = prompt(`Neue Rolle für ${u.name}:`, u.role)
                      if (newRole && allowedRoles(self?.role ?? '').includes(newRole as typeof ALL_ROLES[number])) {
                        handleRoleChange(u, newRole)
                      }
                    },
                  }] : []),
                  {
                    label: 'Löschen',
                    onClick: () => handleDeleteUser(u),
                    variant: 'danger',
                  },
                ]}
              />
            )
          })}
        </div>

        {/* Desktop: Table */}
        <div className="hidden sm:block bg-gray-50 rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden mt-6">
          <div className="px-6 py-4 border-b border-gray-100">
            <h2 className="font-semibold">Registrierte Nutzer ({total})</h2>
          </div>
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-500 uppercase text-xs">
              <tr>
                <th className="px-6 py-3 text-left">Name</th>
                <th className="px-6 py-3 text-left">E-Mail</th>
                <th className="px-6 py-3 text-left">Rolle</th>
                <th className="px-6 py-3 text-left">Team</th>
                <th className="px-6 py-3 text-left"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {users.map(u => {
                const callerRank = ROLE_RANK[self?.role ?? ''] ?? 0
                const canEdit = self?.id !== u.id && (ROLE_RANK[u.role] ?? 0) <= callerRank
                return (
                  <tr key={`user-${u.id}`} className="hover:bg-brand-gray">
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
                        <span className="text-xs text-gray-500">{ROLE_LABELS[u.role]}</span>
                      )}
                    </td>
                    <td className="px-6 py-3 text-gray-600">{u.team_name}</td>
                    <td className="px-6 py-3 text-right">
                      <button
                        onClick={() => handleDeleteUser(u)}
                        disabled={self?.id === u.id}
                        className="text-xs border border-red-300 text-red-600 px-3 py-1 rounded font-medium hover:bg-red-50 hover:border-red-400 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
                      >
                        Löschen
                      </button>
                    </td>
                  </tr>
                )
              })}
              {users.length === 0 && (
                <tr>
                  <td colSpan={5} className="px-6 py-6 text-center text-gray-400">Keine Nutzer vorhanden</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>

        <Pagination currentPage={currentPage} totalPages={totalPages} onPageChange={goToPage} />
      </div>
    </div>
  )
}
