import { describe, test, expect } from 'vitest'
import { render } from '@testing-library/react'
import TorMomentum from './TorMomentum'
import type { EventLine } from '../../lib/staffeln'

let seq = 0
function ev(kind: EventLine['kind'], side: EventLine['side'], sec: number, scorer: string): EventLine {
  seq += 1
  return {
    seq, clockTime: '', gameSecond: sec, scoreHome: null, scoreGuest: null,
    kind, side, playerId: null, playerName: scorer, number: null, rawText: '',
  }
}

// Zahlen aus der Fixture spielbericht_905272: Halbzeitstand bei 21:57,
// zweite Halbzeit ab 25:27.
const EVENTS = [
  ev('goal', 'home', 99, 'Anna'),
  ev('goal', 'guest', 1317, 'Bea'),
  ev('seven_m_goal', 'home', 1527, 'Cem'),
  ev('goal', 'home', 2946, 'Anna'),
]

function momentum(props: Partial<React.ComponentProps<typeof TorMomentum>> = {}) {
  return render(
    <TorMomentum
      events={EVENTS}
      homeTeam="Team Stuttgart 2"
      guestTeam="Fremd"
      homeGoalsHt={1}
      guestGoalsHt={1}
      halfDurationMinutes={25}
      {...props}
    />,
  )
}

describe('TorMomentum', () => {
  test('jeder Kreis traegt eine Beschriftung', () => {
    const { container } = momentum()
    const kreise = container.querySelectorAll('circle')
    expect(kreise).toHaveLength(4)
    kreise.forEach((c) => {
      expect(c.querySelector('title')?.textContent).toBeTruthy()
    })
  })

  test('die Beschriftung nennt Minute, Spielstand und Schuetze', () => {
    const { container } = momentum()
    const erste = container.querySelector('circle title')?.textContent ?? ''
    expect(erste).toContain('1:39')
    expect(erste).toContain('1:0')
    expect(erste).toContain('Anna')
  })

  test('kennzeichnet den Siebenmeter in der Beschriftung', () => {
    const { container } = momentum()
    const titel = [...container.querySelectorAll('circle title')].map((t) => t.textContent ?? '')
    expect(titel.filter((t) => t.includes('Siebenmeter'))).toHaveLength(1)
  })

  test('zwei Halbzeiten bei vorliegendem Halbzeitstand', () => {
    const { container } = momentum()
    expect(container.querySelectorAll('svg')).toHaveLength(2)
  })

  test('ohne Halbzeitstand bleibt eine Achse', () => {
    const { container } = momentum({ homeGoalsHt: null, guestGoalsHt: null })
    expect(container.querySelectorAll('svg')).toHaveLength(1)
  })

  test('ohne Tore erscheint ein Hinweis statt einer leeren Grafik', () => {
    const { container, getByText } = momentum({ events: [] })
    expect(container.querySelector('svg')).toBeNull()
    expect(getByText(/keine Tore im Spielverlauf/)).toBeInTheDocument()
  })
})
