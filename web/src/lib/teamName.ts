export const GENDER_LABEL: Record<string, string> = {
  m: 'männlich',
  f: 'weiblich',
  mixed: 'gemischt',
}

const GENDER_SHORT: Record<string, string> = { m: 'm', f: 'w', mixed: 'g' }

export interface TeamForName {
  id: number
  age_class: string
  gender: string
  team_number: number
  group_count: number
}

function ageInitial(ageClass: string): string {
  const m = ageClass.match(/^([A-F])/i)
  return m ? m[1].toUpperCase() : ageClass.charAt(0)
}


/** Short name for calendar tiles: "mA" or "mA1" — clientseitiger Fallback wenn der Server keine Display-Felder liefert. */
export function buildTeamShortNames<T extends TeamForName>(teams: T[]): Map<number, string> {
  const result = new Map<number, string>()
  for (const t of teams) {
    const base = `${GENDER_SHORT[t.gender] ?? t.gender[0]}${ageInitial(t.age_class)}`
    result.set(t.id, t.group_count > 1 ? `${base}${t.team_number}` : base)
  }
  return result
}

export interface TeamDisplay {
  id: number
  display_short?: string
  display_long?: string
  name?: string
}

export type TeamListMode = 'short' | 'long' | 'kalender'

/**
 * Einheitlicher Render-Pfad für Teamnamen.
 *   'short'    → Kurzform aller Teams, komma-getrennt
 *   'long'     → Langform aller Teams, komma-getrennt
 *   'kalender' → Kurzform bei genau einem Team, sonst der String "Mehrere"
 *                (bewusste Platz-Ausnahme nur fürs Kalender-Tile)
 *
 * Fallback-Reihenfolge: display_short/display_long → name → leerer String.
 */
export function formatTeamList(teams: TeamDisplay[], mode: TeamListMode): string {
  if (teams.length === 0) return ''
  if (mode === 'kalender') {
    if (teams.length > 1) return 'Mehrere'
    return teams[0].display_short ?? teams[0].name ?? ''
  }
  const pick = mode === 'short'
    ? (t: TeamDisplay) => t.display_short ?? t.name ?? ''
    : (t: TeamDisplay) => t.display_long ?? t.name ?? ''
  return teams.map(pick).filter(s => s !== '').join(', ')
}
