import { describe, test, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SpielberichtPanel from './SpielberichtPanel'
import type { EventLine, ReportDetail } from '../lib/staffeln'

function tor(seq: number, side: 'home' | 'guest', sec: number, h: number, g: number): EventLine {
  return {
    seq, clockTime: '', gameSecond: sec, scoreHome: h, scoreGuest: g,
    kind: 'goal', side, playerId: null, playerName: 'Anna', number: null, rawText: '',
  }
}

const REPORT: ReportDetail = {
  reportId: 1, state: 'parsed', homeTeam: 'Team Stuttgart 2', guestTeam: 'Fremd',
  homeGoals: 2, guestGoals: 1, homeGoalsHt: 1, guestGoalsHt: 1,
  spectators: '', referees: '', refereeNames: [], refereesUncertain: false,
  warnings: [], hasPdf: false, players: [],
  events: [tor(1, 'home', 100, 1, 0), tor(2, 'guest', 900, 1, 1), tor(3, 'home', 2000, 2, 1)],
}

vi.mock('../lib/staffeln', async (orig) => ({
  ...(await orig<typeof import('../lib/staffeln')>()),
  fetchReport: vi.fn(() => Promise.resolve(REPORT)),
}))

describe('SpielberichtPanel', () => {
  test('Spielverlauf ist zwischen Kurve und Ablauf umschaltbar', async () => {
    render(<SpielberichtPanel bwhvGameId={1} halfDurationMinutes={25} />)
    const kurve = await screen.findByRole('button', { name: /Kurve/ })
    expect(kurve).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('img', { name: /Torkurve/ })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: /Ablauf/ }))
    expect(screen.queryByRole('img', { name: /Torkurve/ })).not.toBeInTheDocument()
    expect(screen.getAllByRole('img', { name: /Tore je Minute/ })).toHaveLength(2)
  })
})
