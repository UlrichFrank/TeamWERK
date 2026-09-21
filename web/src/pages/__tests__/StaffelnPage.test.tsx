import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, render, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import StaffelnPage from '../StaffelnPage'
import { AuthContext, type AuthCtx } from '../../contexts/AuthContext'

const liveHandlers: Array<(e: string) => void> = []
vi.mock('../../hooks/useLiveUpdates', () => ({
  useLiveUpdates: (fn: (e: string) => void) => { liveHandlers.push(fn) },
}))

let mock: MockAdapter

const staffeln = [
  { id: 1, code: 'mB-RL-BW', name: 'B-Jugend Regionalliga', teamName: 'B-Jugend männlich', kaderId: 10, polled: true },
  { id: 2, code: 'wC-OL-2-BW', name: 'C-Jugend Oberliga', teamName: 'C-Jugend weiblich', kaderId: 11, polled: true },
]

// Zugeordnet, aber noch nie abgerufen: id 0, polled false.
const staffelnOhneAbruf = [
  { id: 0, code: 'mB-RL-BW', name: '', teamName: 'B-Jugend männlich', kaderId: 10, polled: false },
]

const table = [
  { Position: 1, TeamName: 'Verein A', Games: 2, Won: 2, Drawn: 0, Lost: 0, GoalsFor: 58, GoalsAgainst: 44, PointsPlus: 4, PointsMinus: 0 },
  { Position: 2, TeamName: 'Verein B', Games: 2, Won: 0, Drawn: 0, Lost: 2, GoalsFor: 44, GoalsAgainst: 58, PointsPlus: 0, PointsMinus: 4 },
]

const games = [
  { ID: 5, GameNo: '905272', SGID: '3504061', GameID: 7, Date: '2026-09-20', Time: '16:00',
    HomeTeam: 'Verein A', GuestTeam: 'Verein B', HomeGoals: 29, GuestGoals: 25,
    HomeGoalsHT: 14, GuestGoalsHT: 13, HallNumber: '21005', HasReport: true },
  { ID: 6, GameNo: '905275', SGID: '', GameID: null, Date: '2026-09-26', Time: '15:45',
    HomeTeam: 'Verein C', GuestTeam: 'Verein D', HomeGoals: null, GuestGoals: null,
    HomeGoalsHT: null, GuestGoalsHT: null, HallNumber: '5041', HasReport: false },
]

const stats = [
  { playerId: 1, memberId: 3, name: 'Alpha Spieler', teamName: 'Verein A', games: 2, goals: 12,
    sevenMAttempts: 4, sevenMGoals: 3, twoMin: 1, warnings: 0, disq: 0, fairPlayScore: 1 },
  { playerId: 2, memberId: null, name: 'Beta Spieler', teamName: 'Verein B', games: 2, goals: 5,
    sevenMAttempts: 0, sevenMGoals: 0, twoMin: 0, warnings: 2, disq: 0, fairPlayScore: 1 },
]

function mockAll() {
  mock.onGet('/staffeln').reply(200, staffeln)
  mock.onGet(/\/staffeln\/\d+\/tabelle/).reply(200, table)
  mock.onGet(/\/staffeln\/\d+\/spielplan/).reply(200, games)
  mock.onGet(/\/staffeln\/\d+\/ranglisten/).reply(200, stats)
}

const ctx = (caps: string[]): AuthCtx => ({
  user: { id: 1, email: 'a@test.local', role: 'standard', clubFunctions: [], isParent: false },
  loading: false, impersonating: null, mapsProvider: 'auto', setMapsProvider: () => {},
  capabilities: caps, hasCapability: (c: string) => caps.includes(c), navRoutes: [],
  passwordChangeRecommended: false, dismissPasswordChangeHint: () => {},
  keepAlive: () => {}, login: async () => {}, logout: async () => {},
  startImpersonation: async () => {}, stopImpersonation: async () => {},
})

function setup(initial = '/staffeln', caps: string[] = []) {
  return render(
    <AuthContext.Provider value={ctx(caps)}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes><Route path="/staffeln" element={<StaffelnPage />} /></Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  )
}

beforeEach(() => {
  mock = new MockAdapter(api, { onNoMatch: 'passthrough' })
  liveHandlers.length = 0
})
afterEach(() => { mock.restore(); vi.clearAllMocks() })

