/**
 * Massen-Regeneration der Dienst-Slots: Pauschalwahl vs. Zeilen-Override, Ausnahme,
 * destruktive purge-Markierung, Debounce/Abort der Live-Vorschau.
 * Quelle: openspec/changes/duty-bulk-regen/specs/duty-bulk-regen/spec.md
 */
import { describe, test, expect } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import DutyBulkRegenModal from '../DutyBulkRegenModal'
import { renderAsPersonaNoRouter, flushAsync } from '../../test/renderAsPersona'
import { getApiMock } from '../../test/apiMock'

const TEMPLATES = [
  { id: 10, name: 'Standard Heim', template_type: 'heim' },
  { id: 11, name: 'Standard Auswärts', template_type: 'auswärts' },
]

const RANGE = { from: '2026-09-01', to: '2026-09-30' }

function row(over: Record<string, unknown>) {
  return {
    game_id: 1,
    date: '2026-09-05',
    time: '14:00',
    event_name: 'Heimspiel vs. A',
    event_type: 'heim',
    effective_action: 'none',
    excluded: false,
    slots_before: { auto: 0, custom: 0 },
    slots_after: { auto: 0, custom: 0 },
    created: 0,
    deleted_auto: 0,
    deleted_custom: 0,
    assignments_kept: 0,
    assignments_lost: 0,
    conflicts: 0,
    ...over,
  }
}

const ROWS = [
  row({}),
  row({
    game_id: 2, date: '2026-09-06', time: '16:00',
    event_name: 'Auswärtsspiel vs. B', event_type: 'auswärts',
  }),
]

const TOTALS = {
  games: 2, created: 0, deleted: 0, custom_kept: 0, custom_deleted: 0,
  assignments_kept: 0, assignments_lost: 0, conflicts: 0, notified_users: 0,
}

const PREVIEW_RESPONSE = { range: RANGE, rows: ROWS, totals: TOTALS, warnings: [] }

const BASE_MOCKS = [
  { method: 'get' as const, url: '/duty-templates', data: TEMPLATES },
  { method: 'any' as const, url: '/duty-slots/bulk-regen/preview', data: PREVIEW_RESPONSE },
  { method: 'any' as const, url: '/duty-slots/bulk-regen/apply', data: { ...PREVIEW_RESPONSE, applied: true } },
]

function renderModal(onApplied: (r: unknown) => void = () => {}) {
  return renderAsPersonaNoRouter(
    <DutyBulkRegenModal isOpen onClose={() => {}} onApplied={onApplied} />,
    'vorstand',
    { mocks: BASE_MOCKS },
  )
}

function previewRequests() {
  return getApiMock().history.post.filter(r => r.url === '/duty-slots/bulk-regen/preview')
}

async function waitForFirstPreview() {
  await waitFor(() => expect(previewRequests().length).toBeGreaterThan(0))
  await flushAsync()
}

