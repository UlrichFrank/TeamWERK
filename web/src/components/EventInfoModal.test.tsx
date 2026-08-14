import { describe, test, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import EventInfoModal from './EventInfoModal'

vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@example.com', role: 'standard', clubFunctions: [] },
    hasCapability: () => false,
  }),
}))

function baseGame(overrides: Record<string, unknown> = {}) {
  return {
    id: 42,
    date: '2026-09-05',
    time: '15:00',
    opponent: 'TV Beispiel',
    event_type: 'heim',
    confirmed_count: 0,
    declined_count: 0,
    maybe_count: 0,
    slot_count: 3,
    filled_count: 1,
    total_count: 3,
    ...overrides,
  }
}

// Regression: "In Diensten öffnen" navigierte bisher blind zu /dienste, ohne
// das angeklickte Spiel zu markieren — der Nutzer landete oben in der Liste
// statt beim zugehörigen Dienst-Block (siehe DutyPage.tsx focus=game-<id>).
describe('EventInfoModal — In Diensten öffnen', () => {
  test('navigiert zu /dienste?focus=game-<id>', async () => {
    const user = userEvent.setup()
    const router = createMemoryRouter(
      [
        {
          path: '/kalender',
          element: (
            <EventInfoModal type="game" game={baseGame()} onClose={() => {}} />
          ),
        },
        { path: '/dienste', element: <div data-testid="dienste-page" /> },
      ],
      { initialEntries: ['/kalender'] },
    )
    const locations: string[] = []
    router.subscribe(state => locations.push(state.location.pathname + state.location.search))

    render(<RouterProvider router={router} />)

    const button = await screen.findByRole('button', { name: 'In Diensten öffnen' })
    await user.click(button)

    await screen.findByTestId('dienste-page')
    expect(locations).toContain('/dienste?focus=game-42')
  })

  test('Button ist disabled, wenn das Spiel keine Dienst-Slots hat', async () => {
    const router = createMemoryRouter(
      [
        {
          path: '/kalender',
          element: (
            <EventInfoModal type="game" game={baseGame({ slot_count: 0 })} onClose={() => {}} />
          ),
        },
      ],
      { initialEntries: ['/kalender'] },
    )
    render(<RouterProvider router={router} />)

    const button = await screen.findByRole('button', { name: 'In Diensten öffnen' })
    expect((button as HTMLButtonElement).disabled).toBe(true)
  })
})
