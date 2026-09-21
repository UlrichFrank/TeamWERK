import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import MitfahrgelegenheitenPage from './MitfahrgelegenheitenPage'
import { PersonContactProvider } from '../contexts/PersonContactContext'

// Regression, gleiche Klasse wie TerminePage.liveReload: die vier
// Mutations-Handler luden mit `load()` statt `load(true)` nach. Das setzt
// `loading` und ersetzt die Liste durch das Skeleton — der Scrollcontainer
// kollabiert, scrollTop wird auf 0 geklemmt, und nach jedem Klick steht man
// wieder oben. jsdom hat kein Layout; prüfbar ist die Ursache: die Liste darf
// während des Nachladens nicht aus dem DOM verschwinden.

const mockGet = vi.fn()
const mockDelete = vi.fn()

vi.mock('../lib/api', () => ({
  api: {
    get: (...args: unknown[]) => mockGet(...args),
    post: vi.fn(),
    delete: (...args: unknown[]) => mockDelete(...args),
  },
}))
vi.mock('../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, email: 'test@example.com', role: 'standard' } }),
}))

const GAME = {
  game: { id: 1, date: '2026-09-19', time: '15:00', opponent: 'Ludwigsburg', team: 'Team', teamIds: [1], eventType: 'auswärts' },
  biete: [{ id: 11, userId: 1, name: 'Ich', plaetze: 3, treffpunkt: '', notiz: '', isOwn: true }],
  suche: [],
  paarungen: [],
}

describe('Mitfahrgelegenheiten — Mutation lädt still nach', () => {
  beforeEach(() => {
    mockGet.mockReset()
    mockDelete.mockReset()
    mockDelete.mockResolvedValue({ data: {} })
    mockGet.mockImplementation((url: string) => {
      if (url.startsWith('/mitfahrgelegenheiten')) return Promise.resolve({ data: { games: [GAME], children: [] } })
      if (url.startsWith('/teams')) return Promise.resolve({ data: [] })
      return Promise.resolve({ data: [] })
    })
  })

  test('Liste bleibt beim Löschen gemountet — kein Skeleton-Platzhalter', async () => {
    const user = userEvent.setup()
    render(
      <MemoryRouter initialEntries={['/mitfahrten']}>
        <PersonContactProvider><MitfahrgelegenheitenPage /></PersonContactProvider>
      </MemoryRouter>,
    )
    await waitFor(() => expect(document.getElementById('game-1')).toBeTruthy())

    // Antwort auf den Reload offen halten — genau dieses Fenster zeigte vorher
    // das Skeleton statt der Liste.
    let resolveReload: (v: unknown) => void = () => {}
    mockGet.mockImplementation((url: string) => {
      if (url.startsWith('/mitfahrgelegenheiten')) return new Promise(res => { resolveReload = res })
      return Promise.resolve({ data: [] })
    })

    await user.click(screen.getAllByLabelText('Eintrag löschen')[0])
    await waitFor(() => expect(mockDelete).toHaveBeenCalled())

    expect(document.getElementById('game-1')).toBeTruthy()

    await act(async () => { resolveReload({ data: { games: [GAME], children: [] } }) })
    expect(document.getElementById('game-1')).toBeTruthy()
  })
})
