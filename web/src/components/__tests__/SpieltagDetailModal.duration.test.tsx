import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import SpieltagDetailModal from '../SpieltagDetailModal'

// Teil von openspec/changes/dienst-dauer: "+ Dienst hinzufügen" belegt Dauer UND
// Uhrzeit aus dem gewählten Diensttyp vor (Uhrzeit = Zeit des Termins +
// default_offset_minutes, Anker "Start"), beide Felder bleiben editierbar.

vi.mock('../../hooks/useLiveUpdates', () => ({ useLiveUpdates: vi.fn() }))
vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 1, email: 'test@test.local', role: 'admin', clubFunctions: [] },
    hasCapability: () => true,
  }),
}))

let mock: MockAdapter

// Anker "Start", Versatz -30min, Dauer 1,5h — deckt exakt das Szenario aus
// specs/duties/spec.md ("Dienst hinzufügen belegt Dauer und Uhrzeit vor").
const DUTY_TYPES = [
  { id: 9, name: 'Kasse', hours_value: 1.5, default_anchor: 'start', default_offset_minutes: -30, audiences: [] },
]

function mockGame() {
  mock.onGet('/games/50').reply(200, {
    game: {
      id: 50,
      date: '2026-09-13',
      time: '10:00',
      opponent: 'Testgegner',
      event_type: 'heim',
      team_id: 1,
      teams: [{ id: 1, name: 'A-Jugend' }],
      season_id: 2,
      can: { edit: true, delete: true, manage_lineup: false },
    },
    slots: [],
  })
  mock.onGet(/\/duty-board/).reply(200, [])
  mock.onGet('/duty-types').reply(200, DUTY_TYPES)
  mock.onPost('/duty-slots').reply(201)
}

async function openAddModal(gameId = 50) {
  render(
    <MemoryRouter>
      <SpieltagDetailModal gameId={gameId} onClose={() => {}} />
    </MemoryRouter>,
  )
  const user = userEvent.setup()
  await screen.findByText('+ Dienst hinzufügen')
  await user.click(screen.getByText('+ Dienst hinzufügen'))
  return user
}

// openspec/changes/dienst-dauer-dynamisch, Aufgabe 8: ein Diensttyp im Modus
// 'dynamisch' rechnet die Dauer gegen den konkreten Termin aus (Start-Anker +
// Versatz und End-Anker + Versatz gegen game.time bzw. game.end_time), statt
// die gepflegte hours_value zu übernehmen. Der so entstandene Slot bleibt
// trotzdem is_custom=1/absolut — kein Modus-Umschalter im Modal.
const DUTY_TYPES_DYNAMIC = [
  {
    id: 11, name: 'Zeitnehmer', hours_value: 1, default_anchor: 'start', default_offset_minutes: -30,
    duration_mode: 'dynamisch', end_anchor: 'end', end_offset_minutes: 15, audiences: [],
  },
]

function mockDynamicGame() {
  mock.onGet('/games/51').reply(200, {
    game: {
      id: 51,
      date: '2026-09-13',
      time: '10:00',
      end_time: '11:30',
      opponent: 'Vereinsfest',
      event_type: 'generisch',
      team_id: 1,
      teams: [{ id: 1, name: 'A-Jugend' }],
      season_id: 2,
      can: { edit: true, delete: true, manage_lineup: false },
    },
    slots: [],
  })
  mock.onGet(/\/duty-board/).reply(200, [])
  mock.onGet('/duty-types').reply(200, DUTY_TYPES_DYNAMIC)
  mock.onPost('/duty-slots').reply(201)
}

beforeEach(() => {
  mock = new MockAdapter(api)
  mockGame()
})
afterEach(() => {
  mock.restore()
})

describe('SpieltagDetailModal — Dauer und Uhrzeit aus dem Diensttyp vorbelegen', () => {
  test('Diensttyp-Auswahl füllt Uhrzeit (Anker + Versatz) und Dauer', async () => {
    const user = await openAddModal()
    await user.selectOptions(screen.getAllByRole('combobox')[0], '9')

    const timeInput = document.querySelector('input[type="time"]') as HTMLInputElement
    expect(timeInput.value).toBe('09:30')

    const hoursInput = screen.getByPlaceholderText('z.B. 1h 30min') as HTMLInputElement
    expect(hoursInput.value).toBe('1h 30min')
  })

  test('eine manuelle Dauer-Änderung überschreibt die Vorbelegung und überlebt einen erneuten Render', async () => {
    const user = await openAddModal()
    await user.selectOptions(screen.getAllByRole('combobox')[0], '9')

    const hoursInput = screen.getByPlaceholderText('z.B. 1h 30min') as HTMLInputElement
    await user.clear(hoursInput)
    await user.type(hoursInput, '2h')
    await user.tab() // blur -> committed
    expect(hoursInput.value).toBe('2h')

    // Ein anderes Feld ändern löst einen erneuten Render des Modals aus, ohne
    // dass die Diensttyp-Auswahl erneut greift — die manuelle Dauer bleibt.
    const personenInput = screen.getByDisplayValue('1') as HTMLInputElement
    await user.clear(personenInput)
    await user.type(personenInput, '3')

    expect(hoursInput.value).toBe('2h')

    await user.click(screen.getByRole('button', { name: 'Hinzufügen' }))
    await waitFor(() => expect(mock.history.post.length).toBe(1))
    const body = JSON.parse(mock.history.post[0].data)
    expect(body.hours_value).toBe(2)
  })
})

