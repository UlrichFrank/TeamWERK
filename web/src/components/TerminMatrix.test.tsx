import { describe, test, expect, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import TerminMatrix from './TerminMatrix'
import type { RsvpMatrix } from '../lib/terminMatrix'

// Change aufstellung-status-termine: Spielzellen tragen den Aufstellungsstatus
// als Fläche hinter dem Rückmeldesymbol; Trainings bleiben ohne Fläche.

afterEach(cleanup)

const ev = (kind: 'training' | 'game', id: number, date: string, event_type: 'training' | 'heim') =>
  ({ kind, id, date, time: '18:00', event_type, title: '', cancelled: false, rsvp_require_reason: false })

function matrix(): RsvpMatrix {
  return {
    team_id: 1,
    team_name: 'mB1',
    events: [ev('training', 1, '2099-05-01', 'training'), ev('game', 2, '2099-05-02', 'heim'), ev('game', 3, '2099-05-09', 'heim')],
    members: [
      {
        member_id: 10, name: 'Luca Brenner', extended: false, is_self: false, can_respond: false,
        cells: [
          { status: 'confirmed', is_default: false },
          { status: 'confirmed', is_default: false, lineup: 'in' },
          { status: 'confirmed', is_default: false, lineup: 'open' },
        ],
      },
      {
        member_id: 11, name: 'Paul Hahn', extended: false, is_self: false, can_respond: false,
        cells: [
          { status: null, is_default: false },
          { status: null, is_default: false, lineup: 'out' },
          { status: null, is_default: false, lineup: 'open' },
        ],
      },
    ],
  }
}

function renderMatrix() {
  render(
    <MemoryRouter>
      <TerminMatrix matrix={matrix()} columns={[0, 1, 2]} today="2099-01-01" />
    </MemoryRouter>,
  )
}

function cellBox(title: string) {
  const td = screen.getAllByTitle(title)[0]
  return td.querySelector('[data-lineup], span') as HTMLElement
}

describe('TerminMatrix — Aufstellung', () => {
  test('aufgestellt: nur grüner Rahmen (2 px), keine Fläche, Symbol behält seine Farbe', () => {
    renderMatrix()
    const box = cellBox('zugesagt · aufgestellt')
    expect(box.dataset.lineup).toBe('in')
    expect(box.className).toContain('border-2')
    expect(box.className).toContain('border-brand-green')
    expect(box.className).not.toContain('bg-brand-green')
    expect(box.querySelector('svg')!.getAttribute('class')).toContain('text-brand-green')
  })

  test('nicht aufgestellt: gestrichelt mit Diagonale, keine graue Fläche', () => {
    renderMatrix()
    const box = cellBox('keine Rückmeldung · nicht aufgestellt')
    expect(box.className).toContain('border-2')
    expect(box.className).toContain('border-dashed')
    expect(box.className).toContain('linear-gradient(to_bottom_right')
    expect(box.className).not.toContain('bg-brand-border')
  })

  test('offen: nur gestrichelter Rahmen, ohne Diagonale', () => {
    renderMatrix()
    const box = cellBox('zugesagt · Aufstellung offen')
    expect(box.className).toContain('border-dashed')
    expect(box.className).toContain('bg-transparent')
    expect(box.className).not.toContain('linear-gradient')
  })

  test('Trainingszelle ohne Aufstellungsfläche', () => {
    renderMatrix()
    const td = screen.getAllByTitle('zugesagt')[0]
    const box = td.querySelector('span') as HTMLElement
    expect(box.dataset.lineup).toBeUndefined()
    expect(box.className).toContain('border-transparent')
  })

  test('Legende nennt alle drei Zustände', () => {
    renderMatrix()
    const legend = screen.getByRole('list', { name: 'Legende' })
    for (const label of ['aufgestellt', 'nicht aufgestellt', 'Aufstellung offen']) {
      expect(legend.textContent).toContain(label)
    }
  })
})

// Change termin-matrix-kopfzeile-fixiert: die Titelzeile bleibt beim
// vertikalen Scrollen stehen. jsdom rechnet kein Sticky-Layout — geprüft werden
// die Invarianten, ohne die es im Browser nicht greift.
describe('TerminMatrix — fixierte Titelzeile', () => {
  test('Tabelle scrollt in einem eigenen, höhenbegrenzten Bereich', () => {
    renderMatrix()
    const wrapper = screen.getByRole('table').parentElement as HTMLElement
    // overflow-x-auto allein wäre Scroll-Container ohne senkrechten Scroll:
    // sticky top-0 hätte dann keinen Bezug.
    expect(wrapper.className).toMatch(/(^|\s)overflow-auto(\s|$)/)
    expect(wrapper.className).toMatch(/(^|\s)max-h-/)
  })

  test('alle Kopfzellen sind oben fixiert, die Ecke „Spieler" liegt zuoberst', () => {
    renderMatrix()
    const heads = screen.getByRole('table').querySelectorAll('thead th')
    expect(heads.length).toBeGreaterThan(0)
    for (const th of heads) {
      expect(th.className).toMatch(/(^|\s)sticky(\s|$)/)
      expect(th.className).toMatch(/(^|\s)top-0(\s|$)/)
    }
    const corner = screen.getByRole('columnheader', { name: 'Spieler' })
    expect(corner.className).toMatch(/(^|\s)left-0(\s|$)/)
    expect(corner.className).toMatch(/(^|\s)z-30(\s|$)/)
    for (const th of heads) {
      if (th !== corner) expect(th.className).toMatch(/(^|\s)z-20(\s|$)/)
    }
  })

  test('Zeilenköpfe bleiben links fixiert unter der Titelzeile', () => {
    renderMatrix()
    const rowHead = screen.getByRole('rowheader', { name: /Luca Brenner/ })
    expect(rowHead.className).toMatch(/(^|\s)sticky(\s|$)/)
    expect(rowHead.className).toMatch(/(^|\s)left-0(\s|$)/)
    expect(rowHead.className).toMatch(/(^|\s)z-10(\s|$)/)
  })
})
