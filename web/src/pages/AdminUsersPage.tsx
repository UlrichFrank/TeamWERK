import { useState, useEffect, FormEvent } from 'react'
import { X, Check } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { usePagination } from '../lib/usePagination'
import ActionMenu from '../components/ActionMenu'
import Pagination from '../components/Pagination'
import { useEscapeKey } from '../lib/useEscapeKey'

interface User { id: number; first_name: string; last_name: string; email: string; role: string; member_id?: number | null }
interface Invitation { id: number; email: string; role: string; comment: string; expires_at: string }
interface MembershipRequest { id: number; name: string; email: string; comment: string; status: string; created_at: string }

const ROLE_LABELS: Record<string, string> = {
  admin: 'Admin', standard: 'Standard',
}
const ALL_ROLES = ['admin', 'standard'] as const

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

export default function AdminUsersPage() {
  const { user: self, startImpersonation } = useAuth()
  const { items: users, setSearch, total, currentPage, totalPages, goToPage } = usePagination<User>('/admin/users')
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [requests, setRequests] = useState<MembershipRequest[]>([])

  const [createdMemberUserIds, setCreatedMemberUserIds] = useState<Set<number>>(new Set())
  const [createMemberLoading, setCreateMemberLoading] = useState<Set<number>>(new Set())
  const [createMemberErrors, setCreateMemberErrors] = useState<Map<number, string>>(new Map())

  const [showInviteModal, setShowInviteModal] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState('standard')
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
    setInviteRole('standard')
    setInviteComment('')
    setSent(false)
    setInviteError('')
  }

  useEscapeKey(showInviteModal ? closeModal : null)

  const handleDeleteUser = async (u: User) => {
    if (!window.confirm(`Nutzer „${u.first_name} ${u.last_name}" (${u.email}) wirklich löschen?`)) return
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

  const handleCreateMember = async (u: User) => {
    setCreateMemberLoading(prev => new Set(prev).add(u.id))
    setCreateMemberErrors(prev => { const m = new Map(prev); m.delete(u.id); return m })
    try {
      await api.post(`/admin/users/${u.id}/create-member`)
      setCreatedMemberUserIds(prev => new Set(prev).add(u.id))
    } catch {
      setCreateMemberErrors(prev => new Map(prev).set(u.id, 'Fehler beim Anlegen'))
    } finally {
      setCreateMemberLoading(prev => { const s = new Set(prev); s.delete(u.id); return s })
    }
  }

  const handleRoleChange = async (u: User, newRole: string) => {
    await api.put(`/admin/users/${u.id}/role`, { role: newRole })
  }

  const allowedRoles = (callerRole: string) =>
    callerRole === 'admin' ? ALL_ROLES : ALL_ROLES.filter(r => r !== 'admin')

  return (
    <div>
      {/* Header */}
      <div className="sticky top-0 z-10 bg-brand-white pb-4 mb-4 sm:bg-transparent sm:pb-6 sm:mb-0 sm:static sm:z-auto">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-0">
          <h1 className="text-2xl font-bold">Nutzerverwaltung</h1>
          <div className="flex flex-wrap gap-2">
            <input
              type="search"
              placeholder="Suchen…"
              onChange={e => setSearch(e.target.value)}
              className="border border-brand-border rounded-md px-3 py-2.5 sm:py-1.5 text-xs text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-32 sm:w-auto"
            />
            <button
              onClick={() => setShowInviteModal(true)}
              className="text-xs bg-brand-yellow text-brand-black border border-brand-yellow rounded-md px-3 py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors whitespace-nowrap"
            >
              + Einladung
            </button>
          </div>
        </div>
      </div>

      {/* Invite Modal */}
      {showInviteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-lg">Einladung versenden</h2>
              <button onClick={closeModal} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="px-6 py-5 space-y-3">
              {sent && (
                <p className="text-brand-success text-sm flex items-center gap-1">
                  <Check className="w-4 h-4" /> Einladung gesendet
                </p>
              )}
              {inviteError && (
                <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">
                  {inviteError}
                </p>
              )}
              <form onSubmit={handleInvite} className="space-y-3">
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">E-Mail</label>
                  <input value={inviteEmail} onChange={e => setInviteEmail(e.target.value)} type="email" placeholder="name@beispiel.de" required className={INPUT} />
                </div>
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">Rolle</label>
                  <select value={inviteRole} onChange={e => setInviteRole(e.target.value)} className={INPUT}>
                    <option value="standard">Standard</option>
                    <option value="admin">Admin</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-brand-text-muted mb-1">
                    Kommentar <span className="text-brand-text-subtle font-normal">(optional)</span>
                  </label>
                  <input value={inviteComment} onChange={e => setInviteComment(e.target.value)} type="text" placeholder="z.B. Elternteil von Max Mustermann" className={INPUT} />
                </div>
                <div className="flex gap-2 pt-1">
                  <button type="submit" className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
                    Einladung senden
                  </button>
                  <button type="button" onClick={closeModal} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                    Abbrechen
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* Pending requests and invitations */}
      {(requests.length > 0 || invitations.length > 0) && (
        <div className="mb-8">
          <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-x-auto">
            <div className="px-6 py-4 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-brand-text">Ausstehende Anfragen & Einladungen ({requests.length + invitations.length})</h2>
            </div>
            <table className="w-full text-sm">
              <tbody className="divide-y divide-brand-border-subtle">
                {requests.map(req => (
                  <tr key={`req-${req.id}`} className="hover:bg-brand-table-select transition-colors">
                    <td className="px-4 py-3 font-medium text-brand-text">{req.name}</td>
                    <td className="hidden md:table-cell px-4 py-3 text-brand-text-muted">{req.email}</td>
                    <td className="hidden lg:table-cell px-4 py-3 text-brand-text-subtle text-xs">{req.comment || '–'}</td>
                    <td className="px-4 py-3">
                      <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-brand-yellow text-brand-black">Anfrage</span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <ActionMenu actions={[
                        { label: 'Genehmigen', onClick: () => handleApproveRequest(req) },
                        { label: 'Ablehnen', onClick: () => handleRejectRequest(req) },
                        { label: 'Löschen', onClick: () => handleDeleteRequest(req), variant: 'danger' },
                      ]} />
                    </td>
                  </tr>
                ))}
                {invitations.map(inv => (
                  <tr key={`inv-${inv.id}`} className="hover:bg-brand-table-select transition-colors">
                    <td className="px-4 py-3 text-brand-text-muted italic">{inv.email}</td>
                    <td className="hidden md:table-cell px-4 py-3 text-brand-text-subtle">{ROLE_LABELS[inv.role] || inv.role}</td>
                    <td className="hidden lg:table-cell px-4 py-3 text-brand-text-subtle text-xs">{inv.comment || '–'}</td>
                    <td className="px-4 py-3">
                      <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-brand-border-subtle text-brand-text-muted">Einladung</span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <ActionMenu actions={[
                        { label: 'Löschen', onClick: () => handleDeleteInvitation(inv), variant: 'danger' },
                      ]} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Registered users */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-x-auto mt-6">
        <div className="px-6 py-4 border-b border-brand-border-subtle">
          <h2 className="font-semibold text-brand-text">Registrierte Nutzer ({total})</h2>
        </div>
        <table className="w-full text-sm">
          <tbody className="divide-y divide-brand-border-subtle">
            {users.map(u => {
              const canEdit = self?.id !== u.id && self?.role === 'admin'
              return (
                <tr key={`user-${u.id}`} className="hover:bg-brand-table-select transition-colors">
                  <td className="px-4 py-3 font-medium text-brand-text">{u.first_name} {u.last_name}</td>
                  <td className="hidden md:table-cell px-4 py-3 text-brand-text-muted">{u.email}</td>
                  <td className="px-4 py-3">
                    {canEdit ? (
                      <select
                        value={u.role}
                        onChange={e => handleRoleChange(u, e.target.value)}
                        className="border border-brand-border rounded-md px-2 py-1 pr-6 text-xs text-brand-text bg-white focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                      >
                        {allowedRoles(self?.role ?? '').map(r => (
                          <option key={r} value={r}>{ROLE_LABELS[r]}</option>
                        ))}
                      </select>
                    ) : (
                      <span className="text-xs text-brand-text-muted">{ROLE_LABELS[u.role]}</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {createMemberErrors.get(u.id) && (
                      <span className="text-xs text-brand-danger mr-2">{createMemberErrors.get(u.id)}</span>
                    )}
                    <ActionMenu actions={[
                      ...(!u.member_id && !createdMemberUserIds.has(u.id) ? [{
                        label: createMemberLoading.has(u.id) ? 'Wird angelegt…' : 'Mitglied anlegen',
                        onClick: () => handleCreateMember(u),
                      }] : []),
                      ...(self?.role === 'admin' && u.id !== self?.id && u.role !== 'admin' ? [{
                        label: 'Testen als',
                        onClick: () => startImpersonation(u.id, `${u.first_name} ${u.last_name}`.trim()),
                      }] : []),
                      { label: 'Löschen', onClick: () => handleDeleteUser(u), variant: 'danger' as const },
                    ]} />
                  </td>
                </tr>
              )
            })}
            {users.length === 0 && (
              <tr>
                <td colSpan={4} className="px-6 py-6 text-center text-brand-text-subtle">Keine Nutzer vorhanden</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <Pagination currentPage={currentPage} totalPages={totalPages} onPageChange={goToPage} />
    </div>
  )
}
