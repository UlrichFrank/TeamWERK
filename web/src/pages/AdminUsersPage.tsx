import { useState, useEffect, useRef, FormEvent } from 'react'
import { X, Check, Upload, ChevronDown } from 'lucide-react'
import { api } from '../lib/api'
import { useAuth } from '../contexts/AuthContext'
import { useLiveUpdates } from '../hooks/useLiveUpdates'
import { usePagination } from '../lib/usePagination'
import ActionMenu from '../components/ActionMenu'
import Pagination from '../components/Pagination'
import { useEscapeKey } from '../lib/useEscapeKey'

interface User {
  id: number
  first_name: string
  last_name: string
  email: string
  role: string
  member_id?: number | null
  last_login_at?: string | null
  proxy?: boolean
}
interface Invitation {
  id: number
  email: string
  role: string
  comment: string
  expires_at: string
  member_id?: number | null
  member_name?: string
}
interface MembershipRequest { id: number; name: string; email: string; comment: string; status: string; created_at: string }
interface Member { id: number; first_name: string; last_name: string }

const ROLE_LABELS: Record<string, string> = { admin: 'Admin', standard: 'Standard' }
const ALL_ROLES = ['admin', 'standard'] as const

const INPUT = 'w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const m = Math.floor(diff / 60000)
  if (m < 2) return 'gerade eben'
  if (m < 60) return `vor ${m} Min.`
  const h = Math.floor(m / 60)
  if (h < 24) return `vor ${h} Std.`
  const d = Math.floor(h / 24)
  if (d === 1) return 'gestern'
  if (d < 30) return `vor ${d} Tagen`
  const mo = Math.floor(d / 30)
  return mo === 1 ? 'vor 1 Monat' : `vor ${mo} Monaten`
}

