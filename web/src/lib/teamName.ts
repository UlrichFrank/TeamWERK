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


/** Short name for calendar tiles: "mA" or "mA1" */
export function buildTeamShortNames<T extends TeamForName>(teams: T[]): Map<number, string> {
  const result = new Map<number, string>()
  for (const t of teams) {
    const base = `${GENDER_SHORT[t.gender] ?? t.gender[0]}${ageInitial(t.age_class)}`
    result.set(t.id, t.group_count > 1 ? `${base}${t.team_number}` : base)
  }
  return result
}
