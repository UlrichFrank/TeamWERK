import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor, render, fireEvent } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import MockAdapter from 'axios-mock-adapter'
import { api } from '../../lib/api'
import StaffelnPage from '../StaffelnPage'

const liveHandlers: Array<(e: string) => void> = []
vi.mock('../../hooks/useLiveUpdates', () => ({
  useLiveUpdates: (fn: (e: string) => void) => { liveHandlers.push(fn) },
}))

let mock: MockAdapter

const staffeln = [
  { id: 1, code: 'mB-RL-BW', name: 'B-Jugend Regionalliga', teamName: 'B-Jugend männlich', kaderId: 10 },
  { id: 2, code: 'wC-OL-2-BW', name: 'C-Jugend Oberliga', teamName: 'C-Jugend weiblich', kaderId: 11 },
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

function setup(initial = '/staffeln') {
  return render(
    <MemoryRouter initialEntries={[initial]}>
      <Routes><Route path="/staffeln" element={<StaffelnPage />} /></Routes>
    </MemoryRouter>,
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
})
