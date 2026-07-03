/**
 * TermineDetailPage — Multi-Team-Event mit Cross-Team-Filter.
 * Quelle: openspec/changes/profile-cross-team-visibility/specs/spiel-teilnahme/spec.md
 *
 * Backend liefert `{ items, hidden_team_ids }`. Bei `hidden_team_ids.includes(teamID)`
 * wird unter der Team-Sektion „Weitere Mitglieder nicht sichtbar" gerendert.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { Routes, Route } from 'react-router-dom'
import TermineDetailPage from '../TermineDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const GAME_FIXTURE = {
  game: {
    id: 22,
    date: '2026-04-12',
    time: '15:00',
    opponent: 'Vereinsfest',
    event_type: 'generisch',
    is_home: true,
    season_id: 1,
    rsvp_default_players: 'none',
    rsvp_default_extended: 'none',
    rsvp_require_reason: 0,
    teams: [
      { id: 100, name: 'Team A', display_short: 'A', display_long: 'Team A' },
      { id: 200, name: 'Team B', display_short: 'B', display_long: 'Team B' },
    ],
    can: { edit: false, delete: false, manage_lineup: false },
  },
}

describe('TermineDetailPage — Multi-Team-Cross-Team', () => {
  test('zeigt „Weitere Mitglieder nicht sichtbar" pro Team, wenn das Backend hidden_team_ids liefert', async () => {
    const PARTICIPANTS = {
      items: [
        { member_id: 1, member_name: 'Anna Schmidt', is_extended: false, rsvp_status: 'confirmed', in_lineup: false, team_id: 100 },
        { member_id: 2, member_name: 'Ben Müller',   is_extended: false, rsvp_status: null,        in_lineup: false, team_id: 100 },
        { member_id: 3, member_name: 'Carl Beispiel', is_extended: false, rsvp_status: 'confirmed', in_lineup: false, team_id: 200 },
      ],
      hidden_team_ids: [200],
    }
    renderAsPersona(
      <Routes>
        <Route path="/termine/:type/:id" element={<TermineDetailPage />} />
      </Routes>,
      'spieler',
      {
        initialEntries: ['/termine/ereignis/22'],
        mocks: [
          { url: /\/games\/22$/, data: GAME_FIXTURE },
          { url: /\/games\/22\/participants/, data: PARTICIPANTS },
        ],
      },
    )

    await screen.findByText('Anna Schmidt')
    await flushAsync()

    // Beide Sektionen sichtbar (jede hat mindestens eine Zeile).
    expect(screen.getByText('Team A')).toBeTruthy()
    expect(screen.getByText('Team B')).toBeTruthy()
    // Footer nur einmal (nur Team B ist in hidden_team_ids).
    const footers = screen.getAllByText('Weitere Mitglieder nicht sichtbar')
    expect(footers).toHaveLength(1)
  })

  test('rendert leere Sektion (alle Member gefiltert) gar nicht — kein leerer Header', async () => {
    const PARTICIPANTS = {
      items: [
        { member_id: 1, member_name: 'Anna Schmidt', is_extended: false, rsvp_status: 'confirmed', in_lineup: false, team_id: 100 },
      ],
      // Team B hat hidden_team_ids, aber 0 sichtbare Member -> Section weglassen
      hidden_team_ids: [200],
    }
    renderAsPersona(
      <Routes>
        <Route path="/termine/:type/:id" element={<TermineDetailPage />} />
      </Routes>,
      'spieler',
      {
        initialEntries: ['/termine/ereignis/22'],
        mocks: [
          { url: /\/games\/22$/, data: GAME_FIXTURE },
          { url: /\/games\/22\/participants/, data: PARTICIPANTS },
        ],
      },
    )

    await screen.findByText('Anna Schmidt')
    await flushAsync()

    expect(screen.getByText('Team A')).toBeTruthy()
    // Team B Sektion soll NICHT gerendert sein.
    expect(screen.queryByText('Team B')).toBeNull()
    // Auch kein Footer-Hinweis, da Team B keine sichtbaren Zeilen hat.
    expect(screen.queryByText('Weitere Mitglieder nicht sichtbar')).toBeNull()
  })

  test('keine Hinweise, wenn hidden_team_ids leer ist (vollständige Sicht)', async () => {
    const PARTICIPANTS = {
      items: [
        { member_id: 1, member_name: 'Anna Schmidt', is_extended: false, rsvp_status: 'confirmed', in_lineup: false, team_id: 100 },
        { member_id: 3, member_name: 'Carl Beispiel', is_extended: false, rsvp_status: 'confirmed', in_lineup: false, team_id: 200 },
      ],
      hidden_team_ids: [],
    }
    renderAsPersona(
      <Routes>
        <Route path="/termine/:type/:id" element={<TermineDetailPage />} />
      </Routes>,
      'trainer',
      {
        initialEntries: ['/termine/ereignis/22'],
        mocks: [
          { url: /\/games\/22$/, data: GAME_FIXTURE },
          { url: /\/games\/22\/participants/, data: PARTICIPANTS },
        ],
      },
    )

    await screen.findByText('Anna Schmidt')
    await flushAsync()

    expect(screen.queryByText('Weitere Mitglieder nicht sichtbar')).toBeNull()
  })
})
