import { describe, test, expect, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Spielmatrix from './Spielmatrix'
import type { MatrixCell, MatrixGame, TeamMatrix } from '../../lib/staffeln'

const leer: MatrixCell = { goals: 0, sevenMAttempts: 0, sevenMGoals: 0, twoMin: 0, warnings: 0, disq: 0 }
const zelle = (o: Partial<MatrixCell> = {}): MatrixCell => ({ ...leer, ...o })

function game(o: Partial<MatrixGame> = {}): MatrixGame {
  return {
    bwhvGameId: 1, date: '2026-09-20', homeTeam: 'Team Stuttgart 2', guestTeam: 'Fremd',
    isHome: true, homeGoals: 29, guestGoals: 25, hasReport: true, ...o,
  }
}

// Zwei Begegnungen: die erste mit Bericht, die zweite ohne. Anna stand nur in
// der ersten Mannschaftsliste und hat dort nicht getroffen.
function matrix(): TeamMatrix {
  return {
    team: 'Team Stuttgart 2',
    halfDurationMinutes: 25,
    games: [game(), game({ bwhvGameId: 2, date: '2026-09-27', hasReport: false })],
    players: [
      {
        playerId: 7, memberId: null, name: 'Anna Beispiel',
        cells: [zelle({ goals: 0, twoMin: 1 }), null],
        total: zelle({ goals: 0, twoMin: 1 }), games: 1,
      },
    ],
    gameTotals: [zelle({ goals: 29 }), leer],
    total: zelle({ goals: 29 }),
    reportGames: 1,
  }
}

function annaZellen() {
  const zeile = screen.getByRole('row', { name: /Anna Beispiel/ })
  return within(zeile).getAllByRole('cell')
}

describe('Spielmatrix', () => {
  test('unterscheidet nicht im Kader von null Toren', () => {
    render(<Spielmatrix matrix={matrix()} ownPlayers={[]} onOpenGame={() => {}} />)
    const zellen = annaZellen()
    // [0] ist der Name, [1..6] die erste Begegnung, [7..12] die zweite.
    expect(zellen[1]).toHaveTextContent('0')       // dabei, kein Tor
    expect(zellen[7]).toHaveTextContent('–')       // nicht in der Mannschaftsliste
  })

  test('Klick auf die Begegnung springt zu ihr', async () => {
    const onOpenGame = vi.fn()
    render(<Spielmatrix matrix={matrix()} ownPlayers={[]} onOpenGame={onOpenGame} />)
    const [mitBericht, ohneBericht] = screen.getAllByRole('button')
    expect(ohneBericht).toHaveTextContent('kein Bericht')
    await userEvent.click(mitBericht)
    expect(onOpenGame).toHaveBeenCalledWith(1)
    await userEvent.click(ohneBericht)
    expect(onOpenGame).toHaveBeenLastCalledWith(2)
  })

  test('nennt Spiele mit und ohne Bericht getrennt', () => {
    render(<Spielmatrix matrix={matrix()} ownPlayers={[]} onOpenGame={() => {}} />)
    expect(screen.getByText(/2 Spiele, davon\s+1 mit Spielbericht/)).toBeInTheDocument()
  })

  test('hebt die eigene Spielerzeile hervor', () => {
    render(<Spielmatrix matrix={matrix()} ownPlayers={[7]} onOpenGame={() => {}} />)
    expect(screen.getByRole('row', { name: /Anna Beispiel/ })).toHaveAttribute('aria-current', 'true')
  })

  test('ohne gespielte Begegnung steht ein Hinweis statt einer leeren Tabelle', () => {
    const m = { ...matrix(), games: [], players: [], gameTotals: [], reportGames: 0 }
    render(<Spielmatrix matrix={m} ownPlayers={[]} onOpenGame={() => {}} />)
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByText(/noch keine Begegnung gespielt/)).toBeInTheDocument()
  })
})
