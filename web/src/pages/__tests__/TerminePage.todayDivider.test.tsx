/**
 * TerminePage „heute"-Trennlinie:
 * Ein Divider mit Label „heute" wird vor dem ersten nicht-vergangenen Termin
 * gerendert — aber nur, wenn davor mindestens ein vergangener Termin steht.
 * Quelle: openspec/changes/termine-scroll-und-heute-marker/specs/termine-unified-view/spec.md
 *   §"Trennlinie „heute" vor dem ersten nicht-vergangenen Termin"
 *
 * Hinweis: Die Datumsfilterung (from/to) ist serverseitig; der API-Mock liefert die
 * Fixtures unabhängig von Query-Parametern. Der Divider-Index (todayIdx) wird
 * clientseitig aus `date >= heute` berechnet — daher genügen ein klar vergangenes
 * und ein klar zukünftiges Datum.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import TerminePage from '../TerminePage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

function training(id: number, date: string) {
  return {
    id,
    date,
    start_time: '18:00',
    end_time: '19:30',
    status: 'active',
    team_id: 1,
    team_name: 'Test Team',
    confirmed_count: 0,
    declined_count: 0,
    maybe_count: 0,
    my_rsvp: null,
    note: '',
    cancel_reason: '',
    rsvp_default_players: 'none',
    rsvp_default_extended: 'none',
    rsvp_require_reason: 0,
  }
}

describe('TerminePage — „heute"-Trennlinie', () => {
  test('Divider erscheint zwischen vergangenem und zukünftigem Termin', async () => {
    renderAsPersona(<TerminePage />, 'spieler', {
      mocks: [
        { url: /\/training-sessions\?/, data: [training(1, '2020-01-01'), training(2, '2030-06-17')] },
        { url: /\/games\/my/, data: [] },
        { url: /\/teams/, data: [] },
      ],
    })

    // Warten bis beide Termine gerendert sind
    await screen.findAllByText('18:00 – 19:30')
    expect(screen.getByText('heute')).toBeInTheDocument()
    await flushAsync()
  })

  test('kein Divider, wenn alle sichtbaren Termine >= heute liegen', async () => {
    renderAsPersona(<TerminePage />, 'spieler', {
      mocks: [
        { url: /\/training-sessions\?/, data: [training(1, '2030-06-17'), training(2, '2031-06-17')] },
        { url: /\/games\/my/, data: [] },
        { url: /\/teams/, data: [] },
      ],
    })

    await screen.findAllByText('18:00 – 19:30')
    expect(screen.queryByText('heute')).toBeNull()
    await flushAsync()
  })
})
