import { describe, test, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SpielberichtPanel from './SpielberichtPanel'
import { fetchReport, downloadReportPDF } from '../lib/staffeln'
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
  downloadReportPDF: vi.fn(),
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

  // Regression: ein <a href> auf die PDF-Route schickt keinen Bearer-Token mit
  // und lief in Prod in 401. Der Download muss über api (downloadReportPDF).
  test('PDF wird über den angemeldeten API-Client geladen, nicht per Link', async () => {
    vi.mocked(fetchReport).mockResolvedValueOnce({ ...REPORT, reportId: 12, hasPdf: true })
    vi.mocked(downloadReportPDF).mockResolvedValueOnce()
    render(<SpielberichtPanel bwhvGameId={1} />)
    const btn = await screen.findByRole('button', { name: /Bericht als PDF/ })
    expect(screen.queryByRole('link', { name: /Bericht als PDF/ })).not.toBeInTheDocument()
    await userEvent.click(btn)
    expect(downloadReportPDF).toHaveBeenCalledWith(12)
  })

  test('Fehlschlag beim PDF-Download wird angezeigt', async () => {
    vi.mocked(fetchReport).mockResolvedValueOnce({ ...REPORT, hasPdf: true })
    vi.mocked(downloadReportPDF).mockRejectedValueOnce(new Error('401'))
    render(<SpielberichtPanel bwhvGameId={1} />)
    await userEvent.click(await screen.findByRole('button', { name: /Bericht als PDF/ }))
    expect(await screen.findByText(/PDF konnte nicht geladen werden/)).toBeInTheDocument()
  })
})
