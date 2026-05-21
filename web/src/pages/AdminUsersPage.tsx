import { useState, useEffect, FormEvent } from 'react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { usePagination } from '../lib/usePagination'
import MobileCard from '../components/MobileCard'
import Pagination from '../components/Pagination'

interface User { id: number; name: string; email: string; role: string }
interface Invitation { id: number; email: string; role: string; comment: string; expires_at: string }
interface MembershipRequest { id: number; name: string; email: string; comment: string; status: string; created_at: string }

const ROLE_LABELS: Record<string, string> = {
  admin: 'Admin', vorstand: 'Vorstand', trainer: 'Trainer', elternteil: 'Elternteil', spieler: 'Spieler',
}
const ROLE_RANK: Record<string, number> = {
  admin: 5, vorstand: 4, trainer: 3, elternteil: 2, spieler: 1,
}
const ALL_ROLES = ['admin', 'vorstand', 'trainer', 'elternteil', 'spieler'] as const

export default function AdminUsersPage() {
  const { user: self } = useAuth()
  const { items: users, setSearch, total, currentPage, totalPages, goToPage } = usePagination<User>('/admin/users')
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [requests, setRequests] = useState<MembershipRequest[]>([])

  const [showInviteModal, setShowInviteModal] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState('elternteil')
  const [inviteComment, setInviteComment] = useState('')
  const [sent, setSent] = useState(false)
  const [inviteError, setInviteError] = useState('')

  const loadInvitationsAndRequests = () => Promise.all([
    api.get('/admin/invitations').then(r => setInvitations(r.data ?? [])),
    api.get('/admin/membership-requests').then(r => setRequests(r.data ?? [])),
  ])

  useEffect(() => { loadInvitationsAndRequests() }, [])

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault()
    setInviteError('')
    try {
      await api.post('/auth/invite', { email: inviteEmail, role: inviteRole, comment: inviteComment })
      setSent(true)
      setInviteEmail('')
      setInviteComment('')
      setInvitations(prev => [...prev, { id: Date.now(), email: inviteEmail, role: inviteRole, comment: inviteComment, expires_at: '' }])
      setTimeout(() => {
        setSent(false)
        setShowInviteModal(false)
        loadInvitationsAndRequests()
      }, 2000)
    } catch {
      setInviteError('Einladung konnte nicht gesendet werden. Bitte E-Mail-Konfiguration prüfen.')
    }
  }

  const closeModal = () => {
    setShowInviteModal(false)
    setInviteEmail('')
    setInviteRole('elternteil')
    setInviteComment('')
    setSent(false)
    setInviteError('')
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
      {/* Header */}
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0">
          <h1 className="text-2xl font-bold">Nutzerverwaltung</h1>
          <div className="flex flex-col sm:flex-row gap-2">
            <input
              type="search"
              placeholder="Nutzer suchen…"
              onChange={e => setSearch(e.target.value)}
              className="border border-gray-300 rounded-md px-3 py-2.5 sm:py-1.5 text-sm w-full sm:w-auto"
            />
            <button
              onClick={() => setShowInviteModal(true)}
              className="text-sm bg-brand-yellow text-brand-black border border-brand-yellow rounded-md px-3 py-2.5 sm:py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors"
            >
              + Einladung
            </button>
          </div>
        </div>
      </div>

      {/* Invite Modal */}
      {showInviteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4 p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-lg">Einladung versenden</h2>
              <button onClick={closeModal} className="text-gray-400 hover:text-gray-600 text-xl leading-none">&times;</button>
            </div>
            {sent && <p className="text-green-600 text-sm mb-3">Einladung gesendet ✓</p>}
            {inviteError && <p className="text-red-600 text-sm mb-3">{inviteError}</p>}
            <form onSubmit={handleInvite} className="space-y-3">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">E-Mail</label>
                <input
                  value={inviteEmail}
                  onChange={e => setInviteEmail(e.target.value)}
                  type="email"
                  placeholder="name@beispiel.de"
                  required
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Rolle</label>
                <select
                  value={inviteRole}
                  onChange={e => setInviteRole(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                >
                  <option value="elternteil">Elternteil</option>
                  <option value="spieler">Spieler</option>
                  <option value="trainer">Trainer</option>
                  <option value="vorstand">Vorstand</option>
                  <option value="admin">Admin</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Kommentar <span className="text-gray-400 font-normal">(optional)</span>
                </label>
                <input
                  value={inviteComment}
                  onChange={e => setInviteComment(e.target.value)}
                  type="text"
                  placeholder="z.B. Elternteil von Max Mustermann"
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm"
                />
              </div>
              <div className="flex gap-2 pt-1">
                <button
                  type="submit"
                  className="flex-1 bg-brand-yellow text-black rounded-md px-4 py-2 text-sm font-medium hover:bg-black hover:text-brand-yellow transition-colors"
                >
                  Einladung senden
                </button>
                <button
                  type="button"
                  onClick={closeModal}
                  className="px-4 py-2 text-sm border border-gray-300 rounded-md hover:bg-gray-50 transition-colors"
                >
                  Abbrechen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Pending requests and invitations */}
      {(requests.length > 0 || invitations.length > 0) && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-3">Ausstehende Anfragen & Einladungen</h2>
          <div className="sm:hidden space-y-0">
            {requests.map(req => (
              <MobileCard
                key={`req-${req.id}`}
                title={req.name}
                subtitle={req.comment ? `${req.email} · ${req.comment}` : req.email}
                badge={{ label: 'Anfrage', variant: 'yellow' }}
                actions={[
                  { label: 'Genehmigen', onClick: () => handleApproveRequest(req) },
                  { label: 'Ablehnen', onClick: () => handleRejectRequest(req) },
                  { label: 'Löschen', onClick: () => handleDeleteRequest(req), variant: 'danger' },
                ]}
              />
            ))}
            {invitations.map(inv => (
              <MobileCard
                key={`inv-${inv.id}`}
                title={inv.email}
                subtitle={inv.comment || ROLE_LABELS[inv.role] || inv.role}
                badge={{ label: 'Einladung', variant: 'red' }}
                actions={[
                  { label: 'Löschen', onClick: () => handleDeleteInvitation(inv), variant: 'danger' },
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
                    <td className="px-6 py-3 text-gray-500 text-xs">{req.comment || '–'}</td>
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
                    <td className="px-6 py-3 text-gray-400">{ROLE_LABELS[inv.role] || inv.role}</td>
                    <td className="px-6 py-3 text-gray-500 text-xs">{inv.comment || '–'}</td>
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
                <td colSpan={4} className="px-6 py-6 text-center text-gray-400">Keine Nutzer vorhanden</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Mobile cards */}
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
                { label: 'Löschen', onClick: () => handleDeleteUser(u), variant: 'danger' },
              ]}
            />
          )
        })}
      </div>

      <Pagination currentPage={currentPage} totalPages={totalPages} onPageChange={goToPage} />
    </div>
  )
}
