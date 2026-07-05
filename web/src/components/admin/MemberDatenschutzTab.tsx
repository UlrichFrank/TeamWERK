interface Member {
  dsgvo_verarbeitung?: boolean
  dsgvo_verarbeitung_date?: string
  dsgvo_weitergabe?: boolean
  dsgvo_weitergabe_date?: string
  foto_veroeffentlichung?: boolean
  foto_veroeffentlichung_date?: string
  cross_team_visible?: boolean
}

interface Draft {
  id: number
  field_name: string
  old_value: { verarbeitung?: boolean; weitergabe?: boolean; foto_veroeffentlichung?: boolean; [k: string]: unknown } | null
  new_value: { verarbeitung?: boolean; weitergabe?: boolean; foto_veroeffentlichung?: boolean; [k: string]: unknown } | null
}

// Erklärtext je DSGVO-Einwilligung — was der Schalter bedeutet.
const CONSENT_HELP = {
  verarbeitung: 'Erlaubt dem Verein, die Mitgliedsdaten (Stammdaten, Kontakt) zur Vereinsverwaltung zu verarbeiten.',
  weitergabe: 'Erlaubt die Weitergabe von Mitgliedsdaten an Dritte (z. B. Verband, Versicherung), soweit für den Vereinsbetrieb erforderlich.',
  foto: 'Erlaubt die Veröffentlichung von Fotos der Person auf öffentlichen Kanälen des Vereins (Homepage team-stuttgart.org, Spielberichte). Nicht zu verwechseln mit der internen Profilbild-Sichtbarkeit.',
}

interface Props {
  form: Member
  isNew: boolean
  drafts: Draft[]
  onFormChange: (updates: Partial<Member>) => void
  onDraftAccept: (draftId: number) => Promise<void>
  onDraftReject: (draftId: number) => Promise<void>
  onSave: () => Promise<void>
  saving: boolean
  saved: boolean
  error: string
}

export default function MemberDatenschutzTab({ form, isNew, drafts, onFormChange, onDraftAccept, onDraftReject, onSave, saving, saved, error }: Props) {
  const dsgvoDraft = drafts.find(d => d.field_name === 'dsgvo')

  return (
    <div className="space-y-6">
      {/* Sichtbarkeit für Mitglieder */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text mb-1">Sichtbarkeit</h2>
        <p className="text-xs text-brand-text-subtle mb-3">
          Wenn aktiviert, sehen auch Mitglieder anderer Mannschaften Namen und Rückmeldung
          dieses Mitglieds bei gemeinsamen Multi-Team-Terminen.
        </p>
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={form.cross_team_visible || false}
            onChange={e => onFormChange({ cross_team_visible: e.target.checked })}
            className="w-4 h-4 accent-brand-yellow"
          />
          <span className="text-sm text-brand-text">Sichtbarkeit für Mitglieder</span>
        </label>
      </div>

      {/* DSGVO */}
      <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
        <h2 className="font-semibold text-brand-text mb-4">Datenschutz (DSGVO)</h2>
        <div className="space-y-3">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={form.dsgvo_verarbeitung || false}
              onChange={e => onFormChange({ dsgvo_verarbeitung: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">Datenverarbeitung eingewilligt</span>
            {dsgvoDraft && <span className="text-sm text-brand-text-muted">(Änderung ausstehend)</span>}
          </label>
          <p className="ml-6 text-xs text-brand-text-muted">{CONSENT_HELP.verarbeitung}</p>
          {form.dsgvo_verarbeitung_date && (
            <p className="ml-6 text-xs text-brand-text-muted">seit {form.dsgvo_verarbeitung_date}</p>
          )}

          <label className="flex items-center gap-2 cursor-pointer mt-4">
            <input
              type="checkbox"
              checked={form.dsgvo_weitergabe || false}
              onChange={e => onFormChange({ dsgvo_weitergabe: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">Datenweitergabe eingewilligt</span>
            {dsgvoDraft && <span className="text-sm text-brand-text-muted">(Änderung ausstehend)</span>}
          </label>
          <p className="ml-6 text-xs text-brand-text-muted">{CONSENT_HELP.weitergabe}</p>
          {form.dsgvo_weitergabe_date && (
            <p className="ml-6 text-xs text-brand-text-muted">seit {form.dsgvo_weitergabe_date}</p>
          )}

          <label className="flex items-center gap-2 cursor-pointer mt-4">
            <input
              type="checkbox"
              checked={form.foto_veroeffentlichung || false}
              onChange={e => onFormChange({ foto_veroeffentlichung: e.target.checked })}
              className="w-4 h-4 accent-brand-yellow"
            />
            <span className="text-sm text-brand-text">Foto-Veröffentlichung eingewilligt</span>
            {dsgvoDraft && <span className="text-sm text-brand-text-muted">(Änderung ausstehend)</span>}
          </label>
          <p className="ml-6 text-xs text-brand-text-muted">{CONSENT_HELP.foto}</p>
          {form.foto_veroeffentlichung_date && (
            <p className="ml-6 text-xs text-brand-text-muted">seit {form.foto_veroeffentlichung_date}</p>
          )}

          {dsgvoDraft && (
            <div className="p-3 bg-brand-info/10 border border-brand-info/30 rounded-lg text-sm text-brand-text">
              <div className="flex items-center justify-between gap-2 flex-wrap">
                <span>
                  <span className="font-medium">Angeforderte DSGVO-Änderung:</span>{' '}
                  Verarbeitung: {dsgvoDraft.new_value?.verarbeitung ? 'Ja' : 'Nein'}, Weitergabe: {dsgvoDraft.new_value?.weitergabe ? 'Ja' : 'Nein'}, Foto-Veröffentlichung: {dsgvoDraft.new_value?.foto_veroeffentlichung ? 'Ja' : 'Nein'}
                </span>
                <div className="flex gap-2">
                  <button
                    onClick={() => onDraftAccept(dsgvoDraft.id)}
                    className="px-2 py-1 bg-green-100 text-green-700 rounded hover:bg-green-200 font-medium text-xs"
                  >
                    Annehmen
                  </button>
                  <button
                    onClick={() => onDraftReject(dsgvoDraft.id)}
                    className="px-2 py-1 bg-brand-danger-light text-brand-danger rounded hover:opacity-80 font-medium text-xs"
                  >
                    Ablehnen
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Save Button */}
      {!isNew && (
        <div className="flex items-center gap-3">
          <button
            onClick={onSave}
            disabled={saving}
            className="bg-brand-yellow text-brand-black rounded-md px-4 py-2.5 sm:py-2 text-sm font-medium hover:bg-brand-black hover:text-brand-yellow transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {saving ? 'Speichern…' : 'Speichern'}
          </button>
          {saved && <span className="text-sm text-green-600">Gespeichert</span>}
          {error && <span className="text-sm text-brand-danger">{error}</span>}
        </div>
      )}
    </div>
  )
}
