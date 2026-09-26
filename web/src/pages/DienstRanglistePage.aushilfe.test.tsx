import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import DienstRanglistePage from './DienstRanglistePage'

// openspec/changes/dienste-erweiterter-kader: Aushilfen stehen je Block in einem
// eigenen, ungerankten Abschnitt — nur, wenn es welche gibt.

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function renderAt(route: string) {
  const router = createMemoryRouter(
    [{ path: '/dienste/rangliste', element: <DienstRanglistePage /> }],
    { initialEntries: [route] },
  )
  render(<RouterProvider router={router} />)
}

function seed(aushilfen: unknown[] | undefined) {
  mockGet.mockResolvedValue({
    data: {
      teams: [{ id: 9, label: 'mB1' }],
      blocks: [{
        teamId: 9,
        teamLabel: 'mB1',
        soll: 5,
        rows: [{ rank: 1, memberId: null, name: null, isOwn: false, geleistet: 6, vorhersage: 1 }],
        aushilfen,
      }],
    },
  })
}

beforeEach(() => {
  mockGet.mockReset()
})

describe('DienstRanglistePage — Aushilfen', () => {
  test('Abschnitt zeigt eigene Aushilfe benannt, fremde als Strich, ohne Platz', async () => {
    seed([
      { memberId: 11, name: 'Lena Beispiel', isOwn: true, geleistet: 2, vorhersage: 1 },
      { memberId: null, name: null, isOwn: false, geleistet: 1, vorhersage: 0 },
    ])
    renderAt('/dienste/rangliste?team=9')

    const section = await screen.findByTestId('rangliste-aushilfen')
    expect(section).toHaveTextContent('Erw. Kader')
    expect(within(section).getByText('Lena Beispiel')).toBeInTheDocument()
    expect(within(section).getByText('-')).toBeInTheDocument()
    expect(section).toHaveTextContent('2 geleistet · 1 eingetragen')
    expect(section).not.toHaveTextContent('1.')
  })

  test('ohne Aushilfen kein Abschnitt', async () => {
    seed([])
    renderAt('/dienste/rangliste?team=9')
    await screen.findByText('mB1', { selector: 'h2' })
    expect(screen.queryByTestId('rangliste-aushilfen')).not.toBeInTheDocument()
  })
})
