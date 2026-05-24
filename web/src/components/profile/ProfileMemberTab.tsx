import { useState, useEffect } from 'react'
import { api } from '../../lib/api'
import { Member, ChangeDraft } from '../../pages/ProfilePage'

interface Props {
  ownMember: Member | null
  onDraftWithdrawn?: () => void
}

const FIELD_LABELS: Record<string, string> = {
  first_name: 'Vorname',
  last_name: 'Nachname',
  street: 'Straße',
  zip: 'PLZ',
  city: 'Ort',
  iban: 'IBAN',
}

export default function ProfileMemberTab({ ownMember, onDraftWithdrawn }: Props) {
  const [drafts, setDrafts] = useState<ChangeDraft[]>([])
  const [cancelError, setCancelError] = useState('')

  useEffect(() => {
    if (ownMember) loadDrafts()
  }, [ownMember?.id])

  const loadDrafts = async () => {
    if (!ownMember) return
    try {
      const r = await api.get(`/members/${ownMember.id}/change-drafts`)
      setDrafts(r.data?.drafts ?? [])
    } catch {}
  }

  const handleCancelDraft = async (draftId: number) => {
    if (!ownMember) return
    setCancelError('')
    try {
      await api.delete(`/members/${ownMember.id}/change-drafts/${draftId}`)
      setDrafts(prev => prev.filter(d => d.id !== draftId))
      onDraftWithdrawn?.()
    } catch {
      setCancelError('Fehler beim Zurückziehen')
    }
  }

  if (!ownMember) {
    return <div className="text-brand-text-muted">Keine Mitgliedsdaten verfügbar.</div>
  }

  const formatDate = (s: string) => {
    if (!s) return '–'
    return new Date(s).toLocaleDateString('de-DE')
  }

  const profilDraft = drafts.find(d => d.field_name === 'profil')

  return (
    <div className="space-y-6">
      {/* Stammdaten — read-only */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Stammdaten</h2>
        <div className="space-y-3 text-sm">
          <Row label="Vorname" value={ownMember.first_name} />
          <Row label="Nachname" value={ownMember.last_name} />
          <Row label="Geburtsdatum" value={formatDate(ownMember.date_of_birth)} />
          <Row label="Passnummer" value={ownMember.pass_number || '–'} />
          <Row label="Rückennummer" value={ownMember.jersey_number?.toString() ?? '–'} />
          <Row label="Position" value={ownMember.position || '–'} />
          <Row label="Status" value={ownMember.status || '–'} />
          {(ownMember.street || ownMember.zip || ownMember.city) && (
            <Row label="Adresse" value={[ownMember.street, [ownMember.zip, ownMember.city].filter(Boolean).join(' ')].filter(Boolean).join(', ')} />
          )}
        </div>
      </div>

      {/* Ausstehende Anfrage */}
      {profilDraft && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-brand-text-muted mb-4">Ausstehende Anfrage</h2>
          <p className="text-xs text-brand-text-subtle mb-4">Diese Änderungen warten auf Freigabe durch den Verein.</p>
          <div className="space-y-2 text-sm mb-4">
            {Object.entries(FIELD_LABELS).map(([field, label]) => {
              const oldVal = profilDraft.old_value?.[field]
              const newVal = profilDraft.new_value?.[field]
              if (oldVal === undefined && newVal === undefined) return null
              if (oldVal === newVal) return null
              return (
                <div key={field} className="flex gap-2">
                  <span className="text-brand-text-muted w-24 shrink-0">{label}:</span>
                  <span className="text-brand-text-muted line-through mr-1">{oldVal || '–'}</span>
                  <span className="text-brand-text font-medium">{newVal || '–'}</span>
                </div>
              )
            })}
          </div>
          <button
            onClick={() => handleCancelDraft(profilDraft.id)}
            className="bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors"
          >
            Zurückziehen
          </button>
          {cancelError && <p className="mt-2 text-sm text-brand-danger">{cancelError}</p>}
        </div>
      )}
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex gap-2">
      <span className="text-brand-text-muted w-36 shrink-0">{label}:</span>
      <span className="text-brand-text">{value}</span>
    </div>
  )
}
