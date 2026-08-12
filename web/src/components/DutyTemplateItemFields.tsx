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
 * Bewirtungsrotations-Schalter eines Vorlagen-Eintrags (kuchendienst-rotation).
 * Aus (Default) = bisheriges Verhalten, ein Slot pro Team des Spiels. An =
 * tagesweite Team-Warteschlange über alle Heimspiele des Spieltags.
 *
 * Die Obergrenze pro Mannschaft steht bewusst NICHT hier, sondern vereinsweit
 * unter Einstellungen → Bewirtung (bewirtung-cap-global): sie ist eine
 * Vereinsregel, keine Eigenschaft einer einzelnen Vorlage — zwei Vorlagen mit
 * abweichenden Obergrenzen für denselben Diensttyp wären nicht auflösbar.
 *
 * Die Voraussetzung „Normal (immer)" beim Diensttyp prüft nur der Server
 * (400 rotation_requires_normal_behavior), kein Client-Vorab-Check.
 */
export function RotationEnabledField({ id, value, onChange }: {
  id: string
  value: boolean | undefined
  onChange: (v: boolean) => void
}) {
  return (
    <div>
      <label htmlFor={id} className="flex items-center gap-2 text-xs text-brand-text-muted cursor-pointer">
        <input
          id={id}
          type="checkbox"
          checked={value ?? false}
          onChange={e => onChange(e.target.checked)}
          className="accent-brand-yellow"
        />
        Bewirtungsrotation über den ganzen Spieltag
      </label>
      <p className="text-xs text-brand-text-subtle mt-1">
        Max. Kuchen pro Mannschaft wird unter Einstellungen → Bewirtung gepflegt. Setzt voraus,
        dass „Mehrere Spiele am gleichen Tag" und „Spiele am Vortag / Folgetag" beim Diensttyp
        auf „Normal (immer)" stehen.
      </p>
    </div>
  )
}
