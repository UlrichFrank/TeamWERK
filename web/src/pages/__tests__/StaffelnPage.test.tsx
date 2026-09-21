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
  { id: 1, code: 'mB-RL-BW', name: 'B-Jugend Regionalliga', teamName: 'B-Jugend männlich', teamShort: 'mB1', kaderId: 10, polled: true },
  { id: 2, code: 'wC-OL-2-BW', name: 'C-Jugend Oberliga', teamName: 'C-Jugend weiblich', teamShort: 'wC', kaderId: 11, polled: true },
]

// Zugeordnet, aber noch nie abgerufen: id 0, polled false.
const staffelnOhneAbruf = [
  { id: 0, code: 'mB-RL-BW', name: '', teamName: 'B-Jugend männlich', teamShort: 'mB1', kaderId: 10, polled: false },
]

const table = [
  { Position: 1, TeamName: 'Verein A', Games: 2, Won: 2, Drawn: 0, Lost: 0, GoalsFor: 58, GoalsAgainst: 44, PointsPlus: 4, PointsMinus: 0 },
  { Position: 2, TeamName: 'Verein B', Games: 2, Won: 0, Drawn: 0, Lost: 2, GoalsFor: 44, GoalsAgainst: 58, PointsPlus: 0, PointsMinus: 4 },
]

const games = [
  { ID: 5, GameNo: '905272', SGID: '3504061', GameID: 7, Date: '2026-09-20', Time: '16:00',
    HomeTeam: 'Verein A', GuestTeam: 'Verein B', HomeGoals: 29, GuestGoals: 25,
    HomeGoalsHT: 14, GuestGoalsHT: 13, HallNumber: '21005', HasReport: true,
    Venue: { name: 'Sporthalle Nord', street: 'Hallenweg 1', city: 'Stuttgart', postal_code: '70000' } },
  { ID: 6, GameNo: '905275', SGID: '', GameID: null, Date: '2026-09-26', Time: '15:45',
    HomeTeam: 'Verein C', GuestTeam: 'Verein D', HomeGoals: null, GuestGoals: null,
    HomeGoalsHT: null, GuestGoalsHT: null, HallNumber: '5041', HasReport: false, Venue: null },
]

const stats = [
  { playerId: 1, memberId: 3, name: 'Alpha Spieler', teamName: 'Verein A', games: 2, goals: 12,
    sevenMAttempts: 4, sevenMGoals: 3, sevenMMissed: 1, twoMin: 1, warnings: 0, disq: 0, fairPlayScore: 1 },
  { playerId: 2, memberId: null, name: 'Beta Spieler', teamName: 'Verein B', games: 2, goals: 5,
    sevenMAttempts: 0, sevenMGoals: 0, sevenMMissed: 0, twoMin: 0, warnings: 2, disq: 0, fairPlayScore: 1 },
]

const cross = {
  teams: ['Verein A', 'Verein B'],
  rows: [
    {
      team: 'Verein A',
      cells: [null, { bwhvGameId: 5, played: true, homeGoals: 29, guestGoals: 25, date: '' }],
    },
    {
      team: 'Verein B',
      cells: [{ bwhvGameId: 6, played: false, homeGoals: null, guestGoals: null, date: '2026-10-04' }, null],
    },
  ],
}

const progression = [
  {
    date: '2026-09-20',
    entries: [
      { team: 'Verein A', rank: 1, points: 2, games: 1, goalsFor: 29, goalDiff: 4 },
      { team: 'Verein B', rank: 2, points: 0, games: 1, goalsFor: 25, goalDiff: -4 },
    ],
  },
]

