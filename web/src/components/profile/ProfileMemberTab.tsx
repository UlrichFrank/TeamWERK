import { useState, useEffect } from 'react'
import { api } from '../../lib/api'
import { Member, Parent, ChangeDraft } from '../../pages/ProfilePage'

interface MemberEditData {
  first_name: string
  last_name: string
  date_of_birth: string
  jersey_number: number | null
  position: string
}

interface Props {
  ownMember: Member | null
  children?: Member[]
  parents?: Parent[]
  onDraftWithdrawn?: () => void
  onSaveDirect?: (data: MemberEditData) => Promise<void>
}

const CLUB_FUNCTION_LABELS: Record<string, string> = {
  spieler: 'Spieler', trainer: 'Trainer', sportliche_leitung: 'Sportliche Leitung',
  vorstand: 'Vorstand', vorstand_beisitzer: 'Vorstands-Beisitzer', kassierer: 'Kassierer',
}

const FIELD_LABELS: Record<string, string> = {
  first_name: 'Vorname',
  last_name: 'Nachname',
  street: 'Straße',
  zip: 'PLZ',
  city: 'Ort',
  iban: 'IBAN',
}

export default function ProfileMemberTab({ ownMember, children = [], parents = [], onDraftWithdrawn, onSaveDirect }: Props) {
  const [drafts, setDrafts] = useState<ChangeDraft[]>([])
  const [cancelError, setCancelError] = useState('')

  // Edit state (only used when onSaveDirect is provided)
  const [editFirstName, setEditFirstName] = useState('')
  const [editLastName, setEditLastName] = useState('')
  const [editDob, setEditDob] = useState('')
  const [editJersey, setEditJersey] = useState('')
  const [editPosition, setEditPosition] = useState('')
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [saveError, setSaveError] = useState('')

  useEffect(() => {
    if (ownMember && !onSaveDirect) loadDrafts()
    if (ownMember && onSaveDirect) {
      setEditFirstName(ownMember.first_name)
      setEditLastName(ownMember.last_name)
      setEditDob(ownMember.date_of_birth ? ownMember.date_of_birth.slice(0, 10) : '')
      setEditJersey(ownMember.jersey_number?.toString() ?? '')
      setEditPosition(ownMember.position)
    }
  }, [ownMember?.id])

  const loadDrafts = async () => {
    if (!ownMember) return
    try {
      const r = await api.get(`/members/${ownMember.id}/change-drafts`)
      setDrafts(r.data?.drafts ?? [])
    } catch {}
  }

  const handleDirectSave = async () => {
    if (!onSaveDirect || !ownMember) return
    setSaving(true)
    setSaveError('')
    try {
      await onSaveDirect({
        first_name: editFirstName,
        last_name: editLastName,
        date_of_birth: editDob,
        jersey_number: editJersey !== '' ? parseInt(editJersey, 10) : null,
        position: editPosition,
      })
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch {
      setSaveError('Fehler beim Speichern')
    } finally {
      setSaving(false)
    }
  }

  const directChanged = onSaveDirect && ownMember && (
    editFirstName !== ownMember.first_name ||
    editLastName !== ownMember.last_name ||
    editDob !== (ownMember.date_of_birth ? ownMember.date_of_birth.slice(0, 10) : '') ||
    editJersey !== (ownMember.jersey_number?.toString() ?? '') ||
    editPosition !== ownMember.position
  )

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

  const inputCls = `w-full border border-brand-border rounded-md px-3 py-2 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow`

  return (
    <div className="space-y-6">
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text-muted mb-4">Stammdaten</h2>
        {onSaveDirect ? (
          <div className="space-y-3">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Vorname</label>
                <input type="text" value={editFirstName} onChange={e => setEditFirstName(e.target.value)} className={inputCls} />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Nachname</label>
                <input type="text" value={editLastName} onChange={e => setEditLastName(e.target.value)} className={inputCls} />
              </div>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Geburtsdatum</label>
                <input type="date" value={editDob} onChange={e => setEditDob(e.target.value)} className={inputCls} />
              </div>
              <div>
                <label className="block text-sm font-medium text-brand-text-muted mb-1">Rückennummer</label>
                <input type="number" min="1" max="99" value={editJersey} onChange={e => setEditJersey(e.target.value)} placeholder="–" className={inputCls} />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-brand-text-muted mb-1">Position</label>
              <input type="text" value={editPosition} onChange={e => setEditPosition(e.target.value)} placeholder="–" className={inputCls} />
            </div>
            <div className="space-y-3 text-sm pt-2 border-t border-brand-border-subtle">
              <Row label="Passnummer" value={ownMember.pass_number || '–'} />
              <Row label="Status" value={ownMember.status || '–'} />
              {(ownMember.club_functions ?? []).length > 0 && (
                <Row label="Vereinsfunktion" value={(ownMember.club_functions ?? []).map(f => CLUB_FUNCTION_LABELS[f] ?? f).join(', ')} />
              )}
            </div>
            {directChanged && (
              <div className="flex items-center gap-3 pt-1">
                <button
                  onClick={handleDirectSave}
                  disabled={saving}
                  className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {saving ? 'Speichern…' : 'Speichern'}
                </button>
                {saved && <span className="text-sm text-green-600">Gespeichert</span>}
                {saveError && <span className="text-sm text-brand-danger">{saveError}</span>}
              </div>
            )}
          </div>
        ) : (
          <div className="space-y-3 text-sm">
            <Row label="Vorname" value={ownMember.first_name} />
            <Row label="Nachname" value={ownMember.last_name} />
            <Row label="Geburtsdatum" value={formatDate(ownMember.date_of_birth)} />
            <Row label="Passnummer" value={ownMember.pass_number || '–'} />
            <Row label="Rückennummer" value={ownMember.jersey_number?.toString() ?? '–'} />
            <Row label="Position" value={ownMember.position || '–'} />
            <Row label="Status" value={ownMember.status || '–'} />
            {(ownMember.club_functions ?? []).length > 0 && (
              <Row label="Vereinsfunktion" value={(ownMember.club_functions ?? []).map(f => CLUB_FUNCTION_LABELS[f] ?? f).join(', ')} />
            )}
            {(ownMember.street || ownMember.zip || ownMember.city) && (
              <Row label="Adresse" value={[ownMember.street, [ownMember.zip, ownMember.city].filter(Boolean).join(' ')].filter(Boolean).join(', ')} />
            )}
          </div>
        )}
      </div>

      {/* Familie */}
      {(children.length > 0 || parents.length > 0) && (
        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
          <h2 className="font-semibold text-brand-text-muted mb-4">Familie</h2>
          <div className="space-y-3 text-sm">
            {parents.length > 0 && (
              <div>
                <p className="text-brand-text-muted text-xs font-medium mb-1">Erziehungsberechtigte</p>
                <div className="space-y-1">
                  {parents.map(p => (
                    <p key={p.id} className="text-brand-text">• {p.name} ({p.email})</p>
                  ))}
                </div>
              </div>
            )}
            {children.length > 0 && (
              <div>
                <p className="text-brand-text-muted text-xs font-medium mb-1">Meine Kinder</p>
                <div className="space-y-1">
                  {children.map(c => (
                    <p key={c.id} className="text-brand-text">• {c.first_name} {c.last_name}</p>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

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
