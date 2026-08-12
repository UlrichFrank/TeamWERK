/**
 * Ausrichter-Feld im Termin-Wizard (heimspieltag-ausrichter, design.md Decision 9).
 *
 * Der Wert steht zwischen lauter Termin-Feldern, ändert aber den ganzen Spieltag.
 * Geprüft wird deshalb genau das, was diese Falle entschärft: tagesbezogene
 * Beschriftung, Vorbelegung mit dem geltenden Wert und — bei Abweichung — die
 * Vorschau VOR dem Anlegen des Termins.
 */
import { describe, test, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import KalenderPage from '../KalenderPage'
import { renderAsPersona, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))

const DATE = '2026-09-14'

const AUSRICHTER = [
  { id: 1, name: 'TV Ötlingen', aktiv: true, is_default: true, sort_order: 1 },
  { id: 2, name: 'SV Nachbar', aktiv: true, is_default: false, sort_order: 2 },
]

const HOST_GEERBT = { date: DATE, ausrichter_id: 1, ausrichter_name: 'TV Ötlingen', is_explicit: false }

const HOST_PREVIEW = {
  date: DATE, ausrichter_id: 2, ausrichter_name: 'SV Nachbar', is_explicit: true,
  balance: {
    created: 1, deleted: 2, assignments_kept: 0, assignments_lost: 1,
    slots_before: 2, slots_after: 1, assignments_before: 1, assignments_after: 0,
  },
}

const TEAMS = [{ id: 7, name: 'Herren 1', age_class: 'Herren', gender: 'm', team_number: 1, group_count: 1, is_active: true }]

const MOCKS = [
  { url: /\/games/, data: { items: [], total: 0 } },
  { url: /\/training-sessions/, data: { items: [], total: 0 } },
  { url: /\/absences/, data: [] },
  { url: '/teams', data: TEAMS },
  { url: '/teams/names', data: TEAMS },
  { url: '/seasons', data: [{ id: 1, is_active: true }] },
  { url: '/venues', data: [] },
  { url: '/duty-templates', data: [] },
  { url: '/ausrichter', data: { items: AUSRICHTER } },
  { url: `/game-days/${DATE}/host`, data: HOST_GEERBT },
  { method: 'any' as const, url: '/game-days/host/preview', data: HOST_PREVIEW },
  { method: 'any' as const, url: '/game-days/host/apply', data: { ...HOST_PREVIEW, applied: true } },
  { method: 'any' as const, url: '/games', data: { id: 99 } },
]

/** Die Mannschafts-Auswahl des Wizards — erkennbar an der Team-ID als Options-Wert. */
function teamSelect(container: HTMLElement): HTMLSelectElement {
  const found = Array.from(container.querySelectorAll('select'))
    .find(s => Array.from(s.options).some(o => o.value === '7') && Array.from(s.options).some(o => o.textContent === 'Auswählen…'))
  if (!found) throw new Error('Mannschafts-Auswahl im Wizard nicht gefunden')
  return found
}

/** Der Wizard hat je Schritt genau einen „Weiter →"-Knopf; beim Schrittwechsel
 *  können kurzzeitig zwei im DOM stehen — es zählt immer der zuletzt gerenderte. */
function clickWeiter() {
  const buttons = screen.getAllByRole('button', { name: /Weiter/ })
  fireEvent.click(buttons[buttons.length - 1])
}

function postsTo(url: string) {
  return getApiMock().history.post.filter(r => r.url === url)
}

/** Öffnet den Wizard bis zu den Event-Details eines Heimspiels mit gesetztem Datum. */
async function openHeimWizard(container: HTMLElement) {
  fireEvent.click(screen.getByRole('button', { name: /^Event$/i }))
  fireEvent.click(await screen.findByText('Heimspiel'))

  const dateInput = container.querySelector('input[type="date"]') as HTMLInputElement
  fireEvent.change(dateInput, { target: { value: DATE } })
  await flushAsync()
}

describe('KalenderPage — Ausrichter im Termin-Wizard', () => {
  test('Feld ist tagesbezogen beschriftet und mit dem geltenden Wert vorbelegt', async () => {
    const { container } = renderAsPersona(<KalenderPage />, 'vorstand', { mocks: MOCKS })
    await flushAsync()
    await openHeimWizard(container)

    const select = await screen.findByLabelText('Ausrichter am 14.09.') as HTMLSelectElement
    expect(select.value).toBe('1')
    expect(screen.getByText('Gilt für alle Termine dieses Tages.')).toBeTruthy()
  })

  test('abweichende Wahl kündigt die Vorschau an und öffnet sie vor dem Anlegen', async () => {
    const { container } = renderAsPersona(<KalenderPage />, 'vorstand', { mocks: MOCKS })
    await flushAsync()
    await openHeimWizard(container)

    fireEvent.change(await screen.findByLabelText('Ausrichter am 14.09.'), { target: { value: '2' } })
    expect(screen.getByText(/Weicht vom geltenden Wert ab \(TV Ötlingen\)/)).toBeTruthy()

    // Restliche Pflichtfelder: Gegner + Mannschaft, dann durch den Wizard bis zur Bestätigung.
    fireEvent.change(container.querySelector('input[type="text"]') as HTMLInputElement, { target: { value: 'FC Test' } })
    fireEvent.change(teamSelect(container), { target: { value: '7' } })

    clickWeiter()
    await flushAsync()
    // Ohne Vorlage geht es direkt zur Dienste-Bestätigung.
    clickWeiter()
    await flushAsync()
    fireEvent.click(screen.getByRole('button', { name: 'Bestätigen' }))

    await waitFor(() => expect(postsTo('/game-days/host/preview').length).toBe(1))
    await flushAsync()

    // Der Termin ist NICHT angelegt, solange der Wechsel nicht bestätigt wurde.
    expect(screen.getByRole('button', { name: 'Übernehmen' })).toBeTruthy()
    expect(postsTo('/games')).toHaveLength(0)

    // Bestätigen: erst den Tageswert schreiben, dann den Termin anlegen — die
    // Reihenfolge entscheidet, mit welchem Ausrichter der Auto-Regen rechnet.
    fireEvent.click(screen.getByRole('button', { name: 'Übernehmen' }))
    await waitFor(() => expect(postsTo('/game-days/host/apply').length).toBe(1))
    await waitFor(() => expect(postsTo('/games').length).toBe(1))
  })

  test('ohne Abweichung wird der Termin ohne Vorschau angelegt', async () => {
    const { container } = renderAsPersona(<KalenderPage />, 'vorstand', { mocks: MOCKS })
    await flushAsync()
    await openHeimWizard(container)
    await screen.findByLabelText('Ausrichter am 14.09.')

    fireEvent.change(container.querySelector('input[type="text"]') as HTMLInputElement, { target: { value: 'FC Test' } })
    fireEvent.change(teamSelect(container), { target: { value: '7' } })

    clickWeiter()
    await flushAsync()
    clickWeiter()
    await flushAsync()
    fireEvent.click(screen.getByRole('button', { name: 'Bestätigen' }))

    await waitFor(() => expect(postsTo('/games').length).toBe(1))
    expect(postsTo('/game-days/host/preview')).toHaveLength(0)
  })
})
