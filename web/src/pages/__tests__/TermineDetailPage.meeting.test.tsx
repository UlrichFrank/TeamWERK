/**
 * TermineDetailPage — Treffzeit (spiel-treffpunkt): jeder sieht Treffzeit und
 * Ort unter dem Anwurf. Gepflegt wird die Treffzeit ausschließlich im
 * Kalender (EventInfoModal) — die Detailseite zeigt auch mit `can.edit`
 * keinen Editor.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { Routes, Route } from 'react-router-dom'
import TermineDetailPage from '../TermineDetailPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function renderDetail(persona: string, canEdit: boolean, meeting: Record<string, unknown>) {
  renderAsPersona(
    <Routes><Route path="/termine/:type/:id" element={<TermineDetailPage />} /></Routes>,
    persona,
    {
      initialEntries: ['/termine/spiel/31'],
      mocks: [
        {
          url: /\/games\/31$/,
          data: {
            game: {
              id: 31, date: '2099-10-11', time: '15:00', opponent: 'SG Weinstadt',
              event_type: 'auswärts', is_home: false, season_id: 1,
              rsvp_default_players: 'none', rsvp_default_extended: 'none', rsvp_require_reason: 0,
              teams: [{ id: 100, name: 'mB1', display_short: 'mB1', display_long: 'mB1' }],
              can: { edit: canEdit, delete: false, manage_lineup: canEdit },
              ...meeting,
            },
          },
        },
        { url: /\/games\/31\/participants/, data: { items: [{ member_id: 1, member_name: 'Jonas Keller', is_extended: false, rsvp_status: 'confirmed', team_id: 100 }], hidden_team_ids: [] } },
      ],
    },
  )
}

const WITH_MEETING = { meet_time: '13:30', meet_date: '2099-10-11', meet_place: 'Parkplatz Vereinsheim' }

describe('TermineDetailPage — Treffzeit', () => {
  test('Spieler sieht Treffzeit und Ort, aber keinen Editor', async () => {
    renderDetail('spieler', false, WITH_MEETING)
    await screen.findByText('Jonas Keller')
    await flushAsync()
    expect(screen.getByText('Treffen 13:30 Uhr · Parkplatz Vereinsheim')).toBeInTheDocument()
    expect(screen.queryByLabelText('Treffzeit')).toBeNull()
  })

  test('Trainer mit Bearbeitungsrecht sieht Treffzeit, aber keinen Editor (Pflege nur im Kalender)', async () => {
    renderDetail('trainer', true, WITH_MEETING)
    await screen.findByText('Jonas Keller')
    await flushAsync()
    expect(screen.getByText('Treffen 13:30 Uhr · Parkplatz Vereinsheim')).toBeInTheDocument()
    expect(screen.queryByLabelText('Treffzeit')).toBeNull()
    expect(screen.queryByLabelText('Treffpunkt (optional)')).toBeNull()
  })

  test('ohne Treffzeit erscheint keine Treffzeit-Zeile', async () => {
    renderDetail('spieler', false, { meet_time: null, meet_date: null, meet_place: '' })
    await screen.findByText('Jonas Keller')
    await flushAsync()
    expect(screen.queryByText(/^Treffen /)).toBeNull()
  })
})
