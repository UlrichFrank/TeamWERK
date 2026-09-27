import { describe, it, expect } from 'vitest'
import { buildTeamLongName, compareAgeClass, countKaderGroups, type TrainingGroupCategory } from './teamName'

const categories: TrainingGroupCategory[] = [
  { name: 'Perspektivkader', sort_order: 1 },
  { name: 'Förderkader', sort_order: 2 },
]

describe('compareAgeClass', () => {
  it('sorts A–D-Jugend before training groups, then by sort_order (P before F)', () => {
    const input = ['Förderkader', 'C-Jugend', 'Perspektivkader', 'A-Jugend', 'D-Jugend', 'B-Jugend']
    const sorted = [...input].sort((a, b) => compareAgeClass(a, b, categories))
    expect(sorted).toEqual([
      'A-Jugend',
      'B-Jugend',
      'C-Jugend',
      'D-Jugend',
      'Perspektivkader',
      'Förderkader',
    ])
  })

  it('keeps A–D order identical to plain alphabetical when no training groups present', () => {
    const input = ['D-Jugend', 'A-Jugend', 'C-Jugend', 'B-Jugend']
    const sorted = [...input].sort((a, b) => compareAgeClass(a, b, categories))
    expect(sorted).toEqual(['A-Jugend', 'B-Jugend', 'C-Jugend', 'D-Jugend'])
  })

  it('orders training groups purely by sort_order, not alphabetically', () => {
    // Alphabetical would put Förderkader before Perspektivkader; sort_order must win.
    expect(compareAgeClass('Perspektivkader', 'Förderkader', categories)).toBeLessThan(0)
  })

  it('returns 0 for equal age classes', () => {
    expect(compareAgeClass('A-Jugend', 'A-Jugend', categories)).toBe(0)
    expect(compareAgeClass('Förderkader', 'Förderkader', categories)).toBe(0)
  })
})

describe('buildTeamLongName', () => {
  it('hängt bei mehreren Kadern der Kombination die Nummer ans Ende — auch bei Mannschaft 1', () => {
    expect(buildTeamLongName({ age_class: 'C-Jugend', gender: 'm', team_number: 1 }, 2)).toBe('C-Jugend männlich 1')
    expect(buildTeamLongName({ age_class: 'C-Jugend', gender: 'm', team_number: 2 }, 2)).toBe('C-Jugend männlich 2')
  })

  it('lässt die Nummer bei der einzigen Mannschaft der Kombination weg', () => {
    expect(buildTeamLongName({ age_class: 'A-Jugend', gender: 'f', team_number: 1 }, 1)).toBe('A-Jugend weiblich')
  })

  it('beschriftet gemischte Teams', () => {
    expect(buildTeamLongName({ age_class: 'E-Jugend', gender: 'mixed', team_number: 3 }, 3)).toBe('E-Jugend gemischt 3')
  })
})

describe('countKaderGroups', () => {
  it('zählt je Altersklasse und Geschlecht', () => {
    const count = countKaderGroups([
      { age_class: 'C-Jugend', gender: 'm' },
      { age_class: 'C-Jugend', gender: 'm' },
      { age_class: 'C-Jugend', gender: 'f' },
    ])
    expect(count({ age_class: 'C-Jugend', gender: 'm' })).toBe(2)
    expect(count({ age_class: 'C-Jugend', gender: 'f' })).toBe(1)
  })
})
