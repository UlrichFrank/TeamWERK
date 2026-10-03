import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, waitFor, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import MitfahrgelegenheitenPage from './MitfahrgelegenheitenPage'
import { PersonContactProvider } from '../contexts/PersonContactContext'

// Regression, gleiche Klasse wie DutyPage.focus/TerminePage.liveReload: der
// Fokus-Sprung (`#biete-<id>` usw.) hing an `focusGameId`. Rutschte der Eintrag
// bei einem stillen Reload in eine andere Begegnung, sprang die Seite erneut
// zum Fokus — weg von der Stelle, an der der Nutzer gerade arbeitete.

const mockGet = vi.fn()
let liveHandler: ((event: string) => void) | null = null

vi.mock('../lib/api', () => ({
  api: { get: (...args: unknown[]) => mockGet(...args), post: vi.fn(), delete: vi.fn() },
}))
vi.mock('../hooks/useLiveUpdates', () => ({
  useLiveUpdates: (cb: (event: string) => void) => { liveHandler = cb },
}))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, email: 'test@example.com', role: 'standard' } }),
}))

const BIETE = { id: 11, userId: 1, name: 'Ich', plaetze: 3, treffpunkt: '', notiz: '', isOwn: true }

function game(id: number, biete: unknown[]) {
  return {
    game: { id, date: `2026-09-${10 + id}`, time: '15:00', opponent: `Gegner ${id}`, team: 'Team', teamIds: [1], eventType: 'auswärts' },
    biete,
    suche: [],
    paarungen: [],
  }
}

function serve(games: unknown[]) {
  mockGet.mockImplementation((url: string) => {
    if (url.startsWith('/mitfahrgelegenheiten')) return Promise.resolve({ data: { games, children: [] } })
    return Promise.resolve({ data: [] })
  })
}

describe('Mitfahrgelegenheiten — Fokus-Sprung nur einmal je Ziel', () => {
  beforeEach(() => {
    mockGet.mockReset()
    liveHandler = null
  })

  test('springt zum Fokus und nach einem Reload nicht erneut', async () => {
    const scroll = vi.fn()
    Element.prototype.scrollIntoView = scroll
    serve([game(1, [BIETE]), game(2, [])])
    render(
      <MemoryRouter initialEntries={['/mitfahrten#biete-11']}>
        <PersonContactProvider><MitfahrgelegenheitenPage /></PersonContactProvider>
      </MemoryRouter>,
    )
    await waitFor(() => expect(scroll).toHaveBeenCalledTimes(1))

    // Derselbe Eintrag hängt jetzt an einer anderen Begegnung → focusGameId wechselt.
    serve([game(1, []), game(2, [BIETE])])
    await act(async () => { liveHandler?.('mitfahrgelegenheiten') })
    await waitFor(() => expect(document.getElementById('biete-11')).toBeTruthy())

    expect(scroll).toHaveBeenCalledTimes(1)
  })
})
