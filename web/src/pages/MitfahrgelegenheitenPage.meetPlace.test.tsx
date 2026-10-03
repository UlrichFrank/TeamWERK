import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import MitfahrgelegenheitenPage from './MitfahrgelegenheitenPage'
import { PersonContactProvider } from '../contexts/PersonContactContext'

// spiel-treffpunkt: der Treffpunkt-Ort des Spiels befüllt das Feld „Treffpunkt"
// eines NEUEN Eintrags vor; ein bestehender Eintrag behält seinen eigenen Wert.

const mockGet = vi.fn()

vi.mock('../lib/api', () => ({
  api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() },
}))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, email: 'test@example.com', role: 'standard' } }),
}))

function seed(biete: unknown[], suche: unknown[]) {
  const game = {
    game: { id: 1, date: '2099-09-19', time: '15:00', opponent: 'Ludwigsburg', team: 'Team', teamIds: [1], eventType: 'auswärts', meetPlace: 'Parkplatz Vereinsheim' },
    biete, suche, paarungen: [],
  }
  mockGet.mockImplementation((url?: string) => {
    if (url?.startsWith('/mitfahrgelegenheiten')) return Promise.resolve({ data: { games: [game], children: [] } })
    return Promise.resolve({ data: [] })
  })
}

function renderPage() {
  render(
    <MemoryRouter initialEntries={['/mitfahrten']}>
      <PersonContactProvider><MitfahrgelegenheitenPage /></PersonContactProvider>
    </MemoryRouter>,
  )
}

const treffpunktField = () => screen.getByPlaceholderText('z. B. Halle um 09:00 Uhr') as HTMLInputElement

describe('Mitfahrgelegenheiten — Treffpunkt-Vorbefüllung', () => {
  beforeEach(() => mockGet.mockReset())

  test('neues Angebot übernimmt den Treffpunkt des Spiels', async () => {
    seed([], [])
    renderPage()
    await waitFor(() => expect(document.getElementById('game-1')).toBeTruthy())
    await userEvent.click(screen.getByRole('button', { name: /Ich biete Mitfahrt/ }))
    expect(treffpunktField().value).toBe('Parkplatz Vereinsheim')
  })

  test('bestehender Eintrag behält seinen eigenen Treffpunkt', async () => {
    seed(
      [{ id: 11, userId: 1, userName: 'Ich', plaetze: 3, treffpunkt: 'Bahnhof', notiz: '', isOwn: true }],
      [{ id: 12, userId: 1, userName: 'Ich', plaetze: 1, treffpunkt: 'Bahnhof', notiz: '', isOwn: true }],
    )
    renderPage()
    await waitFor(() => expect(document.getElementById('game-1')).toBeTruthy())
    await userEvent.click(screen.getByRole('button', { name: /Eintrag hinzufügen/ }))
    expect(treffpunktField().value).toBe('Bahnhof')
  })
})
