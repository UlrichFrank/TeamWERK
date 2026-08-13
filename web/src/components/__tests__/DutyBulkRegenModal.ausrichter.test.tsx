/**
 * Ausrichter-Spalte im Massenlauf-Dialog (heimspieltag-ausrichter, Task 10.4):
 * die Zeilen kommen flach je Termin, der Ausrichter gehört zum Spieltag — die
 * Tages-Zwischenebene wird clientseitig über row.date gebildet. Eine Änderung
 * geht als host_overrides mit und braucht keinen zweiten Bestätigungsdialog.
 */
import { describe, test, expect } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import DutyBulkRegenModal from '../DutyBulkRegenModal'
import { renderAsPersonaNoRouter, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

const RANGE = { from: '2026-09-01', to: '2026-09-30' }

const AUSRICHTER = [
  { id: 1, name: 'TV Ötlingen', aktiv: true, is_default: true, sort_order: 1 },
  { id: 2, name: 'SV Nachbar', aktiv: true, is_default: false, sort_order: 2 },
]

function row(over: Record<string, unknown>) {
  return {
    game_id: 1, date: '2026-09-05', time: '14:00',
    event_name: 'Heimspiel vs. A', event_type: 'heim',
    effective_action: 'none', excluded: false,
    slots_before: { auto: 0, custom: 0 }, slots_after: { auto: 0, custom: 0 },
    created: 0, deleted_auto: 0, deleted_custom: 0,
    assignments_kept: 0, assignments_lost: 0, conflicts: 0,
    ...over,
  }
}

// Zwei Termine am selben Tag plus einer an einem zweiten Tag: die Tagesebene
// darf nicht einfach „eine Zeile pro Termin" sein.
const ROWS = [
  row({}),
  row({ game_id: 2, time: '16:00', event_name: 'Heimspiel vs. B' }),
  row({ game_id: 3, date: '2026-09-12', event_name: 'Heimspiel vs. C' }),
]

const DAYS = [
  { date: '2026-09-05', effective_ausrichter_id: 1, is_explicit: false },
  { date: '2026-09-12', stored_ausrichter_id: 2, effective_ausrichter_id: 2, is_explicit: true },
]

const TOTALS = {
  games: 3, created: 0, deleted: 0, custom_kept: 0, custom_deleted: 0,
  assignments_kept: 0, assignments_lost: 0, conflicts: 0, notified_users: 0,
}

const PREVIEW_RESPONSE = { range: RANGE, rows: ROWS, days: DAYS, totals: TOTALS, warnings: [] }

const MOCKS = [
  { url: '/duty-templates', data: [] },
  { url: '/ausrichter', data: { items: AUSRICHTER } },
  { method: 'any' as const, url: '/duty-slots/bulk-regen/preview', data: PREVIEW_RESPONSE },
  { method: 'any' as const, url: '/duty-slots/bulk-regen/apply', data: { ...PREVIEW_RESPONSE, applied: true } },
]

function previewRequests() {
  return getApiMock().history.post.filter(r => r.url === '/duty-slots/bulk-regen/preview')
}

function lastPreviewBody() {
  const reqs = previewRequests()
  return JSON.parse(reqs[reqs.length - 1].data as string)
}

async function renderModal(onApplied: (r: unknown) => void = () => {}) {
  renderAsPersonaNoRouter(
    <DutyBulkRegenModal isOpen onClose={() => {}} onApplied={onApplied} />,
    'vorstand',
    { mocks: MOCKS },
  )
  await waitFor(() => expect(previewRequests().length).toBeGreaterThan(0))
  await flushAsync()
}

describe('DutyBulkRegenModal — Ausrichter je Spieltag', () => {
  test('zeigt je Spieltag eine Auswahl mit dem wirksamen Wert und geerbt/festgelegt', async () => {
    await renderModal()

    const tag5 = await screen.findByLabelText('Ausrichter am 05.09.2026') as HTMLSelectElement
    expect(tag5.value).toBe('1')
    expect(screen.getByText('geerbt')).toBeTruthy()
    expect(screen.getByText('festgelegt')).toBeTruthy()

    // Zwei Tage, nicht drei Termine.
    expect(screen.getByLabelText('Ausrichter am 12.09.2026')).toBeTruthy()
    expect(screen.getAllByRole('combobox', { name: /^Ausrichter am/ })).toHaveLength(2)
  })

  test('eine Änderung geht als host_overrides in die Vorschau', async () => {
    await renderModal()

    fireEvent.change(screen.getByLabelText('Ausrichter am 05.09.2026'), { target: { value: '2' } })

    await waitFor(() => {
      expect(lastPreviewBody().host_overrides).toEqual([{ date: '2026-09-05', ausrichter_id: 2 }])
    })
    expect(screen.getByText('wird geändert')).toBeTruthy()
  })

  test('ohne Änderung bleibt host_overrides leer', async () => {
    await renderModal()
    expect(lastPreviewBody().host_overrides).toEqual([])
  })

  test('das Anwenden übernimmt Vorlagen- und Ausrichter-Änderung in einem Schritt', async () => {
    const seen: unknown[] = []
    await renderModal(r => seen.push(r))

    fireEvent.change(screen.getByLabelText('Ausrichter am 05.09.2026'), { target: { value: '2' } })
    await waitFor(() => expect(lastPreviewBody().host_overrides).toHaveLength(1))
    await flushAsync()

    fireEvent.click(screen.getByRole('button', { name: 'Dienste aktualisieren' }))

    await waitFor(() => expect(seen).toHaveLength(1))
    const applyReq = getApiMock().history.post.filter(r => r.url === '/duty-slots/bulk-regen/apply')
    expect(applyReq).toHaveLength(1)
    expect(JSON.parse(applyReq[0].data as string).host_overrides).toEqual([{ date: '2026-09-05', ausrichter_id: 2 }])
  })

  test('ein verschobener Zeitraum verwirft Ausrichter-Wechsel außerhalb des Fensters', async () => {
    await renderModal()

    fireEvent.change(screen.getByLabelText('Ausrichter am 05.09.2026'), { target: { value: '2' } })
    await waitFor(() => expect(lastPreviewBody().host_overrides).toHaveLength(1))

    fireEvent.change(screen.getByLabelText('Von'), { target: { value: '2026-09-10' } })

    await waitFor(() => expect(lastPreviewBody().host_overrides).toEqual([]))
  })
})
