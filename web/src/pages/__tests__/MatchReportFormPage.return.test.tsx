import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, screen, waitFor, render, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import { AuthContext, type AuthCtx } from '../../contexts/AuthContext'
import MatchReportFormPage from '../MatchReportFormPage'

// Rückgabe eines eingereichten Berichts mit Kommentar (POST /match-reports/{id}/return).

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../MarkdownRenderer', () => ({ default: () => null }))

let mock: MockAdapter

const report = (over: Record<string, unknown> = {}) => ({
  id: 42, game_id: 7, duty_slot_id: 3, author_user_id: 1, state: 'pending_review' as const,
  title: 'Test', home_goals: null, away_goals: null,
  home_goals_ht: null, away_goals_ht: null,
  tournament: false, abstract: '', body_md: '',
  published_url: null, typo3_page_uid: null, error_message: null,
  review_comment: null, returned_at: null,
  images: [], photo_consent_missing: null, ...over,
})

function ctx(userId: number, clubFunctions: string[]): AuthCtx {
  return {
    user: { id: userId, email: 'x@test.local', role: 'standard', clubFunctions, isParent: false },
    loading: false, impersonating: null, mapsProvider: 'auto', setMapsProvider: () => {},
    capabilities: [], hasCapability: () => false, navRoutes: [],
    passwordChangeRecommended: false, dismissPasswordChangeHint: () => {},
    keepAlive: () => {}, login: async () => {}, logout: async () => {},
    startImpersonation: async () => {}, stopImpersonation: async () => {},
  }
}

async function setup(auth: AuthCtx) {
  render(
    <AuthContext.Provider value={auth}>
      <MemoryRouter initialEntries={['/spielberichte/42']}>
        <Routes>
          <Route path="/spielberichte/:id" element={<MatchReportFormPage />} />
          <Route path="/spielberichte/pruefen" element={<p>Prüfliste</p>} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  )
  await waitFor(() => expect(screen.getByText(/Bilder \(/)).toBeInTheDocument())
}

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  mock.onGet('/games/7/bwhv-report').reply(404)
})
afterEach(() => {
  mock.restore()
  vi.clearAllMocks()
})

describe('Spielbericht zurückgeben', () => {
  test('Freigeber gibt mit Kommentar zurück — erst speichern, dann POST /return', async () => {
    mock.onGet('/match-reports/42').reply(200, report())
    mock.onPut('/match-reports/42').reply(200)
    mock.onPost('/match-reports/42/return').reply(200, { state: 'draft' })
    await setup(ctx(9, ['medien']))
    const user = userEvent.setup()

    await user.click(screen.getByRole('button', { name: /Zurückgeben/ }))
    const dialog = screen.getByRole('dialog')
    const confirm = within(dialog).getByRole('button', { name: 'Zurückgeben' }) as HTMLButtonElement
    expect(confirm.disabled).toBe(true) // ohne Kommentar keine Rückgabe

    await user.type(within(dialog).getByLabelText('Kommentar für den Autor'), 'Halbzeitstand fehlt')
    await user.click(confirm)

    await waitFor(() => expect(mock.history.post.some(r => r.url === '/match-reports/42/return')).toBe(true))
    const post = mock.history.post.find(r => r.url === '/match-reports/42/return')!
    expect(JSON.parse(post.data)).toEqual({ comment: 'Halbzeitstand fehlt' })
    expect(mock.history.put.length).toBe(1)
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    // Danach ist der Bericht ein draft, den der Freigeber nicht mehr lesen darf:
    // kein erneutes GET (→ 403 „forbidden"), sondern zurück zur Prüfliste.
    await waitFor(() => expect(screen.getByText('Prüfliste')).toBeInTheDocument())
    expect(mock.history.get.filter(r => r.url === '/match-reports/42').length).toBe(1)
    expect(screen.queryByText('forbidden')).toBeNull()
  })

  test('Live-Update nach fremder Rückgabe führt zur Prüfliste statt zu „forbidden"', async () => {
    const { useLiveUpdates } = await import('../../hooks/useLiveUpdates')
    mock.onGet('/match-reports/42').replyOnce(200, report())
    mock.onGet('/match-reports/42').reply(403, { error: 'forbidden' })
    await setup(ctx(9, ['medien']))

    const calls = vi.mocked(useLiveUpdates).mock.calls
    const onEvent = calls[calls.length - 1][0] as (evt: string) => void
    act(() => onEvent('match-report-event'))

    await waitFor(() => expect(screen.getByText('Prüfliste')).toBeInTheDocument())
    expect(screen.queryByText('forbidden')).toBeNull()
  })

  test('Autor sieht im zurückgegebenen Entwurf den Kommentar und darf wieder bearbeiten', async () => {
    mock.onGet('/match-reports/42').reply(200, report({ state: 'draft', review_comment: 'Bitte Titel kürzen' }))
    await setup(ctx(1, []))

    expect(screen.getByText('Bitte Titel kürzen')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Zur Prüfung senden/ })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Zurückgeben/ })).toBeNull()
  })

  test('Autor ohne Freigeber-Funktion bekommt im eingereichten Bericht keinen Zurückgeben-Knopf', async () => {
    mock.onGet('/match-reports/42').reply(200, report())
    await setup(ctx(1, []))
    expect(screen.queryByRole('button', { name: /Zurückgeben/ })).toBeNull()
  })
})