const teamStats = {
  fairPlayWeights: { yellow: 1, twoMin: 2, red: 3, blue: 4 },
  teams: [
    { team: 'Verein A', games: 2, goalsFor: 58, goalsAgainst: 44, goalDiff: 14,
      reportGames: 1, twoMin: 1, yellow: 2, red: 0, blue: 0, fairPlayScore: 4,
      distribution: { players: 5, average: 5.8, median: 4, gini: 0.31 } },
    { team: 'Verein B', games: 2, goalsFor: 44, goalsAgainst: 58, goalDiff: -14,
      reportGames: 0, twoMin: 0, yellow: 0, red: 0, blue: 0, fairPlayScore: null,
      distribution: null },
  ],
}

const referees = [
  { name: 'Max Mustermann', games: 2, twoMin: 6, yellow: 2, red: 0, blue: 0, uncertain: false },
  { name: 'Peter Müller', games: 1, twoMin: 3, yellow: 1, red: 0, blue: 0, uncertain: true },
]

// Der Nutzer gehört zu Verein A und ist dort selbst Spieler 1.
const affiliation = { teamNames: ['Verein A'], playerIds: [1] }

function mockStats(aff = affiliation) {
  mock.onGet(/\/staffeln\/\d+\/kreuztabelle/).reply(200, cross)
  mock.onGet(/\/staffeln\/\d+\/tabellenverlauf/).reply(200, progression)
  mock.onGet(/\/staffeln\/\d+\/teamstatistik/).reply(200, teamStats)
  mock.onGet(/\/staffeln\/\d+\/schiedsrichter/).reply(200, referees)
  mock.onGet(/\/staffeln\/\d+\/affiliation/).reply(200, aff)
}

function mockAll(aff = affiliation) {
  mock.onGet('/staffeln').reply(200, staffeln)
  mock.onGet(/\/staffeln\/\d+\/tabelle$/).reply(200, table)
  mock.onGet(/\/staffeln\/\d+\/tabellenverlauf/).reply(200, progression)
  mock.onGet(/\/staffeln\/\d+\/spielplan/).reply(200, games)
  mock.onGet(/\/staffeln\/\d+\/ranglisten/).reply(200, stats)
  mockStats(aff)
}

const ctx = (caps: string[], clubFunctions: string[] = []): AuthCtx => ({
  user: { id: 1, email: 'a@test.local', role: 'standard', clubFunctions, isParent: false },
  loading: false, impersonating: null, mapsProvider: 'auto', setMapsProvider: () => {},
  capabilities: caps, hasCapability: (c: string) => caps.includes(c), navRoutes: [],
  passwordChangeRecommended: false, dismissPasswordChangeHint: () => {},
  keepAlive: () => {}, login: async () => {}, logout: async () => {},
  startImpersonation: async () => {}, stopImpersonation: async () => {},
})

