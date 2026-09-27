/**
 * TermineDetailPage — Aufstellungsstatus (Change aufstellung-status-termine).
 * Jede Spielerzeile trägt eine Checkbox mit drei Zuständen; der Kartenkopf nennt
 * „Aufstellung offen" bzw. „N aufgestellt". Leere Aufstellung ist nie „nicht aufgestellt".
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { Routes, Route } from 'react-router-dom'
import TermineDetailPage from '../TermineDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function gameFixture(eventType: string, manageLineup: boolean) {
  return {
    game: {
      id: 31, date: '2099-10-11', time: '14:00', opponent: 'SG Weinstadt',
      event_type: eventType, is_home: true, season_id: 1,
      rsvp_default_players: 'none', rsvp_default_extended: 'none', rsvp_require_reason: 0,
      teams: [{ id: 100, name: 'mB1', display_short: 'mB1', display_long: 'mB1' }],
      can: { edit: manageLineup, delete: false, manage_lineup: manageLineup },
    },
  }
}

const row = (id: number, name: string, inLineup: boolean, lineup?: string, extra: Record<string, unknown> = {}) =>
  ({ member_id: id, member_name: name, is_extended: false, rsvp_status: 'confirmed', in_lineup: inLineup, lineup, team_id: 100, ...extra })

function renderDetail(persona: string, eventType: string, manageLineup: boolean, participants: unknown) {
  renderAsPersona(
    <Routes><Route path="/termine/:type/:id" element={<TermineDetailPage />} /></Routes>,
    persona,
    {
      initialEntries: [`/termine/${eventType === 'generisch' ? 'ereignis' : 'spiel'}/31`],
      mocks: [
        { url: /\/games\/31$/, data: gameFixture(eventType, manageLineup) },
        { url: /\/games\/31\/participants/, data: participants },
      ],
    },
  )
}

describe('TermineDetailPage — Aufstellung', () => {
  test('Spieler: gespeicherte Aufstellung als nur lesende Checkboxen, Kopf „N aufgestellt"', async () => {
    renderDetail('spieler', 'heim', false, {
      items: [row(1, 'Jonas Keller', true, 'in'), row(2, 'Finn Wagner', false, 'out')],
      hidden_team_ids: [],
      lineup_count: 5, // auch Zeilen, die der Spieler nicht sieht
    })
    await screen.findByText('Jonas Keller')
    await flushAsync()
    expect(screen.getByText('5 aufgestellt')).toBeTruthy()
    const jonas = screen.getByRole('checkbox', { name: 'Jonas Keller: aufgestellt' })
    expect(jonas).toHaveAttribute('aria-checked', 'true')
    expect(jonas).toHaveAttribute('aria-disabled', 'true')
    expect(screen.getByRole('checkbox', { name: 'Finn Wagner: nicht aufgestellt' })).toBeTruthy()
  })

  test('Spieler: leere Aufstellung zeigt nirgends „nicht aufgestellt"', async () => {
    renderDetail('spieler', 'heim', false, {
      items: [row(1, 'Jonas Keller', false, 'open'), row(2, 'Finn Wagner', false, 'open')],
      hidden_team_ids: [],
      lineup_count: 0,
    })
    await screen.findByText('Jonas Keller')
    await flushAsync()
    expect(screen.getByText('Aufstellung offen')).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: 'Jonas Keller: Aufstellung offen' })).toBeTruthy()
    expect(screen.queryByText(/nicht aufgestellt/)).toBeNull()
    expect(screen.queryByRole('checkbox', { name: /nicht aufgestellt/ })).toBeNull()
  })

  test('Trainer: erstes Häkchen wechselt Kopf und macht die übrigen zu „nicht aufgestellt"', async () => {
    renderDetail('trainer', 'heim', true, {
      items: [row(1, 'Jonas Keller', false, 'open'), row(2, 'Finn Wagner', false, 'open')],
      hidden_team_ids: [],
      lineup_count: 0,
    })
    await screen.findByText('Jonas Keller')
    await flushAsync()
    expect(screen.getByText('Aufstellung offen')).toBeTruthy()
    const jonas = screen.getByRole('checkbox', { name: 'Jonas Keller: Aufstellung offen' })
    fireEvent.keyDown(jonas, { key: ' ' })
    await flushAsync()
    expect(screen.getByText('1 aufgestellt')).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: 'Jonas Keller: aufgestellt' })).toBeTruthy()
    expect(screen.getByRole('checkbox', { name: 'Finn Wagner: nicht aufgestellt' })).toBeTruthy()
  })

  test('„Sonstiges" behält die Aufstellungs-Spalte', async () => {
    renderDetail('spieler', 'generisch', false, {
      items: [row(1, 'Jonas Keller', false, 'open')],
      hidden_team_ids: [],
      lineup_count: 0,
    })
    await screen.findByText('Jonas Keller')
    await flushAsync()
    expect(screen.getByRole('checkbox', { name: 'Jonas Keller: Aufstellung offen' })).toBeTruthy()
  })
})
