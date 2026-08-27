import { describe, test, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DutySlotList, { type BoardSlot } from './DutySlotList'

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 1, name: 'Alice', role: 'standard' } }),
}))
vi.mock('../lib/api', () => ({ api: { post: vi.fn(), delete: vi.fn() } }))

function baseSlot(overrides: Partial<BoardSlot> = {}): BoardSlot {
  return {
    id: 100,
    duty_type: 'Kasse',
    duty_type_id: 42,
    has_instruction: false,
    event_time: '10:00',
    hours_value: 1,
    slots_total: 2,
    vacancies: 1,
    claimed_by_me: false,
    assignees: [],
    ...overrides,
  }
}

describe('DutySlotList — Anleitung link', () => {
  test('renders link to instruction page when has_instruction is true', () => {
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ has_instruction: true })]}
          isPast={false}
          canEdit={false}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    const link = screen.getByRole('link', { name: 'Anleitung ansehen' })
    expect(link.getAttribute('href')).toBe('/dienste/anleitung/42')
    // Zeilen-id ist der Anker für den Fokus-Scroll (siehe DutyPage.focus.test.tsx).
    expect(document.getElementById('duty-slot-100')).not.toBeNull()
  })

  test('renders a strikethrough icon button when has_instruction is false, click opens info modal', () => {
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ has_instruction: false })]}
          isPast={false}
          canEdit={false}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    // No link — instead a button with an explanatory aria-label.
    expect(screen.queryByRole('link', { name: 'Anleitung ansehen' })).toBeNull()
    const btn = screen.getByRole('button', { name: 'Keine Anleitung vorhanden' })
    expect(btn).not.toBeNull()

    // Click opens an info modal.
    fireEvent.click(btn)
    expect(screen.getByText(/noch keine Anleitung/)).toBeTruthy()
  })

  // Teil von openspec/changes/zurueck-position-wiederherstellen: der Klick auf
  // die Anleitung muss den Fokus-Marker VOR der Navigation setzen, damit der
  // /dienste-History-Eintrag ihn schon trägt (relevant für „Zurück").
  test('Klick auf Anleitung ruft onFocusSlot vor der Navigation auf', () => {
    const onFocusSlot = vi.fn()
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ id: 777, has_instruction: true, duty_type_id: 42 })]}
          isPast={false}
          canEdit={false}
          onReload={() => {}}
          onFocusSlot={onFocusSlot}
        />
      </MemoryRouter>,
    )
    const link = screen.getByRole('link', { name: 'Anleitung ansehen' })
    fireEvent.click(link)
    expect(onFocusSlot).toHaveBeenCalledWith(777)
    // Link-Ziel bleibt unverändert — onFocusSlot ergänzt die Navigation, ersetzt sie nicht.
    expect(link.getAttribute('href')).toBe('/dienste/anleitung/42')
  })

  test('ohne onFocusSlot-Prop funktioniert der Link weiterhin unverändert (z. B. SpieltagDetailModal)', () => {
    render(
      <MemoryRouter>
        <DutySlotList
          slots={[baseSlot({ has_instruction: true })]}
          isPast={false}
          canEdit={false}
          onReload={() => {}}
        />
      </MemoryRouter>,
    )
    const link = screen.getByRole('link', { name: 'Anleitung ansehen' })
    expect(() => fireEvent.click(link)).not.toThrow()
  })
})