describe('DutyBulkRegenModal', () => {
  test('Pauschalwahl setzt alle Zeilen; Zeilen-Override sticht die Pauschalwahl', async () => {
    renderModal()
    await waitForFirstPreview()

    fireEvent.change(screen.getByLabelText('Heimspiele'), { target: { value: '10' } })
    await waitFor(() => {
      const reqs = previewRequests()
      const body = JSON.parse(reqs[reqs.length - 1].data as string)
      expect(body.defaults.heim).toEqual({ action: 'template', template_id: 10 })
    })

    fireEvent.change(
      screen.getByLabelText('Zustand für Heimspiel vs. A am 05.09.2026'),
      { target: { value: 'none' } },
    )
    await waitFor(() => {
      const reqs = previewRequests()
      const body = JSON.parse(reqs[reqs.length - 1].data as string)
      // Pauschalwahl bleibt unverändert bestehen …
      expect(body.defaults.heim).toEqual({ action: 'template', template_id: 10 })
      // … aber die Zeile trägt ihren eigenen Override.
      expect(body.overrides).toEqual([{ game_id: 1, action: 'none' }])
    })
  })

  test('Ausnahme-Checkbox nimmt die Zeile in excluded_game_ids auf', async () => {
    renderModal()
    await waitForFirstPreview()

    // Beide Zeilen tragen eine "Ausnehmen"-Checkbox — die erste gehört zu game_id 1.
    fireEvent.click(screen.getAllByRole('checkbox', { name: 'Ausnehmen' })[0])
    await waitFor(() => {
      const reqs = previewRequests()
      const body = JSON.parse(reqs[reqs.length - 1].data as string)
      expect(body.excluded_game_ids).toEqual(expect.arrayContaining([1]))
    })

    // Die ausgenommene Zeile sperrt ihr eigenes Zustands-Dropdown.
    const rowSelect = screen.getByLabelText('Zustand für Heimspiel vs. A am 05.09.2026') as HTMLSelectElement
    expect(rowSelect.disabled).toBe(true)
  })

  test('„alle Dienste löschen" ist destruktiv markiert und erzeugt action: purge', async () => {
    renderModal()
    await waitForFirstPreview()

    // Ab hier antwortet die Vorschau so, wie es der Server nach einer purge-Wahl
    // tatsächlich täte (effective_action: 'purge' für die betroffene Zeile) — die
    // Modal-Logik baut die Regen-Ergebnisse nie selbst nach (design.md §3).
    const mock = getApiMock()
    mock.reset()
    mock.onGet('/duty-templates').reply(200, TEMPLATES)
    mock.onPost('/duty-slots/bulk-regen/preview').reply(200, {
      range: RANGE, rows: [row({ effective_action: 'purge' }), ROWS[1]], totals: TOTALS, warnings: [],
    })
    mock.onAny().reply(200, [])

    fireEvent.change(screen.getByLabelText('Heimspiele'), { target: { value: 'purge' } })
    await waitFor(() => {
      const reqs = mock.history.post.filter(r => r.url === '/duty-slots/bulk-regen/preview')
      expect(reqs.length).toBeGreaterThan(0)
      const body = JSON.parse(reqs[reqs.length - 1].data as string)
      expect(body.defaults.heim).toEqual({ action: 'purge' })
    })
    await flushAsync()

    const applyButton = screen.getByRole('button', { name: 'Dienste aktualisieren' })
    expect(applyButton.className).toContain('bg-brand-danger')
  })

  test('schnelle Folge von Änderungen erzeugt genau eine zusätzliche Vorschau-Anfrage', async () => {
    renderModal()
    await waitForFirstPreview()
    const before = previewRequests().length

    // Drei Änderungen ohne Await dazwischen — die ersten zwei Timer werden vom
    // jeweils nächsten Effect-Lauf gecancelt, bevor sie feuern.
    fireEvent.change(screen.getByLabelText('Von'), { target: { value: '2026-09-02' } })
    fireEvent.change(screen.getByLabelText('Von'), { target: { value: '2026-09-03' } })
    fireEvent.change(screen.getByLabelText('Von'), { target: { value: '2026-09-04' } })

    await new Promise(resolve => setTimeout(resolve, 600))
    await flushAsync()

    expect(previewRequests().length).toBe(before + 1)
    const last = previewRequests()[previewRequests().length - 1]
    expect(JSON.parse(last.data as string).from).toBe('2026-09-04')
  })

  test('meldet das Apply-Ergebnis an die aufrufende Seite und schließt', async () => {
    const seen: unknown[] = []
    const onApplied = (r: unknown) => seen.push(r)
    const onClose = () => {}
    renderAsPersonaNoRouter(
      <DutyBulkRegenModal isOpen onClose={onClose} onApplied={onApplied} />,
      'vorstand',
      { mocks: BASE_MOCKS },
    )
    await waitForFirstPreview()

    fireEvent.click(screen.getByRole('button', { name: 'Dienste aktualisieren' }))
    await waitFor(() => expect(seen).toHaveLength(1))
  })

  test('rendert nichts, wenn isOpen false ist', () => {
    const { container } = renderAsPersonaNoRouter(
      <DutyBulkRegenModal isOpen={false} onClose={() => {}} onApplied={() => {}} />,
      'vorstand',
      { mocks: BASE_MOCKS },
    )
    expect(container).toBeEmptyDOMElement()
  })
})
