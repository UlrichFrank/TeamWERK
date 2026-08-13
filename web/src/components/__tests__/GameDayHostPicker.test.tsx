/**
 * Tages-Ausrichter im Termin-Detail-Modal (heimspieltag-ausrichter, design.md
 * Decision 9/10): tagesbezogene Beschriftung, Kennzeichnung geerbt/festgelegt,
 * und — der eigentliche Schutz — kein Schreibvorgang ohne vorherige Vorschau.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import GameDayHostSection from '../GameDayHostPicker'
import { renderAsPersonaNoRouter, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const DATE = '2026-09-14'

const AUSRICHTER = [
  { id: 1, name: 'TV Ötlingen', aktiv: true, is_default: true, sort_order: 1 },
  { id: 2, name: 'SV Nachbar', aktiv: true, is_default: false, sort_order: 2 },
]

const HOST_GEERBT = {
  date: DATE, ausrichter_id: 1, ausrichter_name: 'TV Ötlingen', is_explicit: false,
}

const BALANCE = {
  created: 0, deleted: 3, assignments_kept: 1, assignments_lost: 2,
  slots_before: 3, slots_after: 0, assignments_before: 3, assignments_after: 1,
}

const PREVIEW = {
  date: DATE, ausrichter_id: 2, ausrichter_name: 'SV Nachbar', is_explicit: true, balance: BALANCE,
}

const MOCKS = [
  { url: '/ausrichter', data: { items: AUSRICHTER } },
  { url: `/game-days/${DATE}/host`, data: HOST_GEERBT },
  { method: 'any' as const, url: '/game-days/host/preview', data: PREVIEW },
  { method: 'any' as const, url: '/game-days/host/apply', data: { ...PREVIEW, applied: true } },
]

function postsTo(url: string) {
  return getApiMock().history.post.filter(r => r.url === url)
}

async function renderSection(canEdit: boolean, onApplied: () => void = () => {}) {
  const result = renderAsPersonaNoRouter(
    <GameDayHostSection date={DATE} canEdit={canEdit} onApplied={onApplied} />,
    'vorstand',
    { mocks: MOCKS },
  )
  await flushAsync()
  return result
}

describe('GameDayHostSection', () => {
  test('beschriftet tagesbezogen, zeigt den geltenden Wert und dass er geerbt ist', async () => {
    await renderSection(true)

    const select = await screen.findByLabelText('Ausrichter am 14.09.') as HTMLSelectElement
    expect(select.value).toBe('1')
    expect(screen.getByText('Gilt für alle Termine dieses Tages.')).toBeTruthy()
    expect(screen.getByText('Vom Standard geerbt.')).toBeTruthy()
  })

  test('eine Änderung öffnet erst die Vorschau und schreibt nichts', async () => {
    await renderSection(true)
    const select = await screen.findByLabelText('Ausrichter am 14.09.')

    fireEvent.change(select, { target: { value: '2' } })

    await waitFor(() => expect(postsTo('/game-days/host/preview').length).toBe(1))
    await flushAsync()

    // Die Bilanz benennt die verlorenen Zusagen — der eigentliche Preis des Wechsels.
    expect(screen.getByText(/2 Zuweisungen gehen verloren/)).toBeTruthy()
    expect(screen.getByText('3 Dienste entfallen')).toBeTruthy()
    // Solange nicht bestätigt wurde, ist nichts persistiert.
    expect(postsTo('/game-days/host/apply')).toHaveLength(0)
  })

  test('Bestätigen wendet den Wechsel an und meldet ihn nach oben', async () => {
    const applied: number[] = []
    await renderSection(true, () => applied.push(1))
    fireEvent.change(await screen.findByLabelText('Ausrichter am 14.09.'), { target: { value: '2' } })
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Übernehmen' })).toBeTruthy())

    fireEvent.click(screen.getByRole('button', { name: 'Übernehmen' }))

    await waitFor(() => expect(postsTo('/game-days/host/apply').length).toBe(1))
    const body = JSON.parse(postsTo('/game-days/host/apply')[0].data as string)
    expect(body).toEqual({ date: DATE, ausrichter_id: 2 })
    await waitFor(() => expect(applied).toHaveLength(1))
  })

  test('Abbrechen im Vorschau-Dialog schreibt nicht', async () => {
    await renderSection(true)
    fireEvent.change(await screen.findByLabelText('Ausrichter am 14.09.'), { target: { value: '2' } })
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Abbrechen' })).toBeTruthy())

    fireEvent.click(screen.getByRole('button', { name: 'Abbrechen' }))
    await flushAsync()

    expect(postsTo('/game-days/host/apply')).toHaveLength(0)
    expect(screen.queryByRole('button', { name: 'Übernehmen' })).toBeNull()
  })

  test('ohne manage_games ist der Wert nur lesbar', async () => {
    await renderSection(false)

    await waitFor(() => expect(screen.getByText('TV Ötlingen')).toBeTruthy())
    expect(screen.queryByRole('combobox')).toBeNull()
    expect(screen.getByText('Ausrichter am 14.09.')).toBeTruthy()
  })
})
