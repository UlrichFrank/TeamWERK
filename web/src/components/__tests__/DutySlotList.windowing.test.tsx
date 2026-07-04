import { describe, test, expect, vi, beforeAll } from 'vitest'
import { render, screen, act, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DutySlotList, { type BoardSlot } from '../DutySlotList'

vi.mock('../../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, name: 'Alice', role: 'standard' } }),
}))
vi.mock('../../lib/api', () => ({ api: { post: vi.fn(), delete: vi.fn() } }))

// Fenster-Scroll-Modus: die Position des Listen-Wrappers relativ zum Viewport
// steuert das Windowing. Wir simulieren einen 300px-Viewport und einen
// Wrapper, dessen Oberkante wir über getBoundingClientRect steuern.
const VIEWPORT = 300
const ROW_HEIGHT = 52
const topBox = { value: 0 }

beforeAll(() => {
  Object.defineProperty(window, 'innerHeight', { configurable: true, value: VIEWPORT })
  const orig = HTMLElement.prototype.getBoundingClientRect
  HTMLElement.prototype.getBoundingClientRect = function () {
    // Nur der DutySlotList-Wrapper (erstes DIV mit einer <table> darin) bekommt
    // eine simulierte Position; alles andere fällt auf das Original zurück.
    if (this.querySelector('table')) {
      return { top: topBox.value, left: 0, right: 0, bottom: 0, width: 0, height: 0, x: 0, y: topBox.value, toJSON: () => {} } as DOMRect
    }
    return orig.call(this)
  }
})

function makeSlots(n: number): BoardSlot[] {
  return Array.from({ length: n }, (_, i) => ({
    id: i,
    duty_type: `Dienst ${i}`,
    duty_type_id: i,
    has_instruction: false,
    event_time: '10:00',
    slots_total: 1,
    vacancies: 0,
    claimed_by_me: false,
    assignees: [],
  }))
}

describe('DutySlotList — Windowing langer Slot-Listen', () => {
  test('rendert bei vielen Slots nur die sichtbaren (+ Puffer)', () => {
    topBox.value = 0
    render(
      <MemoryRouter>
        <DutySlotList slots={makeSlots(300)} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )

    // Erste Slots im DOM, weit hinten liegende nicht.
    expect(screen.getByText('Dienst 0')).toBeInTheDocument()
    expect(screen.queryByText('Dienst 250')).toBeNull()

    // Nach unten scrollen (Wrapper-Oberkante -13000px = ~250 Zeilen).
    const wrapper = document.querySelector('[data-windowed-scroll]') as HTMLElement | null
    // DutySlotList nutzt Fenster-Scroll, kein data-windowed-scroll-Attribut → über window scrollen.
    expect(wrapper).toBeNull()
    act(() => {
      topBox.value = -250 * ROW_HEIGHT
      fireEvent.scroll(window)
    })

    expect(screen.getByText('Dienst 250')).toBeInTheDocument()
    expect(screen.queryByText('Dienst 0')).toBeNull()
  })

  test('kurze Slot-Listen werden vollständig gerendert (kein Windowing)', () => {
    topBox.value = 0
    render(
      <MemoryRouter>
        <DutySlotList slots={makeSlots(3)} isPast={false} canEdit={false} onReload={() => {}} />
      </MemoryRouter>,
    )
    expect(screen.getByText('Dienst 0')).toBeInTheDocument()
    expect(screen.getByText('Dienst 1')).toBeInTheDocument()
    expect(screen.getByText('Dienst 2')).toBeInTheDocument()
  })
})
