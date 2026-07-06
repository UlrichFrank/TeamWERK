/**
 * MembersPage — „Neues Mitglied anlegen" sendet join_date im POST-Body.
 * Regression: Das Backend (POST /api/members) verlangt join_date als Pflichtfeld
 * (Beitrags-Halbierung, Migration 014) und antwortet sonst mit HTTP 400. Ohne das
 * Feld im Payload scheiterte die Anlage generisch mit „Anlegen fehlgeschlagen.".
 */
import { describe, test, expect, vi, afterEach } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import MembersPage from '../MembersPage'
import { renderAsPersona } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../lib/usePagination', () => ({
  usePagination: () => ({
    items: [], total: 0, currentPage: 1, totalPages: 1,
    loading: false, error: null,
    setSearch: vi.fn(), goToPage: vi.fn(), refresh: vi.fn(),
  }),
}))

afterEach(() => vi.restoreAllMocks())

describe('MembersPage — Anlegen sendet join_date', () => {
  test('POST /members enthält first_name, last_name und ein nicht-leeres join_date', async () => {
    renderAsPersona(<MembersPage />, 'vorstand')
    const mock = getApiMock()
    mock.onPost('/members').reply(201, { id: 99 })

    fireEvent.click(screen.getByText('+ Neu'))
    fireEvent.change(screen.getByLabelText('Vorname'), { target: { value: 'Max' } })
    fireEvent.change(screen.getByLabelText('Nachname'), { target: { value: 'Muster' } })
    fireEvent.click(screen.getByRole('button', { name: 'Anlegen' }))

    await waitFor(() => expect(mock.history.post.length).toBe(1))
    const body = JSON.parse(mock.history.post[0].data)
    expect(body.first_name).toBe('Max')
    expect(body.last_name).toBe('Muster')
    expect(body.join_date).toBeTruthy()
    expect(body.join_date).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })
})
