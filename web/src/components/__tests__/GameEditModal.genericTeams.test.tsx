import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import GameEditModal from '../GameEditModal'

// Generische Events sind mannschaftsübergreifend: der Picker speist sich aus der
// vereinsweiten Liste (/teams/names), damit auch ein reiner Trainer fremde
// Mannschaften einladen kann. Heim-/Auswärtsspiele bleiben auf die
// nutzergefilterte Liste (/teams) beschränkt. Siehe game-edit-modal-Spec.

let mock: MockAdapter

// /teams — nutzergefiltert: der Trainer sieht nur seine eigene Mannschaft.
const OWN_TEAMS = [
  { id: 1, name: 'mA-Jugend', age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 1, is_active: true },
]

// /teams/names — vereinsweit: alle aktiven Mannschaften der aktiven Saison.
const CLUB_TEAMS = [
  { id: 1, age_class: 'A-Jugend', gender: 'm', team_number: 1, group_count: 1 },
  { id: 2, age_class: 'B-Jugend', gender: 'f', team_number: 1, group_count: 1 },
  { id: 3, age_class: 'C-Jugend', gender: 'm', team_number: 1, group_count: 1 },
]

const GENERIC_EVENT = {
  id: 42,
  date: '2026-06-20',
  time: '14:00',
  opponent: 'Vereinsfest',
  event_type: 'generisch',
  teams: [{ id: 1, name: 'mA-Jugend' }],
}

const HOME_GAME = {
  id: 43,
  date: '2026-09-10',
  time: '18:00',
  opponent: 'FC Test',
  event_type: 'heim',
  teams: [{ id: 1, name: 'mA-Jugend' }],
}

// Der Mannschafts-Block; grenzt die Abfragen gegen die RSVP-Checkbox weiter unten ab.
const teamPicker = (label: 'Mannschaften' | 'Mannschaft') =>
  within(screen.getByText(label).parentElement!)

beforeEach(() => {
  mock = new MockAdapter(api)
  mock.onGet('/teams').reply(200, OWN_TEAMS)
  mock.onGet('/teams/names').reply(200, CLUB_TEAMS)
  mock.onGet('/duty-templates').reply(200, [])
})
afterEach(() => {
  mock.restore()
})

describe('GameEditModal — Mannschafts-Picker nach Event-Typ', () => {
  test('generisches Event bietet alle aktiven Mannschaften des Vereins an', async () => {
    render(<GameEditModal game={GENERIC_EVENT} onClose={() => {}} onSaved={() => {}} />)

    // Kurznamen aus buildTeamShortNames: mA (eigene), wB und mC (fremde).
    await screen.findByText('mA')
    expect(screen.getByText('wB')).toBeInTheDocument()
    expect(screen.getByText('mC')).toBeInTheDocument()

    expect(teamPicker('Mannschaften').getAllByRole('checkbox')).toHaveLength(3)
  })

  test('beteiligte Mannschaft ist vorausgewählt, fremde nicht', async () => {
    render(<GameEditModal game={GENERIC_EVENT} onClose={() => {}} onSaved={() => {}} />)
    await screen.findByText('mA')

    const checked = (label: string) =>
      (screen.getByText(label).closest('label')!.querySelector('input')! as HTMLInputElement).checked

    expect(checked('mA')).toBe(true)
    expect(checked('wB')).toBe(false)
    expect(checked('mC')).toBe(false)
  })

  test('fremde Mannschaft anhaken landet in team_ids des PUT', async () => {
    const onSaved = vi.fn()
    mock.onPut('/games/42').reply(200, {})
    render(<GameEditModal game={GENERIC_EVENT} onClose={() => {}} onSaved={onSaved} />)
    await screen.findByText('wB')

    fireEvent.click(screen.getByText('wB').closest('label')!.querySelector('input')!)
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await waitFor(() => expect(onSaved).toHaveBeenCalled())
    const body = JSON.parse(mock.history.put[0].data)
    expect(body.team_ids).toEqual([1, 2])
  })

  test('Heimspiel zeigt nur die eigenen Mannschaften im Single-Select', async () => {
    render(<GameEditModal game={HOME_GAME} onClose={() => {}} onSaved={() => {}} />)
    await screen.findByText('Gegner')

    await waitFor(() => expect(screen.queryAllByRole('option').length).toBeGreaterThan(0))
    const teamOptions = screen
      .getAllByRole('option')
      .filter(o => (o as HTMLOptionElement).value !== '' && !/^— /.test(o.textContent ?? ''))
      .map(o => o.textContent)

    // Nur die eigene Mannschaft — wB/mC aus /teams/names dürfen hier nicht auftauchen.
    expect(teamOptions).toContain('mA')
    expect(teamOptions).not.toContain('wB')
    expect(teamOptions).not.toContain('mC')
    // Kein Checkbox-Multi-Select bei Heimspielen.
    expect(teamPicker('Mannschaft').queryAllByRole('checkbox')).toHaveLength(0)
  })

  test('403 beim Speichern zeigt die Team-Scope-Meldung, Modal bleibt offen', async () => {
    const onSaved = vi.fn()
    mock.onPut('/games/42').reply(403, 'forbidden')
    render(<GameEditModal game={GENERIC_EVENT} onClose={() => {}} onSaved={onSaved} />)
    await screen.findByText('mA')

    // Eigene Mannschaft abwählen → der Server lehnt ab.
    fireEvent.click(screen.getByText('mA').closest('label')!.querySelector('input')!)
    fireEvent.click(screen.getByRole('button', { name: 'Speichern' }))

    await screen.findByText('Mindestens eine deiner Mannschaften muss am Event beteiligt bleiben.')
    expect(onSaved).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Speichern' })).toBeInTheDocument()
  })
})
