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
}

function ageInitial(ageClass: string): string {
  const m = ageClass.match(/^([A-F])/i)
  return m ? m[1].toUpperCase() : ageClass.charAt(0)
}

function groupByAgeGender<T extends TeamForName>(teams: T[]): Map<string, T[]> {
  const groups = new Map<string, T[]>()
  for (const t of teams) {
    const key = `${t.age_class}|${t.gender}`
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(t)
  }
  return groups
}

/** Full display name: "A-Jugend männlich" or "A-Jugend männlich 1" */
export function buildTeamDisplayNames<T extends TeamForName>(teams: T[]): Map<number, string> {
  const result = new Map<number, string>()
  for (const [, group] of groupByAgeGender(teams)) {
    const multi = group.length > 1
    for (const t of group) {
      const base = `${t.age_class} ${GENDER_LABEL[t.gender] ?? t.gender}`
      result.set(t.id, multi ? `${base} ${t.team_number}` : base)
    }
  }
  return result
}

/** Short name for calendar tiles: "mA" or "mA1" */
export function buildTeamShortNames<T extends TeamForName>(teams: T[]): Map<number, string> {
  const result = new Map<number, string>()
  for (const [, group] of groupByAgeGender(teams)) {
    const multi = group.length > 1
    for (const t of group) {
      const base = `${GENDER_SHORT[t.gender] ?? t.gender[0]}${ageInitial(t.age_class)}`
      result.set(t.id, multi ? `${base}${t.team_number}` : base)
    }
  }
  return result
}
