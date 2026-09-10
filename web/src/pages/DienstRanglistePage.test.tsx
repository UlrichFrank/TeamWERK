import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import DienstRanglistePage from './DienstRanglistePage'

// Teil von openspec/changes/dienste-familien-rangliste: GET
// /api/duty-fairness/rangliste liefert bereits sortierte, rollenabhängig
// maskierte Blöcke — die Seite rendert nur, was ankommt (kein eigenes
// Masking/Sortieren im Frontend). Mocking-Stil wie DutyPage.teamfilter.test.tsx.

const mockGet = vi.fn()
vi.mock('../lib/api', () => ({ api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() } }))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function renderAt(route: string) {
  const router = createMemoryRouter(
    [{ path: '/dienste/rangliste', element: <DienstRanglistePage /> }],
    { initialEntries: [route] },
  )
  render(<RouterProvider router={router} />)
  return router
}

beforeEach(() => {
  mockGet.mockReset()
})

describe('DienstRanglistePage', () => {
  test('Mehrfachauswahl (team=1,2) zeigt zwei getrennte Blöcke, jeweils eigene Zeilenreihenfolge; Request enthält team=1,2', async () => {
    mockGet.mockImplementation((_url: string) => {
      return Promise.resolve({
        data: {
          teams: [{ id: 1, label: 'mA2' }, { id: 2, label: 'wC1' }],
          blocks: [
            {
              teamId: 1,
              teamLabel: 'mA2',
              soll: 4,
              rows: [
                { rank: 1, memberId: 101, name: 'Anna Beispiel', isOwn: true, geleistet: 5, vorhersage: 1 },
                { rank: 2, memberId: null, name: null, isOwn: false, geleistet: 2, vorhersage: 0 },
              ],
            },
            {
              teamId: 2,
              teamLabel: 'wC1',
              soll: 3,
              rows: [
                { rank: 1, memberId: null, name: null, isOwn: false, geleistet: 3, vorhersage: 0 },
                { rank: 2, memberId: 202, name: 'Ben Beispiel', isOwn: false, geleistet: 1, vorhersage: 0 },
              ],
            },
          ],
        },
      })
    })

    renderAt('/dienste/rangliste?team=1,2')

    await screen.findByText('mA2')
    expect(screen.getByText('wC1')).toBeInTheDocument()
    expect(screen.getByText('Anna Beispiel')).toBeInTheDocument()
    expect(screen.getByText('Ben Beispiel')).toBeInTheDocument()
    // Zwei anonymisierte Zeilen (nur „-" statt Name), eine pro Block.
    expect(screen.getAllByText('-')).toHaveLength(2)

    expect(mockGet).toHaveBeenCalled()
    const calledUrl = mockGet.mock.calls[0][0] as string
    expect(calledUrl).toContain('team=1%2C2')
  })

  test('Standard-Nutzer: nur eigene Zeile benannt, andere nur mit "-"', async () => {
    mockGet.mockResolvedValue({
      data: {
        teams: [{ id: 5, label: 'mB2' }],
        blocks: [
          {
            teamId: 5,
            teamLabel: 'mB2',
            soll: 4,
            rows: [
              { rank: 1, memberId: null, name: null, isOwn: false, geleistet: 6, vorhersage: 0 },
              { rank: 2, memberId: 55, name: 'Eigenes Kind', isOwn: true, geleistet: 3, vorhersage: 1 },
              { rank: 3, memberId: null, name: null, isOwn: false, geleistet: 1, vorhersage: 0 },
            ],
          },
        ],
      },
    })

    renderAt('/dienste/rangliste')

    await screen.findByText('Eigenes Kind')
    expect(screen.getAllByText('-')).toHaveLength(2)
    expect(screen.queryByText(/Platz/)).toBeNull()
  })

  test('Vorstand: alle Zeilen benannt', async () => {
    mockGet.mockResolvedValue({
      data: {
        teams: [{ id: 7, label: 'wA1' }],
        blocks: [
          {
            teamId: 7,
            teamLabel: 'wA1',
            soll: 5,
            rows: [
              { rank: 1, memberId: 1, name: 'Familie Meier', isOwn: false, geleistet: 6, vorhersage: 0 },
              { rank: 2, memberId: 2, name: 'Familie Schmidt', isOwn: false, geleistet: 4, vorhersage: 0 },
              { rank: 3, memberId: 3, name: 'Familie Weber', isOwn: false, geleistet: 2, vorhersage: 0 },
            ],
          },
        ],
      },
    })

    renderAt('/dienste/rangliste')

    await screen.findByText('Familie Meier')
    expect(screen.getByText('Familie Schmidt')).toBeInTheDocument()
    expect(screen.getByText('Familie Weber')).toBeInTheDocument()
    expect(screen.queryByText('-')).toBeNull()
  })

  test('teams: [] zeigt Leerzustand', async () => {
    mockGet.mockResolvedValue({ data: { teams: [], blocks: [] } })

    renderAt('/dienste/rangliste')

    await waitFor(() => expect(screen.getByText(/keine Mannschaft/i)).toBeInTheDocument())
  })

  test('403 zeigt Fehler-Alert', async () => {
    mockGet.mockRejectedValue({ response: { status: 403 } })

    renderAt('/dienste/rangliste?team=99')

    await waitFor(() => expect(screen.getByText('Kein Zugriff auf diese Mannschaft.')).toBeInTheDocument())
  })
})
