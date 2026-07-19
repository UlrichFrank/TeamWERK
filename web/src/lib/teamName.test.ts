import { describe, it, expect } from 'vitest'
import { compareAgeClass, type TrainingGroupCategory } from './teamName'

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
