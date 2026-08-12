import type { TeamForName } from '../lib/teamName'

/**
 * Die beiden Zusatzfelder eines Dienstplan-Vorlagen-Eintrags, die es in ZWEI
 * Editoren gibt: der Modal-Editor auf `/dienstplan-vorlagen` (Liste, „Bearbeiten")
 * und die Detailseite `/dienstplan-vorlagen/:id`. Sie liegen bewusst hier statt
 * doppelt in den Seiten — die erste Fassung der Teamauswahl war nur in der
 * Detailseite gelandet und im Modal unsichtbar, obwohl beide dieselbe
 * PUT-Route mit demselben `items`-Array bedienen.
 */

/**
 * Team-Einschränkung eines Vorlagen-Eintrags. Achtung: umgekehrte Leer-Semantik
 * zur Zielgruppe direkt daneben — keine Auswahl heißt hier „gilt für alle Teams",
 * nicht „für niemanden". Deshalb steht „alle" im Hinweis hervorgehoben.
 *
 * Optionen sind die Kaderteams der aktiven Saison; bereits gespeicherte Teams
 * ohne aktuellen Kader-Eintrag tauchen hier nicht auf, bleiben aber erhalten
 * (der Toggle baut das Array nie neu auf).
 */
export function TeamScopeField({ teams, shortNames, selected, onToggle }: {
  teams: TeamForName[]
  shortNames: Map<number, string>
  selected: number[]
  onToggle: (teamID: number, checked: boolean) => void
}) {
  if (teams.length === 0) return null
  return (
    <div>
      <label className="block text-xs text-brand-text-muted mb-1">
        Kaderteams <span className="text-brand-text-subtle">(leer = <strong className="font-semibold">alle</strong> Teams)</span>
      </label>
      <div className="flex flex-wrap gap-x-3 gap-y-1">
        {teams.map(t => (
          <label key={t.id} className="flex items-center gap-1.5 text-xs cursor-pointer whitespace-nowrap">
            <input
              type="checkbox"
              checked={selected.includes(t.id)}
              onChange={e => onToggle(t.id, e.target.checked)}
              className="accent-brand-yellow"
            />
            {shortNames.get(t.id) ?? String(t.id)}
          </label>
        ))}
      </div>
    </div>
  )
}

/**
 * Bewirtungsrotations-Cap eines Vorlagen-Eintrags (kuchendienst-rotation).
 * Leer = Rotation deaktiviert (Default, bisheriges Verhalten). Ein gesetzter
 * Wert aktiviert die tagesweite Team-Warteschlange — setzt serverseitig
 * same_day_behavior/adjacent_day_behavior='normal' beim Diensttyp voraus,
 * die Prüfung selbst läuft aber nur serverseitig (kein Client-Vorab-Check).
 */
export function RotationCapField({ id, value, onChange }: {
  id: string
  value: number | null | undefined
  onChange: (v: number | null) => void
}) {
  return (
    <div>
      <label htmlFor={id} className="block text-xs text-brand-text-muted mb-1">
        Max. Kuchen pro Mannschaft <span className="text-brand-text-subtle">(leer = Rotation deaktiviert)</span>
      </label>
      <input
        id={id}
        type="number"
        min={1}
        value={value ?? ''}
        onChange={e => {
          const raw = e.target.value
          onChange(raw === '' ? null : Number(raw))
        }}
        className="w-24 border border-brand-border rounded px-2 py-1.5 text-sm text-brand-text focus:outline-none focus:ring-1 focus:ring-brand-yellow"
      />
      <p className="text-xs text-brand-text-subtle mt-1">
        Setzt voraus, dass „Mehrere Spiele am gleichen Tag" und „Spiele am Vortag / Folgetag" beim Diensttyp auf „Normal (immer)" stehen.
      </p>
    </div>
  )
}