describe('SpieltagDetailModal — dynamischer Diensttyp wird gegen den Termin ausgerechnet', () => {
  beforeEach(() => {
    mockDynamicGame()
  })

  // Zeitnehmer: Start = Anpfiff 10:00 − 30 min = 09:30, Ende = Spielende 11:30
  // + 15 min = 11:45 → 135 min = 2h 15min. Entscheidend ist, dass NICHT die
  // gepflegte hours_value (1 h) übernommen wird: genau darin unterscheidet sich
  // der dynamische Modus.
  test('Dauer folgt Anker und Versatz statt der gepflegten hours_value', async () => {
    const user = await openAddModal(51)
    await user.selectOptions(screen.getAllByRole('combobox')[0], '11')

    const timeInput = document.querySelector('input[type="time"]') as HTMLInputElement
    expect(timeInput.value).toBe('09:30')

    const hoursInput = screen.getByPlaceholderText('z.B. 1h 30min') as HTMLInputElement
    expect(hoursInput.value).toBe('2h 15min')
  })

  // Der so entstandene Slot ist is_custom=1 und trägt eine feste Zahl (spec.md,
  // „Manuell angelegte Dienste bleiben absolut"): die ausgerechnete Dauer ist
  // eine Vorbelegung, kein gebundener Wert.
  test('die ausgerechnete Dauer bleibt frei editierbar und geht so in den Request', async () => {
    const user = await openAddModal(51)
    await user.selectOptions(screen.getAllByRole('combobox')[0], '11')

    const hoursInput = screen.getByPlaceholderText('z.B. 1h 30min') as HTMLInputElement
    await user.clear(hoursInput)
    await user.type(hoursInput, '45min')
    await user.tab()
    expect(hoursInput.value).toBe('45min')

    await user.click(screen.getByRole('button', { name: 'Hinzufügen' }))
    await waitFor(() => expect(mock.history.post.length).toBe(1))
    const body = JSON.parse(mock.history.post[0].data)
    expect(body.hours_value).toBeCloseTo(0.75)
    // Kein Modus-Feld im Payload — der Modus ist eine Eigenschaft des Diensttyps
    // und der Vorlage, nicht des Slots.
    expect(body.duration_mode).toBeUndefined()
  })
})

// dienst-zeitmodus-strikt, Aufgabe 6: Anlegen UND Bearbeiten im Termin-Dialog setzen
// is_custom=1 — der Dienst verlässt damit die automatische Regeneration. Beim
// Bearbeiten eines automatisch erzeugten Dienstes ist das eine Nebenwirkung des
// Speicherns und war bisher nirgends sichtbar; wer nur die Uhrzeit korrigierte, nahm
// den Dienst ungewollt dauerhaft aus dem Regen.
describe('SpieltagDetailModal — Hinweis auf die Herausnahme aus der Regeneration', () => {
  test('Anlege-Dialog nennt die manuelle Pflege', async () => {
    await openAddModal()
    expect(screen.getByText(/manuell gepflegt/)).toBeTruthy()
    expect(screen.getByText(/automatischen Regeneration/)).toBeTruthy()
  })

  test('Bearbeiten-Dialog warnt vor der Nebenwirkung des Speicherns', async () => {
    mock.onGet('/games/50').reply(200, {
      game: {
        id: 50, date: '2026-09-13', time: '10:00', opponent: 'Testgegner',
        event_type: 'heim', team_id: 1, teams: [{ id: 1, name: 'A-Jugend' }],
        season_id: 2, can: { edit: true, delete: true, manage_lineup: false },
      },
      slots: [{
        id: 77, duty_type_name: 'Kasse', event_time: '09:30', hours_value: 1.5,
        role_description: '', slots_total: 1, slots_filled: 0, audiences: [],
      }],
    })
    mock.onGet(/\/duty-board/).reply(200, [{
      slots: [{
        id: 77, duty_type: 'Kasse', duty_type_id: 9, has_instruction: false,
        event_time: '09:30', hours_value: 1.5, slots_total: 1, vacancies: 1,
        claimed_by_me: false, audiences: [], assignees: [],
      }],
    }])

    render(
      <MemoryRouter>
        <SpieltagDetailModal gameId={50} onClose={() => {}} />
      </MemoryRouter>,
    )
    const user = userEvent.setup()
    await user.click(await screen.findByText('Bearbeiten'))

    await screen.findByText('Dienst bearbeiten')
    expect(screen.getByText(/Nach dem Speichern gilt dieser Dienst als manuell gepflegt/)).toBeTruthy()
  })
})
