/**
 * ProfileDatenschutzTab — Toggle „Sichtbarkeit für Mitglieder" + read-only DSGVO.
 * Quelle: openspec/changes/profile-cross-team-visibility/specs/profile-datenschutz-tab/spec.md
 */
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, act } from '@testing-library/react'
import ProfileDatenschutzTab from '../ProfileDatenschutzTab'
import { setupApiMock, getApiMock } from '../../../test/apiMock'
import type { Member } from '../../../pages/ProfilePage'

beforeEach(() => {
  setupApiMock()
})

const baseMember: Member = {
  id: 42,
  first_name: 'Anna',
  last_name: 'Schmidt',
  date_of_birth: '',
  pass_number: '',
  position: '',
  status: 'aktiv',
  cross_team_visible: false,
  dsgvo_verarbeitung: true,
  dsgvo_verarbeitung_date: '2024-03-12T00:00:00Z',
  dsgvo_weitergabe: false,
  foto_veroeffentlichung: true,
  foto_veroeffentlichung_date: '2026-07-05T00:00:00Z',
}

describe('ProfileDatenschutzTab', () => {
  test('zeigt den aktuellen Toggle-Wert für cross_team_visible', () => {
    render(<ProfileDatenschutzTab ownMember={{ ...baseMember, cross_team_visible: true }} onUpdated={() => {}} />)
    const toggle = screen.getByRole('button', { name: /Sichtbarkeit für Mitglieder/i })
    // Aktivierter Toggle hat bg-brand-yellow.
    expect(toggle.className).toMatch(/bg-brand-yellow/)
  })

  test('zeigt den Toggle als deaktiviert, wenn cross_team_visible=false', () => {
    render(<ProfileDatenschutzTab ownMember={baseMember} onUpdated={() => {}} />)
    const toggle = screen.getByRole('button', { name: /Sichtbarkeit für Mitglieder/i })
    expect(toggle.className).toMatch(/bg-brand-border/)
  })

  test('Toggle-Klick sendet PUT /members/{id}/cross-team-visible mit true', async () => {
    const mock = getApiMock()
    mock.reset()
    mock.onPut(/\/members\/42\/cross-team-visible/).reply(204)
    mock.onAny().reply(200, [])
    const onUpdated = vi.fn()

    render(<ProfileDatenschutzTab ownMember={baseMember} onUpdated={onUpdated} />)
    const toggle = screen.getByRole('button', { name: /Sichtbarkeit für Mitglieder/i })

    await act(async () => {
      fireEvent.click(toggle)
      await new Promise(r => setTimeout(r, 0))
    })

    expect(mock.history.put).toHaveLength(1)
    const sent = mock.history.put[0]
    expect(sent.url).toMatch(/\/members\/42\/cross-team-visible/)
    expect(JSON.parse(sent.data)).toEqual({ cross_team_visible: true })
    expect(onUpdated).toHaveBeenCalled()
  })

  test('DSGVO-Status wird read-only angezeigt', () => {
    render(<ProfileDatenschutzTab ownMember={baseMember} onUpdated={() => {}} />)
    const verarb = screen.getByLabelText('Datenverarbeitung eingewilligt') as HTMLInputElement
    expect(verarb.checked).toBe(true)
    expect(verarb.disabled).toBe(true)

    const weiter = screen.getByLabelText('Datenweitergabe eingewilligt') as HTMLInputElement
    expect(weiter.checked).toBe(false)
    expect(weiter.disabled).toBe(true)

    const foto = screen.getByLabelText('Foto-Veröffentlichung eingewilligt') as HTMLInputElement
    expect(foto.checked).toBe(true)
    expect(foto.disabled).toBe(true)

    // Datum sichtbar (slice auf 10 Zeichen)
    expect(screen.getByText(/seit 2024-03-12/)).toBeTruthy()
    expect(screen.getByText(/seit 2026-07-05/)).toBeTruthy()
  })

  test('Erklärtext zu jedem der drei DSGVO-Schalter', () => {
    render(<ProfileDatenschutzTab ownMember={baseMember} onUpdated={() => {}} />)
    expect(screen.getByText(/zur Vereinsverwaltung zu verarbeiten/i)).toBeTruthy()
    expect(screen.getByText(/Weitergabe deiner Mitgliedsdaten an Dritte/i)).toBeTruthy()
    expect(screen.getByText(/öffentlichen Kanälen des Vereins/i)).toBeTruthy()
  })
})