function setup(initial = '/staffeln', caps: string[] = [], clubFunctions: string[] = []) {
  return render(
    <AuthContext.Provider value={ctx(caps, clubFunctions)}>
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
  test('die Staffelauswahl nennt die Mannschaften mit Kurznamen', async () => {
    mockAll()
    setup()
    const select = await screen.findByLabelText('Mannschaft wählen')
    expect(select).toHaveTextContent('mB1')
    expect(select).toHaveTextContent('wC')
    expect(select).not.toHaveTextContent('B-Jugend männlich')
  })

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

  // Bekannte Hallen erscheinen mit Namen und Karten-Link, unbekannte nur als Nummer.
  test('Spielplan verlinkt die Halle und fällt ohne Halle auf die Nummer zurück', async () => {
    mockAll()
    setup('/staffeln?tab=spielplan')
    const link = await screen.findByRole('link', { name: /Sporthalle Nord/ })
    expect(link.getAttribute('href')).toContain('q=Hallenweg%201%2070000%20Stuttgart')
    expect(screen.getByText(/Halle 5041/)).toBeInTheDocument()
  })

  // Der Bericht ist über den Spielplan erreichbar: Klick klappt ihn auf und
  // lädt ihn für genau diese Begegnung, ein zweiter Klick klappt ihn zu.
  test('Spielplan: „Bericht“ klappt den Spielbericht der Begegnung auf und zu', async () => {
    mockAll()
    mock.onGet('/bwhv-games/5/report').reply(200, {
      reportId: 1, state: 'parsed', homeTeam: 'Verein A', guestTeam: 'Verein B',
      homeGoals: 29, guestGoals: 25, homeGoalsHt: 14, guestGoalsHt: 13, spectators: '120',
      referees: '', refereeNames: [], refereesUncertain: false, warnings: [], hasPdf: false,
      players: [], events: [],
    })
    setup('/staffeln?tab=spielplan')
    const toggle = await screen.findByRole('button', { name: /Bericht/ })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(toggle)
    await waitFor(() => expect(screen.getByText(/Zuschauer:/)).toBeInTheDocument())
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    fireEvent.click(toggle)
    await waitFor(() => expect(screen.queryByText(/Zuschauer:/)).not.toBeInTheDocument())
  })

  // "Nur Audience" ist dem Vorstand und den Trainern vorbehalten und lädt die
  // Liste mit audience=own neu; ein normaler Nutzer sieht den Schalter nicht.
  test('Nur Audience: Schalter für Vorstand, lädt die eigene Staffelliste', async () => {
    // Der spezifischere Handler zuerst: axios-mock-adapter nimmt den ersten Treffer.
    mock.onGet('/staffeln', { params: { audience: 'own' } }).reply(200, [staffeln[1]])
    mockAll()
    setup('/staffeln', [], ['vorstand'])
    const pill = await screen.findByRole('button', { name: 'Nur meine Audience' })
    expect(pill).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(pill)
    await waitFor(() => expect(pill).toHaveAttribute('aria-pressed', 'true'))
    await waitFor(() => {
      const select = screen.getByLabelText('Mannschaft wählen')
      expect(select).toHaveTextContent('wC')
      expect(select).not.toHaveTextContent('mB1')
    })
  })

  test('Nur Audience: kein Schalter ohne Vorstands-/Trainerfunktion', async () => {
    mockAll()
    setup('/staffeln', [], ['spieler'])
    await screen.findByLabelText('Mannschaft wählen')
    expect(screen.queryByRole('button', { name: 'Nur meine Audience' })).not.toBeInTheDocument()
  })

  test('Nur Audience ohne eigene Staffel bietet den Rückweg zu allen Staffeln', async () => {
    mock.onGet('/staffeln', { params: { audience: 'own' } }).reply(200, [])
    mock.onGet('/staffeln').reply(200, staffeln)
    mockStats()
    mock.onGet(/\/staffeln\/\d+\/(tabelle|spielplan|ranglisten)$/).reply(200, [])
    setup('/staffeln?audience=own', [], ['vorstand'])
    fireEvent.click(await screen.findByRole('button', { name: 'Alle Staffeln anzeigen' }))
    await screen.findByLabelText('Mannschaft wählen')
  })

  // Gleichauf heißt gleicher Platz: 8, 8, 3 Tore ergeben 1, 1, 3 — nicht 1, 2, 3.
  test('Torschützen-Rangliste teilt den Platz bei Gleichstand', async () => {
    mockAll()
    const p = (id: number, name: string, goals: number) => ({
      playerId: id, memberId: null, name, teamName: 'Verein A', games: 2, goals,
      sevenMAttempts: 0, sevenMGoals: 0, sevenMMissed: 0, twoMin: 0, warnings: 0, disq: 0, fairPlayScore: 0,
    })
    mock.onGet(/\/staffeln\/\d+\/ranglisten/).reply(200, [p(1, 'Anna', 8), p(2, 'Berta', 8), p(3, 'Cora', 3)])
    setup('/staffeln?tab=ranglisten')
    const card = (await screen.findByText('Torschützen')).closest('.overflow-hidden') as HTMLElement
    const ranks = Array.from(card.querySelectorAll('li')).map((li) => li.querySelector('span span')?.textContent)
    expect(ranks).toEqual(['1', '1', '3'])
  })

  test('Freitextsuche filtert Spielplan nach Mannschaft, Halle und Datum', async () => {
    mockAll()
    setup('/staffeln?tab=spielplan')
    const box = await screen.findByLabelText('Suche in Tabelle und Spielplan')
    await screen.findByText(/Verein C/)
    fireEvent.change(box, { target: { value: 'verein c' } })
    expect(screen.queryByText('29:25')).not.toBeInTheDocument()
    expect(screen.getByText(/Verein C/)).toBeInTheDocument()
    fireEvent.change(box, { target: { value: 'sporthalle 20.09.2026' } })
    expect(screen.getByText('29:25')).toBeInTheDocument()
    expect(screen.queryByText(/Verein C/)).not.toBeInTheDocument()
    fireEvent.change(box, { target: { value: 'gibtesnicht' } })
    expect(screen.getByText('Keine Begegnung passt zur Suche.')).toBeInTheDocument()
  })

  test('Freitextsuche filtert die Tabelle nach Mannschaft', async () => {
    mockAll()
    setup()
    const box = await screen.findByLabelText('Suche in Tabelle und Spielplan')
    await screen.findByText('Verein A')
    fireEvent.change(box, { target: { value: 'verein b' } })
    expect(screen.queryByText('Verein A')).not.toBeInTheDocument()
    expect(screen.getByText('Verein B')).toBeInTheDocument()
  })

  test('Ranglisten sortieren nach Toren und weisen Spiele aus', async () => {
    mockAll()
    setup('/staffeln?tab=ranglisten')
    await waitFor(() => expect(screen.getByText('Torschützen')).toBeInTheDocument())
    expect(screen.getByText('12 Tore in 2 Spielen')).toBeInTheDocument()
  })

  // Die Siebenmeter-Rangliste ist eigenständig: nach Treffern sortiert, mit
  // Fehlversuchen, und ohne Spieler, die nie geworfen haben.
  test('Siebenmeter-Rangliste zeigt Fehlversuche und lässt Spieler ohne Versuch weg', async () => {
    mockAll()
    setup('/staffeln?tab=ranglisten')
    await waitFor(() => expect(screen.getByText('Siebenmeter')).toBeInTheDocument())
    expect(screen.getByText(/3\/4 · 1 daneben · 75 %/)).toBeInTheDocument()
    // Beta Spieler hat keinen Versuch und steht deshalb nur in der
    // Torschützenliste, nicht in der Siebenmeter-Liste.
    expect(screen.getAllByText('Beta Spieler')).toHaveLength(2)
  })

  // Der gewählte Reiter steht in der Adresse: derselbe Aufruf zeigt dieselbe
  // Darstellung.
  test.each([
    ['kreuztabelle', /Zeile = Heimmannschaft/],
    ['verlauf', 'Verein A'],
    ['tore', /Torverhältnis/],
    ['fairplay', /Fair-Play/],
    ['verteilung', /Torverteilung/],
    ['schiedsrichter', /Strafen des Spiels/],
  ])('der Reiter %s wird aus der Adresse gewählt', async (tab, marker) => {
    mockAll()
    setup(`/staffeln?tab=${tab}`)
    await waitFor(() => expect(screen.getAllByText(marker).length).toBeGreaterThan(0))
  })

  // Der Verlauf ist eine Grafik: er wird über sein Label gefunden, nicht über
  // Text.
  test('der Reiter verlauf zeigt die Grafik', async () => {
    mockAll()
    setup('/staffeln?tab=verlauf')
    await waitFor(() => expect(screen.getByRole('img', { name: /Platzierungsverlauf/ })).toBeInTheDocument())
  })

  describe('Kreuztabelle', () => {
    test('zeigt Endstand, Datum und eine leere Diagonale', async () => {
      mockAll()
      setup('/staffeln?tab=kreuztabelle')
      await waitFor(() => expect(screen.getByText('29:25')).toBeInTheDocument())
      // Noch nicht gespielt: die Zelle trägt das angesetzte Datum.
      expect(screen.getByText('04.10.')).toBeInTheDocument()
      // Die Diagonale ist leer und für Hilfsmittel ausgeblendet.
      const hidden = document.querySelectorAll('td[aria-hidden="true"]')
      expect(hidden).toHaveLength(2)
    })

    // In einer Matrix ist die eigene Mannschaft auf beiden Achsen vertreten;
    // nur eine zu markieren versteckte ihre Auswärtsspiele.
    test('markiert Zeile und Spalte der eigenen Mannschaft', async () => {
      mockAll()
      setup('/staffeln?tab=kreuztabelle')
      await waitFor(() => expect(screen.getByText('29:25')).toBeInTheDocument())
      const marked = document.querySelectorAll('[aria-current="true"]')
      // Ein Spaltenkopf, eine Zeile — beide Achsen sind ausgezeichnet.
      expect(marked.length).toBeGreaterThanOrEqual(2)
    })
  })

  describe('Tabellenverlauf', () => {
    test('zeichnet eine Linie je Mannschaft mit Rang 1 oben', async () => {
      mockAll()
      setup('/staffeln?tab=verlauf')
      // Auf data-team eingegrenzt: die Reiter-Icons von lucide bringen eigene
      // <polyline>-Elemente mit.
      await waitFor(() => expect(document.querySelectorAll('polyline[data-team]')).toHaveLength(2))
      const lines = Array.from(document.querySelectorAll('polyline[data-team]'))
      const yOf = (el: Element) => Number(el.getAttribute('points')!.split(',')[1].split(' ')[0])
      const rang1 = lines.find((l) => l.getAttribute('data-team') === 'Verein A')!
      const rang2 = lines.find((l) => l.getAttribute('data-team') === 'Verein B')!
      expect(yOf(rang1)).toBeLessThan(yOf(rang2))
    })

    test('die eigene Linie ist doppelt so stark', async () => {
      mockAll()
      setup('/staffeln?tab=verlauf')
      await waitFor(() => expect(document.querySelectorAll('polyline[data-team]')).toHaveLength(2))
      const own = document.querySelector('polyline[data-team][data-own]')!
      const other = document.querySelector('polyline[data-team]:not([data-own])')!
      expect(Number(own.getAttribute('stroke-width'))).toBeGreaterThan(
        Number(other.getAttribute('stroke-width')),
      )
    })
  })

  describe('Mannschafts-Statistiken', () => {
    test('Tore sortieren nach Differenz und weisen die Spielzahl aus', async () => {
      mockAll()
      setup('/staffeln?tab=tore')
      await waitFor(() => expect(screen.getByText('Torverhältnis')).toBeInTheDocument())
      expect(screen.getByText('+14')).toBeInTheDocument()
      expect(screen.getByText('Bester Angriff')).toBeInTheDocument()
      expect(screen.getByText('Beste Verteidigung')).toBeInTheDocument()
    })

    // Eine Mannschaft ohne Bericht ist nicht straffrei, sondern unbekannt —
    // sie darf nicht mit 0 an die Spitze der aufsteigenden Wertung.
    test('Fair-Play weist die Gewichtung aus und lässt Mannschaften ohne Bericht leer', async () => {
      mockAll()
      setup('/staffeln?tab=fairplay')
      await waitFor(() => expect(screen.getByText(/Gewichtung: Gelb 1, 2 min 2, Rot 3, Blau 4/)).toBeInTheDocument())
      expect(screen.getByText(/Ohne Wertung.*Verein B/)).toBeInTheDocument()
    })

    test('Verteilung zeigt Gini mit Erläuterung', async () => {
      mockAll()
      setup('/staffeln?tab=verteilung')
      await waitFor(() => expect(screen.getByText('Torverteilung')).toBeInTheDocument())
      expect(screen.getByText(/Gini-Wert misst die Ungleichverteilung/)).toBeInTheDocument()
      expect(screen.getByText('0,31')).toBeInTheDocument()
    })
  })

  describe('Schiedsrichter', () => {
    test('kennzeichnet eine unsichere Trennung und nennt den Bezug der Strafen', async () => {
      mockAll()
      setup('/staffeln?tab=schiedsrichter')
      await waitFor(() => expect(screen.getByText('Max Mustermann')).toBeInTheDocument())
      expect(screen.getByText(/keine Bewertung der Person/)).toBeInTheDocument()
      expect(screen.getAllByText('Trennung unsicher')).toHaveLength(1)
    })
  })

  describe('Hervorhebung der eigenen Zugehörigkeit', () => {
    test.each(['tabelle', 'spielplan', 'tore', 'fairplay', 'verteilung', 'ranglisten'])(
      'markiert im Reiter %s genau die eigene Zeile', async (tab) => {
        mockAll()
        setup(`/staffeln?tab=${tab}`)
        await waitFor(() =>
          expect(
            Array.from(document.querySelectorAll('[aria-current="true"]'))
              .filter((el) => el.tagName !== 'BUTTON').length,
          ).toBeGreaterThan(0))
        // Der gewählte Reiter trägt selbst aria-current; alles darüber hinaus
        // ist die Hervorhebung der eigenen Zeile.
        const marked = Array.from(document.querySelectorAll('[aria-current="true"]'))
          .filter((el) => el.tagName !== 'BUTTON')
        expect(marked.length).toBeGreaterThan(0)
        marked.forEach((el) => expect(el.className).toContain('bg-brand-table-select'))
      },
    )

    // Die Auszeichnung muss von einer ohnehin fett gesetzten Wertspalte
    // unterscheidbar bleiben — deshalb trägt die Zeile zusätzlich die
    // Zeilenmarkierung, nicht nur den Schriftschnitt.
    test('die markierte Zeile trägt Schriftschnitt UND Zeilenmarkierung', async () => {
      mockAll()
      setup('/staffeln?tab=ranglisten')
      await waitFor(() =>
        expect(
          Array.from(document.querySelectorAll('[aria-current="true"]'))
            .filter((el) => el.tagName !== 'BUTTON').length,
        ).toBeGreaterThan(0))
      const marked = Array.from(document.querySelectorAll('[aria-current="true"]'))
        .filter((el) => el.tagName !== 'BUTTON')
      marked.forEach((el) => {
        expect(el.className).toContain('font-semibold')
        expect(el.className).toContain('bg-brand-table-select')
      })
    })

    test('ohne Zugehörigkeit ist keine Zeile markiert', async () => {
      mockAll({ teamNames: [], playerIds: [] })
      setup('/staffeln?tab=tabelle')
      await waitFor(() => expect(screen.getByText('Verein A')).toBeInTheDocument())
      const marked = Array.from(document.querySelectorAll('[aria-current="true"]'))
        .filter((el) => el.tagName !== 'BUTTON')
      expect(marked).toHaveLength(0)
    })
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
    mock.onGet(/\/staffeln\/\d+\/tabelle$/).reply(200, table)
    mock.onGet(/\/staffeln\/\d+\/spielplan/).reply(200, games)
    mock.onGet(/\/staffeln\/\d+\/ranglisten/).reply(200, stats)
    mockStats()

    setup()
    await waitFor(() => expect(screen.getByText(/noch nichts beim Verband abgerufen/)).toBeInTheDocument())

    liveHandlers.forEach(fn => fn('bwhv-updated'))

    // Nach dem Nachladen trägt die Staffel eine ID und die Tabelle erscheint.
    await waitFor(() => expect(screen.getByText('Verein A')).toBeInTheDocument())
    expect(screen.queryByText(/noch nichts beim Verband abgerufen/)).not.toBeInTheDocument()
  })
})
