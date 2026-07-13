import { useEffect, useState } from 'react'
import { api } from '../../lib/api'
import Toggle from '../Toggle'
import { Member, ChangeDraft } from '../../pages/ProfilePage'

interface Props {
  ownMember: Member
  onUpdated: () => void
}

interface DsgvoDraftValue {
  verarbeitung?: boolean
  weitergabe?: boolean
  foto_veroeffentlichung?: boolean
}

export default function ProfileDatenschutzTab({ ownMember, onUpdated }: Props) {
  const [crossTeamVisible, setCrossTeamVisible] = useState<boolean>(!!ownMember.cross_team_visible)
  const [savingVisibility, setSavingVisibility] = useState(false)
  const [visibilityError, setVisibilityError] = useState<string | null>(null)

  const [verarbeitung, setVerarbeitung] = useState<boolean>(!!ownMember.dsgvo_verarbeitung)
  const [weitergabe, setWeitergabe] = useState<boolean>(!!ownMember.dsgvo_weitergabe)
  const [fotoVeroeff, setFotoVeroeff] = useState<boolean>(!!ownMember.foto_veroeffentlichung)
  const [dsgvoDraft, setDsgvoDraft] = useState<ChangeDraft | null>(null)
  const [savingDsgvo, setSavingDsgvo] = useState(false)
  const [dsgvoError, setDsgvoError] = useState<string | null>(null)
  const [dsgvoSaved, setDsgvoSaved] = useState(false)

  // Beim Wechsel des Members (z. B. /profil/kind/3 → /profil/kind/4) lokalen
  // State neu aus den Props aufsetzen, sonst zeigt der Toggle weiter den
  // Wert des vorigen Kindes.
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- bewusster Zustand-Sync im Effekt (Prop-/Abhängigkeits-getrieben), kein Ableitungs-Bug
    setCrossTeamVisible(!!ownMember.cross_team_visible)
    setVerarbeitung(!!ownMember.dsgvo_verarbeitung)
    setWeitergabe(!!ownMember.dsgvo_weitergabe)
    setFotoVeroeff(!!ownMember.foto_veroeffentlichung)
    setVisibilityError(null)
    setDsgvoError(null)
  }, [ownMember.id, ownMember.cross_team_visible, ownMember.dsgvo_verarbeitung, ownMember.dsgvo_weitergabe, ownMember.foto_veroeffentlichung])

  const loadDraft = () => {
    api.get(`/members/${ownMember.id}/change-drafts`).then(r => {
      const drafts: ChangeDraft[] = r.data?.drafts ?? []
      setDsgvoDraft(drafts.find(d => d.field_name === 'dsgvo') ?? null)
    }).catch(() => {})
  }

  useEffect(() => {
    loadDraft()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ownMember.id])

  const toggleCrossTeamVisible = async () => {
    const next = !crossTeamVisible
    setCrossTeamVisible(next)
    setSavingVisibility(true)
    setVisibilityError(null)
    try {
      await api.put(`/members/${ownMember.id}/cross-team-visible`, { cross_team_visible: next })
      onUpdated()
    } catch {
      setCrossTeamVisible(!next)
      setVisibilityError('Speichern fehlgeschlagen. Bitte erneut versuchen.')
    } finally {
      setSavingVisibility(false)
    }
  }

  const dsgvoChanged =
    verarbeitung !== !!ownMember.dsgvo_verarbeitung ||
    weitergabe !== !!ownMember.dsgvo_weitergabe ||
    fotoVeroeff !== !!ownMember.foto_veroeffentlichung

  const requestDsgvoChange = async () => {
    setSavingDsgvo(true)
    setDsgvoError(null)
    try {
      await api.post(`/members/${ownMember.id}/change-request`, {
        field_name: 'dsgvo',
        new_value: {
          verarbeitung,
          weitergabe,
          foto_veroeffentlichung: fotoVeroeff,
        },
      })
      loadDraft()
      setDsgvoSaved(true)
      setTimeout(() => setDsgvoSaved(false), 2500)
    } catch {
      setDsgvoError('Anfrage konnte nicht gestellt werden.')
    } finally {
      setSavingDsgvo(false)
    }
  }

  const withdrawDsgvoDraft = async () => {
    if (!dsgvoDraft) return
    setSavingDsgvo(true)
    setDsgvoError(null)
    try {
      await api.delete(`/members/${ownMember.id}/change-drafts/${dsgvoDraft.id}`)
      setDsgvoDraft(null)
      setVerarbeitung(!!ownMember.dsgvo_verarbeitung)
      setWeitergabe(!!ownMember.dsgvo_weitergabe)
      setFotoVeroeff(!!ownMember.foto_veroeffentlichung)
    } catch {
      setDsgvoError('Zurückziehen fehlgeschlagen.')
    } finally {
      setSavingDsgvo(false)
    }
  }

  const draftNew = (dsgvoDraft?.new_value ?? null) as DsgvoDraftValue | null

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
              {visibilityError && <p className="mt-2 text-xs text-brand-danger">{visibilityError}</p>}
            </div>
            <Toggle
              enabled={crossTeamVisible}
              onToggle={() => { if (!savingVisibility) toggleCrossTeamVisible() }}
              label="Sichtbarkeit für Mitglieder"
            />
          </div>
        </div>
      </div>

      {/* DSGVO — Änderungen laufen über Change-Request */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text mb-1">Datenschutz (DSGVO)</h2>
        <p className="text-xs text-brand-text-subtle mb-4">
          Diese Einwilligungen werden vom Verein dokumentiert. Änderungen musst du anfragen —
          der Vorstand nimmt sie an oder lehnt sie ab.
        </p>

        {dsgvoDraft && (
          <div className="mb-4 p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
            Änderungsanfrage ausstehend — wird beim erneuten „Änderung anfragen" aktualisiert.
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={verarbeitung}
                onChange={e => setVerarbeitung(e.target.checked)}
                className="w-4 h-4 accent-brand-yellow"
                aria-label="Datenverarbeitung eingewilligt"
              />
              <span className="text-sm text-brand-text">Datenverarbeitung eingewilligt</span>
              {draftNew && draftNew.verarbeitung !== !!ownMember.dsgvo_verarbeitung && (
                <span className="text-xs text-brand-text-muted">(angefragt: {draftNew.verarbeitung ? 'Ja' : 'Nein'})</span>
              )}
            </label>
            <p className="ml-6 text-xs text-brand-text-muted">
              Erlaubt dem Verein, deine Mitgliedsdaten (Stammdaten, Kontakt) zur Vereinsverwaltung zu verarbeiten.
            </p>
            {ownMember.dsgvo_verarbeitung_date && (
              <p className="ml-6 text-xs text-brand-text-muted">seit {ownMember.dsgvo_verarbeitung_date.slice(0, 10)}</p>
            )}
          </div>

          <div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={weitergabe}
                onChange={e => setWeitergabe(e.target.checked)}
                className="w-4 h-4 accent-brand-yellow"
                aria-label="Datenweitergabe eingewilligt"
              />
              <span className="text-sm text-brand-text">Datenweitergabe eingewilligt</span>
              {draftNew && draftNew.weitergabe !== !!ownMember.dsgvo_weitergabe && (
                <span className="text-xs text-brand-text-muted">(angefragt: {draftNew.weitergabe ? 'Ja' : 'Nein'})</span>
              )}
            </label>
            <p className="ml-6 text-xs text-brand-text-muted">
              Erlaubt die Weitergabe deiner Mitgliedsdaten an Dritte (z. B. Verband, Versicherung), soweit für den Vereinsbetrieb erforderlich.
            </p>
            {ownMember.dsgvo_weitergabe_date && (
              <p className="ml-6 text-xs text-brand-text-muted">seit {ownMember.dsgvo_weitergabe_date.slice(0, 10)}</p>
            )}
          </div>

          <div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={fotoVeroeff}
                onChange={e => setFotoVeroeff(e.target.checked)}
                className="w-4 h-4 accent-brand-yellow"
                aria-label="Foto-Veröffentlichung eingewilligt"
              />
              <span className="text-sm text-brand-text">Foto-Veröffentlichung eingewilligt</span>
              {draftNew && draftNew.foto_veroeffentlichung !== !!ownMember.foto_veroeffentlichung && (
                <span className="text-xs text-brand-text-muted">(angefragt: {draftNew.foto_veroeffentlichung ? 'Ja' : 'Nein'})</span>
              )}
            </label>
            <p className="ml-6 text-xs text-brand-text-muted">
              Erlaubt die Veröffentlichung von Fotos auf öffentlichen Kanälen des Vereins (Homepage team-stuttgart.org, Spielberichte). Nicht zu verwechseln mit der internen Profilbild-Sichtbarkeit.
            </p>
            {ownMember.foto_veroeffentlichung_date && (
              <p className="ml-6 text-xs text-brand-text-muted">seit {ownMember.foto_veroeffentlichung_date.slice(0, 10)}</p>
            )}
          </div>
        </div>

        <div className="flex items-center gap-3 mt-6">
          <button
            onClick={requestDsgvoChange}
            disabled={!dsgvoChanged || savingDsgvo}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {savingDsgvo ? 'Sende…' : 'Änderung anfragen'}
          </button>
          {dsgvoDraft && (
            <button
              onClick={withdrawDsgvoDraft}
              disabled={savingDsgvo}
              className="bg-brand-danger text-white rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-danger/90 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Anfrage zurückziehen
            </button>
          )}
          {dsgvoSaved && <span className="text-sm text-green-600">Anfrage gesendet</span>}
          {dsgvoError && <span className="text-sm text-brand-danger">{dsgvoError}</span>}
        </div>
      </div>
    </div>
  )
}