export default function AdminUsersPage() {
  const { user: self, startImpersonation } = useAuth()
  const { items: users, setSearch, total, currentPage, totalPages, goToPage, refresh: refreshUsers } = usePagination<User>('/users')
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [requests, setRequests] = useState<MembershipRequest[]>([])
  const [filterText, setFilterText] = useState('')

  const [createdMemberUserIds, setCreatedMemberUserIds] = useState<Set<number>>(new Set())
  const [createMemberLoading, setCreateMemberLoading] = useState<Set<number>>(new Set())
  const [createMemberErrors, setCreateMemberErrors] = useState<Map<number, string>>(new Map())

  const [sendingInvitation, setSendingInvitation] = useState<Set<number>>(new Set())
  const [invitationFeedback, setInvitationFeedback] = useState<Map<number, { ok: boolean; msg: string }>>(new Map())

  // + Neu modal (single invite)
  const [showInviteModal, setShowInviteModal] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState('standard')
  const [inviteComment, setInviteComment] = useState('')
  const [inviteSent, setInviteSent] = useState(false)
  const [inviteError, setInviteError] = useState('')

  // Split-button dropdown
  const [showDropdown, setShowDropdown] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // CSV import modal
  const [showCsvModal, setShowCsvModal] = useState(false)
  const [csvFile, setCsvFile] = useState<File | null>(null)
  const [csvLoading, setCsvLoading] = useState(false)
  const [csvResult, setCsvResult] = useState<{ created: number; skipped: number } | null>(null)
  const [csvError, setCsvError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Member link modal
  const [linkModal, setLinkModal] = useState<{ invId: number; email: string } | null>(null)
  const [memberSearch, setMemberSearch] = useState('')
  const [memberResults, setMemberResults] = useState<Member[]>([])
  const [linkLoading, setLinkLoading] = useState(false)
  const [linkError, setLinkError] = useState('')

  // Proxy account activate modal
  const [activateModal, setActivateModal] = useState<{ userId: number; name: string } | null>(null)
  const [activateEmail, setActivateEmail] = useState('')
  const [activateLoading, setActivateLoading] = useState(false)
  const [activateError, setActivateError] = useState('')

  const loadInvitationsAndRequests = () => Promise.all([
    api.get('/invitations').then(r => setInvitations(r.data ?? [])),
    api.get('/membership-requests').then(r => setRequests(r.data ?? [])),
  ])

  useEffect(() => { loadInvitationsAndRequests() }, [])

  useLiveUpdates(event => { if (event === 'members') loadInvitationsAndRequests() })

  useEffect(() => {
    if (!showDropdown) return
    const handler = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node))
        setShowDropdown(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [showDropdown])

  useEffect(() => {
    if (!linkModal) return
    const t = setTimeout(async () => {
      const r = await api.get('/members', { params: { search: memberSearch, limit: 20 } })
      setMemberResults(r.data?.items ?? r.data ?? [])
    }, 300)
    return () => clearTimeout(t)
  }, [memberSearch, linkModal])

  const closeInviteModal = () => {
    setShowInviteModal(false)
    setInviteEmail('')
    setInviteRole('standard')
    setInviteComment('')
    setInviteSent(false)
    setInviteError('')
  }

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault()
    setInviteError('')
    try {
      await api.post('/auth/invite', { email: inviteEmail, role: inviteRole, comment: inviteComment })
      setInviteSent(true)
      setInviteEmail('')
      setInviteComment('')
      setTimeout(() => { closeInviteModal(); loadInvitationsAndRequests() }, 2000)
    } catch {
      setInviteError('Einladung konnte nicht gesendet werden. Bitte E-Mail-Konfiguration prüfen.')
    }
  }

  const closeCsvModal = () => {
    setShowCsvModal(false)
    setCsvFile(null)
    setCsvResult(null)
    setCsvError('')
  }

  const closeLinkModal = () => {
    setLinkModal(null)
    setMemberSearch('')
    setMemberResults([])
    setLinkError('')
  }

  const closeActivateModal = () => {
    setActivateModal(null)
    setActivateEmail('')
    setActivateError('')
  }

  const handleActivateProxy = async (e: FormEvent) => {
    e.preventDefault()
    if (!activateModal) return
    setActivateLoading(true)
    setActivateError('')
    try {
      await api.put(`/users/${activateModal.userId}`, { can_login: 1, email: activateEmail })
      closeActivateModal()
      refreshUsers()
    } catch (err: any) {
      if (err?.response?.status === 409) {
        setActivateError('Diese E-Mail-Adresse ist bereits vergeben.')
      } else {
        setActivateError('Aktivierung fehlgeschlagen.')
      }
    } finally {
      setActivateLoading(false)
    }
  }

  useEscapeKey(showInviteModal ? closeInviteModal : showCsvModal ? closeCsvModal : linkModal ? closeLinkModal : activateModal ? closeActivateModal : null)

  const handleCsvUpload = async () => {
    if (!csvFile) return
    setCsvLoading(true)
    setCsvError('')
    setCsvResult(null)
    try {
      const form = new FormData()
      form.append('file', csvFile)
      const r = await api.post('/invitations/import-csv', form)
      setCsvResult(r.data)
      loadInvitationsAndRequests()
    } catch (e: any) {
      setCsvError(e?.response?.data || 'Fehler beim Import. Bitte CSV-Datei prüfen.')
    } finally {
      setCsvLoading(false)
    }
  }

  const handleSendInvitation = async (inv: Invitation) => {
    setSendingInvitation(prev => new Set(prev).add(inv.id))
    try {
      await api.post(`/invitations/${inv.id}/send`)
      setInvitationFeedback(prev => new Map(prev).set(inv.id, { ok: true, msg: 'Gesendet' }))
      setTimeout(() => setInvitationFeedback(prev => { const m = new Map(prev); m.delete(inv.id); return m }), 3000)
      loadInvitationsAndRequests()
    } catch {
      setInvitationFeedback(prev => new Map(prev).set(inv.id, { ok: false, msg: 'Fehler beim Versand' }))
    } finally {
      setSendingInvitation(prev => { const s = new Set(prev); s.delete(inv.id); return s })
    }
  }

  const handleLinkMember = async (memberId: number) => {
    if (!linkModal) return
    setLinkLoading(true)
    setLinkError('')
    try {
      await api.put(`/invitations/${linkModal.invId}/member`, { member_id: memberId })
      closeLinkModal()
      loadInvitationsAndRequests()
    } catch (e: any) {
      setLinkError(e?.response?.status === 409 ? 'Mitglied ist bereits mit einem Nutzer verknüpft.' : 'Fehler beim Verknüpfen.')
    } finally {
      setLinkLoading(false)
    }
  }

  const handleUnlinkMember = async (inv: Invitation) => {
    await api.put(`/invitations/${inv.id}/member`, { member_id: null })
    loadInvitationsAndRequests()
  }

  const handleDeleteUser = async (u: User) => {
    if (!window.confirm(`Nutzer „${u.first_name} ${u.last_name}" (${u.email}) wirklich löschen?`)) return
    await api.delete(`/users/${u.id}`)
  }

  const handleDeleteInvitation = async (inv: Invitation) => {
    if (!window.confirm(`Einladung für ${inv.email} widerrufen?`)) return
    await api.delete(`/invitations/${inv.id}`)
    setInvitations(prev => prev.filter(x => x.id !== inv.id))
  }

  const handleApproveRequest = async (req: MembershipRequest) => {
    await api.post(`/membership-requests/${req.id}/approve`)
    setRequests(prev => prev.filter(x => x.id !== req.id))
  }

  const handleRejectRequest = async (req: MembershipRequest) => {
    await api.post(`/membership-requests/${req.id}/reject`)
    setRequests(prev => prev.filter(x => x.id !== req.id))
  }

  const handleDeleteRequest = async (req: MembershipRequest) => {
    if (!window.confirm(`Beitrittsanfrage von ${req.name} löschen?`)) return
    await api.delete(`/membership-requests/${req.id}`)
    setRequests(prev => prev.filter(x => x.id !== req.id))
  }

  const f = filterText.toLowerCase()
  const filteredRequests = f
    ? requests.filter(r => r.name.toLowerCase().includes(f) || r.email.toLowerCase().includes(f))
    : requests
  const filteredInvitations = f
    ? invitations.filter(i => i.email.toLowerCase().includes(f) || (i.member_name ?? '').toLowerCase().includes(f))
    : invitations

  const handleCreateMember = async (u: User) => {
    setCreateMemberLoading(prev => new Set(prev).add(u.id))
    setCreateMemberErrors(prev => { const m = new Map(prev); m.delete(u.id); return m })
    try {
      await api.post(`/users/${u.id}/create-member`)
      setCreatedMemberUserIds(prev => new Set(prev).add(u.id))
    } catch {
      setCreateMemberErrors(prev => new Map(prev).set(u.id, 'Fehler beim Anlegen'))
    } finally {
      setCreateMemberLoading(prev => { const s = new Set(prev); s.delete(u.id); return s })
    }
  }

  const handleRoleChange = async (u: User, newRole: string) => {
    await api.put(`/users/${u.id}/role`, { role: newRole })
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
              onChange={e => { setFilterText(e.target.value); setSearch(e.target.value) }}
              className="border border-brand-border rounded-md px-3 py-2.5 sm:py-1.5 text-xs text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow w-32 sm:w-auto"
            />
            <div ref={dropdownRef} className="relative">
              <div className="flex">
                <button
                  onClick={() => setShowInviteModal(true)}
                  className="text-xs bg-brand-yellow text-brand-black border border-brand-yellow rounded-l-md px-3 py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors whitespace-nowrap"
                >
                  + Neu
                </button>
                <button
                  onClick={() => setShowDropdown(v => !v)}
                  aria-label="Weitere Optionen"
                  className="text-xs bg-brand-yellow text-brand-black border border-brand-yellow border-l-brand-black/20 rounded-r-md px-2 py-1.5 font-medium hover:bg-brand-black hover:text-brand-yellow hover:border-brand-black transition-colors border-l"
                >
                  <ChevronDown className="w-3.5 h-3.5" />
                </button>
              </div>
              {showDropdown && (
                <div className="absolute right-0 mt-1 w-44 bg-white border border-brand-border rounded-md shadow-lg z-20">
                  <button
                    onClick={() => { setShowDropdown(false); setShowCsvModal(true) }}
                    className="w-full text-left px-4 py-2.5 text-xs text-brand-text hover:bg-brand-surface-card transition-colors flex items-center gap-2"
                  >
                    <Upload className="w-3.5 h-3.5 text-brand-text-muted" />
                    CSV importieren
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* + Neu Modal */}
      {showInviteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-lg">Einladung versenden</h2>
              <button onClick={closeInviteModal} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="px-6 py-5 space-y-3">
              {inviteSent && (
                <p className="text-sm flex items-center gap-1 text-brand-text">
                  <Check className="w-4 h-4 text-brand-info" /> Einladung gesendet
                </p>
              )}
              {inviteError && (
                <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{inviteError}</p>
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
                  <button type="button" onClick={closeInviteModal} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                    Abbrechen
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* CSV Import Modal */}
      {showCsvModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-lg">CSV importieren</h2>
              <button onClick={closeCsvModal} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="px-6 py-5 space-y-4">
              {csvResult ? (
                <div className="space-y-3">
                  <p className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
                    <span className="font-semibold">{csvResult.created}</span> Einladungen angelegt,{' '}
                    <span className="font-semibold">{csvResult.skipped}</span> übersprungen (bereits vorhanden)
                  </p>
                  <button onClick={closeCsvModal} className="w-full bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors">
                    Schließen
                  </button>
                </div>
              ) : (
                <div className="space-y-3">
                  <p className="text-sm text-brand-text-muted">
                    Liest die Spalten <code className="bg-brand-surface-card px-1 rounded">Email</code> und <code className="bg-brand-surface-card px-1 rounded">Email 2</code> aus der CSV-Datei. Bereits vorhandene Adressen werden übersprungen.
                  </p>
                  {csvError && (
                    <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{csvError}</p>
                  )}
                  <div
                    onClick={() => fileInputRef.current?.click()}
                    className="border-2 border-dashed border-brand-border rounded-lg p-6 text-center cursor-pointer hover:border-brand-yellow transition-colors"
                  >
                    <Upload className="w-6 h-6 mx-auto mb-2 text-brand-text-muted" />
                    {csvFile
                      ? <p className="text-sm font-medium text-brand-text">{csvFile.name}</p>
                      : <p className="text-sm text-brand-text-muted">CSV-Datei auswählen</p>
                    }
                    <input
                      ref={fileInputRef}
                      type="file"
                      accept=".csv"
                      className="hidden"
                      onChange={e => { setCsvFile(e.target.files?.[0] ?? null); setCsvError('') }}
                    />
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={handleCsvUpload}
                      disabled={!csvFile || csvLoading}
                      className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                    >
                      {csvLoading ? 'Wird importiert…' : 'Importieren'}
                    </button>
                    <button onClick={closeCsvModal} className="px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                      Abbrechen
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Member Link Modal */}
      {linkModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-lg">Mit Mitglied verknüpfen</h2>
              <button onClick={closeLinkModal} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="px-6 py-5 space-y-3">
              <p className="text-xs text-brand-text-muted">Einladung: <span className="font-medium text-brand-text">{linkModal.email}</span></p>
              {linkError && (
                <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{linkError}</p>
              )}
              <input
                type="search"
                placeholder="Mitglied suchen…"
                value={memberSearch}
                onChange={e => setMemberSearch(e.target.value)}
                className={INPUT}
                autoFocus
              />
              <div className="max-h-48 overflow-y-auto divide-y divide-brand-border-subtle border border-brand-border rounded-md">
                {memberResults.length === 0 && (
                  <p className="px-3 py-4 text-sm text-brand-text-subtle text-center">Keine Mitglieder gefunden</p>
                )}
                {memberResults.map(m => (
                  <button
                    key={m.id}
                    disabled={linkLoading}
                    onClick={() => handleLinkMember(m.id)}
                    className="w-full text-left px-3 py-2.5 text-sm text-brand-text hover:bg-brand-surface-card transition-colors disabled:opacity-40"
                  >
                    {m.first_name} {m.last_name}
                  </button>
                ))}
              </div>
              <button onClick={closeLinkModal} className="w-full px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                Abbrechen
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Proxy account activate modal */}
      {activateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-xl shadow-xl border-t-4 border-brand-yellow w-full max-w-sm mx-4">
            <div className="flex items-center justify-between px-6 py-4 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-lg">Proxy-Account aktivieren</h2>
              <button onClick={closeActivateModal} aria-label="Schließen" className="text-brand-text-muted hover:text-brand-text transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={handleActivateProxy} className="px-6 py-5 space-y-4">
              <p className="text-sm text-brand-text-muted">
                Account <span className="font-medium text-brand-text">{activateModal.name}</span> wird login-fähig.
                Eine E-Mail-Adresse ist erforderlich.
              </p>
              {activateError && (
                <p className="p-3 bg-brand-danger-light border border-brand-danger/30 rounded-lg text-sm text-brand-danger">{activateError}</p>
              )}
              <input
                type="email"
                required
                placeholder="E-Mail-Adresse"
                value={activateEmail}
                onChange={e => setActivateEmail(e.target.value)}
                className="w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow"
                autoFocus
              />
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={activateLoading || !activateEmail}
                  className="flex-1 bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {activateLoading ? 'Aktivieren…' : 'Aktivieren'}
                </button>
                <button type="button" onClick={closeActivateModal} className="flex-1 px-4 py-2 text-sm border border-brand-border rounded-md text-brand-text-muted hover:text-brand-text hover:border-brand-text-muted transition-colors">
                  Abbrechen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Pending requests and invitations */}
      {(filteredRequests.length > 0 || filteredInvitations.length > 0) && (
        <div className="mb-8">
          <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-x-auto">
            <div className="px-6 py-4 border-b border-brand-border-subtle">
              <h2 className="font-semibold text-brand-text">Ausstehende Anfragen & Einladungen ({filteredRequests.length + filteredInvitations.length})</h2>
            </div>
            <table className="w-full text-sm">
              <tbody className="divide-y divide-brand-border-subtle">
                {filteredRequests.map(req => (
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
                {filteredInvitations.map(inv => {
                  const feedback = invitationFeedback.get(inv.id)
                  return (
                    <tr key={`inv-${inv.id}`} className="hover:bg-brand-table-select transition-colors">
                      <td className="px-4 py-3">
                        <p className="text-brand-text-muted italic">{inv.email}</p>
                        {inv.member_name && (
                          <p className="text-xs text-brand-text-subtle mt-0.5 flex items-center gap-1">
                            <Check className="w-3 h-3 text-brand-info" />
                            {inv.member_name}
                          </p>
                        )}
                      </td>
                      <td className="hidden md:table-cell px-4 py-3 text-brand-text-subtle">{ROLE_LABELS[inv.role] || inv.role}</td>
                      <td className="hidden lg:table-cell px-4 py-3 text-brand-text-subtle text-xs">{inv.comment || '–'}</td>
                      <td className="px-4 py-3">
                        {feedback ? (
                          <span className={`text-xs font-medium ${feedback.ok ? 'text-brand-info' : 'text-brand-danger'}`}>{feedback.msg}</span>
                        ) : (
                          <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-brand-border-subtle text-brand-text-muted">Einladung</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <ActionMenu actions={[
                          {
                            label: sendingInvitation.has(inv.id) ? 'Wird gesendet…' : 'Einladung senden',
                            onClick: () => handleSendInvitation(inv),
                          },
                          inv.member_id
                            ? { label: 'Verknüpfung aufheben', onClick: () => handleUnlinkMember(inv) }
                            : { label: 'Mit Mitglied verknüpfen', onClick: () => { setLinkModal({ invId: inv.id, email: inv.email }); setMemberSearch('') } },
                          { label: 'Löschen', onClick: () => handleDeleteInvitation(inv), variant: 'danger' },
                        ]} />
                      </td>
                    </tr>
                  )
                })}
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
                  <td className="px-4 py-3 font-medium text-brand-text">
                    {u.first_name} {u.last_name}
                    {u.proxy && (
                      <span className="ml-2 inline-block px-1.5 py-0.5 rounded text-xs font-medium bg-brand-border-subtle text-brand-text-muted">Proxy</span>
                    )}
                  </td>
                  <td className="hidden md:table-cell px-4 py-3 text-brand-text-muted">{u.email || '–'}</td>
                  <td className="px-4 py-3">
                    {!u.proxy && canEdit ? (
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
                      <span className="text-xs text-brand-text-muted">{u.proxy ? '–' : ROLE_LABELS[u.role]}</span>
                    )}
                  </td>
                  <td className="hidden lg:table-cell px-4 py-3 text-xs text-brand-text-subtle">
                    {u.proxy ? '–' : u.last_login_at ? relativeTime(u.last_login_at) : '–'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {createMemberErrors.get(u.id) && (
                      <span className="text-xs text-brand-danger mr-2">{createMemberErrors.get(u.id)}</span>
                    )}
                    <ActionMenu actions={[
                      ...(u.proxy ? [{
                        label: 'Aktivieren',
                        onClick: () => { setActivateModal({ userId: u.id, name: `${u.first_name} ${u.last_name}`.trim() }); setActivateEmail('') },
                      }] : []),
                      ...(!u.proxy && !u.member_id && !createdMemberUserIds.has(u.id) ? [{
                        label: createMemberLoading.has(u.id) ? 'Wird angelegt…' : 'Mitglied anlegen',
                        onClick: () => handleCreateMember(u),
                      }] : []),
                      ...(!u.proxy && self?.role === 'admin' && u.id !== self?.id && u.role !== 'admin' ? [{
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
                <td colSpan={5} className="px-6 py-6 text-center text-brand-text-subtle">Keine Nutzer vorhanden</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <Pagination currentPage={currentPage} totalPages={totalPages} onPageChange={goToPage} />
    </div>
  )
}
