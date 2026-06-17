// Persona-Definitionen für Frontend-Permission-Tests.
// Spiegelbildlich zu internal/permissions/personas_test.go —
// bei Änderungen beide Dateien aktualisieren.
// Quelle der Wahrheit: openspec/changes/permissions-baseline-tests/specs/permissions/spec.md §1

export type Persona = {
  id: string
  label: string
  role: 'admin' | 'standard'
  clubFunctions: string[]
  isParent: boolean
}

export const PERSONAS: Persona[] = [
  { id: 'admin', label: 'Admin', role: 'admin', clubFunctions: [], isParent: false },
  { id: 'vorstand', label: 'Vorstand', role: 'standard', clubFunctions: ['vorstand'], isParent: false },
  { id: 'vorstand_elternteil', label: 'Vorstand-Elternteil', role: 'standard', clubFunctions: ['vorstand'], isParent: true },
  { id: 'vorstand_beisitzer', label: 'Vorstand Beisitzer', role: 'standard', clubFunctions: ['vorstand_beisitzer'], isParent: false },
  { id: 'kassierer', label: 'Kassierer', role: 'standard', clubFunctions: ['kassierer'], isParent: false },
  { id: 'trainer', label: 'Trainer', role: 'standard', clubFunctions: ['trainer'], isParent: false },
  { id: 'trainer_elternteil', label: 'Trainer-Elternteil', role: 'standard', clubFunctions: ['trainer'], isParent: true },
  { id: 'sportliche_leitung', label: 'Sportliche Leitung', role: 'standard', clubFunctions: ['sportliche_leitung'], isParent: false },
  { id: 'sportliche_leitung_elternteil', label: 'Sportliche Leitung-Elternteil', role: 'standard', clubFunctions: ['sportliche_leitung'], isParent: true },
  { id: 'spieler', label: 'Spieler', role: 'standard', clubFunctions: ['spieler'], isParent: false },
  { id: 'elternteil', label: 'Elternteil', role: 'standard', clubFunctions: [], isParent: true },
]

export function personaById(id: string): Persona {
  const p = PERSONAS.find(p => p.id === id)
  if (!p) throw new Error(`Unknown persona: ${id}`)
  return p
}
