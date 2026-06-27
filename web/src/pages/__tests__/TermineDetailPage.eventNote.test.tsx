/**
 * TermineDetailPage — event-notes: die „Hinweis"-Sektion zeigt den Hinweistext
 * an und blendet den Inline-Editor (Textarea) nur für Berechtigte ein
 * (isTrainer = admin || trainer || sportliche_leitung).
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { Routes, Route } from 'react-router-dom'
import TermineDetailPage from '../TermineDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const SESSION_WITH_NOTE = {
  id: 1,
  date: '2020-06-17',
  start_time: '18:00',
  end_time: '19:30',
  status: 'active',
  team_id: 1,
  team_name: 'Test Team',
  confirmed_count: 1,
  declined_count: 0,
  maybe_count: 0,
  my_rsvp: null,
  note: 'Halle gesperrt, wir joggen am See',
  cancel_reason: '',
  rsvp_opt_out: 0,
  rsvp_require_reason: 0,
}

const ATTENDANCE_FIXTURE = [
  { member_id: 1, member_name: 'Max Mustermann', present: null, rsvp_status: 'confirmed', reason: null },
]

function renderDetail(personaId: string) {
  renderAsPersona(
    <Routes>
      <Route path="/termine/:type/:id" element={<TermineDetailPage />} />
    </Routes>,
    personaId,
    {
      initialEntries: ['/termine/training/1'],
      mocks: [
        { url: /training-sessions\/1$/, data: SESSION_WITH_NOTE },
        { url: /training-sessions\/1\/attendances/, data: ATTENDANCE_FIXTURE },
      ],
    },
  )
}

describe('TermineDetailPage — Hinweis-Sektion', () => {
  test('Trainer sieht Hinweistext und Inline-Editor', async () => {
    renderDetail('trainer')
    await screen.findByText('Max Mustermann')
    await flushAsync()

    // Hinweistext erscheint (Indikator + Editor-Textarea-Wert)
    expect(screen.getAllByText('Halle gesperrt, wir joggen am See').length).toBeGreaterThan(0)
    // Editor-Textarea ist sichtbar (per Placeholder eindeutig)
    expect(screen.getByPlaceholderText(/Hinweis für die Mannschaft/)).toBeInTheDocument()
  })

  test('Spieler sieht Hinweistext, aber keinen Editor', async () => {
    renderDetail('spieler')
    await screen.findByText('Max Mustermann')
    await flushAsync()

    expect(screen.getByText('Halle gesperrt, wir joggen am See')).toBeInTheDocument()
    expect(screen.queryByPlaceholderText(/Hinweis für die Mannschaft/)).toBeNull()
  })
})
