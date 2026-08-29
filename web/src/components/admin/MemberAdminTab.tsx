import { useState } from 'react'
import { CheckCircle, Mail } from 'lucide-react'
import { api } from '../../lib/api'
import { isLikelyNameMatch } from '../../lib/nameMatch'
import SearchableSelect from '../SearchableSelect'
import { BTN_PRIMARY } from '../../lib/buttonStyles'

interface User {
  id: number
  first_name: string
  last_name: string
  email: string
}

interface PendingInvitation {
  id: number
  email: string
  member_id?: number | null
}

interface Props {
  isNew: boolean
  memberId?: number
  memberName: { first_name: string; last_name: string }
  users: User[]
  invitations: PendingInvitation[]
  currentUserId: number | null
  welcomeEmailSentAt: string | null
  onWelcomeEmailSent: (sentAt: string) => void
  onLinkUser: (userId: number | null) => Promise<void>
  onLinkInvitation: (invitationId: number | null) => Promise<void>
  saving: boolean
  saved: boolean
  error: string
}

function formatSentAt(iso: string) {
  const d = new Date(iso)
  return d.toLocaleDateString('de-DE', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

export default function MemberAdminTab({
  isNew, memberId, memberName, users, invitations, currentUserId, welcomeEmailSentAt, onWelcomeEmailSent,
  onLinkUser, onLinkInvitation, saving, saved, error,
}: Props) {
  const linkedInvitation = memberId != null
    ? invitations.find(i => i.member_id === memberId) ?? null
    : null

  const initialValue = currentUserId
    ? `u:${currentUserId}`
    : linkedInvitation
      ? `i:${linkedInvitation.id}`
      : ''

  const [selected, setSelected] = useState<string>(initialValue)
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState('')

  const currentUser = currentUserId ? users.find(u => u.id === currentUserId) : null

  const userItems = users
    .slice()
    .sort((a, b) => (a.last_name + a.first_name).localeCompare(b.last_name + b.first_name))
    .map(u => ({
      value: `u:${u.id}`,
      label: `${u.first_name} ${u.last_name} (${u.email})`,
      searchText: `${u.first_name} ${u.last_name} ${u.email}`.toLowerCase(),
      likely: isLikelyNameMatch(u, memberName),
    }))
  const invitationItems = invitations
    .slice()
    .sort((a, b) => a.email.localeCompare(b.email))
    .map(i => ({
      value: `i:${i.id}`,
      label: `${i.email} (Einladung ausstehend)`,
      searchText: i.email.toLowerCase(),
      likely: false,
    }))
  const linkItems = [...userItems, ...invitationItems]

  const handleSave = async () => {
    if (!selected) {
      if (currentUserId) await onLinkUser(null)
      else if (linkedInvitation) await onLinkInvitation(null)
      return
    }
    const [type, rawId] = selected.split(':')
    if (type === 'u') await onLinkUser(Number(rawId))
    else await onLinkInvitation(Number(rawId))
  }

  const handleSendWelcome = async () => {
    if (!memberId) return
    setSending(true)
    setSendError('')
    try {
      const res = await api.post<{ sent_at: string }>(`/members/${memberId}/welcome-email`)
      onWelcomeEmailSent(res.data.sent_at)
    } catch {
      setSendError('Fehler beim Senden der Willkommensmail. Bitte erneut versuchen.')
    } finally {
      setSending(false)
    }
  }

  if (isNew) {
    return <div className="text-brand-text-muted">Nutzer-Verknüpfung kann nach dem Erstellen vorgenommen werden.</div>
  }

  return (
    <div className="space-y-6">
      {/* Nutzer verknüpfen */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text mb-4">Nutzer verknüpfen</h2>

        {(currentUser || linkedInvitation) && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm">
            <p className="font-medium text-brand-text">Aktuell verknüpft:</p>
            {currentUser
              ? <p className="text-brand-text-muted">{currentUser.first_name} {currentUser.last_name} ({currentUser.email})</p>
              : <p className="text-brand-text-muted">{linkedInvitation!.email} <span className="italic">(Einladung ausstehend)</span></p>
            }
          </div>
        )}

        <div>
          <label className="block text-sm font-medium text-brand-text-muted mb-1">Nutzer ändern</label>
          <SearchableSelect
            items={linkItems}
            value={selected}
            onChange={setSelected}
            placeholder="– Keine Verknüpfung –"
            clearLabel="– Keine Verknüpfung –"
            className="mb-3"
          />

          <button
            onClick={handleSave}
            disabled={saving}
            className={BTN_PRIMARY}
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
        </div>

        {saved && <p className="text-sm text-brand-success mt-3">Gespeichert</p>}
        {error && <p className="text-sm text-brand-danger mt-3">{error}</p>}
      </div>

      {/* Willkommensmail */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text mb-4">Willkommensmail</h2>

        {welcomeEmailSentAt ? (
          <div className="flex items-center gap-2 p-3 bg-green-50 border border-green-200 rounded-lg text-sm">
            <CheckCircle className="w-4 h-4 text-green-600 flex-shrink-0" />
            <span className="text-green-800">
              Mail wurde am {formatSentAt(welcomeEmailSentAt)} versendet.
            </span>
          </div>
        ) : (
          <div className="space-y-3">
            <p className="text-sm text-brand-text-muted">
              Sendet die Willkommensmail mit Vereinssatzung, Gebührenordnung und Leitbild an das Mitglied.
            </p>
            {!currentUserId && (
              <p className="text-sm text-brand-text-subtle italic">
                Bitte zuerst einen Nutzeraccount verknüpfen.
              </p>
            )}
            <button
              onClick={handleSendWelcome}
              disabled={!currentUserId || sending}
              className={`${BTN_PRIMARY} inline-flex items-center gap-2`}
            >
              <Mail className="w-4 h-4" />
              {sending ? 'Wird gesendet…' : 'Willkommensmail senden'}
            </button>
            {sendError && <p className="text-sm text-brand-danger">{sendError}</p>}
          </div>
        )}
      </div>
    </div>
  )
}
