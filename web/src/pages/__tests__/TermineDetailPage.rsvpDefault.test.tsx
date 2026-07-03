/**
 * TermineDetailPage: Zeilen mit virtuellem Default (rsvp_is_default=true) werden
 * dezent (text-brand-text-subtle italic) gerendert, aktive Antworten normal.
 * Quelle: openspec/changes/rsvp-defaults-per-rolle/specs/termine-detail/spec.md
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { Routes, Route } from 'react-router-dom'
import TermineDetailPage from '../TermineDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const SESSION_FIXTURE = {
  id: 1,
  date: '2020-06-17',
  start_time: '18:00',
  end_time: '19:30',
  status: 'active',
  team_id: 1,
  team_name: 'Test Team',
  confirmed_count: 1,
  declined_count: 1,
  maybe_count: 0,
  my_rsvp: null,
  note: '',
  cancel_reason: '',
  rsvp_default_players: 'declined',
  rsvp_default_extended: 'none',
  rsvp_require_reason: 0,
}

const ATTENDANCE_FIXTURE = [
  { member_id: 1, member_name: 'Default Spieler', present: null, rsvp_status: 'declined', rsvp_is_default: true, reason: null },
  { member_id: 2, member_name: 'Aktiv Spieler', present: null, rsvp_status: 'confirmed', rsvp_is_default: false, reason: null },
]

describe('TermineDetailPage — virtuelle Default-Zeile dezent', () => {
  test('Default-Zeile ist kursiv, aktive Antwort nicht', async () => {
    renderAsPersona(
      <Routes>
        <Route path="/termine/:type/:id" element={<TermineDetailPage />} />
      </Routes>,
      'admin',
      {
        initialEntries: ['/termine/training/1'],
        mocks: [
          { url: /training-sessions\/1$/, data: SESSION_FIXTURE },
          { url: /training-sessions\/1\/attendances/, data: ATTENDANCE_FIXTURE },
        ],
      },
    )

    await screen.findByText('Default Spieler')
    await flushAsync()

    const defaultRow = screen.getByText('Default Spieler').closest('tr')!
    const activeRow = screen.getByText('Aktiv Spieler').closest('tr')!

    expect(defaultRow.querySelector('.italic')).not.toBeNull()
    expect(activeRow.querySelector('.italic')).toBeNull()
  })
})
