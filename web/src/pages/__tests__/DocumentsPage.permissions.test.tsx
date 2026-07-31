/**
 * PermissionsModal — Team-/Eltern-Principals und der Eigentümer-Eintrag.
 *
 * Quelle: openspec/changes/folder-owner-and-team-principals/specs/folder-permission-ux/spec.md
 */
import { describe, test, expect, beforeEach, afterEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import { PermissionsModal } from '../DocumentsPage'

// group_count = 2 → Kurzname trägt die Team-Nummer: id 7 → "mA1", id 8 → "mA2".
const TEAMS = [
  { id: 7, age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 2 },
  { id: 8, age_class: 'A-Jugend', gender: 'm', team_number: 2, group_count: 2 },
]

const OWNER_ENTRY = {
  id: 0,
  principal_type: 'owner',
  principal_ref: '3',
  display_name: 'Florian Steinle',
  can_read: true,
  can_write: true,
}

let mock: MockAdapter

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  mock.onGet('/teams/names').reply(200, TEAMS)
})

afterEach(() => {
  mock.restore()
})

function renderModal() {
  return render(<PermissionsModal folderId={9} canWrite onClose={() => {}} />)
}

describe('PermissionsModal — Mannschaftsauswahl', () => {
  test('Wechsel auf "Team" rendert das Mannschafts-Dropdown; /teams/names nur einmal', async () => {
    mock.onGet('/folders/9/permissions').reply(200, [OWNER_ENTRY])
    renderModal()
    await screen.findByText(/Eigentümer/)

    const typeSelect = screen.getByDisplayValue('Alle Nutzer')
    fireEvent.change(typeSelect, { target: { value: 'team' } })

    await screen.findByText('Mannschaft wählen…')
    expect(screen.getByText('mA1')).toBeTruthy()
    expect(screen.getByText('mA2')).toBeTruthy()

    // Erneuter Typwechsel darf nicht nachladen (loadTeams ist idempotent).
    fireEvent.change(typeSelect, { target: { value: 'user' } })
    fireEvent.change(typeSelect, { target: { value: 'team_parents' } })
    await screen.findByText('Mannschaft wählen…')

    const teamCalls = mock.history.get.filter(r => r.url === '/teams/names')
    expect(teamCalls.length).toBe(1)
  })

  test('Absenden mit "Eltern" sendet team_parents und die teams.id', async () => {
    mock.onGet('/folders/9/permissions').reply(200, [OWNER_ENTRY])
    mock.onPost('/folders/9/permissions').reply(201, { id: 5 })
    renderModal()
    await screen.findByText(/Eigentümer/)

    fireEvent.change(screen.getByDisplayValue('Alle Nutzer'), { target: { value: 'team_parents' } })
    await screen.findByText('Mannschaft wählen…')
    fireEvent.change(screen.getByDisplayValue('Mannschaft wählen…'), { target: { value: '7' } })
    fireEvent.click(screen.getByText('Hinzufügen'))

    await waitFor(() => {
      expect(mock.history.post.length).toBe(1)
    })
    expect(JSON.parse(mock.history.post[0].data)).toMatchObject({
      principal_type: 'team_parents',
      principal_ref: '7',
      can_read: true,
      can_write: false,
    })
  })

  test('Bestandseintrag team/7 wird als "Team: mA1" gerendert', async () => {
    mock.onGet('/folders/9/permissions').reply(200, [
      OWNER_ENTRY,
      { id: 4, principal_type: 'team', principal_ref: '7', display_name: 'mA-Jugend', can_read: true, can_write: false },
    ])
    renderModal()

    await waitFor(() => {
      expect(screen.getByText('Team: mA1')).toBeTruthy()
    })
  })

  test('Ohne geladene Teamliste fällt die Anzeige auf display_name zurück', async () => {
    mock.onGet('/teams/names').reply(500)
    mock.onGet('/folders/9/permissions').reply(200, [
      { id: 4, principal_type: 'team', principal_ref: '7', display_name: 'mA-Jugend', can_read: true, can_write: false },
    ])
    renderModal()

    await waitFor(() => {
      expect(screen.getByText('Team: mA-Jugend')).toBeTruthy()
    })
  })
})

describe('PermissionsModal — Eigentümer-Eintrag', () => {
  test('wird ohne Löschen-Button gerendert, normale Einträge behalten ihn', async () => {
    mock.onGet('/folders/9/permissions').reply(200, [
      OWNER_ENTRY,
      { id: 4, principal_type: 'everyone', principal_ref: '', can_read: true, can_write: false },
    ])
    renderModal()

    const ownerRow = (await screen.findByText('Eigentümer: Florian Steinle')).closest('div')?.parentElement
    expect(ownerRow?.querySelector('[aria-label="Entfernen"]')).toBeNull()

    // "Alle Nutzer" steht auch als <option> im Typ-Select — nur die Listenzeile (<span>) zählt.
    const everyoneLabel = screen.getAllByText('Alle Nutzer').find(el => el.tagName === 'SPAN')
    const everyoneRow = everyoneLabel?.closest('div')?.parentElement
    expect(everyoneRow?.querySelector('[aria-label="Entfernen"]')).not.toBeNull()
  })
})
