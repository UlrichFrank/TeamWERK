import { useEffect, useState } from 'react'
import { api } from '../../lib/api'
import Toggle from '../Toggle'
import { Member } from '../../pages/ProfilePage'

interface Props {
  ownMember: Member
  onUpdated: () => void
}

export default function ProfileDatenschutzTab({ ownMember, onUpdated }: Props) {
  const [crossTeamVisible, setCrossTeamVisible] = useState<boolean>(!!ownMember.cross_team_visible)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Beim Wechsel des Members (z. B. /profil/kind/3 → /profil/kind/4) lokalen
  // State neu aus den Props aufsetzen, sonst zeigt der Toggle weiter den
  // Wert des vorigen Kindes.
  useEffect(() => {
    setCrossTeamVisible(!!ownMember.cross_team_visible)
    setError(null)
  }, [ownMember.id, ownMember.cross_team_visible])

  const toggleCrossTeamVisible = async () => {
    const next = !crossTeamVisible
    setCrossTeamVisible(next)
    setSaving(true)
    setError(null)
    try {
      await api.put(`/members/${ownMember.id}/cross-team-visible`, { cross_team_visible: next })
      onUpdated()
    } catch {
      setCrossTeamVisible(!next)
      setError('Speichern fehlgeschlagen. Bitte erneut versuchen.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Sichtbarkeit */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow overflow-hidden">
        <div className="p-6 pb-2">
          <h2 className="font-semibold text-brand-text-muted mb-1">Sichtbarkeit</h2>
          <p className="text-xs text-brand-text-subtle mb-3">Steuere, ob du auch für Mitglieder anderer Mannschaften sichtbar bist.</p>
        </div>
        <div className="divide-y divide-brand-border-subtle">
          <div className="flex items-start justify-between gap-4 px-6 py-4">
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-brand-text">Sichtbarkeit für Mitglieder</p>
              <p className="mt-1 text-xs text-brand-text-muted">
                Wenn aktiviert, sehen auch Mitglieder anderer Mannschaften deinen Namen und
                deine Rückmeldung bei gemeinsamen Terminen mit mehreren Mannschaften
                (z. B. Vereinsfeier). Standard: aus.
              </p>
              {error && <p className="mt-2 text-xs text-brand-danger">{error}</p>}
            </div>
            <Toggle
              enabled={crossTeamVisible}
              onToggle={() => { if (!saving) toggleCrossTeamVisible() }}
              label="Sichtbarkeit für Mitglieder"
            />
          </div>
        </div>
      </div>

      {/* DSGVO (read-only) */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text mb-1">Datenschutz (DSGVO)</h2>
        <p className="text-xs text-brand-text-subtle mb-4">
          Diese Einwilligungen werden vom Verein dokumentiert. Änderungen kannst du über den
          Tab „Kontakt" anfragen.
        </p>
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={!!ownMember.dsgvo_verarbeitung}
              readOnly
              disabled
              className="w-4 h-4 accent-brand-yellow cursor-default opacity-70"
              aria-label="Datenverarbeitung eingewilligt"
            />
            <span className="text-sm text-brand-text">Datenverarbeitung eingewilligt</span>
          </div>
          {ownMember.dsgvo_verarbeitung_date && (
            <p className="-mt-2 ml-6 text-xs text-brand-text-muted">seit {ownMember.dsgvo_verarbeitung_date.slice(0, 10)}</p>
          )}

          <div className="flex items-center gap-2 mt-4">
            <input
              type="checkbox"
              checked={!!ownMember.dsgvo_weitergabe}
              readOnly
              disabled
              className="w-4 h-4 accent-brand-yellow cursor-default opacity-70"
              aria-label="Datenweitergabe eingewilligt"
            />
            <span className="text-sm text-brand-text">Datenweitergabe eingewilligt</span>
          </div>
          {ownMember.dsgvo_weitergabe_date && (
            <p className="-mt-2 ml-6 text-xs text-brand-text-muted">seit {ownMember.dsgvo_weitergabe_date.slice(0, 10)}</p>
          )}
        </div>
      </div>
    </div>
  )
}
