import { describe, test, expect } from 'vitest'
import { render } from '@testing-library/react'
import StandingsChart from './StandingsChart'
import { gespieltePunkte, verlaufPunkte } from '../../lib/staffeln'
import type { ProgressionDay, ProgressionEntry } from '../../lib/staffeln'

const e = (team: string, rank: number, games: number): ProgressionEntry =>
  ({ team, rank, games, points: 0, goalsFor: 0, goalDiff: 0 })

// A spielt an beiden Tagen, B nur am ersten (am zweiten spielfrei, aber mit Platz).
const DAYS: ProgressionDay[] = [
  { date: '2026-09-20', entries: [e('A', 1, 1), e('B', 2, 1)] },
  { date: '2026-09-27', entries: [e('A', 2, 2), e('B', 1, 1), e('C', 3, 1)] },
]

describe('StandingsChart', () => {
  test('Punkt nur an Spieltagen, an denen die Mannschaft gespielt hat', () => {
    expect(gespieltePunkte(DAYS, 'A')).toEqual([{ day: 0, rank: 1 }, { day: 1, rank: 2 }])
    expect(gespieltePunkte(DAYS, 'B')).toEqual([{ day: 0, rank: 2 }])
    expect(gespieltePunkte(DAYS, 'C')).toEqual([{ day: 1, rank: 3 }])
  })

  test('Platzierung an jedem Datum, gespielt nur wo die Spielzahl steigt', () => {
    expect(verlaufPunkte(DAYS, 'B')).toEqual([
      { day: 0, rank: 2, played: true }, { day: 1, rank: 1, played: false },
    ])
  })

  test('Linie durch alle Tage, Kreis nur an Spieltagen der Mannschaft', () => {
    const { container } = render(<StandingsChart days={DAYS} ownTeams={[]} />)
    const linie = (t: string) => container.querySelector(`g[data-team="${t}"] polyline`)
    const punkte = (t: string) => container.querySelectorAll(`g[data-team="${t}"] circle`)
    expect(linie('A')).not.toBeNull()
    expect(punkte('A')).toHaveLength(2)
    expect(linie('B')).not.toBeNull()
    expect(punkte('B')).toHaveLength(1)
    expect(linie('C')).toBeNull()
    expect(punkte('C')).toHaveLength(1)
  })

  test('jedes Spieldatum steht an der Achse, Abstände fest', () => {
    const { container, getByText } = render(<StandingsChart days={DAYS} ownTeams={[]} />)
    expect(getByText('20.09.')).toBeInTheDocument()
    expect(getByText('27.09.')).toBeInTheDocument()
    const svg = container.querySelector('svg')!
    // 1 Spaltenschritt + Ränder, 2 Zeilenschritte + Ränder.
    expect(svg.getAttribute('width')).toBe(String(28 + 36 + 16))
    expect(svg.getAttribute('height')).toBe(String(12 + 2 * 36 + 30))
  })
})
