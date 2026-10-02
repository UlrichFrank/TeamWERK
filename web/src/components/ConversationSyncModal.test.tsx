import { describe, test, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import type MockAdapter from 'axios-mock-adapter'
import ConversationSyncModal from './ConversationSyncModal'
import { setupApiMock } from '../test/apiMock'

const spielerT1 = { groupType: 'team', refId: 7, kind: 'spieler', label: 'Spieler mC1', total: 3, alreadyIn: 2 }
const elternT1 = { groupType: 'team', refId: 7, kind: 'eltern', label: 'Eltern mC1', total: 4, alreadyIn: 1 }

function previewWith(over: Record<string, unknown> = {}) {
  return {
    sources: [spielerT1],
    add: [{ id: 11, name: 'Neu Spieler' }],
    remove: [{ id: 21, name: 'Alt Eins' }, { id: 22, name: 'Alt Zwei' }],
    blocked: false,
    suggestions: [spielerT1, elternT1],
    ...over,
  }
}

let mock: MockAdapter
let previewBodies: unknown[]
let applyBodies: unknown[]

function stub(preview: (body: { sources?: unknown[] }) => unknown, applyStatus = 200, applyData: unknown = { added: 0, removed: 0 }) {
  mock.onPost('/chat/conversations/5/sync/preview').reply(config => {
    const body = JSON.parse(config.data ?? '{}')
    previewBodies.push(body)
    return [200, preview(body)]
  })
  mock.onPost('/chat/conversations/5/sync/apply').reply(config => {
    applyBodies.push(JSON.parse(config.data))
    return [applyStatus, applyData]
  })
}

beforeEach(() => {
  mock = setupApiMock()
  mock.reset()
  previewBodies = []
  applyBodies = []
})

describe('ConversationSyncModal', () => {
  test('erste Vorschau läuft ohne sources (gespeicherte Herkunft)', async () => {
    stub(() => previewWith())
    render(<ConversationSyncModal convId={5} onClose={() => {}} onApplied={() => {}} />)
    await screen.findByText('Neu Spieler')
    expect(previewBodies[0]).toEqual({})
    expect(screen.getByText(/Fehlende Personen/)).toBeTruthy()
  })

  test('Abbrechen ruft apply nicht', async () => {
    stub(() => previewWith())
    const onClose = vi.fn()
    render(<ConversationSyncModal convId={5} onClose={onClose} onApplied={() => {}} />)
    await screen.findByText('Alt Eins')
    fireEvent.click(screen.getByLabelText('Alt Eins'))
    fireEvent.click(screen.getByText('Abbrechen'))
    expect(onClose).toHaveBeenCalled()
    expect(applyBodies).toHaveLength(0)
  })

  test('OK schickt nur angehakte Personen und die Herkunft', async () => {
    stub(() => previewWith(), 200, { added: 1, removed: 1 })
    const onApplied = vi.fn()
    const onClose = vi.fn()
    render(<ConversationSyncModal convId={5} onClose={onClose} onApplied={onApplied} />)
    await screen.findByText('Alt Zwei')
    fireEvent.click(screen.getByLabelText('Alt Zwei'))
    fireEvent.click(screen.getByRole('button', { name: 'OK' }))
    await waitFor(() => expect(onApplied).toHaveBeenCalled())
    expect(applyBodies[0]).toEqual({
      sources: [{ groupType: 'team', refId: 7, kind: 'spieler' }],
      addUserIds: [11],
      removeUserIds: [21],
    })
    expect(onClose).toHaveBeenCalled()
  })

  test('Bestandsgruppe ohne Vorauswahl zeigt Überlappung', async () => {
    stub(() => previewWith({ sources: [], add: [], remove: [] }))
    render(<ConversationSyncModal convId={5} onClose={() => {}} onApplied={() => {}} />)
    await screen.findByText(/noch nicht festgelegt/)
    expect(screen.getByText('2 von 3 schon drin')).toBeTruthy()
    expect(screen.getByText('1 von 4 schon drin')).toBeTruthy()
    expect(screen.queryByRole('checkbox')).toBeNull()
  })

  test('Kachel hinzufügen lädt Vorschau neu, Abwahl bleibt erhalten', async () => {
    stub(body =>
      body.sources && body.sources.length === 2
        ? previewWith({ sources: [spielerT1, elternT1], add: [{ id: 11, name: 'Neu Spieler' }, { id: 12, name: 'Neu Elternteil' }] })
        : previewWith(),
    )
    render(<ConversationSyncModal convId={5} onClose={() => {}} onApplied={() => {}} />)
    await screen.findByText('Neu Spieler')
    fireEvent.click(screen.getByLabelText('Neu Spieler'))
    fireEvent.click(screen.getByText('Eltern mC1'))
    await screen.findByText('Neu Elternteil')
    expect(previewBodies[1]).toEqual({
      sources: [
        { groupType: 'team', refId: 7, kind: 'spieler' },
        { groupType: 'team', refId: 7, kind: 'eltern' },
      ],
    })
    expect((screen.getByLabelText('Neu Spieler') as HTMLInputElement).checked).toBe(false)
    expect((screen.getByLabelText('Neu Elternteil') as HTMLInputElement).checked).toBe(true)
  })

  test('blocked deaktiviert OK und zeigt das Problem an der Kachel', async () => {
    stub(() => previewWith({ sources: [{ ...spielerT1, total: 0, alreadyIn: 0, problem: 'empty' }], add: [], remove: [], blocked: true }))
    render(<ConversationSyncModal convId={5} onClose={() => {}} onApplied={() => {}} />)
    await screen.findByText('(derzeit leer)')
    expect((screen.getByRole('button', { name: 'OK' }) as HTMLButtonElement).disabled).toBe(true)
  })

  test('409 sync_stale lädt neu und zeigt einen Hinweis statt zu schließen', async () => {
    stub(() => previewWith(), 409, { error: 'sync_stale' })
    const onClose = vi.fn()
    render(<ConversationSyncModal convId={5} onClose={onClose} onApplied={() => {}} />)
    await screen.findByText('Neu Spieler')
    fireEvent.click(screen.getByRole('button', { name: 'OK' }))
    await screen.findByText(/inzwischen geändert/)
    expect(onClose).not.toHaveBeenCalled()
    expect(previewBodies).toHaveLength(2)
  })
})
