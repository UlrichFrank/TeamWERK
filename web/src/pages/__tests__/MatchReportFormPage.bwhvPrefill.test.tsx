import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, render, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import { AuthContext, type AuthCtx } from '../../contexts/AuthContext'
import MatchReportFormPage from '../MatchReportFormPage'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../MarkdownRenderer', () => ({ default: () => null }))

let mock: MockAdapter

const draft = (over: Record<string, unknown> = {}) => ({
  id: 42, game_id: 7, duty_slot_id: 3, author_user_id: 1, state: 'draft' as const,
  title: 'Test', home_goals: null, away_goals: null,
  home_goals_ht: null, away_goals_ht: null,
  tournament: false, abstract: '', body_md: '',
  published_url: null, typo3_page_uid: null, error_message: null,
  images: [], photo_consent_missing: null, ...over,
})

const bwhvReport = {
  reportId: 9, state: 'parsed', homeTeam: 'Verein A', guestTeam: 'Verein B',
  homeGoals: 29, guestGoals: 25, homeGoalsHt: 14, guestGoalsHt: 13,
  spectators: 'k.A.', referees: '', warnings: [], hasPdf: true,
  players: [], events: [],
}

const adminCtx: AuthCtx = {
  user: { id: 1, email: 'admin@test.local', role: 'admin', clubFunctions: [], isParent: false },
  loading: false, impersonating: null, mapsProvider: 'auto', setMapsProvider: () => {},
  capabilities: [], hasCapability: () => true, navRoutes: [],
  passwordChangeRecommended: false, dismissPasswordChangeHint: () => {},
  keepAlive: () => {}, login: async () => {}, logout: async () => {},
  startImpersonation: async () => {}, stopImpersonation: async () => {},
}

async function setup() {
  const result = render(
    <AuthContext.Provider value={adminCtx}>
      <MemoryRouter initialEntries={['/spielberichte/42']}>
        <Routes>
          <Route path="/spielberichte/:id" element={<MatchReportFormPage />} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  )
  await waitFor(() => expect(screen.getByText(/Bilder \(/)).toBeInTheDocument())
  return result
}

/** Liefert die vier Ergebnis-Eingabefelder in Dokumentreihenfolge. */
function goalInputs(): HTMLInputElement[] {
  return Array.from(document.querySelectorAll('input[type="number"]')) as HTMLInputElement[]
}

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
})
afterEach(() => {
  mock.restore()
  vi.clearAllMocks()
})

describe('Vorbefüllung aus dem BWHV-Spielbericht', () => {
  test('leere Ergebnisfelder werden aus dem Bericht gefüllt', async () => {
    mock.onGet('/match-reports/42').reply(200, draft())
    mock.onGet('/games/7/bwhv-report').reply(200, bwhvReport)
    await setup()

    await waitFor(() => {
      const values = goalInputs().map(i => i.value)
      expect(values).toContain('29')
      expect(values).toContain('25')
      expect(values).toContain('14')
      expect(values).toContain('13')
    })
  })

  test('ohne Bericht bleiben die Felder leer', async () => {
    mock.onGet('/match-reports/42').reply(200, draft())
    mock.onGet('/games/7/bwhv-report').reply(404, { error: 'not_found' })
    await setup()

    await waitFor(() => expect(goalInputs().length).toBeGreaterThan(0))
    expect(goalInputs().every(i => i.value === '')).toBe(true)
  })

  // Der Autor hat Vorrang: hat er bereits ein Ergebnis eingetragen, wird der
  // Bericht gar nicht erst abgerufen.
  test('bereits eingetragene Werte werden nicht überschrieben', async () => {
    mock.onGet('/match-reports/42').reply(200, draft({ home_goals: 30, away_goals: 20 }))
    let abgerufen = false
    mock.onGet('/games/7/bwhv-report').reply(() => { abgerufen = true; return [200, bwhvReport] })
    await setup()

    await waitFor(() => {
      const values = goalInputs().map(i => i.value)
      expect(values).toContain('30')
      expect(values).toContain('20')
    })
    expect(abgerufen).toBe(false)
  })

  test('eine Änderung des Autors überlebt die Vorbefüllung', async () => {
    mock.onGet('/match-reports/42').reply(200, draft())
    mock.onGet('/games/7/bwhv-report').reply(200, bwhvReport)
    await setup()

    await waitFor(() => expect(goalInputs().map(i => i.value)).toContain('29'))
    const feld = goalInputs().find(i => i.value === '29')!
    fireEvent.change(feld, { target: { value: '31' } })
    expect(feld.value).toBe('31')
  })
})
