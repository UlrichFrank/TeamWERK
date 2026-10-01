/**
 * TermineDetailPage: ein Spieler beantwortet einen Termin über das Drei-Punkte-Menü
 * seiner eigenen Zeile. Fremde Zeilen bekommen kein Menü (das bleibt Trainern
 * vorbehalten), vergangene Termine ebenfalls nicht; Sperren meldet der Server.
 * Die eigene Zeile erkennt die Seite an `user_id` (renderAsPersona: User-ID 1).
 */
import { describe, test, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent, waitFor, within } from '@testing-library/react'
import { Routes, Route } from 'react-router-dom'
import TermineDetailPage from '../TermineDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { api } from '../../lib/api'
import { AxiosError, type AxiosResponse } from 'axios'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const FUTURE = '2099-06-17'

function session(overrides: Record<string, unknown> = {}) {
  return {
    id: 710,
    date: FUTURE,
    start_time: '18:00',
    end_time: '19:30',
    status: 'active',
    team_id: 1,
    team_name: 'Test Team',
    confirmed_count: 1,
    declined_count: 0,
    maybe_count: 0,
    my_rsvp: 'confirmed',
    note: '',
    cancel_reason: '',
    rsvp_default_players: 'none',
    rsvp_default_extended: 'none',
    rsvp_require_reason: 0,
    ...overrides,
  }
}

const ATTENDANCES = [
  { member_id: 11, member_name: 'Ich Selbst', user_id: 1, present: null, rsvp_status: 'confirmed', reason: null },
  { member_id: 12, member_name: 'Andere Spielerin', user_id: 2, present: null, rsvp_status: null, reason: null },
]

function renderPage(sessionOverrides: Record<string, unknown> = {}) {
  const mocks = [
    { url: /training-sessions\/710$/, data: session(sessionOverrides) },
    { url: /training-sessions\/710\/attendances/, data: ATTENDANCES },
  ]
  renderAsPersona(
    <Routes>
      <Route path="/termine/:type/:id" element={<TermineDetailPage />} />
    </Routes>,
    'spieler',
    { initialEntries: ['/termine/training/710'], mocks },
  )
}

function rowOf(name: string) {
  return screen.getByText(name).closest('tr')!
}

describe('TermineDetailPage — Spieler antwortet über das Zeilenmenü', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  // Der Catch-all des API-Mocks antwortet auf jeden POST mit 200; den Respond-
  // Aufruf fangen wir deshalb direkt an api.post ab.
  function capturePosts(error?: { status: number; data: unknown }) {
    return vi.spyOn(api, 'post').mockImplementation(async () => {
      if (error) {
        throw new AxiosError('fail', String(error.status), undefined, undefined,
          { status: error.status, data: error.data } as AxiosResponse)
      }
      return { status: 204, data: undefined } as AxiosResponse
    })
  }

  test('nur die eigene Zeile hat ein Menü', async () => {
    renderPage()
    await screen.findByText('Ich Selbst')
    await flushAsync()

    expect(within(rowOf('Ich Selbst')).getByRole('button', { name: 'Aktionen' })).toBeTruthy()
    expect(within(rowOf('Andere Spielerin')).queryByRole('button', { name: 'Aktionen' })).toBeNull()
  })

  test('Absagen schickt die Antwort für das eigene Mitglied mit Grund', async () => {
    renderPage()
    await screen.findByText('Ich Selbst')
    await flushAsync()
    const post = capturePosts()

    fireEvent.click(within(rowOf('Ich Selbst')).getByRole('button', { name: 'Aktionen' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Absagen' }))
    fireEvent.change(await screen.findByPlaceholderText('z.B. Krank, Urlaub…'), { target: { value: 'krank' } })
    fireEvent.click(screen.getByRole('button', { name: 'Absagen' }))

    await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
    expect(post).toHaveBeenCalledWith('/training-sessions/710/respond', { member_id: 11, status: 'declined', reason: 'krank' })
  })

  test('Pflichtgrund: Vielleicht öffnet das Modal und sendet ohne Grund nicht', async () => {
    renderPage({ rsvp_require_reason: 1 })
    await screen.findByText('Ich Selbst')
    await flushAsync()
    const post = capturePosts()

    fireEvent.click(within(rowOf('Ich Selbst')).getByRole('button', { name: 'Aktionen' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Vielleicht' }))

    const confirm = await screen.findByRole('button', { name: 'Vielleicht' })
    expect((confirm as HTMLButtonElement).disabled).toBe(true)

    fireEvent.change(screen.getByPlaceholderText('z.B. Krank, Urlaub…'), { target: { value: 'Prüfung' } })
    fireEvent.click(confirm)
    await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
    expect(post).toHaveBeenCalledWith('/training-sessions/710/respond', { member_id: 11, status: 'maybe', reason: 'Prüfung' })
  })

  test('Server-Sperre wird mit ihrer Meldung angezeigt', async () => {
    renderPage()
    await screen.findByText('Ich Selbst')
    await flushAsync()
    capturePosts({
      status: 422,
      data: { error: 'rsvp_locked', message: 'Training kann nur bis 2 Stunden vor Beginn umgesagt werden.' },
    })

    fireEvent.click(within(rowOf('Ich Selbst')).getByRole('button', { name: 'Aktionen' }))
    fireEvent.click(await screen.findByRole('menuitem', { name: 'Zusagen' }))

    expect(await screen.findByRole('alert')).toHaveProperty(
      'textContent',
      'Training kann nur bis 2 Stunden vor Beginn umgesagt werden.',
    )
  })

  test('vergangener Termin: kein Menü für den Spieler', async () => {
    renderPage({ date: '2020-06-17' })
    await screen.findByText('Ich Selbst')
    await flushAsync()

    expect(screen.queryByRole('button', { name: 'Aktionen' })).toBeNull()
  })
})
