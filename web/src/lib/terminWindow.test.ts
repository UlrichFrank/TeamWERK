import { describe, it, expect } from 'vitest'
import { terminLoadWindow } from './terminWindow'

// Fester Bezugspunkt, damit die Erwartungen Datumsliterale sein können.
const NOW = new Date('2026-09-19T10:00:00Z')

describe('terminLoadWindow', () => {
  it('lädt Termine jenseits des Saisonendes', () => {
    // Genau der Prod-Fall, der die Übungsgruppen-Termine verschluckt hat: die
    // Serie „Förderkinder" läuft bis 03.07.2027, die Saison endet am 30.06.2027.
    const { to } = terminLoadWindow({ start_date: '2026-05-01', end_date: '2027-06-30' }, false, NOW)
    expect(to).toBe('2027-09-19')
    expect(to >= '2027-07-03').toBe(true)
  })

  it('reicht bis zum Saisonende, wenn die Saison länger läuft als das rollierende Jahr', () => {
    const { to } = terminLoadWindow({ start_date: '2026-05-01', end_date: '2028-01-31' }, false, NOW)
    expect(to).toBe('2028-01-31')
  })

  it('beginnt ohne "Vergangene" bei heute', () => {
    const { from } = terminLoadWindow({ start_date: '2026-05-01', end_date: '2027-06-30' }, false, NOW)
    expect(from).toBe('2026-09-19')
  })

  it('reicht mit "Vergangene" mindestens ein Jahr zurück', () => {
    const { from } = terminLoadWindow({ start_date: '2026-05-01', end_date: '2027-06-30' }, true, NOW)
    expect(from).toBe('2025-09-19')
  })

  it('reicht mit "Vergangene" bis zum Saisonstart, wenn der vor dem Jahresrückblick liegt', () => {
    const { from } = terminLoadWindow({ start_date: '2025-03-01', end_date: '2027-06-30' }, true, NOW)
    expect(from).toBe('2025-03-01')
  })
})