describe('StaffelnPage', () => {
  test('zeigt die Tabelle der ersten Staffel', async () => {
    mockAll()
    setup()
    await waitFor(() => expect(screen.getByText('Verein A')).toBeInTheDocument())
    expect(screen.getByText('58:44')).toBeInTheDocument()
    expect(screen.getByText('4:0')).toBeInTheDocument()
  })

  test('Umschalter listet alle Mannschaften und wechselt die Staffel', async () => {
    mockAll()
    setup()
    await waitFor(() => expect(screen.getByLabelText('Mannschaft wählen')).toBeInTheDocument())
    const select = screen.getByLabelText('Mannschaft wählen') as HTMLSelectElement
    expect(select.options).toHaveLength(2)

    fireEvent.change(select, { target: { value: '2' } })
    await waitFor(() => expect(screen.getByText(/C-Jugend Oberliga/)).toBeInTheDocument())
  })

  // Der Spielplan enthält fremde Begegnungen; nur eigene tragen eine
  // Verknüpfung und werden als solche markiert.
  test('Spielplan zeigt fremde Begegnungen und markiert eigene', async () => {
    mockAll()
    setup('/staffeln?tab=spielplan')
    await waitFor(() => expect(screen.getByText(/Verein C/)).toBeInTheDocument())
    expect(screen.getByText('29:25')).toBeInTheDocument()
    expect(screen.getByText(/eigenes Spiel/)).toBeInTheDocument()
  })

  test('Ranglisten sortieren nach Toren und zeigen die Siebenmeter-Quote', async () => {
    mockAll()
    setup('/staffeln?tab=ranglisten')
    await waitFor(() => expect(screen.getByText('Torschützen')).toBeInTheDocument())
    expect(screen.getByText('12 Tore')).toBeInTheDocument()
    expect(screen.getByText('75 % (3/4)')).toBeInTheDocument()
  })

  test('ohne zugeordnete Staffel erscheint ein Hinweis statt einer leeren Seite', async () => {
    mock.onGet('/staffeln').reply(200, [])
    setup()
    await waitFor(() => expect(screen.getByText(/noch keiner Mannschaft eine Staffel/)).toBeInTheDocument())
  })

  test('lädt bei bwhv-updated nach', async () => {
    mockAll()
    setup()
    await waitFor(() => expect(screen.getByText('Verein A')).toBeInTheDocument())
    const vorher = mock.history.get.length

    liveHandlers.forEach(fn => fn('bwhv-updated'))
    await waitFor(() => expect(mock.history.get.length).toBeGreaterThan(vorher))
  })

  test('fremde SSE-Ereignisse lösen keinen Nachladevorgang aus', async () => {
    mockAll()
    setup()
    await waitFor(() => expect(screen.getByText('Verein A')).toBeInTheDocument())
    const vorher = mock.history.get.length

    liveHandlers.forEach(fn => fn('games'))
    expect(mock.history.get.length).toBe(vorher)
  })

  // Der manuelle Abruf greift nach außen und hängt deshalb an der Capability
  // poll_bwhv (Vorstand/Admin) — das Lesen steht allen offen.
  test('Abruf-Knopf nur mit Capability poll_bwhv', async () => {
    mockAll()
    setup('/staffeln')
    await waitFor(() => expect(screen.getByText('Verein A')).toBeInTheDocument())
    expect(screen.queryByText('Jetzt abrufen')).not.toBeInTheDocument()
  })

  test('mit Capability wird der Abruf angestoßen', async () => {
    mockAll()
    mock.onPost(/\/staffeln\/\d+\/poll/).reply(200, { status: 'gestartet' })
    setup('/staffeln', ['poll_bwhv'])
    await waitFor(() => expect(screen.getByText('Jetzt abrufen')).toBeInTheDocument())

    fireEvent.click(screen.getByText('Jetzt abrufen'))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(await screen.findByText(/Abruf gestartet/)).toBeInTheDocument()
  })

  test('ein fehlgeschlagener Abruf wird gemeldet', async () => {
    mockAll()
    mock.onPost(/\/staffeln\/\d+\/poll/).reply(403, { error: 'forbidden' })
    setup('/staffeln', ['poll_bwhv'])
    await waitFor(() => expect(screen.getByText('Jetzt abrufen')).toBeInTheDocument())

    fireEvent.click(screen.getByText('Jetzt abrufen'))
    expect(await screen.findByText(/konnte nicht gestartet werden/)).toBeInTheDocument()
  })

  // Der gemeldete Fehler: eine im Kader gepflegte Staffel erschien nicht, weil
  // die Liste aus dem Abruf-Snapshot statt aus der Zuordnung kam.
  test('zeigt eine zugeordnete Staffel auch ohne Abruf', async () => {
    mock.onGet('/staffeln').reply(200, staffelnOhneAbruf)
    setup()
    await waitFor(() => expect(screen.getByText('mB-RL-BW')).toBeInTheDocument())
    expect(screen.queryByText(/noch keiner Mannschaft eine Staffel zugeordnet/)).not.toBeInTheDocument()
    expect(screen.getByText(/noch nichts beim Verband abgerufen/)).toBeInTheDocument()
  })

  test('ohne Snapshot stößt der Knopf die Einrichtung an', async () => {
    mock.onGet('/staffeln').reply(200, staffelnOhneAbruf)
    mock.onPost('/staffeln/sync').reply(200, { status: 'gestartet' })
    setup('/staffeln', ['poll_bwhv'])
    await waitFor(() => expect(screen.getByText('Jetzt abrufen')).toBeInTheDocument())

    fireEvent.click(screen.getByText('Jetzt abrufen'))
    await waitFor(() => expect(mock.history.post).toHaveLength(1))
    expect(mock.history.post[0].url).toBe('/staffeln/sync')
  })

  // Der gemeldete Fehler: nach dem Abruf aktualisierte sich nichts. Ursache
  // war, dass das Live-Update nur die Daten der gewählten Staffel nachlud,
  // nicht die Zuordnungsliste — dort wechselt eine Staffel aber von id 0 auf
  // eine echte ID, und bis dahin blockt die Reload-Logik.
  test('Live-Update lädt auch die Zuordnungsliste nach', async () => {
    mock.onGet('/staffeln').replyOnce(200, staffelnOhneAbruf)
    mock.onGet('/staffeln').reply(200, staffeln)
    mock.onGet(/\/staffeln\/\d+\/tabelle/).reply(200, table)
    mock.onGet(/\/staffeln\/\d+\/spielplan/).reply(200, games)
    mock.onGet(/\/staffeln\/\d+\/ranglisten/).reply(200, stats)

    setup()
    await waitFor(() => expect(screen.getByText(/noch nichts beim Verband abgerufen/)).toBeInTheDocument())

    liveHandlers.forEach(fn => fn('bwhv-updated'))

    // Nach dem Nachladen trägt die Staffel eine ID und die Tabelle erscheint.
    await waitFor(() => expect(screen.getByText('Verein A')).toBeInTheDocument())
    expect(screen.queryByText(/noch nichts beim Verband abgerufen/)).not.toBeInTheDocument()
  })
})
